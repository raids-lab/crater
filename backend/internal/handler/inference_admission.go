package handler

import (
	"context"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

func (mgr *KthenaMgr) validateServingQuota(
	ctx context.Context,
	serving *unstructured.Unstructured,
	token util.JWTMessage,
) error {
	deployment, err := kthena.ReadDeployment(serving)
	if err != nil {
		return err
	}
	requested := deployment.TotalResources()
	resources := make(map[string]string, len(requested))
	for key, value := range requested {
		resources[string(key)] = value.String()
	}
	queue := vcqueue.ResolveJobQueueName(token)
	if mgr.quota != nil {
		request, err := mgr.quota.CheckRequestedResourceLimit(ctx, token.UserID, token.AccountID, queue, resources)
		if err != nil {
			return err
		}
		if request.Exceeded {
			return bizerr.BadRequest.ParameterError.New("deployment total resources exceed the user queue quota")
		}
	}
	// Volcano admits each serving group's PodGroup through the shared extender.
	// Resource contention must leave those groups Pending instead of hiding the deployment at zero replicas.
	return nil
}

func queuedServingReplicaCount(serving *unstructured.Unstructured) int64 {
	count, _ := strconv.ParseInt(serving.GetAnnotations()[kthena.QueuedReplicasAnnotation], 10, 64)
	return count
}
