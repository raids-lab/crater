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

func MatchingPodsByNodeName(nodeName string) client.MatchingFields {
	return client.MatchingFields{PodNodeNameIndex: nodeName}
}
