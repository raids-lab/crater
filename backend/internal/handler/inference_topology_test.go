package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:gocyclo // Each topology checks the same render, read, resource and repair invariants.
func TestServingTopologyRoundTripAndRepair(t *testing.T) {
	for _, tc := range []struct{ engine, layout, execution string }{{"vLLM", "combined", "single"}, {"SGLang", "combined", "single"}, {"vLLM", "combined", kthena.ExecutionRay}, {"vLLM", kthena.LayoutPD, "single"}, {"SGLang", kthena.LayoutPD, "single"}, {"vLLM", kthena.LayoutPD, kthena.ExecutionRay}} {
		name := tc.engine + "-" + tc.layout + "-" + tc.execution
		t.Run(name, func(t *testing.T) {
			req := nativeTestRequest(t)
			req.ModelRevision = "0123456789abcdef0123456789abcdef01234567"
			req.BackendType = tc.engine
			req.Layout = tc.layout
			if tc.layout == kthena.LayoutPD {
				req.Roles[0].Name = "prefill"
				decode := req.Roles[0]
				decode.Name = "decode"
				decode.Instances = 2
				req.Roles = append(req.Roles, decode)
			}
			option := "max-model-len"
			image := kthena.VLLMImage
			if tc.engine == kthena.EngineSGLang {
				option = "max-running-requests"
				image = "lmsysorg/sglang:v0.4.10.post2"
			}
			var expectedGPU int64
			for i := range req.Roles {
				r := &req.Roles[i]
				r.Execution = tc.execution
				r.Entry.Image = image
				r.EngineOptions = map[string]string{option: "1024"}
				if tc.execution == kthena.ExecutionRay {
					r.NodesPerInstance = 2
					r.PipelineParallel = 2
					worker := r.Entry
					worker.CPU = "3"
					r.Worker = &worker
				}
				expectedGPU += req.Replicas * r.Instances * r.NodesPerInstance
			}
			if err := validateCreateKthenaReq(t.Context(), req, util.JWTMessage{}); err != nil {
				t.Fatal(err)
			}
			obj := nativeTestServing(t, req)
			d, err := kthena.ReadDeployment(obj)
			if err != nil {
				t.Fatal(err)
			}
			if d.Engine != tc.engine || d.Layout != tc.layout || len(d.Roles) != len(req.Roles) {
				t.Fatalf("lost topology: %+v", d)
			}
			gpu := d.TotalResources()["nvidia.com/gpu"]
			if gpu.Value() != expectedGPU {
				t.Fatalf("GPU reservation=%s, want %d", gpu.String(), expectedGPU)
			}
			for i, role := range d.Roles {
				if role.Execution != tc.execution || role.NodesPerInstance != req.Roles[i].NodesPerInstance ||
					role.EngineOptions[option] != "1024" {
					t.Fatalf("lost role: %+v", role)
				}
				if tc.execution == kthena.ExecutionRay && role.Worker.CPU != "3" {
					t.Fatal("worker resource override lost")
				}
			}
			children := buildKthenaNetworking(obj, req)
			engine, _, _ := unstructured.NestedString(children[0].Object, "spec", "inferenceEngine")
			selector, _, _ := unstructured.NestedStringMap(children[0].Object, "spec", "workloadSelector", "matchLabels")
			if engine != tc.engine || selector[kthena.EntryLabel] != "true" {
				t.Fatal("router targets workers or wrong engine")
			}
			pd, found, _ := unstructured.NestedMap(children[0].Object, "spec", "workloadSelector", "pdGroup")
			if (tc.layout == kthena.LayoutPD) != found {
				t.Fatalf("PD selector=%v", pd)
			}
			c := nativeTestClient(obj)
			mgr := &KthenaMgr{client: c}
			if err := mgr.reconcileKthenaNetworking(t.Context(), obj); err != nil {
				t.Fatal(err)
			}
			for _, expected := range children {
				actual := &unstructured.Unstructured{}
				actual.SetGroupVersionKind(expected.GroupVersionKind())
				if err := c.Get(t.Context(), client.ObjectKeyFromObject(expected), actual); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(actual.Object["spec"], expected.Object["spec"]) {
					t.Fatal("repair changed topology")
				}
			}
			if dir := os.Getenv("CRATER_KTHENA_MANIFEST_DIR"); dir != "" {
				dir = filepath.Join(dir, name)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				for _, cr := range append([]*unstructured.Unstructured{obj}, children...) {
					data, err := json.MarshalIndent(cr.Object, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, cr.GetKind()+".json"), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

func TestServingRejectsTopologyAndReservedOptions(t *testing.T) {
	for name, mutate := range map[string]func(*CreateKthenaReq){
		"missing decode": func(r *CreateKthenaReq) { r.Layout = kthena.LayoutPD },
		"ray sglang": func(r *CreateKthenaReq) {
			r.BackendType = "SGLang"
			r.Roles[0].Execution = kthena.ExecutionRay
			r.Roles[0].NodesPerInstance = 2
		},
		"unmatched nodes":    func(r *CreateKthenaReq) { r.Roles[0].NodesPerInstance = 2 },
		"parallelism":        func(r *CreateKthenaReq) { r.Roles[0].TensorParallel = 8 },
		"model abbreviation": func(r *CreateKthenaReq) { r.Roles[0].EngineOptions = map[string]string{"mod": "foreign"} },
		"credential":         func(r *CreateKthenaReq) { r.Roles[0].Entry.Env = map[string]string{"HF_TOKEN": "do-not-store"} },
		"selector": func(r *CreateKthenaReq) {
			r.Roles[0].Entry.Selectors = []corev1.NodeSelectorRequirement{{Key: "gpu", Operator: corev1.NodeSelectorOpIn}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := nativeTestRequest(t)
			mutate(req)
			if err := validateCreateKthenaReq(t.Context(), req, util.JWTMessage{}); err == nil {
				t.Fatal("invalid topology accepted")
			}
		})
	}
}

func TestServingRuntimeUsesArgvWithoutShellExpansion(t *testing.T) {
	req := nativeTestRequest(t)
	req.Roles[0].EngineOptions = map[string]string{"chat-template": "$(touch /tmp/not-executed); `echo unsafe`"}
	obj := nativeTestServing(t, req)
	d, err := kthena.ReadDeployment(obj)
	if err != nil {
		t.Fatal(err)
	}
	command := d.Pods[0].Entry.Containers[0].Command
	var launch struct {
		Argv []string `json:"argv"`
	}
	if err := json.Unmarshal([]byte(command[4]), &launch); err != nil {
		t.Fatal(err)
	}
	if command[0] != "python3" || launch.Argv[len(launch.Argv)-1] != req.Roles[0].EngineOptions["chat-template"] {
		t.Fatal("engine options must be opaque argv")
	}
}
