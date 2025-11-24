package reconciler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
)

// ModelDownloadReconciler reconciles model download Jobs
type ModelDownloadReconciler struct {
	client.Client
	KubeClient kubernetes.Interface
	Scheme     *runtime.Scheme
	log        logr.Logger
}

// NewModelDownloadReconciler returns a new reconciler
func NewModelDownloadReconciler(crClient client.Client, kubeClient kubernetes.Interface, scheme *runtime.Scheme) *ModelDownloadReconciler {
	return &ModelDownloadReconciler{
		Client:     crClient,
		KubeClient: kubeClient,
		Scheme:     scheme,
		log:        ctrl.Log.WithName("ModelDownload-reconciler"),
	}
}

// SetupWithManager sets up the controller with the Manager
func (r *ModelDownloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("modeldownload-reconciler").
		For(&batchv1.Job{}).
		Owns(&v1.Pod{}).
		WithOptions(controller.Options{}).
		Complete(r)
}

var (
	PlatformSpace = config.GetConfig().Namespaces.Job
)

const (
	podLogTailLines               int64 = 500
	logVerboseLevelDebug                = 4
	bytesInKB                     int64 = 1024
	bytesInMB                           = bytesInKB * 1024
	bytesInGB                           = bytesInMB * 1024
	progressReportIntervalSeconds int64 = 5
)

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get

// Reconcile reconciles model download Jobs
func (r *ModelDownloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if req.Namespace != PlatformSpace {
		return ctrl.Result{}, nil
	}

	job, result, err := r.fetchJob(ctx, req, logger)
	if err != nil || job == nil {
		return result, err
	}

	if job.Labels["app"] != "model-download" {
		return ctrl.Result{}, nil
	}

	download, result, err := r.fetchDownloadRecord(ctx, job, logger)
	if err != nil || download == nil {
		return result, err
	}

	return r.syncDownloadWithJob(ctx, job, download, logger)
}

func (r *ModelDownloadReconciler) fetchJob(
	ctx context.Context, req ctrl.Request, logger logr.Logger,
) (*batchv1.Job, ctrl.Result, error) {
	var job batchv1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if k8serrors.IsNotFound(err) {
			result, handleErr := r.handleJobNotFound(ctx, req.Name)
			return nil, result, handleErr
		}
		logger.Error(err, "unable to fetch job")
		return nil, ctrl.Result{Requeue: true}, err
	}

	return &job, ctrl.Result{}, nil
}

func (r *ModelDownloadReconciler) fetchDownloadRecord(
	ctx context.Context, job *batchv1.Job, logger logr.Logger,
) (*model.ModelDownload, ctrl.Result, error) {
	q := query.ModelDownload
	download, err := q.WithContext(ctx).Where(q.JobName.Eq(job.Name)).First()
	if err == nil {
		return download, ctrl.Result{}, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Info("download record not found, cleaning up orphaned job", "jobName", job.Name)

		if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
			deletePolicy := metav1.DeletePropagationBackground
			if err := r.KubeClient.BatchV1().Jobs(job.Namespace).Delete(
				ctx, job.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy},
			); err != nil {
				logger.Error(err, "failed to delete orphaned job", "jobName", job.Name)
			} else {
				logger.Info("successfully deleted orphaned job", "jobName", job.Name)
			}
		}

		return nil, ctrl.Result{}, nil
	}

	logger.Error(err, "unable to fetch download record")
	return nil, ctrl.Result{Requeue: true}, err
}

func (r *ModelDownloadReconciler) syncDownloadWithJob(
	ctx context.Context, job *batchv1.Job, download *model.ModelDownload, logger logr.Logger,
) (ctrl.Result, error) {
	oldStatus := download.Status
	newStatus := r.getJobStatus(job)

	if newStatus == model.ModelDownloadStatusDownloading {
		if err := r.updateProgress(ctx, job, download); err != nil {
			logger.Error(err, "failed to update progress")
		}
	}

	if newStatus == model.ModelDownloadStatusReady && download.Status != model.ModelDownloadStatusReady {
		if err := r.extractFinalResult(ctx, job, download); err != nil {
			logger.Error(err, "failed to extract final result")
		}

		if err := r.createDatasetForModel(ctx, download); err != nil {
			logger.Error(err, "failed to create dataset for model")
		}
	}

	if newStatus != oldStatus {
		if err := r.updateDownloadStatus(ctx, download, newStatus); err != nil {
			logger.Error(err, "failed to update download status")
			return ctrl.Result{Requeue: true}, err
		}
		logger.Info(fmt.Sprintf("model download: %s, status: %s -> %s", job.Name, oldStatus, newStatus))
	}

	return ctrl.Result{}, nil
}

func (r *ModelDownloadReconciler) handleJobNotFound(ctx context.Context, jobName string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	q := query.ModelDownload
	download, err := q.WithContext(ctx).Where(q.JobName.Eq(jobName)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Job和数据库记录都不存在，这是正常的（可能是旧Job或已清理的任务）
			logger.V(1).Info("Job not found in both k8s and database", "jobName", jobName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch download record")
		return ctrl.Result{Requeue: true}, err
	}

	logger.Info("Job not found in k8s but exists in database", "jobName", jobName, "status", download.Status)

	// If already in terminal state, no update needed
	if download.Status == model.ModelDownloadStatusReady ||
		download.Status == model.ModelDownloadStatusFailed ||
		download.Status == model.ModelDownloadStatusPaused {
		return ctrl.Result{}, nil
	}

	// Job deleted but status not terminal, mark as failed
	if err := r.updateDownloadStatus(ctx, download, model.ModelDownloadStatusFailed); err != nil {
		logger.Error(err, "failed to update download status to failed")
		return ctrl.Result{Requeue: true}, err
	}

	_, _ = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Update(q.Message, "Job was deleted")

	return ctrl.Result{}, nil
}

func (r *ModelDownloadReconciler) getJobStatus(job *batchv1.Job) model.ModelDownloadStatus {
	if job.Status.Succeeded == 1 {
		return model.ModelDownloadStatusReady
	} else if job.Status.Failed >= 1 {
		return model.ModelDownloadStatusFailed
	}

	// Job is active - check if running or pending
	if job.Status.Active > 0 {
		return model.ModelDownloadStatusDownloading
	}

	return model.ModelDownloadStatusPending
}

func (r *ModelDownloadReconciler) updateDownloadStatus(
	ctx context.Context, download *model.ModelDownload, status model.ModelDownloadStatus,
) error {
	q := query.ModelDownload
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(download.ID)).
		Update(q.Status, status)
	return err
}

func (r *ModelDownloadReconciler) updateProgress(ctx context.Context, job *batchv1.Job, download *model.ModelDownload) error {
	// Get pod logs to extract progress
	podList := &v1.PodList{}
	labelSelector := client.MatchingLabels{
		"job-name": job.Name,
	}
	err := r.List(ctx, podList, client.InNamespace(job.Namespace), labelSelector)
	if err != nil || len(podList.Items) == 0 {
		return err
	}

	pod := &podList.Items[0]
	if pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded {
		return nil
	}

	// Get pod logs
	logs, err := r.getPodLogs(ctx, pod)
	if err != nil {
		return err
	}

	// Parse progress from logs: [PROGRESS] downloaded_bytes=12345
	progressPattern := regexp.MustCompile(`\[PROGRESS\] downloaded_bytes=(\d+)`)
	matches := progressPattern.FindAllStringSubmatch(logs, -1)
	if len(matches) > 0 {
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) > 1 {
			downloadedBytes, _ := strconv.ParseInt(lastMatch[1], 10, 64)

			// Calculate speed if we have previous data
			q := query.ModelDownload
			//nolint:gofmt // gofmt incorrectly formats this map literal
			updates := map[string]interface{}{
				"downloaded_bytes": downloadedBytes,
			}

			// Simple speed calculation based on downloaded bytes change
			if download.DownloadedBytes > 0 && downloadedBytes > download.DownloadedBytes {
				bytesPerInterval := downloadedBytes - download.DownloadedBytes
				speedBytesPerSec := bytesPerInterval / progressReportIntervalSeconds
				updates["download_speed"] = formatSpeed(speedBytesPerSec)
			}

			_, err = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Updates(updates)
			return err
		}
	}

	return nil
}

func (r *ModelDownloadReconciler) extractFinalResult(ctx context.Context, job *batchv1.Job, download *model.ModelDownload) error {
	// Get pod logs to extract final size
	podList := &v1.PodList{}
	labelSelector := client.MatchingLabels{
		"job-name": job.Name,
	}
	err := r.List(ctx, podList, client.InNamespace(job.Namespace), labelSelector)
	if err != nil || len(podList.Items) == 0 {
		return err
	}

	pod := &podList.Items[0]
	logs, err := r.getPodLogs(ctx, pod)
	if err != nil {
		return err
	}

	// Parse result: [RESULT] size_bytes=12345 duration_seconds=60 speed_bytes_per_sec=205
	resultPattern := regexp.MustCompile(`\[RESULT\] size_bytes=(\d+)(?:\s+duration_seconds=(\d+)\s+speed_bytes_per_sec=(\d+))?`)
	matches := resultPattern.FindStringSubmatch(logs)
	if len(matches) > 1 {
		sizeBytes, _ := strconv.ParseInt(matches[1], 10, 64)

		q := query.ModelDownload
		//nolint:gofmt // gofmt incorrectly formats this map literal
		updates := map[string]interface{}{
			"size_bytes":       sizeBytes,
			"downloaded_bytes": sizeBytes,
		}

		if len(matches) > 3 && matches[3] != "" {
			speedBytesPerSec, _ := strconv.ParseInt(matches[3], 10, 64)
			updates["download_speed"] = formatSpeed(speedBytesPerSec)
		}

		_, err = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Updates(updates)
		return err
	}

	return nil
}

func (r *ModelDownloadReconciler) getPodLogs(ctx context.Context, pod *v1.Pod) (string, error) {
	// Get pod logs using Kubernetes clientset
	req := r.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{
		Container: "downloader",
		TailLines: func() *int64 { i := podLogTailLines; return &i }(),
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		klog.V(logVerboseLevelDebug).Infof("Failed to get logs from pod %s/%s: %v", pod.Namespace, pod.Name, err)
		return "", err
	}
	defer stream.Close()

	buf, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func formatSpeed(bytesPerSec int64) string {
	if bytesPerSec < bytesInKB {
		return fmt.Sprintf("%d B/s", bytesPerSec)
	} else if bytesPerSec < bytesInMB {
		return fmt.Sprintf("%.2f KB/s", float64(bytesPerSec)/float64(bytesInKB))
	} else if bytesPerSec < bytesInGB {
		return fmt.Sprintf("%.2f MB/s", float64(bytesPerSec)/float64(bytesInMB))
	}
	return fmt.Sprintf("%.2f GB/s", float64(bytesPerSec)/float64(bytesInGB))
}

func (r *ModelDownloadReconciler) createDatasetForModel(ctx context.Context, download *model.ModelDownload) error {
	// Create a dataset record for the downloaded model or dataset
	qDataset := query.Dataset
	qUserDataset := query.UserDataset
	qAccountDataset := query.AccountDataset

	// 根据 category 确定数据类型
	var dataType model.DataType
	var resourceLabel string
	if download.Category == model.DownloadCategoryDataset {
		dataType = model.DataTypeDataset
		resourceLabel = "数据集"
	} else {
		dataType = model.DataTypeModel
		resourceLabel = "模型"
	}

	// Check if dataset already exists for this resource (check by name only, regardless of type)
	// This prevents creating duplicate records with different types
	existingDataset, _ := qDataset.WithContext(ctx).
		Where(qDataset.Name.Eq(download.Name)).
		First()

	if existingDataset != nil {
		// If exists but type is different, update it to the correct type
		if existingDataset.Type != dataType {
			klog.Warningf("Dataset %s exists with wrong type %s, updating to %s", download.Name, existingDataset.Type, dataType)
			_, err := qDataset.WithContext(ctx).
				Where(qDataset.ID.Eq(existingDataset.ID)).
				Update(qDataset.Type, dataType)
			if err != nil {
				klog.Errorf("Failed to update dataset type: %v", err)
			}
			// Also update the description
			_, _ = qDataset.WithContext(ctx).
				Where(qDataset.ID.Eq(existingDataset.ID)).
				Update(qDataset.Describe, fmt.Sprintf("从 %s 下载的%s",
					map[model.ModelSource]string{model.ModelSourceModelScope: "ModelScope", model.ModelSourceHuggingFace: "HuggingFace"}[download.Source],
					resourceLabel))
		}
		klog.V(logVerboseLevelDebug).Infof("Dataset already exists for %s %s (dataset ID: %d)", resourceLabel, download.Name, existingDataset.ID)
		return nil
	}

	// 将前端路径(如public/222/...)转换为物理路径(如sugon-gpu-incoming/222/...)用于存储访问
	datasetURL := r.convertToPhysicalPath(download.Path)

	// 根据来源格式化描述信息
	var sourceLabel string
	if download.Source == model.ModelSourceModelScope {
		sourceLabel = "ModelScope"
	} else {
		sourceLabel = "HuggingFace"
	}

	// Create dataset record
	dataset := &model.Dataset{
		Name:     download.Name,
		URL:      datasetURL,
		Describe: fmt.Sprintf("从 %s 下载的%s", sourceLabel, resourceLabel),
		Type:     dataType,
		UserID:   download.CreatorID,
		Extra: datatypes.NewJSONType(model.ExtraContent{
			Tags:     []string{string(download.Source), "auto-download"},
			Editable: false,
		}),
	}

	if err := qDataset.WithContext(ctx).Create(dataset); err != nil {
		return fmt.Errorf("failed to create dataset: %w", err)
	}

	// Create user-dataset association
	userDataset := &model.UserDataset{
		UserID:    download.CreatorID,
		DatasetID: dataset.ID,
	}
	if err := qUserDataset.WithContext(ctx).Create(userDataset); err != nil {
		return fmt.Errorf("failed to create user-dataset association: %w", err)
	}

	// 所有模型都下载到公共空间,创建account-dataset关联(AccountID: 1 is public)
	accountDataset := &model.AccountDataset{
		AccountID: 1,
		DatasetID: dataset.ID,
	}
	if err := qAccountDataset.WithContext(ctx).Create(accountDataset); err != nil {
		klog.Warningf("Failed to create account-dataset association: %v", err)
		// Don't fail if public association fails
	}

	klog.Infof("Created dataset for %s %s (dataset ID: %d)", resourceLabel, download.Name, dataset.ID)
	return nil
}

// convertToPhysicalPath 将前端路径转换为物理存储路径
func (r *ModelDownloadReconciler) convertToPhysicalPath(frontendPath string) string {
	// public -> sugon-gpu-incoming
	if strings.HasPrefix(frontendPath, "public/") || frontendPath == "public" {
		return strings.Replace(frontendPath, "public", config.GetConfig().Storage.Prefix.Public, 1)
	}
	// user -> sugon-gpu-home-lab (if needed in future)
	if strings.HasPrefix(frontendPath, "user/") || frontendPath == "user" {
		return strings.Replace(frontendPath, "user", config.GetConfig().Storage.Prefix.User, 1)
	}
	return frontendPath
}
