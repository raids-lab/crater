package checkpoint

import (
	"fmt"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"
)

type ValidationContext struct {
	Config       *Config
	VolumeMounts []v1.VolumeMount
	Policies     FrameworkPolicyRegistry
}

type ValidationRule interface {
	Validate(ValidationContext) error
}

type ValidationRuleFunc func(ValidationContext) error

func (fn ValidationRuleFunc) Validate(ctx ValidationContext) error {
	return fn(ctx)
}

type Validator struct {
	Rules []ValidationRule
}

func DefaultValidator() Validator {
	return Validator{Rules: []ValidationRule{
		ValidationRuleFunc(validateFramework),
		ValidationRuleFunc(validateResumeMode),
		ValidationRuleFunc(validateManualResume),
		directoryRule{name: "checkpoint.outputDir", value: func(cfg *Config) string { return cfg.OutputDir }},
		directoryRule{name: "checkpoint.checkpointDir", value: func(cfg *Config) string { return cfg.CheckpointDir }},
		ValidationRuleFunc(validateResumeFromPath),
	}}
}

func (v Validator) Validate(ctx ValidationContext) error {
	if ctx.Config == nil || !ctx.Config.Enabled {
		return nil
	}
	if len(ctx.Policies.policies) == 0 {
		ctx.Policies = DefaultFrameworkPolicyRegistry()
	}
	if len(v.Rules) == 0 {
		v = DefaultValidator()
	}
	for _, rule := range v.Rules {
		if err := rule.Validate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func validateFramework(ctx ValidationContext) error {
	if _, ok := ctx.Policies.Get(ctx.Config.Framework); ok {
		return nil
	}
	return unknownFrameworkError(ctx.Config.Framework, ctx.Policies.AllowedFrameworks())
}

func validateResumeMode(ctx ValidationContext) error {
	switch ctx.Config.ResumeMode {
	case ResumeModeNone, ResumeModeManual, ResumeModeLatest, ResumeModeAuto:
		return nil
	default:
		return fmt.Errorf("checkpoint.resumeMode %q is unsupported; allowed values: [none manual latest auto]", ctx.Config.ResumeMode)
	}
}

func validateManualResume(ctx ValidationContext) error {
	if ctx.Config.ResumeMode == ResumeModeManual && ctx.Config.ResumeFrom == "" {
		return fmt.Errorf("checkpoint.resumeFrom is required when resumeMode is manual")
	}
	return nil
}

type directoryRule struct {
	name  string
	value func(*Config) string
}

func (r directoryRule) Validate(ctx ValidationContext) error {
	value := r.value(ctx.Config)
	if value == "" {
		return fmt.Errorf("%s is required when checkpoint is enabled", r.name)
	}
	if !filepath.IsAbs(value) {
		return fmt.Errorf("%s must be an absolute path", r.name)
	}
	if isForbiddenPath(value) {
		return fmt.Errorf("%s cannot be under a system or ephemeral directory", r.name)
	}
	mount, ok := bestMountForPath(value, ctx.VolumeMounts)
	if !ok {
		return fmt.Errorf("%s must be under a mounted persistent volume", r.name)
	}
	if mount.ReadOnly {
		return fmt.Errorf("%s must be under a writable volume mount", r.name)
	}
	return nil
}

func validateResumeFromPath(ctx ValidationContext) error {
	if ctx.Config.ResumeFrom == "" {
		return nil
	}
	if !filepath.IsAbs(ctx.Config.ResumeFrom) {
		return fmt.Errorf("checkpoint.resumeFrom must be an absolute path")
	}
	if !isPathUnderOrEqual(ctx.Config.ResumeFrom, ctx.Config.CheckpointDir) {
		return fmt.Errorf("checkpoint.resumeFrom must be under checkpoint.checkpointDir")
	}
	return nil
}

func firstWritableMount(volumeMounts []v1.VolumeMount) string {
	for _, mount := range volumeMounts {
		mountPath := filepath.Clean(strings.TrimSpace(mount.MountPath))
		if mountPath == "." || mount.ReadOnly || isForbiddenPath(mountPath) {
			continue
		}
		return mountPath
	}
	return ""
}

func bestMountForPath(cleanPath string, volumeMounts []v1.VolumeMount) (v1.VolumeMount, bool) {
	cleanPath = filepath.Clean(cleanPath)
	var best v1.VolumeMount
	bestLen := -1
	for _, mount := range volumeMounts {
		mountPath := filepath.Clean(strings.TrimSpace(mount.MountPath))
		if mountPath == "." || !isPathUnderOrEqual(cleanPath, mountPath) {
			continue
		}
		if len(mountPath) > bestLen {
			best = mount
			bestLen = len(mountPath)
		}
	}
	return best, bestLen >= 0
}

func isPathUnderOrEqual(cleanPath, cleanBase string) bool {
	cleanPath = filepath.Clean(cleanPath)
	cleanBase = filepath.Clean(cleanBase)
	rel, err := filepath.Rel(cleanBase, cleanPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}

func isForbiddenPath(cleanPath string) bool {
	cleanPath = filepath.Clean(cleanPath)
	if cleanPath == "/" {
		return true
	}
	for _, forbidden := range []string{"/tmp", "/dev/shm", "/proc", "/sys", "/etc", "/var/run"} {
		if cleanPath == forbidden || strings.HasPrefix(cleanPath, forbidden+"/") {
			return true
		}
	}
	return false
}
