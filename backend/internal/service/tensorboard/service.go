package tensorboard

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

const (
	tensorboardLogDirEnv     = "TENSORBOARD_LOGDIR"
	multiSourceLogRoot       = "/tensorboard-runs"
	maxTensorboardSourceJobs = 10
	maxActiveTensorboards    = 10
	tensorboardHTTPPortName  = "http"
)

type TensorboardService struct {
	crClient       client.Client
	serviceManager crclient.ServiceManagerInterface
}

func NewTensorboardService(
	crClient client.Client,
	serviceManager crclient.ServiceManagerInterface,
) *TensorboardService {
	return &TensorboardService{
		crClient:       crClient,
		serviceManager: serviceManager,
	}
}

func getTensorboardLogDir(jobDB *model.Job) string {
	job := jobDB.Attributes.Data()
	if job == nil {
		return ""
	}

	for taskIndex := range job.Spec.Tasks {
		task := &job.Spec.Tasks[taskIndex]
		for containerIndex := range task.Template.Spec.Containers {
			container := &task.Template.Spec.Containers[containerIndex]
			for _, env := range container.Env {
				if env.Name == tensorboardLogDirEnv {
					return strings.TrimSpace(env.Value)
				}
			}
		}
	}
	return ""
}

func getSourceJob(ctx context.Context, jobName string, userID uint) (*model.Job, error) {
	return query.Job.WithContext(ctx).
		Where(query.Job.JobName.Eq(jobName), query.Job.UserID.Eq(userID)).
		First()
}

func wrapSourceJobLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerr.NotFound.DataBaseNotFound.Wrap(err, "source job not found")
	}
	return bizerr.Internal.DatabaseError.Wrap(err, "get source job failed")
}

func wrapTensorboardLookupError(err error) error {
	if k8serrors.IsNotFound(err) {
		return bizerr.NotFound.K8sResourceNotFound.Wrap(err, "TensorBoard panel not found")
	}
	return bizerr.Internal.K8sServiceError.Wrap(err, "get TensorBoard panel failed")
}

func isLogDirMounted(logDir string, volumeMounts []corev1.VolumeMount) bool {
	for _, mount := range volumeMounts {
		if pathWithinMount(logDir, mount.MountPath) {
			return true
		}
	}
	return false
}

func isActiveTensorboard(deploy *appsv1.Deployment, now time.Time) bool {
	if deploy.DeletionTimestamp != nil {
		return false
	}

	expiration := deploy.Annotations[interutil.AnnotationKeyExpirationTime]
	if expiration == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, expiration)
	if err != nil {
		return true
	}
	return now.Before(expiresAt)
}

func isTensorboardOwner(deploy *appsv1.Deployment, username string) bool {
	return deploy.Labels != nil && deploy.Labels[crclient.LabelKeyTaskUser] == username
}

func (svc *TensorboardService) activeTensorboardCount(ctx context.Context, username string) (int, error) {
	cfg := config.GetConfig()
	var deployList appsv1.DeploymentList
	if err := svc.crClient.List(ctx, &deployList,
		client.InNamespace(cfg.Namespaces.Job),
		client.MatchingLabels{
			crclient.LabelKeyTaskType: interutil.LabelKeyTypeTensorboard,
			crclient.LabelKeyTaskUser: username,
		},
	); err != nil {
		return 0, err
	}

	count := 0
	now := time.Now()
	for i := range deployList.Items {
		if isActiveTensorboard(&deployList.Items[i], now) {
			count++
		}
	}
	return count, nil
}

type sourceJobConfig struct {
	name   string
	logDir string
}

func sourceJobs(req *payload.CreateTensorboardReq) []sourceJobConfig {
	sources := make([]sourceJobConfig, 0, len(req.SourceJobs))
	if len(req.SourceJobs) > 0 {
		for _, source := range req.SourceJobs {
			sources = append(sources, sourceJobConfig{
				name:   source.JobName,
				logDir: source.LogDir,
			})
		}
	} else {
		names := req.SourceJobNames
		if len(names) == 0 && strings.TrimSpace(req.SourceJobName) != "" {
			names = []string{req.SourceJobName}
		}
		for _, name := range names {
			sources = append(sources, sourceJobConfig{name: name})
		}
	}

	seen := make(map[string]struct{}, len(sources))
	result := make([]sourceJobConfig, 0, len(sources))
	for _, source := range sources {
		source.name = strings.TrimSpace(source.name)
		source.logDir = strings.TrimSpace(source.logDir)
		if source.name == "" {
			continue
		}
		if _, ok := seen[source.name]; ok {
			continue
		}
		seen[source.name] = struct{}{}
		result = append(result, source)
	}

	// Legacy clients provide the single-source override through top-level LogDir.
	if len(result) == 1 && result[0].logDir == "" {
		result[0].logDir = strings.TrimSpace(req.LogDir)
	}
	return result
}

func getSourceStorage(jobDB *model.Job) ([]corev1.Volume, []corev1.VolumeMount, error) {
	job := jobDB.Attributes.Data()
	if job == nil || len(job.Spec.Tasks) == 0 {
		return nil, nil, bizerr.BadRequest.ParameterError.New("job has no usable pod configuration")
	}

	podSpec := job.Spec.Tasks[0].Template.Spec
	if len(podSpec.Containers) == 0 {
		return nil, nil, bizerr.BadRequest.ParameterError.New("job has no usable container configuration")
	}
	return podSpec.Volumes, podSpec.Containers[0].VolumeMounts, nil
}

func pathWithinMount(targetPath, mountPath string) bool {
	cleanTarget := path.Clean(targetPath)
	cleanMount := path.Clean(mountPath)
	if cleanMount == "/" {
		return path.IsAbs(cleanTarget)
	}
	return cleanTarget == cleanMount || strings.HasPrefix(cleanTarget, cleanMount+"/")
}

func getLogStorage(jobDB *model.Job, logDir string) (corev1.Volume, corev1.VolumeMount, error) {
	volumes, mounts, err := getSourceStorage(jobDB)
	if err != nil {
		return corev1.Volume{}, corev1.VolumeMount{}, err
	}

	cleanLogDir := path.Clean(logDir)
	selectedMount := -1
	for i := range mounts {
		cleanMountPath := path.Clean(mounts[i].MountPath)
		if pathWithinMount(cleanLogDir, cleanMountPath) &&
			(selectedMount == -1 || len(cleanMountPath) > len(path.Clean(mounts[selectedMount].MountPath))) {
			selectedMount = i
		}
	}
	if selectedMount == -1 {
		return corev1.Volume{}, corev1.VolumeMount{}, bizerr.BadRequest.ParameterError.New(
			"log directory is not inside a job data mount",
		)
	}

	sourceMount := mounts[selectedMount]
	if sourceMount.SubPathExpr != "" {
		return corev1.Volume{}, corev1.VolumeMount{}, bizerr.BadRequest.ParameterError.New(
			"dynamic SubPathExpr mounts are not supported",
		)
	}

	for i := range volumes {
		if volumes[i].Name == sourceMount.Name {
			sourceMount.ReadOnly = true
			return volumes[i], sourceMount, nil
		}
	}

	return corev1.Volume{}, corev1.VolumeMount{}, bizerr.BadRequest.ParameterError.New(
		"volume for the log directory was not found",
	)
}

func buildRunMount(jobDB *model.Job, jobName string, index int, requestedLogDir string) (corev1.Volume, corev1.VolumeMount, error) {
	logDir := strings.TrimSpace(requestedLogDir)
	if logDir == "" {
		logDir = getTensorboardLogDir(jobDB)
	}
	if logDir == "" {
		return corev1.Volume{}, corev1.VolumeMount{}, bizerr.BadRequest.ParameterError.New(
			"job does not declare " + tensorboardLogDirEnv + " and no log directory was provided",
		)
	}
	if !path.IsAbs(logDir) {
		return corev1.Volume{}, corev1.VolumeMount{}, bizerr.BadRequest.ParameterError.New(
			"log directory must be an absolute path",
		)
	}

	sourceVolume, sourceMount, err := getLogStorage(jobDB, logDir)
	if err != nil {
		return corev1.Volume{}, corev1.VolumeMount{}, err
	}

	cleanLogDir := path.Clean(logDir)
	relativeLogDir := strings.TrimPrefix(cleanLogDir, path.Clean(sourceMount.MountPath))
	relativeLogDir = strings.TrimPrefix(relativeLogDir, "/")
	subPath := path.Join(sourceMount.SubPath, relativeLogDir)
	if subPath == "." {
		subPath = ""
	}

	volumeName := fmt.Sprintf("tb-run-%d", index)
	return corev1.Volume{
			Name:         volumeName,
			VolumeSource: sourceVolume.VolumeSource,
		}, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: path.Join(multiSourceLogRoot, jobName),
			SubPath:   subPath,
			ReadOnly:  true,
		}, nil
}

func (svc *TensorboardService) GetSourceConfig(
	ctx context.Context,
	userID uint,
	jobName string,
) (*payload.TensorboardSourceConfigResp, error) {
	jobDB, err := getSourceJob(ctx, jobName, userID)
	if err != nil {
		return nil, wrapSourceJobLookupError(err)
	}

	return &payload.TensorboardSourceConfigResp{
		LogDir: getTensorboardLogDir(jobDB),
	}, nil
}

type tensorboardStorage struct {
	logDir       string
	volumes      []corev1.Volume
	volumeMounts []corev1.VolumeMount
	multiSource  bool
}

func (svc *TensorboardService) prepareSingleSourceStorage(
	ctx context.Context,
	userID uint,
	source sourceJobConfig,
) (*tensorboardStorage, error) {
	jobDB, err := getSourceJob(ctx, source.name, userID)
	if err != nil {
		return nil, wrapSourceJobLookupError(err)
	}

	logDir := source.logDir
	if configuredLogDir := getTensorboardLogDir(jobDB); logDir == "" && configuredLogDir != "" {
		logDir = configuredLogDir
	}
	if logDir == "" {
		return nil, bizerr.BadRequest.MissingParameter.New(
			"the source job does not declare TENSORBOARD_LOGDIR; provide a log directory",
		)
	}

	volume, mount, err := getLogStorage(jobDB, logDir)
	if err != nil {
		return nil, bizerr.BadRequest.ParameterError.Wrap(
			err,
			"the source job log directory is not accessible from its data mounts",
		)
	}

	return &tensorboardStorage{
		logDir:       logDir,
		volumes:      []corev1.Volume{volume},
		volumeMounts: []corev1.VolumeMount{mount},
	}, nil
}

func (svc *TensorboardService) prepareMultiSourceStorage(
	ctx context.Context,
	userID uint,
	sources []sourceJobConfig,
) (*tensorboardStorage, error) {
	volumes := make([]corev1.Volume, 0, len(sources))
	volumeMounts := make([]corev1.VolumeMount, 0, len(sources))
	for i, source := range sources {
		jobDB, err := getSourceJob(ctx, source.name, userID)
		if err != nil {
			return nil, wrapSourceJobLookupError(err)
		}
		volume, mount, err := buildRunMount(jobDB, source.name, i, source.logDir)
		if err != nil {
			return nil, bizerr.BadRequest.ParameterError.Wrap(
				err,
				fmt.Sprintf("source job %q cannot be used by TensorBoard", source.name),
			)
		}
		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, mount)
	}

	return &tensorboardStorage{
		logDir:       multiSourceLogRoot,
		volumes:      volumes,
		volumeMounts: volumeMounts,
		multiSource:  true,
	}, nil
}

func (svc *TensorboardService) prepareStorage(
	ctx context.Context,
	userID uint,
	req *payload.CreateTensorboardReq,
) (*tensorboardStorage, error) {
	sources := sourceJobs(req)
	if len(sources) > maxTensorboardSourceJobs {
		return nil, bizerr.BadRequest.ParameterError.New(fmt.Sprintf(
			"a TensorBoard panel can reference at most %d source jobs",
			maxTensorboardSourceJobs,
		))
	}

	switch len(sources) {
	case 0:
		return nil, bizerr.BadRequest.MissingParameter.New("at least one source job is required")
	case 1:
		return svc.prepareSingleSourceStorage(ctx, userID, sources[0])
	default:
		return svc.prepareMultiSourceStorage(ctx, userID, sources)
	}
}

func (svc *TensorboardService) Create(
	ctx context.Context,
	userID uint,
	username string,
	req *payload.CreateTensorboardReq,
) (*payload.CreateTensorboardResp, error) {
	activeCount, err := svc.activeTensorboardCount(ctx, username)
	if err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "check TensorBoard panel quota failed")
	}
	if activeCount >= maxActiveTensorboards {
		return nil, bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf(
			"you can have at most %d active TensorBoard panels; delete one or wait for it to expire",
			maxActiveTensorboards,
		))
	}

	storage, err := svc.prepareStorage(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	if len(storage.volumes) == 0 {
		return nil, bizerr.BadRequest.ParameterError.New(
			"the selected source jobs do not provide an accessible data mount",
		)
	}
	if storage.logDir == "" {
		return nil, bizerr.BadRequest.MissingParameter.New(
			"the source job does not declare TENSORBOARD_LOGDIR; provide a log directory",
		)
	}
	if !path.IsAbs(storage.logDir) ||
		(!storage.multiSource && !isLogDirMounted(storage.logDir, storage.volumeMounts)) {
		return nil, bizerr.BadRequest.ParameterError.New(
			"the TensorBoard log directory must be an absolute path inside a source job data mount",
		)
	}

	tbID := uuid.New().String()[:8]
	prefix := fmt.Sprintf("%s-%s", username, tbID) // Use an exclusive route prefix for each panel.
	ingressPrefixPath := fmt.Sprintf("/ingress/%s", prefix)

	cfg := config.GetConfig()
	ns := cfg.Namespaces.Job
	builder := interutil.NewBuilder(ns)

	// Build the Kubernetes deployment.
	deploy := builder.BuildDeployment(
		tbID,
		username,
		storage.logDir,
		ingressPrefixPath,
		req.TTLHours,
		storage.volumes,
		storage.volumeMounts,
	)

	if err := svc.crClient.Create(ctx, deploy); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "create TensorBoard deployment failed")
	}

	// Use owner references so network resources are garbage-collected with the deployment.
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(deploy, appsv1.SchemeGroupVersion.WithKind("Deployment")),
	}

	// TensorBoard listens on port 6006 inside the container.
	port := &corev1.ServicePort{
		Name:       tensorboardHTTPPortName,
		Port:       interutil.TensorboardPort,
		TargetPort: intstr.FromInt(interutil.TensorboardPort),
		Protocol:   corev1.ProtocolTCP,
	}

	// Reuse ServiceManager to create the service and ingress.
	host := cfg.Host // Global domain mapped from server config
	urlPath, err := svc.serviceManager.CreateIngressWithPrefix(
		ctx,
		ownerRefs,
		deploy.Labels, // Select the pods mapped to the deployment
		port,
		host,
		prefix,
	)
	if err != nil {
		_ = svc.crClient.Delete(ctx, deploy)
		return nil, bizerr.Internal.K8sServiceError.Wrap(
			err,
			"create TensorBoard service and ingress failed",
		)
	}

	return &payload.CreateTensorboardResp{
		TensorboardID: tbID,
		AccessPath:    urlPath,
	}, nil
}

func (svc *TensorboardService) ExtendTTL(
	ctx context.Context,
	username string,
	tbID string,
	req *payload.ExtendTTLReq,
) (string, error) {
	cfg := config.GetConfig()
	ns := cfg.Namespaces.Job

	// Find the deployment for this panel.
	var deploy appsv1.Deployment
	err := svc.crClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: fmt.Sprintf("tb-%s", tbID)}, &deploy)
	if err != nil {
		return "", wrapTensorboardLookupError(err)
	}

	// Verify ownership before mutating the panel.
	if !isTensorboardOwner(&deploy, username) {
		return "", bizerr.Forbidden.PermissionDenied.New(
			"you do not have permission to modify this TensorBoard panel",
		)
	}

	// Calculate the new expiration time from now.
	newExpiration := time.Now().Add(time.Duration(req.TTLHours) * time.Hour).Format(time.RFC3339)
	if deploy.Annotations == nil {
		deploy.Annotations = make(map[string]string)
	}
	deploy.Annotations["crater.raids.io/expiration-time"] = newExpiration

	if err := svc.crClient.Update(ctx, &deploy); err != nil {
		return "", bizerr.Internal.K8sServiceError.Wrap(
			err,
			"update TensorBoard panel expiration failed",
		)
	}

	return newExpiration, nil
}

func (svc *TensorboardService) Delete(ctx context.Context, username, tbID string) error {
	cfg := config.GetConfig()
	ns := cfg.Namespaces.Job

	var deploy appsv1.Deployment
	err := svc.crClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: fmt.Sprintf("tb-%s", tbID)}, &deploy)
	if err != nil {
		return wrapTensorboardLookupError(err)
	}

	if !isTensorboardOwner(&deploy, username) {
		return bizerr.Forbidden.PermissionDenied.New(
			"you do not have permission to delete this TensorBoard panel",
		)
	}

	if err := svc.crClient.Delete(ctx, &deploy); err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(err, "delete TensorBoard panel failed")
	}

	return nil
}

func getStatus(deploy *appsv1.Deployment) (payload.TensorboardStatus, payload.TensorboardStatusReason, string) {
	for _, condition := range deploy.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse {
			return payload.TensorboardStatusFailed,
				payload.TensorboardStatusReasonDeploymentFailed,
				"The panel failed to start. Check the Pod events or contact an administrator."
		}
	}

	if deploy.Status.AvailableReplicas == 0 || deploy.Status.ReadyReplicas == 0 {
		return payload.TensorboardStatusStarting,
			payload.TensorboardStatusReasonDeploymentStarting,
			"The panel is starting. The first startup may take several minutes."
	}

	return payload.TensorboardStatusReady, payload.TensorboardStatusReasonReady, "The panel is ready."
}

func (svc *TensorboardService) List(
	ctx context.Context,
	username string,
) ([]payload.TensorboardInfo, error) {
	cfg := config.GetConfig()
	ns := cfg.Namespaces.Job

	var deployList appsv1.DeploymentList
	err := svc.crClient.List(ctx, &deployList,
		client.InNamespace(ns),
		client.MatchingLabels{
			crclient.LabelKeyTaskUser: username,
			crclient.LabelKeyTaskType: "tensorboard",
		},
	)
	if err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "list TensorBoard panels failed")
	}

	// Build the response from the owned deployments.
	items := make([]payload.TensorboardInfo, 0, len(deployList.Items))
	for i := range deployList.Items {
		deploy := &deployList.Items[i]
		tbID := deploy.Labels["crater.raids.io/tensorboard-id"]
		status, statusReason, statusMessage := getStatus(deploy)
		items = append(items, payload.TensorboardInfo{
			ID:            tbID,
			Expiration:    deploy.Annotations["crater.raids.io/expiration-time"],
			CreatedAt:     deploy.CreationTimestamp.Format(time.RFC3339),
			AccessPath:    fmt.Sprintf("https://%s/ingress/%s-%s", cfg.Host, username, tbID),
			Status:        status,
			StatusReason:  statusReason,
			StatusMessage: statusMessage,
		})
	}

	return items, nil
}
