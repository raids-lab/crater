package checkpoint

import (
	"fmt"
	"path/filepath"
)

type FrameworkPolicy interface {
	Name() string
	DefaultDirectory(input PrepareInput, cfg *Config, mountPath string) string
}

type FrameworkPolicyRegistry struct {
	policies map[string]FrameworkPolicy
}

func NewFrameworkPolicyRegistry(policies ...FrameworkPolicy) FrameworkPolicyRegistry {
	registry := FrameworkPolicyRegistry{policies: make(map[string]FrameworkPolicy, len(policies))}
	for _, policy := range policies {
		registry.policies[policy.Name()] = policy
	}
	return registry
}

func (r FrameworkPolicyRegistry) Get(name string) (FrameworkPolicy, bool) {
	policy, ok := r.policies[name]
	return policy, ok
}

func (r FrameworkPolicyRegistry) AllowedFrameworks() []string {
	frameworks := make([]string, 0, len(r.policies))
	for framework := range r.policies {
		frameworks = append(frameworks, framework)
	}
	return frameworks
}

func DefaultFrameworkPolicyRegistry() FrameworkPolicyRegistry {
	defaultPolicy := sharedVolumePolicy{}
	return NewFrameworkPolicyRegistry(
		defaultPolicy.named(FrameworkPytorch),
		hfTrainerPolicy{},
		defaultPolicy.named(FrameworkDeepSpeed),
		defaultPolicy.named(FrameworkVerl),
		defaultPolicy.named(FrameworkLightning),
		defaultPolicy.named(FrameworkCustom),
	)
}

type namedSharedVolumePolicy struct {
	sharedVolumePolicy
	name string
}

type sharedVolumePolicy struct{}

func (p sharedVolumePolicy) named(name string) FrameworkPolicy {
	return namedSharedVolumePolicy{sharedVolumePolicy: p, name: name}
}

func (p namedSharedVolumePolicy) Name() string {
	return p.name
}

func (p namedSharedVolumePolicy) DefaultDirectory(input PrepareInput, cfg *Config, mountPath string) string {
	return p.sharedVolumePolicy.DefaultDirectory(input, cfg, mountPath)
}

func (p sharedVolumePolicy) DefaultDirectory(_ PrepareInput, cfg *Config, mountPath string) string {
	return filepath.Join(
		mountPath,
		"checkpoints",
		pathSegment(cfg.ProjectName, "project"),
		pathSegment(cfg.ExperimentName, "experiment"),
	)
}

type hfTrainerPolicy struct {
	sharedVolumePolicy
}

func (p hfTrainerPolicy) Name() string {
	return FrameworkHFTrainer
}

func (p hfTrainerPolicy) DefaultDirectory(input PrepareInput, cfg *Config, mountPath string) string {
	// HF Trainer writes checkpoint-* under output_dir by default, so outputDir and
	// checkpointDir intentionally share the same root unless the caller overrides.
	return p.sharedVolumePolicy.DefaultDirectory(input, cfg, mountPath)
}

func unknownFrameworkError(framework string, allowed []string) error {
	return fmt.Errorf("checkpoint.framework %q is unsupported; allowed values: %v", framework, allowed)
}
