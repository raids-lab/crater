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

	"github.com/go-logr/logr"
	harbormodelv2 "github.com/mittwald/goharbor-client/v5/apiv2/model"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	imagepackv1 "github.com/raids-lab/crater/pkg/apis/imagepack/v1"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/logutils"
	"gorm.io/gen"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// VcJobReconciler reconciles a AIJob object
type ImagePackReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	log          logr.Logger
	harborClient *crclient.HarborClient
}

// NewVcJobReconciler returns a new reconcile.Reconciler
func NewImagePackReconciler(crClient client.Client, scheme *runtime.Scheme) *ImagePackReconciler {
	harborClient := crclient.NewHarborClient()
	return &ImagePackReconciler{
		Client:       crClient,
		Scheme:       scheme,
		log:          ctrl.Log.WithName("imagepack-reconciler"),
		harborClient: &harborClient,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImagePackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&imagepackv1.ImagePack{}).
		Owns(&v1.Pod{}).
		WithOptions(controller.Options{}).
		Complete(r)
}

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
//
//nolint:gocyclo // refactor later
func (r *ImagePackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	kanikoQuery := query.Kaniko

	// TODO(user): your logic here
	var imagepack imagepackv1.ImagePack
	err := r.Get(ctx, req.NamespacedName, &imagepack)

	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error(err, "unable to fetch ImagePack")
		return ctrl.Result{}, nil
	}

	if k8serrors.IsNotFound(err) {
		// set job status to deleted
		var kaniko *model.Kaniko
		kaniko, err = kanikoQuery.WithContext(ctx).Where(kanikoQuery.ImagePackName.Eq(req.Name)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error(err, "unable to fetch kaniko entity")
			return ctrl.Result{Requeue: true}, err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("kaniko not found in database")
			return ctrl.Result{}, nil
		}

		if kaniko.Status == imagepackv1.PackJobFailed || kaniko.Status == imagepackv1.PackJobFinished {
			return ctrl.Result{}, nil
		}

		var info gen.ResultInfo
		info, err = kanikoQuery.WithContext(ctx).Where(kanikoQuery.ImagePackName.Eq(req.Name)).Delete()
		if err != nil {
			logger.Error(err, "unable to delete kaniko entity in database")
			return ctrl.Result{Requeue: true}, err
		}
		if info.RowsAffected == 0 {
			logger.Info("kaniko not found in database")
		}
		return ctrl.Result{}, nil
	}

	// create or update db record
	// if kaniko not found, return
	var kaniko *model.Kaniko
	kaniko, err = kanikoQuery.WithContext(ctx).Preload(query.Kaniko.User).Where(kanikoQuery.ImagePackName.Eq(imagepack.Name)).First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "unable to fetch kaniko entity in database")
		return ctrl.Result{Requeue: true}, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error(err, "kaniko entity not existed in database")
		return ctrl.Result{}, nil
	}

	// if kaniko found, update the status and decide whether to create image
	if _, err = kanikoQuery.WithContext(ctx).
		Where(kanikoQuery.ImagePackName.Eq(imagepack.Name)).
		Update(kanikoQuery.Status, imagepack.Status.Stage); err != nil {
		logger.Error(err, "kaniko entity status updated failed")
	}
	logger.Info(fmt.Sprintf("kakino pod: %s , new stage: %s", imagepack.Name, imagepack.Status.Stage))

	if imagepack.Status.Stage == imagepackv1.PackJobFinished {
		size := r.getKankikoSize(ctx, kaniko)
		if _, err = kanikoQuery.WithContext(ctx).
			Where(kanikoQuery.ImagePackName.Eq(imagepack.Name)).
			Update(kanikoQuery.Size, size); err != nil {
			logger.Error(err, "kaniko entity size updated failed")
		}
		imageQuery := query.Image
		// avoid duplicatedly creating image entity result from the same one finished kaniko crd
		if _, err = imageQuery.WithContext(ctx).
			Where(imageQuery.ImagePackName.Eq(kaniko.ImagePackName)).
			First(); errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("create new image entity with kaniko name: " + kaniko.ImagePackName)
			image := &model.Image{
				User:          kaniko.User,
				UserID:        kaniko.User.ID,
				ImageLink:     kaniko.ImageLink,
				Description:   kaniko.Description,
				ImagePackName: kaniko.ImagePackName,
				IsPublic:      false,
				TaskType:      model.JobTypeCustom,
				ImageSource:   model.ImageCreateType,
				Size:          size,
			}
			if err = imageQuery.WithContext(ctx).Create(image); err != nil {
				logger.Error(err, "image entity size update failed")
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *ImagePackReconciler) getKankikoSize(ctx context.Context, kaniko *model.Kaniko) int64 {
	var imageArtifact *harbormodelv2.Artifact
	var err error
	projectName := fmt.Sprintf("user-%s", kaniko.User.Name)
	name, tag := handler.GetImageNameAndTag(kaniko.ImageLink)
	if imageArtifact, err = r.harborClient.GetArtifact(ctx, projectName, name, tag); err != nil {
		logutils.Log.Errorf("get imagepack artifact failed! err:%+v", err)
		return 0
	}
	return imageArtifact.Size
}
