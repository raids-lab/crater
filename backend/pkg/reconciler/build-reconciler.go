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
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/logutils"
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

	// 1. if job not found in k8s, maybe deleted by user or automatically deleted by k8s
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if k8serrors.IsNotFound(err) {
			return r.handleJobNotFound(ctx, req.Name)
		} else {
			logger.Error(err, "unable to fetch job, maybe k8s api server is down")
			return ctrl.Result{Requeue: true}, err
		}
	}

	// 2. fetch kaniko record from database
	kaniko, err := k.WithContext(ctx).Preload(query.Kaniko.User).Where(k.ImagePackName.Eq(job.Name)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 3. if kaniko record not found, then create a new one.
			return r.handleKanikoRecordNotFound(ctx, &job)
		}
		// 4. if fetch failed, it may be caused by database connection error.
		logger.Error(err, "unable to fetch kaniko record in database")
		return ctrl.Result{Requeue: true}, err
	}

	// 5. get the job status from k8s job
	jobStatus := r.getJobBuildStatus(ctx, &job)

	// 6. if buildkit job finished
	if jobStatus == model.BuildJobFinished && kaniko.Status != model.BuildJobFinished {
		var size int64
		size, err = r.imageRegistry.GetImageSize(ctx, kaniko.ImageLink)
		if err != nil {
			logger.Error(err, "get image size failed")
			return ctrl.Result{Requeue: true}, err
		}
		// 7. update kaniko record size
		if err = r.updateKanikoSize(ctx, kaniko, size); err != nil {
			logger.Error(err, "kaniko record size updated failed")
			return ctrl.Result{Requeue: true}, err
		}
		// 8. create image record
		if err = r.createImageRecord(ctx, kaniko); err != nil {
			logger.Error(err, "create image record failed")
			return ctrl.Result{Requeue: true}, err
		}
	}

	if err = r.updateKanikoStatus(ctx, kaniko, jobStatus); err != nil {
		logger.Error(err, "kaniko record status updated failed")
		return ctrl.Result{Requeue: true}, err
	}
	logger.Info(fmt.Sprintf("buildkit pod: %s , new stage: %s", job.Name, jobStatus))

	return ctrl.Result{}, nil
}

func (r *BuildKitReconciler) handleJobNotFound(ctx context.Context, jobName string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Job not found in k8s, maybe deleted by user or automatically deleted by k8s" + jobName)
	return ctrl.Result{}, nil
}

func (r *BuildKitReconciler) handleKanikoRecordNotFound(ctx context.Context, job *batchv1.Job) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	k := query.Kaniko
	userID, _ := strconv.Atoi(job.Annotations["build-data/UserID"])
	dockerfile := job.Annotations["build-data/Dockerfile"]
	description := job.Annotations["build-data/Description"]
	envd := job.Annotations["build-data/Envd"]
	var buildSource model.BuildSource
	if dockerfile != "" {
		buildSource = model.BuildKit
	} else if envd != "" {
		buildSource = model.Envd
	} else {
		buildSource = model.Snapshot
	}
	kanikorecord := model.Kaniko{
		//nolint:gosec // userID is very small, integer overflow conversion won't happen
		UserID:        uint(userID),
		ImagePackName: job.Name,
		ImageLink:     job.Annotations["build-data/ImageLink"],
		NameSpace:     job.Namespace,
		Status:        model.BuildJobInitial,
		Dockerfile:    &dockerfile,
		Description:   &description,
		BuildSource:   buildSource,
	}
	if err := k.WithContext(ctx).Create(&kanikorecord); err != nil {
		logutils.Log.Errorf("create imagepack record failed, params: %+v, %+v", kanikorecord, err)
	} else {
		logger.Info("successfully created kaniko record")
	}
	return ctrl.Result{}, nil
}

func (r *BuildKitReconciler) updateKanikoStatus(ctx context.Context, kaniko *model.Kaniko, status model.BuildStatus) error {
	k := query.Kaniko
	_, err := k.WithContext(ctx).
		Where(k.ImagePackName.Eq(kaniko.ImagePackName)).
		Update(k.Status, status)
	return err
}

func (r *BuildKitReconciler) updateKanikoSize(ctx context.Context, kaniko *model.Kaniko, size int64) error {
	k := query.Kaniko
	_, err := k.WithContext(ctx).
		Where(k.ImagePackName.Eq(kaniko.ImagePackName)).
		Update(k.Size, size)
	return err
}

func (r *BuildKitReconciler) createImageRecord(ctx context.Context, kaniko *model.Kaniko) error {
	logger := log.FromContext(ctx)
	im := query.Image

	// Check if the image record already exists
	_, err := im.WithContext(ctx).Where(im.ImagePackName.Eq(kaniko.ImagePackName)).First()
	if err == nil {
		logger.Error(err, "Image record already existed")
		// skip if image record already exists
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "encounter other error when querying image record")
		return err
	}

	// Create a new image record
	taskType := model.JobTypeCustom
	if kaniko.BuildSource == model.Snapshot {
		taskType = model.JobTypeJupyter
	}
	image := &model.Image{
		UserID:        kaniko.UserID,
		ImageLink:     kaniko.ImageLink,
		Description:   kaniko.Description,
		ImagePackName: &kaniko.ImagePackName,
		IsPublic:      false,
		TaskType:      taskType,
		ImageSource:   model.ImageCreateType,
		Size:          kaniko.Size,
	}
	err = im.WithContext(ctx).Create(image)
	if err != nil {
		logger.Error(err, "Image record creation failed")
		return err
	}

	logger.Info("Image record created successfully: " + kaniko.ImagePackName)
	return nil
}

func (r *BuildKitReconciler) getJobBuildStatus(ctx context.Context, job *batchv1.Job) model.BuildStatus {
	// Check if the job has succeeded or failed
	if job.Status.Succeeded == 1 {
		return model.BuildJobFinished
	} else if job.Status.Failed == 1 {
		return model.BuildJobFailed
	}

	// If job is active, check pod status to determine if it's running or pending
	if job.Status.Active > 0 {
		podList := &v1.PodList{}
		labelSelector := client.MatchingLabels{
			"job-name": job.Name,
		}
		err := r.List(ctx, podList, client.InNamespace(job.Namespace), labelSelector)
		if err != nil {
			r.log.Error(err, "Failed to list pods for job", "job", job.Name)
			return model.BuildJobPending
		}

		// If any pod is running, the job is running
		for i := range podList.Items {
			if podList.Items[i].Status.Phase == v1.PodRunning {
				return model.BuildJobRunning
			}
		}

		// All pods are pending, so job is pending
		return model.BuildJobPending
	}

	return model.BuildJobPending
}
