package handler

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

func (mgr *KthenaMgr) shouldQueueServing(
	ctx context.Context,
	serving *unstructured.Unstructured,
	token util.JWTMessage,
) (bool, error) {
	deployment, err := kthena.ReadDeployment(serving)
	if err != nil {
		return false, err
	}
	requested := deployment.TotalResources()
	ctx = service.WithServingReplacementResources(ctx, requested)
	resources := make(map[string]string, len(requested))
	for key, value := range requested {
		resources[string(key)] = value.String()
	}
	queue := vcqueue.ResolveJobQueueName(token)
	if mgr.quota != nil {
		request, err := mgr.quota.CheckRequestedResourceLimit(ctx, token.UserID, token.AccountID, queue, resources)
		if err != nil {
			return false, err
		}
		if request.Exceeded {
			return false, bizerr.BadRequest.ParameterError.New("deployment total resources exceed the user queue quota")
		}
		usage, err := mgr.quota.CheckUserResourceLimit(ctx, token.UserID, token.AccountID, queue, resources)
		if err != nil {
			return false, err
		}
		if usage.Exceeded {
			return true, nil
		}
	}
	if mgr.prequeue != nil && mgr.configService != nil {
		cfg, err := mgr.configService.GetPrequeueConfig(ctx)
		if err != nil {
			return false, err
		}
		if cfg.ShouldBlockByTimedOutPendingNormalJob() {
			candidate := &batch.Job{Spec: batch.JobSpec{
				Queue: queue,
			}}
			for i := range deployment.Pods {
				role := &deployment.Pods[i]
				candidate.Spec.Tasks = append(candidate.Spec.Tasks, batch.TaskSpec{Template: corev1.PodTemplateSpec{Spec: role.Entry}})
				if role.Workers > 0 {
					candidate.Spec.Tasks = append(
						candidate.Spec.Tasks,
						batch.TaskSpec{Template: corev1.PodTemplateSpec{Spec: role.Worker}},
					)
				}
			}
			return mgr.prequeue.HasBlockingTimedOutPendingNormalJob(ctx, token.AccountID, candidate, requested)
		}
	}
	return false, nil
}

func queuedServingReplicaCount(serving *unstructured.Unstructured) int64 {
	count, _ := strconv.ParseInt(serving.GetAnnotations()[kthena.QueuedReplicasAnnotation], 10, 64)
	return count
}
