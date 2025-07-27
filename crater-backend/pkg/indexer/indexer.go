package indexer

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// PodNodeNameIndex is the index for pods by their node name.
	PodNodeNameIndex = "spec.nodeName"
)

// SetupIndexers sets up the indexers for the manager.
func SetupIndexers(mgr manager.Manager) error {
	// Indexer pods by node name (not include failed and completed)
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, PodNodeNameIndex, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return nil
		}
		if pod.Spec.NodeName == "" {
			return nil
		}
		return []string{pod.Spec.NodeName}
	}); err != nil {
		return fmt.Errorf("unable to index field spec.nodeName: %w", err)
	}
	return nil
}

func QueryPodsByNodeName(ctx context.Context, c client.Client, nodeName string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	if err := c.List(ctx, podList, client.MatchingFields{PodNodeNameIndex: nodeName}); err != nil {
		return nil, fmt.Errorf("failed to list pods by node name %s: %w", nodeName, err)
	}
	return podList.Items, nil
}
