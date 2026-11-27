// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extender

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/utils"
)

const servingGroupPrefix = "serving/"

// Kthena v1.0.0 creates one PodGroup per serving group, controlled by its ModelServing UID.
func (s *Server) workloadName(ctx context.Context, req *requestJobInfo) (string, error) {
	if name := req.ownerJobName(); name != "" {
		return name, nil
	}
	if req == nil || req.PodGroup == nil {
		return "", nil
	}
	for _, owner := range req.PodGroup.ownerReferences() {
		if owner.Controller == nil || !*owner.Controller || owner.Kind != kthena.ServingKind ||
			owner.APIVersion != kthena.ServingGVK.GroupVersion().String() {
			continue
		}
		serving := &unstructured.Unstructured{}
		serving.SetGroupVersionKind(kthena.ServingGVK)
		if err := s.reader.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: owner.Name}, serving); err != nil {
			return "", err
		}
		if kthena.IsNative(serving) {
			if owner.UID == "" || serving.GetUID() != owner.UID {
				return "", fmt.Errorf("managed serving owner UID does not match")
			}
			return servingGroupPrefix + req.podGroupName(), nil
		}
	}
	return "", nil
}

func (s *Server) addServingViews(ctx context.Context, snap *snapshot, groups *scheduling.PodGroupList, namespace string) error {
	labels := client.MatchingLabels{kthena.ManagedByLabel: kthena.ManagedByValue}
	servings, err := kthena.ListDeployments(ctx, s.reader, namespace, labels)
	if err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) && !runtime.IsNotRegisteredError(err) {
		return err
	}
	pods := &v1.PodList{}
	if err := s.reader.List(ctx, pods, client.InNamespace(namespace), labels); err != nil {
		return err
	}
	for i := range servings {
		serving := &servings[i]
		deployment, err := kthena.ReadDeployment(serving)
		if err != nil {
			return fmt.Errorf("read serving group resources: %w", err)
		}
		for i := range groups.Items {
			group := &groups.Items[i]
			if !kthena.IsDesiredPodGroup(serving, group) {
				continue
			}
			view := servingGroupView(serving, deployment, group)
			snap.views = append(snap.views, view)
			snap.byName[view.name] = view
		}
	}
	reservations, err := service.BuildServingQuotaReservations(ctx, servings, pods, groups)
	if err != nil {
		return err
	}
	for _, reservation := range reservations {
		key := ownerKey{userID: reservation.UserID, queue: reservation.Queue}
		snap.usageByOwner[key] = utils.SumResources(snap.usageByOwner[key], reservation.Resources.Data())
	}
	return nil
}

func servingGroupView(serving *unstructured.Unstructured, deployment *kthena.Deployment, group *scheduling.PodGroup) *jobView {
	job := &batch.Job{}
	for i := range deployment.Pods {
		role := &deployment.Pods[i]
		job.Spec.Tasks = append(job.Spec.Tasks, batch.TaskSpec{Template: v1.PodTemplateSpec{Spec: role.Entry}})
		if role.Workers > 0 {
			job.Spec.Tasks = append(job.Spec.Tasks, batch.TaskSpec{Template: v1.PodTemplateSpec{Spec: role.Worker}})
		}
	}
	resources := deployment.GroupResources()
	return &jobView{
		name:      servingGroupPrefix + group.Name,
		userID:    annotationUserID(serving.GetLabels()),
		queue:     group.Spec.Queue,
		resources: resources, minimum: resources,
		resourceDomain: utils.GetResourceDomain(resources),
		nodes:          utils.GetJobExplicitNodeNames(job),
		jobPhase:       batch.Pending, podGroupPhase: group.Status.Phase,
		createdAt: group.CreationTimestamp.Time, tolerance: annotationTolerance(serving.GetAnnotations()),
	}
}
