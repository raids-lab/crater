package controller

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobControlledPodList filter pod list owned by the job.
func JobControlledPodList(list []corev1.Pod, job metav1.Object) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		if !metav1.IsControlledBy(&list[i], job) {
			continue
		}
		ret = append(ret, &list[i])
	}
	return ret
}
