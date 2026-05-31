package checkpoint

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/raids-lab/crater/dao/model"
)

const (
	AnnotationKeyEnabled    = "crater.raids.io/checkpoint-enabled"
	AnnotationKeyFramework  = "crater.raids.io/checkpoint-framework"
	AnnotationKeyProject    = "crater.raids.io/checkpoint-project"
	AnnotationKeyExperiment = "crater.raids.io/checkpoint-experiment"
	AnnotationKeyDir        = "crater.raids.io/checkpoint-dir"
	AnnotationKeyOutputDir  = "crater.raids.io/output-dir"
	AnnotationKeyResumeMode = "crater.raids.io/resume-mode"
	AnnotationKeyResumeFrom = "crater.raids.io/resume-from"
	AnnotationKeySaveSteps  = "crater.raids.io/checkpoint-save-steps"
	AnnotationKeyMaxToKeep  = "crater.raids.io/checkpoint-max-to-keep"
	AnnotationKeyMaxBytes   = "crater.raids.io/checkpoint-max-bytes"
	AnnotationKeyLatest     = "crater.raids.io/latest-checkpoint"
	AnnotationKeyConfig     = "crater.raids.io/checkpoint-config"
)

func ApplyAnnotations(annotations map[string]string, info *model.CheckpointInfo) error {
	if annotations == nil || info == nil || !info.Enabled {
		return nil
	}

	annotations[AnnotationKeyEnabled] = strconv.FormatBool(info.Enabled)
	annotations[AnnotationKeyFramework] = info.Framework
	annotations[AnnotationKeyProject] = info.ProjectName
	annotations[AnnotationKeyExperiment] = info.ExperimentName
	annotations[AnnotationKeyDir] = info.CheckpointDir
	annotations[AnnotationKeyOutputDir] = info.OutputDir
	annotations[AnnotationKeyResumeMode] = info.ResumeMode
	annotations[AnnotationKeyResumeFrom] = info.ResumeFrom
	annotations[AnnotationKeySaveSteps] = strconv.Itoa(info.SaveSteps)
	annotations[AnnotationKeyMaxToKeep] = strconv.Itoa(info.MaxToKeep)
	annotations[AnnotationKeyMaxBytes] = strconv.FormatInt(info.MaxBytes, 10)
	if info.LatestCheckpoint != "" {
		annotations[AnnotationKeyLatest] = info.LatestCheckpoint
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint config: %w", err)
	}
	annotations[AnnotationKeyConfig] = string(data)
	return nil
}

func ParseAnnotations(annotations map[string]string) (*model.CheckpointInfo, error) {
	if annotations == nil {
		return nil, nil
	}

	if raw := annotations[AnnotationKeyConfig]; raw != "" {
		var info model.CheckpointInfo
		if err := json.Unmarshal([]byte(raw), &info); err != nil {
			return nil, fmt.Errorf("failed to parse checkpoint config annotation: %w", err)
		}
		if !info.Enabled {
			return nil, nil
		}
		return &info, nil
	}

	enabled, err := strconv.ParseBool(annotations[AnnotationKeyEnabled])
	if err != nil || !enabled {
		return nil, nil
	}

	saveSteps, _ := strconv.Atoi(annotations[AnnotationKeySaveSteps])
	maxToKeep, _ := strconv.Atoi(annotations[AnnotationKeyMaxToKeep])
	maxBytes, _ := strconv.ParseInt(annotations[AnnotationKeyMaxBytes], 10, 64)
	return &model.CheckpointInfo{
		Enabled:          enabled,
		Framework:        annotations[AnnotationKeyFramework],
		ProjectName:      annotations[AnnotationKeyProject],
		ExperimentName:   annotations[AnnotationKeyExperiment],
		OutputDir:        annotations[AnnotationKeyOutputDir],
		CheckpointDir:    annotations[AnnotationKeyDir],
		ResumeMode:       annotations[AnnotationKeyResumeMode],
		ResumeFrom:       annotations[AnnotationKeyResumeFrom],
		LatestCheckpoint: annotations[AnnotationKeyLatest],
		SaveSteps:        saveSteps,
		MaxToKeep:        maxToKeep,
		MaxBytes:         maxBytes,
	}, nil
}
