package packer

import (
	"context"
)

type ImageRegistry struct {
	Server  string
	User    string
	Pass    string
	Project string
}

type BuildKitReq struct {
	Namespace  string
	JobName    string
	Dockerfile string

	Registry *ImageRegistry // If nil, use default registry

	ImageLink string
}

type SnapshotReq struct {
	Namespace     string
	PodName       string
	ContainerName string

	Registry *ImageRegistry // If nil, use default registry

	ImageLink string
}

type ImagePackerInterface interface {
	CreateFromDockerfile(ctx context.Context, data *BuildKitReq) error
	CreateFromSnapshot(ctx context.Context, data *SnapshotReq) error
}
