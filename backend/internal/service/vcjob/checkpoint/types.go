package checkpoint

import (
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
)

const (
	FrameworkPytorch   = "pytorch"
	FrameworkHFTrainer = "hf-trainer"
	FrameworkDeepSpeed = "deepspeed"
	FrameworkVerl      = "verl"
	FrameworkLightning = "lightning"
	FrameworkCustom    = "custom"

	ResumeModeNone   = "none"
	ResumeModeManual = "manual"
	ResumeModeLatest = "latest"
	ResumeModeAuto   = "auto"
)

const (
	defaultFramework = FrameworkCustom
	defaultResume    = ResumeModeNone
	defaultSaveSteps = 500
	defaultMaxToKeep = 3
)

type Config struct {
	Enabled          bool   `json:"enabled"`
	Framework        string `json:"framework"`
	ProjectName      string `json:"projectName"`
	ExperimentName   string `json:"experimentName"`
	OutputDir        string `json:"outputDir"`
	CheckpointDir    string `json:"checkpointDir"`
	ResumeMode       string `json:"resumeMode"`
	ResumeFrom       string `json:"resumeFrom"`
	LatestCheckpoint string `json:"latestCheckpoint,omitempty"`
	SaveSteps        int    `json:"saveSteps"`
	MaxToKeep        int    `json:"maxToKeep"`
	MaxBytes         int64  `json:"maxBytes"`
}

type PrepareInput struct {
	Config       *Config
	RequestName  string
	AccountID    uint
	AccountName  string
	VolumeMounts []v1.VolumeMount
}

func (cfg *Config) ToCheckpointInfo() *model.CheckpointInfo {
	if cfg == nil {
		return nil
	}
	return &model.CheckpointInfo{
		Enabled:          cfg.Enabled,
		Framework:        cfg.Framework,
		ProjectName:      cfg.ProjectName,
		ExperimentName:   cfg.ExperimentName,
		OutputDir:        cfg.OutputDir,
		CheckpointDir:    cfg.CheckpointDir,
		ResumeMode:       cfg.ResumeMode,
		ResumeFrom:       cfg.ResumeFrom,
		LatestCheckpoint: cfg.LatestCheckpoint,
		SaveSteps:        cfg.SaveSteps,
		MaxToKeep:        cfg.MaxToKeep,
		MaxBytes:         cfg.MaxBytes,
	}
}

func ConfigFromInfo(info *model.CheckpointInfo) *Config {
	if info == nil || !info.Enabled {
		return nil
	}
	return &Config{
		Enabled:          info.Enabled,
		Framework:        info.Framework,
		ProjectName:      info.ProjectName,
		ExperimentName:   info.ExperimentName,
		OutputDir:        info.OutputDir,
		CheckpointDir:    info.CheckpointDir,
		ResumeMode:       info.ResumeMode,
		ResumeFrom:       info.ResumeFrom,
		LatestCheckpoint: info.LatestCheckpoint,
		SaveSteps:        info.SaveSteps,
		MaxToKeep:        info.MaxToKeep,
		MaxBytes:         info.MaxBytes,
	}
}
