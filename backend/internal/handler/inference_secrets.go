package handler

import (
	"context"
	"strconv"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Secret references are limited to the same owner and account as the deployment.
// Secret values never enter the deployment DTO, template or resource preview.
func (mgr *KthenaMgr) validateServingSecrets(ctx context.Context, req *CreateKthenaReq, token util.JWTMessage) error {
	for i := range req.Roles {
		role := &req.Roles[i]
		profiles := []kthena.PodProfile{role.Entry}
		if role.Worker != nil {
			profiles = append(profiles, *role.Worker)
		}
		for i := range profiles {
			profile := &profiles[i]
			for _, ref := range profile.SecretEnv {
				var secret corev1.Secret
				if err := mgr.client.Get(ctx, client.ObjectKey{Namespace: mgr.namespace, Name: ref.Name}, &secret); err != nil {
					return bizerr.BadRequest.ParameterError.New("serving credential is unavailable")
				}
				if secret.Labels[inferenceServiceLabelUserID] != strconv.FormatUint(uint64(token.UserID), 10) ||
					secret.Labels[inferenceServiceLabelAccountID] != strconv.FormatUint(uint64(token.AccountID), 10) {
					return bizerr.BadRequest.ParameterError.New("serving credential is unavailable")
				}
				if _, ok := secret.Data[ref.Key]; !ok {
					return bizerr.BadRequest.ParameterError.New("serving credential key is unavailable")
				}
			}
		}
	}
	return nil
}
