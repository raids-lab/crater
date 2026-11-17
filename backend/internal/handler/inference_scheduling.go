package handler

import (
	"context"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

// Kthena creates a PodGroup per serving group and copies this queue annotation
// into PodGroup.spec.queue. Reuse the same queues as ordinary Volcano jobs.
func (mgr *KthenaMgr) prepareKthenaScheduling(ctx context.Context, token util.JWTMessage) error {
	queueName := strings.TrimSpace(vcqueue.ResolveJobQueueName(token))
	if token.UserID == 0 || token.AccountID == 0 || queueName == "" {
		return bizerr.BadRequest.ParameterError.New("current user and account queue are required")
	}
	// Do not allow Kthena's optional PodGroup support to silently fall back to
	// individually scheduled pods in a cluster without Volcano's scheduling API.
	if err := mgr.client.List(ctx, &scheduling.PodGroupList{}, client.InNamespace(mgr.namespace), client.Limit(1)); err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(err, "Volcano PodGroup API is required for model deployments")
	}
	if err := mgr.checkKthenaScheduling(ctx, token, true); err != nil {
		return err
	}
	if err := mgr.ensureKthenaAccountQueues(ctx, token, queueName); err != nil {
		return err
	}

	return mgr.checkKthenaScheduling(ctx, token, false)
}

func (mgr *KthenaMgr) checkKthenaScheduling(ctx context.Context, token util.JWTMessage, allowMissing bool) error {
	queueName := strings.TrimSpace(vcqueue.ResolveJobQueueName(token))
	if token.UserID == 0 || token.AccountID == 0 || queueName == "" {
		return bizerr.BadRequest.ParameterError.New("current user and account queue are required")
	}
	if err := mgr.client.List(ctx, &scheduling.PodGroupList{}, client.InNamespace(mgr.namespace), client.Limit(1)); err != nil {
		return err
	}
	names := []string{queueName}
	if token.AccountID != model.DefaultAccountID {
		names = append(names, vcqueue.GetAccountLogicQueueName(token.AccountID))
	}
	for _, name := range names {
		if err := mgr.checkServingQueue(ctx, token, name, allowMissing); err != nil {
			return err
		}
	}
	return nil
}

func (mgr *KthenaMgr) checkServingQueue(ctx context.Context, token util.JWTMessage, queueName string, allowMissing bool) error {
	queue, err := vcqueue.GetQueue(ctx, mgr.client, queueName)
	if apierrors.IsNotFound(err) && allowMissing && token.AccountID != model.DefaultAccountID {
		return nil
	}
	if err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(err, "inference Volcano queue is unavailable")
	}
	if strings.EqualFold(string(queue.Status.State), "Closed") || strings.EqualFold(string(queue.Status.State), "Closing") {
		return bizerr.Conflict.ResourceStatusError.New("inference Volcano queue is closed")
	}
	if token.AccountID != model.DefaultAccountID && queueName == vcqueue.ResolveJobQueueName(token) &&
		queue.Spec.Parent != vcqueue.GetAccountLogicQueueName(token.AccountID) {
		return bizerr.Conflict.ResourceStatusError.New("inference user queue does not belong to the current account")
	}
	return nil
}

func (mgr *KthenaMgr) ensureKthenaAccountQueues(ctx context.Context, token util.JWTMessage, queueName string) error {
	if token.AccountID == model.DefaultAccountID {
		return nil
	}
	parent := vcqueue.GetAccountLogicQueueName(token.AccountID)
	if _, err := vcqueue.GetQueue(ctx, mgr.client, parent); apierrors.IsNotFound(err) {
		if err := vcqueue.EnsureAccountQueueExists(ctx, mgr.client, token, token.AccountID); err != nil {
			return bizerr.Internal.K8sServiceError.Wrap(err, "ensure inference account queue failed")
		}
	} else if err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(err, "read inference account queue failed")
	}
	if _, err := vcqueue.GetQueue(ctx, mgr.client, queueName); apierrors.IsNotFound(err) {
		if err := vcqueue.EnsureUserQueueExists(ctx, mgr.client, token, token.AccountID, token.UserID); err != nil {
			return bizerr.Internal.K8sServiceError.Wrap(err, "ensure inference user queue failed")
		}
	} else if err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(err, "read inference user queue failed")
	}
	return nil
}
