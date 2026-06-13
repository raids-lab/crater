package vcjob

import (
	v1 "k8s.io/api/core/v1"

	checkpointsvc "github.com/raids-lab/crater/internal/service/vcjob/checkpoint"
	"github.com/raids-lab/crater/internal/util"
)

type CheckpointConfig = checkpointsvc.Config

func prepareCheckpointConfig(
	token util.JWTMessage,
	req *CreateJobCommon,
	volumeMounts []v1.VolumeMount,
) (*CheckpointConfig, error) {
	if req == nil {
		return nil, nil
	}
	cfg, err := checkpointsvc.Prepare(checkpointsvc.PrepareInput{
		Config:       req.Checkpoint,
		RequestName:  req.Name,
		AccountID:    token.AccountID,
		AccountName:  token.AccountName,
		VolumeMounts: volumeMounts,
	})
	if err != nil {
		return nil, err
	}
	req.Checkpoint = cfg
	return cfg, nil
}

func AppendCheckpointEnvs(envs []v1.EnvVar, cfg *CheckpointConfig, jobName string) []v1.EnvVar {
	return checkpointsvc.AppendEnvs(envs, cfg, jobName)
}

func ApplyCheckpointAnnotations(annotations map[string]string, cfg *CheckpointConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	return checkpointsvc.ApplyAnnotations(annotations, cfg.ToCheckpointInfo())
}
