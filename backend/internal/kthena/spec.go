package kthena

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	EngineVLLM           = "vLLM"
	EngineSGLang         = "SGLang"
	LayoutCombined       = "combined"
	LayoutPD             = "disaggregated"
	ExecutionSingle      = "single"
	ExecutionRay         = "ray"
	EntryValue           = "true"
	MaxEngineConfigBytes = 64 << 10
	EngineAnnotation     = "crater.raids.io/engine"
	LayoutAnnotation     = "crater.raids.io/layout"
	ModelAnnotation      = "crater.raids.io/served-model"
	RoleConfigAnnotation = "crater.raids.io/role-config"
	EntryLabel           = "modelserving.volcano.sh/entry"
	RoleLabel            = "modelserving.volcano.sh/role"
	GroupLabel           = "modelserving.volcano.sh/group-name"
	SGLangImage          = "lmsysorg/sglang:v0.4.10.post2"
	VLLMImage            = "ghcr.io/volcano-sh/vllm-openai:v0.10.0-cu128-nixl-v0.4.1-lmcache-0.3.2"
)

// RoleSpec describes one independently replicated stage. Entry counts as a node.
// Pod profiles contain only fields Crater owns; callers cannot inject PodSpecs.
type RoleSpec struct {
	Name             string            `json:"name"`
	Instances        int64             `json:"instances"`
	Execution        string            `json:"execution"`
	NodesPerInstance int64             `json:"nodesPerInstance"`
	Entry            PodProfile        `json:"entry"`
	Worker           *PodProfile       `json:"worker,omitempty"`
	TensorParallel   int64             `json:"tensorParallel"`
	PipelineParallel int64             `json:"pipelineParallel"`
	EngineOptions    map[string]string `json:"engineOptions"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type PodProfile struct {
	SecretEnv         map[string]SecretKeyRef          `json:"secretEnv,omitempty"`
	Image             string                           `json:"image"`
	CPU               string                           `json:"cpu"`
	Memory            string                           `json:"memory"`
	GPU               string                           `json:"gpu"`
	GPUModel          string                           `json:"gpuModel"`
	ExtendedResources map[string]string                `json:"extendedResources,omitempty"`
	Env               map[string]string                `json:"env,omitempty"`
	ImageArchs        []string                         `json:"imageArchs,omitempty"`
	Selectors         []corev1.NodeSelectorRequirement `json:"selectors,omitempty"`
	Tolerations       []corev1.Toleration              `json:"tolerations,omitempty"`
}

func (p *PodProfile) Requests() corev1.ResourceList {
	r := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(p.CPU), corev1.ResourceMemory: resource.MustParse(p.Memory)}
	if p.GPU != "" && p.GPU != "0" {
		r[corev1.ResourceName(p.GPUModel)] = resource.MustParse(p.GPU)
	}
	for k, v := range p.ExtendedResources {
		r[corev1.ResourceName(k)] = resource.MustParse(v)
	}
	return r
}

var credentialEnv = regexp.MustCompile(`(?i)(^|_)(TOKEN|PASSWORD|SECRET|API_KEY|ACCESS_KEY)(_|$)`)

var optionName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var reservedOptions = map[string]bool{
	"config": true, "model": true, "model-path": true, "host": true, "port": true, "served-model-name": true,
	"tensor-parallel-size": true, "pipeline-parallel-size": true, "tp": true, "pp": true, "tp-size": true, "pp-size": true,
	"distributed-executor-backend": true, "kv-transfer-config": true, "disaggregation-mode": true,
	"disaggregation-transfer-backend": true, "disaggregation-bootstrap-port": true, "nnodes": true, "node-rank": true,
	"dist-init-addr": true, "enable-metrics": true,
}

func ValidateRoles(engine, layout string, groups int64, roles []RoleSpec) error {
	if engine != EngineVLLM && engine != "SGLang" {
		return fmt.Errorf("backendType must be vLLM or SGLang")
	}
	if groups < 1 || groups > 10000 {
		return fmt.Errorf("replicas must be between 1 and 10000")
	}
	expected := map[string]bool{"server": true}
	if layout == LayoutPD {
		expected = map[string]bool{"prefill": true, "decode": true}
	} else if layout != "combined" {
		return fmt.Errorf("layout must be combined or disaggregated")
	}
	if len(roles) != len(expected) {
		return fmt.Errorf("roles do not match layout")
	}
	for i := range roles {
		r := &roles[i]
		if !expected[r.Name] {
			return fmt.Errorf("roles[%d].name: unexpected or duplicate role", i)
		}
		delete(expected, r.Name)
		if err := validateRole(engine, r); err != nil {
			return fmt.Errorf("roles[%d]: %w", i, err)
		}
	}

	if layout == LayoutPD {
		return validatePD(roles)
	}
	return nil
}

func validatePD(roles []RoleSpec) error {
	for i := range roles {
		if roles[i].Entry.GPU == "0" {
			return fmt.Errorf("P/D requires GPU resources")
		}
		if roles[i].Entry.Image != roles[0].Entry.Image {
			return fmt.Errorf("P/D roles must use the same engine image")
		}
	}
	for _, key := range []string{
		"dtype", "kv-cache-dtype", "tokenizer", "revision", "tokenizer-revision", "quantization", "trust-remote-code",
	} {
		if roles[0].EngineOptions[key] != roles[1].EngineOptions[key] {
			return fmt.Errorf("P/D roles require identical %s", key)
		}
	}
	return nil
}

func validateProfile(p *PodProfile) error {
	p.Image = strings.TrimSpace(p.Image)
	if p.Image == "" {
		return fmt.Errorf("image is required")
	}
	for _, v := range []string{p.CPU, p.Memory} {
		q, e := resource.ParseQuantity(v)
		if e != nil || q.Sign() <= 0 {
			return fmt.Errorf("CPU and memory must be positive quantities")
		}
	}
	if p.GPU == "" {
		p.GPU = "0"
	}
	if p.GPUModel == "" {
		p.GPUModel = "nvidia.com/gpu"
	}
	if err := validateExtendedResources(p); err != nil {
		return err
	}

	if err := validateProfileEnvironment(p); err != nil {
		return err
	}
	return validateProfileScheduling(p)
}
func validateProfileEnvironment(p *PodProfile) error {
	for name, ref := range p.SecretEnv {
		if _, duplicate := p.Env[name]; duplicate {
			return fmt.Errorf("duplicate environment variable %s", name)
		}
		if invalidEnvironmentName(name) {
			return fmt.Errorf("reserved environment variable %s", name)
		}
		if len(validation.IsDNS1123Subdomain(ref.Name)) > 0 || len(validation.IsConfigMapKey(ref.Key)) > 0 {
			return fmt.Errorf("invalid secret reference")
		}
	}
	for k := range p.Env {
		if credentialEnv.MatchString(k) {
			return fmt.Errorf("credential %s must use secretEnv", k)
		}
		if invalidEnvironmentName(k) {
			return fmt.Errorf("reserved or invalid environment variable %s", k)
		}
	}
	return nil
}
func validateProfileScheduling(p *PodProfile) error {
	for _, t := range p.Tolerations {
		if t.Key == "" || t.Key == "crater.raids.io/account" {
			return fmt.Errorf("account tolerations are server controlled")
		}
	}

	for _, s := range p.Selectors {
		if len(validation.IsQualifiedName(s.Key)) > 0 {
			return fmt.Errorf("invalid node selector key")
		}
		switch s.Operator {
		case corev1.NodeSelectorOpIn, corev1.NodeSelectorOpNotIn:
			if len(s.Values) == 0 {
				return fmt.Errorf("node selector requires values")
			}
		case corev1.NodeSelectorOpExists, corev1.NodeSelectorOpDoesNotExist:
			if len(s.Values) != 0 {
				return fmt.Errorf("exists selector cannot have values")
			}
		default:
			return fmt.Errorf("unsupported node selector operator")
		}
		for _, v := range s.Values {
			if len(validation.IsValidLabelValue(v)) > 0 {
				return fmt.Errorf("invalid node selector value")
			}
		}
	}
	for _, a := range p.ImageArchs {
		if a != "amd64" && a != "arm64" {
			return fmt.Errorf("unsupported architecture %s", a)
		}
	}
	return nil
}

func reservedEngineOption(key string) bool {
	for name := range reservedOptions {
		if strings.HasPrefix(name, key) {
			return true
		}
	}
	return false
}

func validateRole(engine string, r *RoleSpec) error {
	if r.Instances < 1 || r.Instances > 10000 || r.NodesPerInstance < 1 || r.NodesPerInstance > 128 {
		return fmt.Errorf("%s: invalid instance/node count", r.Name)
	}
	if r.TensorParallel < 1 || r.PipelineParallel < 1 {
		return fmt.Errorf("%s: parallelism must be positive", r.Name)
	}
	if r.Execution != "single" && r.Execution != ExecutionRay {
		return fmt.Errorf("%s.execution: unsupported executor", r.Name)
	}
	if r.Execution == "single" && (r.NodesPerInstance != 1 || r.Worker != nil) {
		return fmt.Errorf("%s: single execution requires one entry and no worker profile", r.Name)
	}
	if r.Execution == ExecutionRay && (engine != EngineVLLM || r.NodesPerInstance < 2) {
		return fmt.Errorf("%s: Ray requires vLLM and at least two nodes", r.Name)
	}
	return validateRoleProfiles(r)
}
func validateRoleProfiles(r *RoleSpec) error {
	if err := validateProfile(&r.Entry); err != nil {
		return fmt.Errorf("%s.entry: %w", r.Name, err)
	}
	if r.Worker != nil {
		if err := validateProfile(r.Worker); err != nil {
			return fmt.Errorf("%s.worker: %w", r.Name, err)
		}
	}

	if err := validateParallelism(r); err != nil {
		return err
	}
	return validateEngineOptions(r)
}
func validateParallelism(r *RoleSpec) error {
	if r.Execution == ExecutionRay {
		w := &r.Entry
		if r.Worker != nil {
			w = r.Worker
		}
		if w.Image != r.Entry.Image || w.GPU != r.Entry.GPU || w.GPUModel != r.Entry.GPUModel {
			return fmt.Errorf("%s: Ray profile requires identical image and GPU allocation on all nodes", r.Name)
		}
		gpu := resource.MustParse(r.Entry.GPU)
		if gpu.Value() < 1 || r.TensorParallel != gpu.Value() || r.PipelineParallel != r.NodesPerInstance {
			return fmt.Errorf("%s: Ray profile requires TP=GPUs per node and PP=nodes per instance", r.Name)
		}
	} else {
		gpu := resource.MustParse(r.Entry.GPU)
		if r.PipelineParallel != 1 || (gpu.Value() > 0 && r.TensorParallel != gpu.Value()) || (gpu.IsZero() && r.TensorParallel != 1) {
			return fmt.Errorf("%s: single-node profile requires PP=1 and TP=allocated GPUs (or 1 for CPU)", r.Name)
		}
	}
	return nil
}
func validateEngineOptions(r *RoleSpec) error {
	for k := range r.EngineOptions {
		if !optionName.MatchString(k) || reservedEngineOption(k) {
			return fmt.Errorf("%s.engineOptions: reserved or invalid option %q", r.Name, k)
		}
	}
	raw, _ := json.Marshal(r.EngineOptions)
	if len(raw) > MaxEngineConfigBytes {
		return fmt.Errorf("%s.engineOptions: too large", r.Name)
	}
	return nil
}

func invalidEnvironmentName(name string) bool {
	return len(validation.IsEnvVarName(name)) > 0 || name == "ENTRY_ADDRESS" || name == "POD_IP" || name == "HF_REVISION" ||
		name == "MS_REVISION" ||
		strings.HasPrefix(name, "CRATER_")
}

func validateExtendedResources(p *PodProfile) error {
	ext := map[string]string{p.GPUModel: p.GPU}
	for k, v := range p.ExtendedResources {
		if k == p.GPUModel {
			return fmt.Errorf("duplicate GPU resource")
		}
		ext[k] = v
	}
	for k, v := range ext {
		q, e := resource.ParseQuantity(v)
		n := q.Value()
		exact := q.Cmp(*resource.NewQuantity(n, resource.DecimalSI)) == 0
		if e != nil || !exact || n < 0 || !strings.Contains(k, "/") || len(validation.IsQualifiedName(k)) > 0 {
			return fmt.Errorf("invalid extended resource %s", k)
		}
		if k == p.GPUModel {
			p.GPU = strconv.FormatInt(n, 10)
		} else {
			p.ExtendedResources[k] = strconv.FormatInt(n, 10)
		}
	}

	return nil
}
