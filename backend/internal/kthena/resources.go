// Package kthena defines the resource contract shared by deployment and workload APIs.
package kthena

import (
	"context"
	"encoding/json"
	"fmt"

	resourcehelper "k8s.io/component-helpers/resource"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	QueuedReplicasAnnotation = "crater.raids.io/queued-replicas"
	apiVersion               = "v1alpha1"
	ManagedByLabel           = "crater.raids.io/managed-by"
	ManagedByValue           = "inference-service"
	LayoutLabel              = "crater.raids.io/inference-layout"
	NativeLayout             = "model-serving-v1"
	PodDeploymentLabel       = "crater.raids.io/inference-deployment"
	ModelURIAnnotation       = "crater.raids.io/model-uri"
	CacheURIAnnotation       = "crater.raids.io/cache-uri"
	ServingKind              = "ModelServing"
	EngineContainer          = "engine"
)

var (
	ServingGVK = schema.GroupVersionKind{Group: "workload.serving.volcano.sh", Version: apiVersion, Kind: ServingKind}
	ServerGVK  = schema.GroupVersionKind{Group: "networking.serving.volcano.sh", Version: apiVersion, Kind: "ModelServer"}
	RouteGVK   = schema.GroupVersionKind{Group: "networking.serving.volcano.sh", Version: apiVersion, Kind: "ModelRoute"}
)

func IsNative(obj *unstructured.Unstructured) bool {
	return obj.GetKind() == ServingKind && obj.GetLabels()[LayoutLabel] == NativeLayout &&
		obj.GetLabels()[ManagedByLabel] == ManagedByValue
}

// GetDeployment only returns a ModelServing managed by Crater.
func GetDeployment(ctx context.Context, c client.Reader, key client.ObjectKey) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(ServingGVK)
	if err := c.Get(ctx, key, obj); err != nil {
		return nil, err
	}
	if IsNative(obj) {
		return obj, nil
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{Group: ServingGVK.Group, Resource: "modelservings"}, key.Name)
}

// ListDeployments returns managed ModelServings and propagates observation failures.
func ListDeployments(
	ctx context.Context, c client.Reader, namespace string, labels client.MatchingLabels,
) ([]unstructured.Unstructured, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(ServingGVK.GroupVersion().WithKind(ServingKind + "List"))
	if err := c.List(ctx, list, client.InNamespace(namespace), labels); err != nil {
		return nil, err
	}
	result := make([]unstructured.Unstructured, 0, len(list.Items))
	for _, obj := range list.Items {
		if IsNative(&obj) {
			result = append(result, obj)
		}
	}
	return result, nil
}

// Deployment reads actual native templates; runtime annotations contain only
// engine options not represented by Kubernetes fields.
type Deployment struct {
	ModelURI, ModelRevision, CacheURI, Scheduler, Engine, Layout, ServedModel string
	Replicas, Port                                                            int64
	Roles                                                                     []RoleSpec
	Pods                                                                      []RolePods
}

type RolePods struct {
	Instances int64
	Workers   int64
	Entry     corev1.PodSpec
	Worker    corev1.PodSpec
}

func ReadDeployment(obj *unstructured.Unstructured) (*Deployment, error) {
	roles, _, err := unstructured.NestedSlice(obj.Object, "spec", "template", "roles")
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("serving roles missing")
	}
	a := obj.GetAnnotations()
	d := &Deployment{
		ModelURI:      a[ModelURIAnnotation],
		ModelRevision: a["crater.raids.io/model-revision"],
		CacheURI:      a[CacheURIAnnotation],
		Engine:        a[EngineAnnotation],
		Layout:        a[LayoutAnnotation],
		ServedModel:   a[ModelAnnotation],
	}
	d.Replicas, _, _ = unstructured.NestedInt64(obj.Object, "spec", "replicas")
	d.Scheduler, _, _ = unstructured.NestedString(obj.Object, "spec", "schedulerName")
	for _, raw := range roles {
		r, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid role")
		}
		name, _, _ := unstructured.NestedString(r, "name")
		count, _, _ := unstructured.NestedInt64(r, "replicas")
		workers, _, _ := unstructured.NestedInt64(r, "workerReplicas")
		pod, profile, meta, port, err := readPodProfile(r, "entryTemplate")
		if err != nil {
			return nil, err
		}
		role := RoleSpec{Name: name, Instances: count, NodesPerInstance: workers + 1, Entry: profile}
		if err := json.Unmarshal([]byte(meta), &role); err != nil {
			return nil, fmt.Errorf("role %s runtime metadata: %w", name, err)
		}
		pods := RolePods{Instances: count, Workers: workers, Entry: pod}
		if workers > 0 {
			wp, w, _, _, err := readPodProfile(r, "workerTemplate")
			if err != nil {
				return nil, err
			}
			role.Worker = &w
			pods.Worker = wp
		}
		if d.Port != 0 && d.Port != port {
			return nil, fmt.Errorf("P/D role ports must match")
		}
		d.Port = port
		d.Roles = append(d.Roles, role)
		d.Pods = append(d.Pods, pods)
	}
	return d, nil
}

func readPodProfile(role map[string]any, key string) (pod corev1.PodSpec, profile PodProfile, meta string, port int64, err error) {
	profile = PodProfile{SecretEnv: map[string]SecretKeyRef{}, Env: map[string]string{}, ExtendedResources: map[string]string{}}
	raw, _, err := unstructured.NestedMap(role, key, "spec")
	if err != nil {
		return pod, profile, "", 0, err
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw, &pod); err != nil {
		return pod, profile, "", 0, err
	}
	meta, _, _ = unstructured.NestedString(role, key, "metadata", "annotations", RoleConfigAnnotation)
	var runtimeMeta struct {
		GPUModel string `json:"gpuModel"`
	}
	if err := json.Unmarshal([]byte(meta), &runtimeMeta); err != nil {
		return pod, profile, "", 0, err
	}
	profile.GPUModel = runtimeMeta.GPUModel
	profile.GPU = "0"
	found := false
	for i := range pod.Containers {
		c := &pod.Containers[i]
		if c.Name != EngineContainer {
			continue
		}
		found = true
		port = readEngineProfile(c, &profile)
	}
	if !found {
		return pod, profile, "", 0, fmt.Errorf("engine container missing")
	}
	if err := readProfileScheduling(&pod, &profile); err != nil {
		return pod, profile, "", 0, err
	}
	return pod, profile, meta, port, nil
}

func (d *Deployment) TotalResources() corev1.ResourceList {
	result := corev1.ResourceList{}
	for i := range d.Pods {
		p := &d.Pods[i]
		addResources(result, PodRequests(&p.Entry), d.Replicas*p.Instances)
		addResources(result, PodRequests(&p.Worker), d.Replicas*p.Instances*p.Workers)
	}
	return result
}

// PodRequests includes restartable init sidecars and Pod overhead, using the
// same peak-init versus steady-state rule as Kubernetes scheduling.
func PodRequests(pod *corev1.PodSpec) corev1.ResourceList {
	podCopy := pod.DeepCopy()
	for i := range podCopy.Containers {
		podCopy.Containers[i].Resources.Requests = effectiveRequests(podCopy.Containers[i].Resources)
	}
	for i := range podCopy.InitContainers {
		podCopy.InitContainers[i].Resources.Requests = effectiveRequests(podCopy.InitContainers[i].Resources)
	}
	return resourcehelper.PodRequests(&corev1.Pod{Spec: *podCopy}, resourcehelper.PodResourcesOptions{})
}
func addResources(dst, src corev1.ResourceList, multiplier int64) {
	for k, v := range src {
		v = v.DeepCopy()
		v.Mul(multiplier)
		q := dst[k]
		q.Add(v)
		dst[k] = q
	}
}

func Phase(obj *unstructured.Unstructured) string {
	if obj.GetAnnotations()["crater.raids.io/paused-replicas"] != "" {
		return "Suspended"
	}
	if obj.GetAnnotations()[QueuedReplicasAnnotation] != "" {
		return "Prequeue"
	}
	if !IsNative(obj) {
		return ""
	}
	desired, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	available, _, _ := unstructured.NestedInt64(obj.Object, "status", "availableReplicas")
	observed, _, _ := unstructured.NestedInt64(obj.Object, "status", "observedGeneration")
	if observed < obj.GetGeneration() {
		return "Progressing"
	}
	if desired > 0 && available >= desired {
		return "Ready"
	}
	if available > 0 {
		return "Degraded"
	}
	return "Pending"
}

func effectiveRequests(resources corev1.ResourceRequirements) corev1.ResourceList {
	requests := resources.Limits.DeepCopy()
	if requests == nil {
		requests = corev1.ResourceList{}
	}
	for k, v := range resources.Requests {
		requests[k] = v.DeepCopy()
	}
	return requests
}

func readEngineProfile(c *corev1.Container, profile *PodProfile) int64 {
	var port int64
	profile.Image = c.Image
	for k, v := range effectiveRequests(c.Resources) {
		switch k {
		case corev1.ResourceCPU:
			profile.CPU = v.String()
		case corev1.ResourceMemory:
			profile.Memory = v.String()
		default:
			if string(k) == profile.GPUModel {
				profile.GPU = v.String()
			} else {
				profile.ExtendedResources[string(k)] = v.String()
			}
		}
	}
	for _, e := range c.Env {
		if e.ValueFrom == nil {
			profile.Env[e.Name] = e.Value
		} else if ref := e.ValueFrom.SecretKeyRef; ref != nil {
			profile.SecretEnv[e.Name] = SecretKeyRef{Name: ref.Name, Key: ref.Key}
		}
	}
	for _, p := range c.Ports {
		if p.Name == "http" {
			port = int64(p.ContainerPort)
		}
	}
	return port
}
func readProfileScheduling(pod *corev1.PodSpec, profile *PodProfile) error {
	for _, t := range pod.Tolerations {
		if t.Key != "crater.raids.io/account" {
			profile.Tolerations = append(profile.Tolerations, t)
		}
	}
	if pod.Affinity != nil && pod.Affinity.NodeAffinity != nil &&
		pod.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		terms := pod.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		if len(terms) != 1 {
			return fmt.Errorf("unsupported node affinity")
		}
		for _, s := range terms[0].MatchExpressions {
			if s.Key == "kubernetes.io/arch" {
				profile.ImageArchs = s.Values
			} else {
				profile.Selectors = append(profile.Selectors, s)
			}
		}
	}
	return nil
}
