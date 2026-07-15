package handler

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/prequeuewatcher"
)

// Manager is the interface that wraps the basic methods for a handler manager.
type Manager interface {
	GetName() string
	RegisterPublic(group *gin.RouterGroup)
	RegisterProtected(group *gin.RouterGroup)
	RegisterAdmin(group *gin.RouterGroup)
}

// RegisterConfig is a struct that holds the configuration for a Manager.
type RegisterConfig struct {
	// Client is the controller-runtime client.
	Client client.Client

	// KubeConfig is the kubernetes client config.
	KubeConfig *rest.Config

	// KubeClient is the kubernetes client.
	KubeClient kubernetes.Interface

	// PrometheusClient is the prometheus client.
	PrometheusClient monitor.PrometheusInterface

	// AITaskCtrl is the aitask controller.
	AITaskCtrl aitaskctl.TaskControllerInterface

	// ImagePacker is the image packer.
	ImagePacker packer.ImagePackerInterface

	// ImageRegistry is the image registry.
	ImageRegistry imageregistry.ImageRegistryInterface

	// ServiceManager 用于创建 Service 和 Ingress
	ServiceManager crclient.ServiceManagerInterface

	CronJobManager  *cronjob.CronJobManager
	PrequeueWatcher *prequeuewatcher.PrequeueWatcher

	// services
	ConfigService      *service.ConfigService
	PrequeueService    *service.PrequeueService
	BillingService     *service.BillingService
	GpuAnalysisService *service.GpuAnalysisService
}

type JobMutationSubmitter interface {
	SubmitJupyterJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
	SubmitWebIDEJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
	SubmitTrainingJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
	SubmitPytorchJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
	SubmitTensorflowJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
	DeleteJob(ctx context.Context, token util.JWTMessage, jobName string) (any, error)
	StopJob(ctx context.Context, token util.JWTMessage, jobName string) (any, error)
	ResubmitJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
}

type JobInsightReader interface {
	FindScopedJob(ctx context.Context, token util.JWTMessage, jobName string) (*model.Job, error)
	BuildJobDetail(job *model.Job) any
	GetJobEvents(ctx context.Context, token util.JWTMessage, jobName string) (any, error)
	GetJobLog(ctx context.Context, token util.JWTMessage, jobName string, tailLines int64, keyword string) (map[string]string, error)
	GetDiagnosticContext(
		ctx context.Context,
		token util.JWTMessage,
		jobName string,
		includeLog bool,
		tailLines int64,
	) (JobContextResp, error)
}

type ImageAccessRecord struct {
	Image       *model.Image
	ShareStatus model.ImageShareType
}

type ImageInsightReader interface {
	ListAccessibleImages(ctx context.Context, token util.JWTMessage) ([]ImageAccessRecord, error)
}

var jobMutationSubmitterFactory func(conf *RegisterConfig) JobMutationSubmitter
var jobInsightReaderFactory func(conf *RegisterConfig) JobInsightReader
var imageInsightReaderFactory func(conf *RegisterConfig) ImageInsightReader

func RegisterJobMutationSubmitterFactory(factory func(conf *RegisterConfig) JobMutationSubmitter) {
	jobMutationSubmitterFactory = factory
}

func RegisterJobInsightReaderFactory(factory func(conf *RegisterConfig) JobInsightReader) {
	jobInsightReaderFactory = factory
}

func RegisterImageInsightReaderFactory(factory func(conf *RegisterConfig) ImageInsightReader) {
	imageInsightReaderFactory = factory
}

func NewJobMutationSubmitter(conf *RegisterConfig) JobMutationSubmitter {
	if jobMutationSubmitterFactory == nil {
		return nil
	}
	return jobMutationSubmitterFactory(conf)
}

func NewJobInsightReader(conf *RegisterConfig) JobInsightReader {
	if jobInsightReaderFactory == nil {
		return nil
	}
	return jobInsightReaderFactory(conf)
}

func NewImageInsightReader(conf *RegisterConfig) ImageInsightReader {
	if imageInsightReaderFactory == nil {
		return nil
	}
	return imageInsightReaderFactory(conf)
}

// InternalRouter is an optional interface for managers that expose internal-only endpoints
// (e.g. service-to-service callbacks authenticated via X-Agent-Internal-Token).
// Managers that do not need internal routes do not need to implement this interface.
type InternalRouter interface {
	RegisterInternal(group *gin.RouterGroup)
}

// Registers is a slice of Manager Init functions.
// Each Manager should register itself by appending its Init function to this slice.
var Registers = []func(config *RegisterConfig) Manager{}
