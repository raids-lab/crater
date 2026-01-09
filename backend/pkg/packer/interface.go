package packer

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
)

type ImageRegistrySecret struct {
	Server  string
	User    string
	Pass    string
	Project string
}

type BuildKitReq struct {
	UserID       uint
	Namespace    string
	JobName      string
	Dockerfile   *string
	Requirements *string
	Description  *string
	Registry     *ImageRegistrySecret // If nil, use default registry
	ImageLink    string
	Tags         []string
	Template     string
	BuildSource  model.BuildSource
	Archs        []string
}

type SnapshotReq struct {
	UserID        uint
	IsAdmin       bool
	Namespace     string
	PodName       string
	ContainerName string
	Description   string
	NodeName      string
	Registry      *ImageRegistrySecret // If nil, use default registry
	ImageLink     string
	BuildSource   model.BuildSource
	// Tolerations to apply to the snapshot job's PodSpec. Optional.
	Tolerations []corev1.Toleration
}

type EnvdReq struct {
	UserID       uint
	Namespace    string
	JobName      string
	Envd         *string
	Requirements *string
	Description  *string
	Registry     *ImageRegistrySecret // If nil, use default registry
	ImageLink    string
	Tags         []string
	Template     string
	BuildSource  model.BuildSource
	Archs        []string
}

type ImagePackerInterface interface {
	CreateFromDockerfile(ctx context.Context, data *BuildKitReq) error
	CreateFromSnapshot(ctx context.Context, data *SnapshotReq) error
	CreateFromEnvd(ctx context.Context, data *EnvdReq) error
	DeleteJob(ctx context.Context, jobName, ns string) error
}

type imagePacker struct {
	client client.Client
}

var (
	buildkitdArmName string = "buildkitd-arm"
	buildkitdAmdName string = "buildkitd-x86"
	runAsUerNumber   int64  = 1000
	runAsGroupNumber int64  = 1000
	fsAsGroupNumber  int64  = 1000

	harborCreditSecretName string = "buildkit-secret"

	JobCleanTime       int32 = 259200
	BackoffLimitNumber int32 = 0
	CompletionNumber   int32 = 1
	ParallelismNumber  int32 = 1
)

const (
	// cpuLimit      = "2"
	// memoryLimit   = "4Gi"
	// cpuRequest    = "1"
	// memoryRequest = "2Gi"

	AnnotationKeyUserID      = "build-data/UserID"      // 用户名ID
	AnnotationKeyImageLink   = "build-data/ImageLink"   // 镜像链接
	AnnotationKeyScript      = "build-data/Script"      // 镜像构建脚本（Dockerfile or Envd 类型）
	AnnotationKeyDescription = "build-data/Description" // 镜像描述
	AnnotationKeyTags        = "build-data/Tags"        // 镜像标签
	AnnotationKeySource      = "build-data/Source"      // 镜像构建来源
	AnnotationKeyTemplate    = "build-data/Template"    // 镜像模板（提交表单转化为Json格式）
	AnnotationKeyArchs       = "build-data/Archs"       // 镜像架构
)

func GetImagePackerMgr(cli client.Client) ImagePackerInterface {
	b := &imagePacker{
		client: cli,
	}
	return b
}
