package packer

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ImageRegistrySecret struct {
	Server  string
	User    string
	Pass    string
	Project string
}

type BuildKitReq struct {
	UserID      uint
	Namespace   string
	JobName     string
	Dockerfile  *string
	Description *string
	Registry    *ImageRegistrySecret // If nil, use default registry
	ImageLink   string
}

type SnapshotReq struct {
	UserID        uint
	Namespace     string
	PodName       string
	ContainerName string
	Description   string

	NodeName string

	Registry *ImageRegistrySecret // If nil, use default registry

	ImageLink string
}

type ImagePackerInterface interface {
	CreateFromDockerfile(ctx context.Context, data *BuildKitReq) error
	CreateFromSnapshot(ctx context.Context, data *SnapshotReq) error
}

type imagePacker struct {
	client client.Client
}

var (
	runAsUerNumber   int64 = 1000
	runAsGroupNumber int64 = 1000
	fsAsGroupNumber  int64 = 1000

	harborCreditSecretName   string = "buildkit-secret"
	buildkitClientSecretName string = "buildkit-client-certs"

	JobCleanTime       int32 = 259200
	BackoffLimitNumber int32 = 0
	CompletionNumber   int32 = 1
	ParallelismNumber  int32 = 1
)

const (
	cpuLimit    = "2"
	memoryLimit = "4Gi"

	cpuRequest    = "1"
	memoryRequest = "2Gi"
)

func GetImagePackerMgr(cli client.Client) ImagePackerInterface {
	b := &imagePacker{
		client: cli,
	}
	return b
}
