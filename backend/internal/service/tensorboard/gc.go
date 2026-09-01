package tensorboard

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/bizerr"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

const tensorboardGCInterval = 5 * time.Minute

// TensorboardGarbageCollector removes TensorBoard deployments after their TTL expires.
type TensorboardGarbageCollector struct {
	crClient  client.Client
	namespace string
	logger    logr.Logger
}

func NewTensorboardGarbageCollector(
	crClient client.Client,
	namespace string,
) *TensorboardGarbageCollector {
	return &TensorboardGarbageCollector{
		crClient:  crClient,
		namespace: namespace,
		logger:    ctrl.Log.WithName("tensorboard-garbage-collector"),
	}
}

// NeedLeaderElection ensures only the elected manager performs TTL cleanup.
func (gc *TensorboardGarbageCollector) NeedLeaderElection() bool {
	return true
}

// Start runs TensorBoard TTL cleanup under the controller manager lifecycle.
func (gc *TensorboardGarbageCollector) Start(ctx context.Context) error {
	if gc.crClient == nil {
		return bizerr.Internal.ServiceError.New("tensorboard garbage collector requires a Kubernetes client")
	}
	if gc.namespace == "" {
		return bizerr.Internal.ServiceError.New("tensorboard garbage collector requires a job namespace")
	}

	ticker := time.NewTicker(tensorboardGCInterval)
	defer ticker.Stop()

	gc.logger.Info("tensorboard garbage collector started", "interval", tensorboardGCInterval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			gc.cleanExpiredTensorboards(ctx)
		}
	}
}

func (gc *TensorboardGarbageCollector) cleanExpiredTensorboards(ctx context.Context) {
	var deployList appsv1.DeploymentList
	err := gc.crClient.List(ctx, &deployList,
		client.InNamespace(gc.namespace),
		client.MatchingLabels{
			crclient.LabelKeyTaskType: interutil.LabelKeyTypeTensorboard,
		},
	)
	if err != nil {
		gc.logger.Error(err, "failed to list TensorBoard deployments for TTL cleanup")
		return
	}

	now := time.Now()
	for i := range deployList.Items {
		deploy := &deployList.Items[i]
		expiration, ok := deploy.Annotations[interutil.AnnotationKeyExpirationTime]
		if !ok {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, expiration)
		if err != nil || !now.After(expiresAt) {
			continue
		}

		if err := gc.crClient.Delete(ctx, deploy); err != nil && !k8serrors.IsNotFound(err) {
			gc.logger.Error(err, "failed to delete expired TensorBoard deployment", "deployment", deploy.Name)
		}
	}
}
