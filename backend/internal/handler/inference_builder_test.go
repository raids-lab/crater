package handler

import (
	"testing"

	"github.com/raids-lab/crater/internal/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestKthenaSchedulingObjectIsJSONCompatible(t *testing.T) {
	seconds := int64(30)
	req := nativeTestRequest(t)
	req.Roles[0].Entry.GPU = "2"
	req.Roles[0].TensorParallel = 2
	req.Roles[0].Entry.Selectors = []corev1.NodeSelectorRequirement{
		{Key: "accelerator", Operator: corev1.NodeSelectorOpIn, Values: []string{"a100"}},
	}
	req.Roles[0].Entry.Tolerations = []corev1.Toleration{
		{Key: "gpu", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: &seconds},
	}
	if err := validateCreateKthenaReq(t.Context(), req, util.JWTMessage{}); err != nil {
		t.Fatal(err)
	}
	// Kubernetes clients/cache copy unstructured objects; typed slices panic here.
	obj := nativeTestServing(t, req).DeepCopy()
	roles, _, err := unstructured.NestedSlice(obj.Object, "spec", "template", "roles")
	if err != nil {
		t.Fatal(err)
	}
	pod, _, err := unstructured.NestedMap(roles[0].(map[string]any), "entryTemplate", "spec")
	if err != nil {
		t.Fatal(err)
	}
	worker := pod["containers"].([]any)[0].(map[string]any)
	for _, kind := range []string{"requests", "limits"} {
		gpu, _, err := unstructured.NestedString(worker, "resources", kind, "nvidia.com/gpu")
		if err != nil || gpu != "2" {
			t.Fatalf("%s GPU = %q, %v", kind, gpu, err)
		}
	}
	tolerations, _, err := unstructured.NestedSlice(pod, "tolerations")
	if err != nil || tolerations[0].(map[string]any)["tolerationSeconds"] != seconds {
		t.Fatalf("tolerations = %v, %v", tolerations, err)
	}
	terms,
		_,
		err := unstructured.NestedSlice(pod,
		"affinity",
		"nodeAffinity",
		"requiredDuringSchedulingIgnoredDuringExecution",
		"nodeSelectorTerms")
	if err != nil {
		t.Fatal(err)
	}
	expressions := terms[0].(map[string]any)["matchExpressions"].([]any)
	if expressions[0].(map[string]any)["values"].([]any)[0] != "a100" {
		t.Fatalf("expressions = %v", expressions)
	}
}

func TestKthenaRejectsInvalidWorkerBeforeCreation(t *testing.T) {
	tests := map[string]func(*CreateKthenaReq){
		"external PVC": func(r *CreateKthenaReq) { r.ModelURI = "pvc:///shared/other-user" },
		"hostPath":     func(r *CreateKthenaReq) { r.CacheURI = "hostpath:///etc" },
		"account toleration": func(r *CreateKthenaReq) {
			r.Roles[0].Entry.Tolerations = []corev1.Toleration{{Key: "crater.raids.io/account", Value: "other-account"}}
		},
		"wildcard toleration": func(r *CreateKthenaReq) {
			r.Roles[0].Entry.Tolerations = []corev1.Toleration{{Operator: corev1.TolerationOpExists}}
		},
		"architecture":    func(r *CreateKthenaReq) { r.Roles[0].Entry.ImageArchs = []string{"unsupported"} },
		"cpu":             func(r *CreateKthenaReq) { r.Roles[0].Entry.CPU = "garbage" },
		"negative memory": func(r *CreateKthenaReq) { r.Roles[0].Entry.Memory = "-1Gi" },
		"fractional gpu":  func(r *CreateKthenaReq) { r.Roles[0].Entry.GPU = "0.5" },
		"invalid gpu":     func(r *CreateKthenaReq) { r.Roles[0].Entry.GPU = "invalid" },
		"gpu name":        func(r *CreateKthenaReq) { r.Roles[0].Entry.GPU = "1"; r.Roles[0].Entry.GPUModel = "cpu" },
		"model uri":       func(r *CreateKthenaReq) { r.ModelURI = "hf://" },
		"cache uri":       func(r *CreateKthenaReq) { r.CacheURI = "pvc://" },
		"env":             func(r *CreateKthenaReq) { r.Roles[0].Entry.Env = map[string]string{"bad=name": "value"} },
		"probe port":      func(r *CreateKthenaReq) { r.Roles[0].EngineOptions = map[string]string{"port": "70000"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			req := nativeTestRequest(t)
			mutate(req)
			if err := validateCreateKthenaReq(t.Context(), req, util.JWTMessage{}); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}
