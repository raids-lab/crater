/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VcJobReconciler reconciles a AIJob object
type BuildKitReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	log           logr.Logger
	imageRegistry imageregistry.ImageRegistryInterface
}

// NewVcJobReconciler returns a new reconcile.Reconciler
func NewBuildKitReconciler(crClient client.Client, scheme *runtime.Scheme,
	imageRegistry imageregistry.ImageRegistryInterface) *BuildKitReconciler {
	return &BuildKitReconciler{
		Client:        crClient,
		Scheme:        scheme,
		log:           ctrl.Log.WithName("BuildKit-reconciler"),
		imageRegistry: imageRegistry,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildKitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Owns(&v1.Pod{}).
		WithOptions(controller.Options{}).
		Complete(r)
}

var (
	ImageSpace = config.GetConfig().Workspace.ImageNamespace
)

//nolint:lll // kubebuilder rbac declares
//+kubebuilder:rbac:groups=hsy.hsy.crd;"",resources=imagepacks;pods;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hsy.hsy.crd,resources=imagepacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hsy.hsy.crd,resources=imagepacks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AIJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile

// Reconcile 主要用于同步 ImagePack 的状态到数据库中
func (r *BuildKitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k := query.Kaniko

	// TODO(user): your logic here
	if req.Namespace != ImageSpace {
		return ctrl.Result{}, nil
	}

	var job batchv1.Job

	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if k8serrors.IsNotFound(err) {
			// if job record in db is not finished, update the status to canceled
			return r.handleJobNotFound(ctx, req.Name)
		} else {
			logger.Error(err, "unable to fetch job")
			return ctrl.Result{Requeue: true}, err
		}
	}

	// create or update db record
	// if kaniko not found, return
	kaniko, err := k.WithContext(ctx).Preload(query.Kaniko.User).Where(k.ImagePackName.Eq(job.Name)).First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "unable to fetch kaniko entity in database")
		return ctrl.Result{Requeue: true}, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		userID, _ := strconv.Atoi(job.Annotations["buildkit-data/UserID"])
		dockerfile := job.Annotations["buildkit-data/Dockerfile"]
		description := job.Annotations["buildkit-data/Description"]
		kanikoEntity := &model.Kaniko{
			//nolint:gosec // userID is very small, integer overflow conversion won't happen
			UserID:        uint(userID),
			ImagePackName: job.Name,
			ImageLink:     job.Annotations["buildkit-data/ImageLink"],
			NameSpace:     job.Namespace,
			Status:        model.BuildJobInitial,
			Dockerfile:    &dockerfile,
			Description:   &description,
			BuildSource:   model.BuildKit,
		}
		if err = k.WithContext(ctx).Create(kanikoEntity); err != nil {
			logutils.Log.Errorf("create imagepack entity failed, params: %+v", kanikoEntity)
		}
		logger.Info("successfully created kaniko entity")
		return ctrl.Result{Requeue: true}, nil
	}

	jobStatus := r.getJobBuildStatus(&job)

	// if kaniko found, update the status and decide whether to create image
	if _, err = k.WithContext(ctx).
		Where(k.ImagePackName.Eq(job.Name)).
		Update(k.Status, jobStatus); err != nil {
		logger.Error(err, "kaniko entity status updated failed")
		return ctrl.Result{Requeue: true}, nil
	}
	logger.Info(fmt.Sprintf("buildkit pod: %s , new stage: %s", job.Name, jobStatus))

	if jobStatus == model.BuildJobFinished {
		size, err := r.imageRegistry.GetImageSize(ctx, kaniko.ImageLink)
		if err != nil {
			logger.Error(err, "get image size failed")
		}
		if _, err = k.WithContext(ctx).
			Where(k.ImagePackName.Eq(job.Name)).
			Update(k.Size, size); err != nil {
			logger.Error(err, "kaniko entity size updated failed")
			return ctrl.Result{Requeue: true}, err
		}
		im := query.Image
		// avoid duplicatedly creating image entity result from the same one finished kaniko crd
		if _, err = im.WithContext(ctx).
			Where(im.ImagePackName.Eq(kaniko.ImagePackName)).
			First(); errors.Is(err, gorm.ErrRecordNotFound) {
			image := &model.Image{
				User:          kaniko.User,
				UserID:        kaniko.User.ID,
				ImageLink:     kaniko.ImageLink,
				Description:   kaniko.Description,
				ImagePackName: &kaniko.ImagePackName,
				IsPublic:      false,
				TaskType:      model.JobTypeCustom,
				ImageSource:   model.ImageCreateType,
				Size:          size,
			}
			err = im.WithContext(ctx).Create(image)
			if err != nil {
				logger.Error(err, "image entity created failed")
				return ctrl.Result{Requeue: true}, err
			} else {
				logger.Info("kaniko entity created successfully: " + kaniko.ImagePackName)
			}
		} else if err == nil {
			logger.Error(err, "image entity already existed")
		}
	}
	return ctrl.Result{}, nil
}

func (r *BuildKitReconciler) handleJobNotFound(ctx context.Context, jobName string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k := query.Kaniko
	kaniko, err := k.WithContext(ctx).Where(k.ImagePackName.Eq(jobName)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("kaniko not found in database")
			return ctrl.Result{}, nil
		} else {
			logger.Error(err, "unable to fetch kaniko entity")
			return ctrl.Result{Requeue: true}, err
		}
	}

	// if kaniko status is finished or failed, do nothing
	if kaniko.Status == model.BuildJobFailed || kaniko.Status == model.BuildJobFinished {
		return ctrl.Result{}, nil
	}

	if _, err := k.WithContext(ctx).Where(k.ImagePackName.Eq(jobName)).UpdateColumn(k.Status, model.BuildJobCanceled); err != nil {
		logger.Error(err, "unable to update kaniko entity status")
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (r *BuildKitReconciler) getJobBuildStatus(job *batchv1.Job) model.BuildStatus {
	var status model.BuildStatus
	if job.Status.Succeeded == 1 {
		status = model.BuildJobFinished
	} else if job.Status.Failed == 1 {
		status = model.BuildJobFailed
	} else if job.Status.Active == 1 {
		status = model.BuildJobRunning
	} else {
		status = model.BuildJobPending
	}
	return status
}
