package checkpoint

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Normalizer struct {
	Policies FrameworkPolicyRegistry
}

func DefaultNormalizer() Normalizer {
	return Normalizer{Policies: DefaultFrameworkPolicyRegistry()}
}

func (n Normalizer) Normalize(input PrepareInput) (*Config, error) {
	if input.Config == nil || !input.Config.Enabled {
		return nil, nil
	}
	if len(n.Policies.policies) == 0 {
		n.Policies = DefaultFrameworkPolicyRegistry()
	}

	cfg := *input.Config
	cfg.Enabled = true
	cfg.Framework = normalizeKeyword(cfg.Framework, defaultFramework)
	cfg.ResumeMode = normalizeKeyword(cfg.ResumeMode, defaultResume)

	cfg.ProjectName = strings.TrimSpace(cfg.ProjectName)
	if cfg.ProjectName == "" {
		cfg.ProjectName = strings.TrimSpace(input.AccountName)
	}
	if cfg.ProjectName == "" {
		cfg.ProjectName = fmt.Sprintf("account-%d", input.AccountID)
	}

	cfg.ExperimentName = strings.TrimSpace(cfg.ExperimentName)
	if cfg.ExperimentName == "" {
		cfg.ExperimentName = strings.TrimSpace(input.RequestName)
	}
	if cfg.ExperimentName == "" {
		cfg.ExperimentName = "experiment"
	}

	if cfg.SaveSteps <= 0 {
		cfg.SaveSteps = defaultSaveSteps
	}
	if cfg.MaxToKeep <= 0 {
		cfg.MaxToKeep = defaultMaxToKeep
	}

	cfg.OutputDir = strings.TrimSpace(cfg.OutputDir)
	cfg.CheckpointDir = strings.TrimSpace(cfg.CheckpointDir)
	if cfg.OutputDir == "" && cfg.CheckpointDir == "" {
		if policy, ok := n.Policies.Get(cfg.Framework); ok {
			if mountPath := firstWritableMount(input.VolumeMounts); mountPath != "" {
				defaultDir := policy.DefaultDirectory(input, &cfg, mountPath)
				cfg.OutputDir = defaultDir
				cfg.CheckpointDir = defaultDir
			}
		}
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = cfg.CheckpointDir
	}
	if cfg.CheckpointDir == "" {
		cfg.CheckpointDir = cfg.OutputDir
	}

	cfg.OutputDir = cleanOptionalPath(cfg.OutputDir)
	cfg.CheckpointDir = cleanOptionalPath(cfg.CheckpointDir)
	cfg.ResumeFrom = cleanOptionalPath(strings.TrimSpace(cfg.ResumeFrom))
	return &cfg, nil
}

func normalizeKeyword(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func cleanOptionalPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func pathSegment(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return fallback
	}
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	if value == "" {
		return fallback
	}
	return value
}
