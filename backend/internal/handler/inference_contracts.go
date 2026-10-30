package handler

import (
	"time"

	"github.com/raids-lab/crater/internal/kthena"

	corev1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
)

const (
	kthenaChatCompletionsPath      = "v1/chat/completions"
	inferenceServiceLabelManagedBy = "crater.raids.io/managed-by"
	inferenceServiceLabelUserID    = "crater.raids.io/user-id"
	inferenceServiceLabelAccountID = "crater.raids.io/account-id"
	inferenceServiceManagedByValue = "inference-service"

	inferenceServiceAnnotationUsername = "crater.raids.io/user"
	inferenceServiceAnnotationAccount  = "crater.raids.io/account"
	inferenceServiceAnnotationSource   = "crater.raids.io/model-source"
	inferenceServiceAnnotationModelID  = "crater.raids.io/platform-model-id"

	kthenaNamespace     = "kthena-system"
	kthenaRouterService = "kthena-router"
	kthenaProxyPrefix   = "openai"
	kthenaSchedulerName = "volcano"
	kthenaSpecTypeKey   = "type"

	kthenaKindModelServing = "ModelServing"
	kthenaKindModelRoute   = "ModelRoute"

	kthenaKindModelServer = "ModelServer"

	kthenaBackendVLLM = "vLLM"

	kthenaResourceCPUKey         = "cpu"
	kthenaResourceMemoryKey      = "memory"
	kthenaPhasePending           = "Pending"
	kthenaPhaseReady             = "Ready"
	kthenaPhaseActive            = "Active"
	kthenaPhaseDegraded          = "Degraded"
	kthenaPhaseProgressing       = "Progressing"
	kthenaDiagnosticLevelWarning = "warning"

	kthenaLogVerbosity              = 4
	kthenaMaxDiagnostics            = 10
	kthenaDefaultLogTailLines int64 = 80

	inferenceModelSourcePlatform = "platform"
	inferenceModelSourceExternal = "external"
)

type CreateKthenaReq struct {
	ModelRevision   string            `json:"modelRevision"`
	Name            string            `json:"name"            binding:"required"`
	ModelSource     string            `json:"modelSource"`
	PlatformModelID uint              `json:"platformModelId"`
	ModelURI        string            `json:"modelURI"`
	ServedModel     string            `json:"servedModel"`
	BackendType     string            `json:"backendType"`
	CacheURI        string            `json:"cacheURI"`
	Replicas        int64             `json:"replicas"`
	Layout          string            `json:"layout"`
	Port            int64             `json:"port"`
	Roles           []kthena.RoleSpec `json:"roles"           binding:"required"`
	ModelSubPath    string            `json:"-"`
	ModelPVC        string            `json:"-"`
	BillingGate     bool              `json:"-"`
}

type KthenaServiceResp struct {
	ModelRevision     string              `json:"modelRevision"`
	DesiredReplicas   int64               `json:"desiredReplicas"`
	Name              string              `json:"name"`
	Namespace         string              `json:"namespace"`
	Owner             string              `json:"owner"`
	UserInfo          model.UserInfo      `json:"userInfo"`
	ModelSource       string              `json:"modelSource"`
	PlatformModelID   uint                `json:"platformModelId"`
	ModelURI          string              `json:"modelURI"`
	ServedModel       string              `json:"servedModel"`
	BackendType       string              `json:"backendType"`
	CacheURI          string              `json:"cacheURI"`
	Replicas          int64               `json:"replicas"`
	Queue             string              `json:"queue"`
	Layout            string              `json:"layout"`
	Port              int64               `json:"port"`
	Roles             []kthena.RoleSpec   `json:"roles"`
	TotalResources    corev1.ResourceList `json:"totalResources"`
	Phase             string              `json:"phase"`
	Conditions        []map[string]any    `json:"conditions"`
	Resources         []KthenaResource    `json:"resources"`
	RuntimePods       []KthenaRuntimePod  `json:"runtimePods"`
	Diagnostics       []KthenaDiagnostic  `json:"diagnostics"`
	Access            KthenaAccess        `json:"access"`
	Labels            map[string]string   `json:"labels"`
	CreationTimestamp time.Time           `json:"createdAt"`
}

type KthenaResource struct {
	Kind       string           `json:"kind"`
	Name       string           `json:"name"`
	Namespace  string           `json:"namespace"`
	Phase      string           `json:"phase"`
	Ready      bool             `json:"ready"`
	Conditions []map[string]any `json:"conditions"`
}

type KthenaRuntimePod struct {
	Group           string `json:"group"`
	Role            string `json:"role"`
	Instance        string `json:"instance"`
	Entry           bool   `json:"entry"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	NodeName        string `json:"nodeName"`
	PodIP           string `json:"podIP,omitempty"`
	HostIP          string `json:"hostIP,omitempty"`
	Phase           string `json:"phase"`
	Ready           bool   `json:"ready"`
	Restarts        int32  `json:"restarts"`
	ReadyContainers int    `json:"readyContainers"`
	TotalContainers int    `json:"totalContainers"`
}

type KthenaAccess struct {
	ModelName       string `json:"modelName"`
	ProxyBaseURL    string `json:"proxyBaseURL"`
	InternalBaseURL string `json:"internalBaseURL"`
	NodePortURL     string `json:"nodePortURL,omitempty"`
	RouterService   string `json:"routerService"`
	RouteName       string `json:"routeName,omitempty"`
	ServerName      string `json:"serverName,omitempty"`
}

type KthenaDiagnostic struct {
	Level     string    `json:"level"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Pod       string    `json:"pod,omitempty"`
	Container string    `json:"container,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}
