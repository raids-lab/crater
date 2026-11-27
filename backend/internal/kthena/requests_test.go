package kthena

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

func TestResourcesIncludeInitSidecarsOverheadAndAllRoles(t *testing.T) {
	container := func(cpu string) corev1.Container {
		return corev1.Container{
			Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)}},
		}
	}
	sidecar := container("1")
	sidecar.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)
	entry := corev1.PodSpec{
		Containers:     []corev1.Container{container("2")},
		InitContainers: []corev1.Container{sidecar, container("4")},
		Overhead:       corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")},
	}
	worker := corev1.PodSpec{Containers: []corev1.Container{container("4")}}
	d := Deployment{
		Replicas: 2,
		Pods:     []RolePods{{Instances: 3, Workers: 1, Entry: entry, Worker: worker}, {Instances: 1, Entry: worker}},
	}
	// 2 groups * (3 instances * (5.5 entry + 4 worker) + 1 decode * 4).
	total := d.TotalResources()
	if cpu := total.Cpu(); cpu.Cmp(resource.MustParse("65")) != 0 {
		t.Fatalf("total CPU=%s", cpu.String())
	}
}
