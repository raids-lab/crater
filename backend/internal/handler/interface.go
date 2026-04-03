package handler

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/cronjob"
	"github.com/raids-lab/crater/pkg/imageregistry"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/packer"
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

	CronJobManager *cronjob.CronJobManager

	ConfigService      *service.ConfigService
	GpuAnalysisService *service.GpuAnalysisService
}

type JobMutationSubmitter interface {
	SubmitJupyterJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
	SubmitTrainingJob(ctx context.Context, token util.JWTMessage, req json.RawMessage) (any, error)
}

var jobMutationSubmitterFactory func(conf *RegisterConfig) JobMutationSubmitter

func RegisterJobMutationSubmitterFactory(factory func(conf *RegisterConfig) JobMutationSubmitter) {
	jobMutationSubmitterFactory = factory
}

func NewJobMutationSubmitter(conf *RegisterConfig) JobMutationSubmitter {
	if jobMutationSubmitterFactory == nil {
		return nil
	}
	return jobMutationSubmitterFactory(conf)
}

// Registers is a slice of Manager Init functions.
// Each Manager should register itself by appending its Init function to this slice.
var Registers = []func(config *RegisterConfig) Manager{}
