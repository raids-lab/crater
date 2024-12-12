package buildkit

import (
	"context"
)

type BuildKitData struct {
	Namespace  string
	JobName    string
	Dockerfile string

	RegistryServer  string
	RegistryUser    string
	RegistryPass    string
	RegistryProject string

	ImageLink string
}

type BuildKitInterface interface {
	CreateFromDockerfile(ctx context.Context, data *BuildKitData) error
}
