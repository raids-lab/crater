/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

const (
	tensorboardLogDirEnv     = "TENSORBOARD_LOGDIR"
	multiSourceLogRoot       = "/tensorboard-runs"
	personalVolumeName       = "tensorboard-personal"
	maxTensorboardSourceJobs = 10
	maxActiveTensorboards    = 10
	tensorboardHTTPPortName  = "http"
	tensorboardAuthPath      = "/api/tensorboard/auth"
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

func isEligibleSourceJob(jobDB *model.Job) bool {
	return jobDB != nil && jobDB.JobType != model.JobType(labelKeyTypeTensorboard)
}

func getSourceJob(ctx context.Context, jobName string, userID uint) (*model.Job, error) {
	jobDB, err := query.Job.WithContext(ctx).
		Where(query.Job.JobName.Eq(jobName), query.Job.UserID.Eq(userID)).
		First()
	if err != nil {
		return nil, err
	}
	if !isEligibleSourceJob(jobDB) {
		return nil, gorm.ErrRecordNotFound
	}
	return jobDB, nil
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

	expiration := deploy.Annotations[annotationKeyExpirationTime]
	if expiration == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, expiration)
	if err != nil {
		return true
	}
	return now.Before(expiresAt)
}

func isTensorboardOwner(object metav1.Object, username string) bool {
	labels := object.GetLabels()
	return labels != nil && labels[crclient.LabelKeyTaskUser] == username
}

func isActiveTensorboardJob(job *batch.Job) bool {
	if job.DeletionTimestamp != nil {
		return false
	}
	switch string(job.Status.State.Phase) {
	case "Completed", "Failed", "Aborted", "Terminated":
		return false
	default:
		return true
	}
}

func (svc *TensorboardService) activeTensorboardCount(ctx context.Context, username string) (int, error) {
	cfg := config.GetConfig()
	labels := client.MatchingLabels{
		crclient.LabelKeyTaskType: labelKeyTypeTensorboard,
		crclient.LabelKeyTaskUser: username,
	}
	var jobList batch.JobList
	if err := svc.crClient.List(ctx, &jobList,
		client.InNamespace(cfg.Namespaces.Job),
		labels,
	); err != nil {
		return 0, err
	}

	var deployList appsv1.DeploymentList
	if err := svc.crClient.List(ctx, &deployList,
		client.InNamespace(cfg.Namespaces.Job),
		labels,
	); err != nil {
		return 0, err
	}

	count := 0
	for i := range jobList.Items {
		if isActiveTensorboardJob(&jobList.Items[i]) {
			count++
		}
	}
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

func buildPersonalStorage(
	user *model.User,
	logDir string,
	claimName string,
	userStoragePrefix string,
) (*tensorboardStorage, error) {
	logDir = strings.TrimSpace(logDir)
	if logDir == "" {
		return nil, bizerr.BadRequest.MissingParameter.New("provide a TensorBoard log directory")
	}
	if !path.IsAbs(logDir) {
		return nil, bizerr.BadRequest.ParameterError.New("log directory must be an absolute path")
	}

	cleanLogDir := path.Clean(logDir)
	personalMountPath := path.Join("/home", user.Name)
	if !pathWithinMount(cleanLogDir, personalMountPath) {
		return nil, bizerr.BadRequest.ParameterError.New(
			"a panel without a source job can only read the current user's personal workspace",
		)
	}
	if strings.TrimSpace(claimName) == "" || strings.TrimSpace(user.Space) == "" {
		return nil, bizerr.Internal.K8sServiceError.New("personal storage is not configured")
	}

	return &tensorboardStorage{
		logDir: cleanLogDir,
		volumes: []corev1.Volume{{
			Name: personalVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		}},
		volumeMounts: []corev1.VolumeMount{{
			Name:      personalVolumeName,
			MountPath: personalMountPath,
			SubPath:   path.Join(userStoragePrefix, user.Space),
			ReadOnly:  true,
		}},
	}, nil
}

func (svc *TensorboardService) preparePersonalStorage(
	ctx context.Context,
	userID uint,
	logDir string,
) (*tensorboardStorage, error) {
	user, err := query.User.WithContext(ctx).Where(query.User.ID.Eq(userID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerr.NotFound.DataBaseNotFound.Wrap(err, "user not found")
		}
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "get user storage failed")
	}

	cfg := config.GetConfig()
	claimName := cfg.Storage.PVC.ReadWriteMany
	if cfg.Storage.PVC.ReadOnlyMany != nil && strings.TrimSpace(*cfg.Storage.PVC.ReadOnlyMany) != "" {
		claimName = *cfg.Storage.PVC.ReadOnlyMany
	}
	return buildPersonalStorage(user, logDir, claimName, cfg.Storage.Prefix.User)
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
		return svc.preparePersonalStorage(ctx, userID, req.LogDir)
	case 1:
		return svc.prepareSingleSourceStorage(ctx, userID, sources[0])
	default:
		return svc.prepareMultiSourceStorage(ctx, userID, sources)
	}
}

func (svc *TensorboardService) createTensorboardJob(
	ctx context.Context,
	token interutil.JWTMessage,
	tbID string,
	ingressPrefixPath string,
	storage *tensorboardStorage,
) (*batch.Job, error) {
	cfg := config.GetConfig()
	if strings.TrimSpace(cfg.Tensorboard.Image) == "" || cfg.Tensorboard.ImagePullPolicy == "" {
		return nil, bizerr.Internal.K8sServiceError.New("TensorBoard image configuration is incomplete")
	}

	imagePullSecrets := make([]corev1.LocalObjectReference, 0, len(cfg.Tensorboard.ImagePullSecrets))
	for _, secret := range cfg.Tensorboard.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: secret.Name})
	}
	if err := vcqueue.EnsureAccountQueueExists(ctx, svc.crClient, token, token.AccountID); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "ensure TensorBoard account queue failed")
	}
	if err := vcqueue.EnsureUserQueueExists(ctx, svc.crClient, token, token.AccountID, token.UserID); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "ensure TensorBoard user queue failed")
	}

	builder := newJobBuilder(cfg.Namespaces.Job, &workloadConfig{
		Image:            cfg.Tensorboard.Image,
		ImagePullPolicy:  corev1.PullPolicy(cfg.Tensorboard.ImagePullPolicy),
		ImagePullSecrets: imagePullSecrets,
		NodeSelector:     cfg.Tensorboard.NodeSelector,
		Tolerations:      cfg.Tensorboard.Tolerations,
		Affinity:         cfg.Tensorboard.Affinity,
	})
	job := builder.buildJob(
		tbID,
		token.Username,
		vcqueue.ResolveJobQueueName(token),
		storage.logDir,
		ingressPrefixPath,
		storage.volumes,
		storage.volumeMounts,
	)
	if err := svc.crClient.Create(ctx, job); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "create TensorBoard Volcano Job failed")
	}
	return job, nil
}

func (svc *TensorboardService) Create(
	ctx context.Context,
	token interutil.JWTMessage,
	req *payload.CreateTensorboardReq,
) (*payload.CreateTensorboardResp, error) {
	activeCount, err := svc.activeTensorboardCount(ctx, token.Username)
	if err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "check TensorBoard panel quota failed")
	}
	if activeCount >= maxActiveTensorboards {
		return nil, bizerr.Conflict.ResourceStatusError.New(fmt.Sprintf(
			"you can have at most %d active TensorBoard panels; delete one or wait for it to expire",
			maxActiveTensorboards,
		))
	}

	storage, err := svc.prepareStorage(ctx, token.UserID, req)
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
	prefix := fmt.Sprintf("%s-%s", token.Username, tbID) // Use an exclusive route prefix for each panel.
	ingressPrefixPath := fmt.Sprintf("/ingress/%s", prefix)

	job, err := svc.createTensorboardJob(ctx, token, tbID, ingressPrefixPath, storage)
	if err != nil {
		return nil, err
	}

	// Use owner references so network resources are garbage-collected with the Volcano Job.
	ownerRefs := []metav1.OwnerReference{
		*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
	}

	// TensorBoard listens on port 6006 inside the container.
	port := &corev1.ServicePort{
		Name:       tensorboardHTTPPortName,
		Port:       tensorboardPort,
		TargetPort: intstr.FromInt(tensorboardPort),
		Protocol:   corev1.ProtocolTCP,
	}

	// Reuse ServiceManager to create the service and ingress.
	cfg := config.GetConfig()
	host := cfg.Host // Global domain mapped from server config
	ingressOptions := make([]crclient.IngressOptions, 0, 1)
	if cfg.Tensorboard.IsIngressAuthEnabled() {
		ingressOptions = append(ingressOptions, crclient.IngressOptions{Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/auth-url":    "https://$host" + tensorboardAuthPath,
			"nginx.ingress.kubernetes.io/auth-method": "GET",
		}})
	}
	urlPath, err := svc.serviceManager.CreateIngressWithPrefix(
		ctx,
		ownerRefs,
		job.Labels,
		port,
		host,
		prefix,
		ingressOptions...,
	)
	if err != nil {
		_ = svc.crClient.Delete(ctx, job)
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

type tensorboardPanel struct {
	job        *batch.Job
	deployment *appsv1.Deployment
}

func (svc *TensorboardService) getTensorboardPanel(ctx context.Context, tbID string) (*tensorboardPanel, error) {
	cfg := config.GetConfig()
	key := client.ObjectKey{Namespace: cfg.Namespaces.Job, Name: fmt.Sprintf("tb-%s", tbID)}

	var job batch.Job
	if err := svc.crClient.Get(ctx, key, &job); err == nil {
		return &tensorboardPanel{job: &job}, nil
	} else if !k8serrors.IsNotFound(err) {
		return nil, wrapTensorboardLookupError(err)
	}

	var deploy appsv1.Deployment
	if err := svc.crClient.Get(ctx, key, &deploy); err != nil {
		return nil, wrapTensorboardLookupError(err)
	}
	return &tensorboardPanel{deployment: &deploy}, nil
}

func (panel *tensorboardPanel) object() client.Object {
	if panel.job != nil {
		return panel.job
	}
	return panel.deployment
}

func (svc *TensorboardService) getTensorboardPod(ctx context.Context, tbID string) (*corev1.Pod, error) {
	var pods corev1.PodList
	err := svc.crClient.List(ctx, &pods,
		client.InNamespace(config.GetConfig().Namespaces.Job),
		client.MatchingLabels{
			labelKeyTensorboardID:     tbID,
			crclient.LabelKeyTaskType: labelKeyTypeTensorboard,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, nil
	}
	return &pods.Items[0], nil
}

// GetAccessPath verifies panel ownership and activity before creating a browser session.
func (svc *TensorboardService) GetAccessPath(
	ctx context.Context,
	username string,
	tbID string,
) (string, error) {
	cfg := config.GetConfig()
	panel, err := svc.getTensorboardPanel(ctx, tbID)
	if err != nil {
		return "", err
	}
	if !isTensorboardOwner(panel.object(), username) {
		return "", bizerr.Forbidden.PermissionDenied.New(
			"you do not have permission to access this TensorBoard panel",
		)
	}
	if panel.job != nil && !isActiveTensorboardJob(panel.job) {
		return "", bizerr.Conflict.ResourceStatusError.New(
			"this TensorBoard panel is no longer running",
		)
	}
	if panel.job != nil {
		pod, podErr := svc.getTensorboardPod(ctx, tbID)
		if podErr != nil {
			return "", bizerr.Internal.K8sServiceError.Wrap(
				podErr,
				"check TensorBoard Pod failed",
			)
		}
		if tensorboardPodExpired(pod) ||
			(pod != nil && pod.Status.StartTime != nil &&
				time.Now().After(pod.Status.StartTime.Add(
					time.Duration(tensorboardMaxRuntimeSeconds)*time.Second,
				))) {
			return "", bizerr.Conflict.ResourceStatusError.New(
				"this TensorBoard panel has expired",
			)
		}
	}
	if panel.deployment != nil && !isActiveTensorboard(panel.deployment, time.Now()) {
		return "", bizerr.Conflict.ResourceStatusError.New(
			"this TensorBoard panel has expired",
		)
	}

	return fmt.Sprintf("https://%s/ingress/%s-%s", cfg.Host, username, tbID), nil
}

func (svc *TensorboardService) Delete(ctx context.Context, username, tbID string) error {
	panel, err := svc.getTensorboardPanel(ctx, tbID)
	if err != nil {
		return err
	}

	if !isTensorboardOwner(panel.object(), username) {
		return bizerr.Forbidden.PermissionDenied.New(
			"you do not have permission to delete this TensorBoard panel",
		)
	}

	if err := svc.crClient.Delete(ctx, panel.object()); err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(err, "delete TensorBoard panel failed")
	}

	return nil
}

func getDeploymentStatus(deploy *appsv1.Deployment) (payload.TensorboardStatus, payload.TensorboardStatusReason, string) {
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

func tensorboardPodReady(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func tensorboardPodExpired(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}
	if pod.Status.Reason == "DeadlineExceeded" {
		return true
	}
	for i := range pod.Status.ContainerStatuses {
		containerStatus := &pod.Status.ContainerStatuses[i]
		if containerStatus.State.Terminated != nil &&
			containerStatus.State.Terminated.Reason == "DeadlineExceeded" {
			return true
		}
	}
	return false
}

func getJobStatus(
	job *batch.Job,
	pod *corev1.Pod,
) (payload.TensorboardStatus, payload.TensorboardStatusReason, string) {
	switch job.Status.State.Phase {
	case batch.Completed, batch.Failed, batch.Aborted, batch.Terminated,
		batch.Aborting, batch.Completing, batch.Terminating:
		if tensorboardPodExpired(pod) {
			return payload.TensorboardStatusExpired,
				payload.TensorboardStatusReasonRuntimeExpired,
				"The panel reached its four-day runtime limit and is being removed."
		}
		return payload.TensorboardStatusFailed,
			payload.TensorboardStatusReasonJobFailed,
			"The panel stopped unexpectedly. Check the Pod events or contact an administrator."
	case batch.Running:
		if tensorboardPodReady(pod) {
			return payload.TensorboardStatusReady,
				payload.TensorboardStatusReasonReady,
				"The panel is ready."
		}
		return payload.TensorboardStatusStarting,
			payload.TensorboardStatusReasonPodStarting,
			"The panel was scheduled and TensorBoard is starting."
	case batch.Restarting:
		return payload.TensorboardStatusStarting,
			payload.TensorboardStatusReasonPodStarting,
			"The panel is restarting."
	default:
		return payload.TensorboardStatusPending,
			payload.TensorboardStatusReasonWaitingForSchedule,
			"The panel is waiting for resources in your scheduling queue."
	}
}

func tensorboardExpiration(pod *corev1.Pod) string {
	if pod == nil || pod.Status.StartTime == nil {
		return ""
	}
	return pod.Status.StartTime.Add(
		time.Duration(tensorboardMaxRuntimeSeconds) * time.Second,
	).Format(time.RFC3339)
}

func (svc *TensorboardService) List(
	ctx context.Context,
	username string,
) ([]payload.TensorboardInfo, error) {
	cfg := config.GetConfig()
	ns := cfg.Namespaces.Job

	selector := client.MatchingLabels{
		crclient.LabelKeyTaskUser: username,
		crclient.LabelKeyTaskType: labelKeyTypeTensorboard,
	}
	var jobList batch.JobList
	if err := svc.crClient.List(ctx, &jobList,
		client.InNamespace(ns),
		selector,
	); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "list TensorBoard Volcano Jobs failed")
	}

	var podList corev1.PodList
	if err := svc.crClient.List(ctx, &podList,
		client.InNamespace(ns),
		selector,
	); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "list TensorBoard Pods failed")
	}
	podsByID := make(map[string]*corev1.Pod, len(podList.Items))
	for i := range podList.Items {
		pod := &podList.Items[i]
		podsByID[pod.Labels[labelKeyTensorboardID]] = pod
	}

	var deployList appsv1.DeploymentList
	if err := svc.crClient.List(ctx, &deployList,
		client.InNamespace(ns),
		selector,
	); err != nil {
		return nil, bizerr.Internal.K8sServiceError.Wrap(err, "list TensorBoard panels failed")
	}

	// New panels are Volcano Jobs. Legacy Deployments remain visible during migration.
	items := make([]payload.TensorboardInfo, 0, len(jobList.Items)+len(deployList.Items))
	for i := range jobList.Items {
		job := &jobList.Items[i]
		tbID := job.Labels[labelKeyTensorboardID]
		pod := podsByID[tbID]
		status, statusReason, statusMessage := getJobStatus(job, pod)
		items = append(items, payload.TensorboardInfo{
			ID:            tbID,
			Expiration:    tensorboardExpiration(pod),
			CreatedAt:     job.CreationTimestamp.Format(time.RFC3339),
			AccessPath:    fmt.Sprintf("https://%s/ingress/%s-%s", cfg.Host, username, tbID),
			Status:        status,
			StatusReason:  statusReason,
			StatusMessage: statusMessage,
		})
	}
	for i := range deployList.Items {
		deploy := &deployList.Items[i]
		tbID := deploy.Labels[labelKeyTensorboardID]
		status, statusReason, statusMessage := getDeploymentStatus(deploy)
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
