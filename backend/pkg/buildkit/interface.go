package buildkit

import (
	"context"

	"github.com/raids-lab/crater/dao/model"
)

// PrometheusClient is a client for Prometheus
type BuildkitSubmitterInterface interface {
	CreateFromDockerfile(ctx context.Context, user *model.User, dockerfile string) (string, error)
}
