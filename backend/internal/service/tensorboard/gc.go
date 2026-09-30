/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

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
	"github.com/raids-lab/crater/pkg/crclient"
)

const tensorboardGCInterval = 5 * time.Minute

// TensorboardGarbageCollector removes legacy TensorBoard Deployments after their TTL expires.
// VCJob-based panels use activeDeadlineSeconds and Volcano TTL instead.
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

	gc.logger.Info("legacy TensorBoard Deployment garbage collector started", "interval", tensorboardGCInterval)
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
			crclient.LabelKeyTaskType: labelKeyTypeTensorboard,
		},
	)
	if err != nil {
		gc.logger.Error(err, "failed to list TensorBoard deployments for TTL cleanup")
		return
	}

	now := time.Now()
	for i := range deployList.Items {
		deploy := &deployList.Items[i]
		expiration, ok := deploy.Annotations[annotationKeyExpirationTime]
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
