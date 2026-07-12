package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	progressRequeueInterval             = 5 * time.Second
	// maxStoredLogBytes caps the log tail persisted on the download record when
	// the job reaches a terminal state (the K8s Job itself is GC'd after 7 days).
	maxStoredLogBytes = 64 * 1024
)

// Markers printed by the download job script (see buildDownloadCommand).
var (
	progressPattern = regexp.MustCompile(`\[PROGRESS\] downloaded_bytes=(\d+)`)
	totalPattern    = regexp.MustCompile(`\[TOTAL\] size_bytes=(\d+)`)
	resultPattern   = regexp.MustCompile(`\[RESULT\] size_bytes=(\d+)(?:\s+duration_seconds=(\d+)\s+speed_bytes_per_sec=(\d+))?`)
	descPattern     = regexp.MustCompile(`\[DESC\] (.+)`)
	metadataPattern = regexp.MustCompile(`\[META\] (.+)`)
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

		deletePolicy := metav1.DeletePropagationBackground
		if err := r.KubeClient.BatchV1().Jobs(job.Namespace).Delete(
			ctx, job.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy},
		); err != nil && !k8serrors.IsNotFound(err) {
			logger.Error(err, "failed to delete orphaned job", "jobName", job.Name)
		} else {
			logger.Info("successfully deleted orphaned job", "jobName", job.Name)
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

		// Persist the log tail before the Job/Pod is GC'd, and reuse it to pick
		// up the [DESC] summary the job extracted from the repo's README.
		logs := r.persistFinalLogs(ctx, job, download)
		if err := r.persistRepositoryMetadata(ctx, download, logs); err != nil {
			logger.Error(err, "failed to persist repository metadata")
		}

		metadata := parseRepositoryMetadata(logs)
		if err := r.createDatasetForModel(ctx, download, parseDescriptionFromLogs(logs), metadata.Tags); err != nil {
			logger.Error(err, "failed to create dataset for model")
			return ctrl.Result{RequeueAfter: progressRequeueInterval}, err
		}
	}

	// When a download fails, capture the reason from pod logs so the user can
	// see why (gated repo, auth, network, OOM, ...) instead of an empty message.
	if newStatus == model.ModelDownloadStatusFailed && download.Status != model.ModelDownloadStatusFailed {
		r.persistFinalLogs(ctx, job, download)
		if err := r.extractFailureReason(ctx, job, download); err != nil {
			logger.Error(err, "failed to extract failure reason")
		}
	}

	if newStatus != oldStatus {
		if err := r.updateDownloadStatus(ctx, download, newStatus); err != nil {
			logger.Error(err, "failed to update download status")
			return ctrl.Result{Requeue: true}, err
		}
		logger.Info(fmt.Sprintf("model download: %s, status: %s -> %s", job.Name, oldStatus, newStatus))
	}
	if newStatus == model.ModelDownloadStatusDownloading {
		return ctrl.Result{RequeueAfter: progressRequeueInterval}, nil
	}

	return ctrl.Result{}, nil
}

type repositoryMetadata struct {
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	License        string   `json:"license"`
	Task           string   `json:"task"`
	Library        string   `json:"library"`
	ModelType      string   `json:"model_type"`
	ParameterCount int64    `json:"parameter_count"`
	Private        bool     `json:"private"`
	Gated          bool     `json:"gated"`
	LoginRequired  bool     `json:"login_required"`
	Downloads      int64    `json:"downloads"`
	Likes          int64    `json:"likes"`
	LogoURL        string   `json:"logo_url"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	Tags           []string `json:"tags"`
}

func parseRepositoryMetadata(logs string) repositoryMetadata {
	matches := metadataPattern.FindAllStringSubmatch(logs, -1)
	if len(matches) == 0 {
		return repositoryMetadata{}
	}
	var metadata repositoryMetadata
	_ = json.Unmarshal([]byte(matches[len(matches)-1][1]), &metadata)
	return metadata
}

func (r *ModelDownloadReconciler) persistRepositoryMetadata(
	ctx context.Context, download *model.ModelDownload, logs string,
) error {
	organization := strings.SplitN(download.Name, "/", 2)[0]
	metadata := parseRepositoryMetadata(logs)
	updates := map[string]any{
		"organization":          organization,
		"logo_url":              metadata.LogoURL,
		"source_url":            downloadSourceURL(download),
		"display_name":          metadata.DisplayName,
		"source_description":    metadata.Description,
		"license":               metadata.License,
		"task":                  metadata.Task,
		"library":               metadata.Library,
		"model_type":            metadata.ModelType,
		"parameter_count":       metadata.ParameterCount,
		"source_private":        metadata.Private,
		"source_gated":          metadata.Gated,
		"source_login_required": metadata.LoginRequired,
		"source_downloads":      metadata.Downloads,
		"source_likes":          metadata.Likes,
	}
	if metadata.UpdatedAt != "" {
		if updatedAt, err := time.Parse(time.RFC3339, metadata.UpdatedAt); err == nil {
			updates["source_updated_at"] = updatedAt
		}
	}
	if metadata.CreatedAt != "" {
		if createdAt, err := time.Parse(time.RFC3339, metadata.CreatedAt); err == nil {
			updates["source_created_at"] = createdAt
		}
	}

	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		source := model.ModelDatasetSource{
			Provider:       model.ModelDatasetProvider(download.Source),
			ResourceType:   model.DataType(download.Category),
			RepositoryID:   download.Name,
			RepositoryURL:  downloadSourceURL(download),
			Organization:   organization,
			LogoURL:        metadata.LogoURL,
			DisplayName:    metadata.DisplayName,
			Description:    metadata.Description,
			License:        metadata.License,
			Task:           metadata.Task,
			Library:        metadata.Library,
			ModelType:      metadata.ModelType,
			ParameterCount: metadata.ParameterCount,
			Private:        metadata.Private,
			Gated:          metadata.Gated,
			LoginRequired:  metadata.LoginRequired,
			Downloads:      metadata.Downloads,
			Likes:          metadata.Likes,
		}
		if value, ok := updates["source_updated_at"].(time.Time); ok {
			source.SourceUpdatedAt = &value
		}
		if value, ok := updates["source_created_at"].(time.Time); ok {
			source.SourceCreatedAt = &value
		}
		var persisted model.ModelDatasetSource
		lookup := tx.Where(
			"provider = ? AND resource_type = ? AND repository_id = ?",
			source.Provider, source.ResourceType, source.RepositoryID,
		).First(&persisted)
		if errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			if err := tx.Create(&source).Error; err != nil {
				return err
			}
			persisted = source
		} else if lookup.Error != nil {
			return lookup.Error
		} else if err := tx.Model(&persisted).Updates(source).Error; err != nil {
			return err
		}
		updates["model_dataset_source_id"] = persisted.ID
		if err := tx.Model(&model.ModelDownload{}).Where("id = ?", download.ID).Updates(updates).Error; err != nil {
			return err
		}
		download.ModelDatasetSourceID = &persisted.ID
		return nil
	})
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
	// Prefer terminal Job conditions so that retries (BackoffLimit > 0) do not
	// flip the record to Failed while attempts are still being made.
	for i := range job.Status.Conditions {
		cond := job.Status.Conditions[i]
		if cond.Status != v1.ConditionTrue {
			continue
		}
		switch cond.Type {
		case batchv1.JobComplete:
			return model.ModelDownloadStatusReady
		case batchv1.JobFailed:
			return model.ModelDownloadStatusFailed
		}
	}

	if job.Status.Succeeded >= 1 {
		return model.ModelDownloadStatusReady
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

func (r *ModelDownloadReconciler) latestPodForJob(ctx context.Context, job *batchv1.Job) (*v1.Pod, error) {
	podList := &v1.PodList{}
	if err := r.List(
		ctx,
		podList,
		client.InNamespace(job.Namespace),
		client.MatchingLabels{"job-name": job.Name},
	); err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		return nil, nil
	}

	latest := &podList.Items[0]
	for i := 1; i < len(podList.Items); i++ {
		if podList.Items[i].CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = &podList.Items[i]
		}
	}
	return latest, nil
}

func (r *ModelDownloadReconciler) updateProgress(ctx context.Context, job *batchv1.Job, download *model.ModelDownload) error {
	pod, err := r.latestPodForJob(ctx, job)
	if err != nil || pod == nil {
		return err
	}
	if pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded {
		return nil
	}

	// Get pod logs
	logs, err := r.getPodLogs(ctx, pod)
	if err != nil {
		return err
	}

	// Persist the latest log tail on every progress reconciliation. This keeps
	// logs available through the record-level API even while the Pod is still
	// running, and decouples readers from direct Pod-log permissions.
	updates := map[string]any{
		"logs":          truncateLogTail(logs, maxStoredLogBytes),
		"logs_saved_at": time.Now(),
	}

	// The job prints the repo's total size once, up front: [TOTAL] size_bytes=N.
	// Only fill it while unknown; the final [RESULT] (actual on-disk size) stays
	// authoritative via extractFinalResult.
	if download.SizeBytes == 0 {
		if m := totalPattern.FindStringSubmatch(logs); len(m) > 1 {
			if total, parseErr := strconv.ParseInt(m[1], 10, 64); parseErr == nil && total > 0 {
				updates["size_bytes"] = total
			}
		}
	}

	// Parse progress from logs: [PROGRESS] downloaded_bytes=12345
	matches := progressPattern.FindAllStringSubmatch(logs, -1)
	if len(matches) > 0 {
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) > 1 {
			downloadedBytes, _ := strconv.ParseInt(lastMatch[1], 10, 64)
			updates["downloaded_bytes"] = downloadedBytes

			// Simple speed calculation based on downloaded bytes change
			if download.DownloadedBytes > 0 && downloadedBytes > download.DownloadedBytes {
				bytesPerInterval := downloadedBytes - download.DownloadedBytes
				speedBytesPerSec := bytesPerInterval / progressReportIntervalSeconds
				updates["download_speed"] = formatSpeed(speedBytesPerSec)
			}
		}
	}

	q := query.ModelDownload
	_, err = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Updates(updates)
	return err
}

func (r *ModelDownloadReconciler) extractFinalResult(ctx context.Context, job *batchv1.Job, download *model.ModelDownload) error {
	pod, err := r.latestPodForJob(ctx, job)
	if err != nil || pod == nil {
		return err
	}
	logs, err := r.getPodLogs(ctx, pod)
	if err != nil {
		return err
	}

	// Parse result: [RESULT] size_bytes=12345 duration_seconds=60 speed_bytes_per_sec=205
	matches := resultPattern.FindStringSubmatch(logs)
	if len(matches) > 1 {
		sizeBytes, _ := strconv.ParseInt(matches[1], 10, 64)
		download.SizeBytes = sizeBytes
		download.DownloadedBytes = sizeBytes

		q := query.ModelDownload
		updates := map[string]any{
			"size_bytes":       sizeBytes,
			"downloaded_bytes": sizeBytes,
		}

		if len(matches) > 3 && matches[3] != "" {
			speedBytesPerSec, _ := strconv.ParseInt(matches[3], 10, 64)
			download.DownloadSpeed = formatSpeed(speedBytesPerSec)
			updates["download_speed"] = download.DownloadSpeed
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

// persistFinalLogs stores the pod's log tail on the download record so it can
// still be inspected after the K8s Job (TTL 7 days) and its pods are GC'd.
// Returns the raw logs so callers can parse markers without a second fetch.
func (r *ModelDownloadReconciler) persistFinalLogs(ctx context.Context, job *batchv1.Job, download *model.ModelDownload) string {
	pod, err := r.latestPodForJob(ctx, job)
	if err != nil || pod == nil {
		return ""
	}

	logs, err := r.getPodLogs(ctx, pod)
	if err != nil || logs == "" {
		return ""
	}

	now := time.Now()
	q := query.ModelDownload
	if _, err := q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Updates(map[string]any{
		"logs":          truncateLogTail(logs, maxStoredLogBytes),
		"logs_saved_at": now,
	}); err != nil {
		klog.Warningf("failed to persist final logs for download %d: %v", download.ID, err)
	}
	return logs
}

// truncateLogTail keeps at most maxBytes from the end of the logs, starting at
// a line boundary so the stored tail stays readable.
func truncateLogTail(logs string, maxBytes int) string {
	if len(logs) <= maxBytes {
		return logs
	}
	tail := logs[len(logs)-maxBytes:]
	if idx := strings.IndexByte(tail, '\n'); idx >= 0 && idx+1 < len(tail) {
		tail = tail[idx+1:]
	}
	return tail
}

// parseDescriptionFromLogs extracts the repo summary the download job printed
// as a "[DESC] ..." line (taken from the downloaded README). Empty if absent.
func parseDescriptionFromLogs(logs string) string {
	if logs == "" {
		return ""
	}
	matches := descPattern.FindAllStringSubmatch(logs, -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.TrimSpace(matches[len(matches)-1][1])
}

// extractFailureReason reads the failed pod's logs (and termination state) and
// stores a concise, human-readable reason on the download record so users can
// understand why a download failed instead of seeing an empty message.
func (r *ModelDownloadReconciler) extractFailureReason(ctx context.Context, job *batchv1.Job, download *model.ModelDownload) error {
	q := query.ModelDownload

	pod, err := r.latestPodForJob(ctx, job)
	if err != nil || pod == nil {
		_, _ = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Update(q.Message, "Download job failed (pod not found)")
		return err
	}

	// Prefer the container termination reason when available (e.g. OOMKilled).
	for i := range pod.Status.ContainerStatuses {
		if term := pod.Status.ContainerStatuses[i].State.Terminated; term != nil && term.Reason == "OOMKilled" {
			_, _ = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Update(q.Message,
				"Download failed: out of memory (OOMKilled). Try again later or contact an admin to raise the job memory limit.")
			return nil
		}
	}

	logs, logErr := r.getPodLogs(ctx, pod)
	if logErr != nil {
		_, _ = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Update(q.Message, "Download job failed (logs unavailable)")
		return logErr
	}

	_, _ = q.WithContext(ctx).Where(q.ID.Eq(download.ID)).Update(q.Message, classifyDownloadFailure(logs))
	return nil
}

// downloadFailureRule maps log keywords to a concise, actionable reason.
type downloadFailureRule struct {
	keywords []string
	reason   string
}

// downloadFailureRules are evaluated in order; the first matching rule wins.
var downloadFailureRules = []downloadFailureRule{
	{
		keywords: []string{"gated", "awaiting a review", "access to model", "you must be authenticated"},
		reason:   "Download failed: this repository is gated and requires authorization/login on the source site.",
	},
	{
		keywords: []string{"401", "403", "unauthorized", "forbidden"},
		reason:   "Download failed: access denied (401/403). The repository may be private or require a token.",
	},
	{
		keywords: []string{"404", "not found", "repository not found", "does not exist"},
		reason:   "Download failed: repository or revision not found (404). Check the name and revision.",
	},
	{
		keywords: []string{"no space left"},
		reason:   "Download failed: no space left on the storage volume.",
	},
	{
		keywords: []string{"timed out", "timeout", "connection reset", "temporary failure in name resolution", "max retries exceeded"},
		reason:   "Download failed: network error while reaching the source. Try again later.",
	},
}

// classifyDownloadFailure maps raw download logs to a concise, actionable reason.
func classifyDownloadFailure(logs string) string {
	lower := strings.ToLower(logs)
	for _, rule := range downloadFailureRules {
		for _, kw := range rule.keywords {
			if strings.Contains(lower, kw) {
				return rule.reason
			}
		}
	}
	return fallbackFailureReason(logs)
}

// fallbackFailureReason returns the last non-empty log line, which usually
// carries the underlying error, truncated to a reasonable length.
func fallbackFailureReason(logs string) string {
	const maxReasonLen = 300
	lines := strings.Split(strings.TrimRight(logs, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if len(line) > maxReasonLen {
			line = line[:maxReasonLen] + "..."
		}
		return "Download failed: " + line
	}
	return "Download failed (no log output captured)."
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

// downloadSourceURL returns the public page of the downloaded repo on its
// source site, used as the dataset's WebURL and description fallback.
func downloadSourceURL(download *model.ModelDownload) string {
	if download.Source == model.ModelSourceHuggingFace {
		endpoint := config.GetConfig().HuggingFaceDownloadEndpoint()
		if download.Category == model.DownloadCategoryDataset {
			return endpoint + "/datasets/" + download.Name
		}
		return endpoint + "/" + download.Name
	}
	endpoint := config.GetConfig().ModelScopeDownloadEndpoint()
	if download.Category == model.DownloadCategoryDataset {
		return endpoint + "/datasets/" + download.Name
	}
	return endpoint + "/models/" + download.Name
}

// datasetDescriptionForDownload builds the dataset description. It prefers the
// summary extracted from the repo's README by the download job; otherwise it
// falls back to a description tied to the specific repo (name/source/revision).
func datasetDescriptionForDownload(download *model.ModelDownload, readmeDesc string) string {
	if readmeDesc != "" {
		return readmeDesc
	}

	var sourceLabel string
	if download.Source == model.ModelSourceModelScope {
		sourceLabel = "ModelScope"
	} else {
		sourceLabel = "HuggingFace"
	}
	var resourceLabel string
	if download.Category == model.DownloadCategoryDataset {
		resourceLabel = "数据集"
	} else {
		resourceLabel = "模型"
	}

	desc := fmt.Sprintf("%s 上的%s %s", sourceLabel, resourceLabel, download.Name)
	if download.Revision != "" {
		desc += fmt.Sprintf("（版本 %s）", download.Revision)
	}
	return desc
}

func datasetExtraForDownload(
	existing model.ExtraContent, download *model.ModelDownload, sourceURL string, repositoryTags []string,
) model.ExtraContent {
	filteredTags := existing.Tags[:0]
	for _, tag := range existing.Tags {
		if tag != "auto-download" {
			filteredTags = append(filteredTags, tag)
		}
	}
	existing.Tags = filteredTags
	for _, candidate := range append([]string{string(download.Source)}, repositoryTags...) {
		if candidate == "" {
			continue
		}
		hasTag := false
		for _, tag := range existing.Tags {
			if tag == candidate {
				hasTag = true
				break
			}
		}
		if !hasTag {
			existing.Tags = append(existing.Tags, candidate)
		}
	}
	existing.WebURL = &sourceURL
	existing.Editable = false
	return existing
}

func (r *ModelDownloadReconciler) createDatasetForModel(
	ctx context.Context, download *model.ModelDownload, readmeDesc string, repositoryTags []string,
) error {
	// Create a dataset record for the downloaded model or dataset
	qDataset := query.Dataset

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

	describe := datasetDescriptionForDownload(download, readmeDesc)
	sourceURL := downloadSourceURL(download)

	// Check if dataset already exists for this resource (check by name only, regardless of type)
	// This prevents creating duplicate records with different types
	// First check for non-deleted records
	existingDataset, _ := qDataset.WithContext(ctx).
		Where(qDataset.Name.Eq(download.Name)).
		First()

	if existingDataset != nil {
		if existingDataset.Type != dataType {
			klog.Warningf("Dataset %s exists with wrong type %s, updating to %s", download.Name, existingDataset.Type, dataType)
		}
		extra := datasetExtraForDownload(existingDataset.Extra.Data(), download, sourceURL, repositoryTags)
		if _, err := qDataset.WithContext(ctx).Where(qDataset.ID.Eq(existingDataset.ID)).Updates(map[string]any{
			"type":                    dataType,
			"describe":                describe,
			"extra":                   datatypes.NewJSONType(extra),
			"size_bytes":              download.SizeBytes,
			"model_dataset_source_id": download.ModelDatasetSourceID,
		}); err != nil {
			return fmt.Errorf("failed to update existing dataset metadata: %w", err)
		}
		klog.V(logVerboseLevelDebug).Infof("Dataset already exists for %s %s (dataset ID: %d)", resourceLabel, download.Name, existingDataset.ID)
		return r.ensureDatasetAssociations(ctx, existingDataset.ID, download.CreatorID)
	}

	// Check for soft-deleted records
	softDeletedDataset, _ := qDataset.WithContext(ctx).Unscoped().
		Where(qDataset.Name.Eq(download.Name), qDataset.DeletedAt.IsNotNull()).
		First()

	if softDeletedDataset != nil {
		// Restore the soft-deleted dataset
		klog.Infof("Restoring soft-deleted dataset for %s %s (dataset ID: %d)", resourceLabel, download.Name, softDeletedDataset.ID)
		_, err := qDataset.WithContext(ctx).Unscoped().
			Where(qDataset.ID.Eq(softDeletedDataset.ID)).
			Update(qDataset.DeletedAt, nil)
		if err != nil {
			return fmt.Errorf("failed to restore soft-deleted dataset: %w", err)
		}

		extra := datasetExtraForDownload(softDeletedDataset.Extra.Data(), download, sourceURL, repositoryTags)
		updates := map[string]any{
			"type":                    dataType,
			"describe":                describe,
			"extra":                   datatypes.NewJSONType(extra),
			"size_bytes":              download.SizeBytes,
			"model_dataset_source_id": download.ModelDatasetSourceID,
		}
		if _, err := qDataset.WithContext(ctx).
			Where(qDataset.ID.Eq(softDeletedDataset.ID)).
			Updates(updates); err != nil {
			return fmt.Errorf("failed to update restored dataset metadata: %w", err)
		}

		return r.ensureDatasetAssociations(ctx, softDeletedDataset.ID, download.CreatorID)
	}

	// 将前端路径(如public/222/...)转换为物理路径(如sugon-gpu-incoming/222/...)用于存储访问
	datasetURL := r.convertToPhysicalPath(download.Path)

	// Create dataset record
	dataset := &model.Dataset{
		Name:                 download.Name,
		URL:                  datasetURL,
		Describe:             describe,
		Type:                 dataType,
		UserID:               download.CreatorID,
		SizeBytes:            download.SizeBytes,
		ModelDatasetSourceID: download.ModelDatasetSourceID,
		Extra: datatypes.NewJSONType(model.ExtraContent{
			Tags:     datasetExtraForDownload(model.ExtraContent{}, download, sourceURL, repositoryTags).Tags,
			WebURL:   &sourceURL,
			Editable: false,
		}),
	}

	if err := qDataset.WithContext(ctx).Create(dataset); err != nil {
		return fmt.Errorf("failed to create dataset: %w", err)
	}

	if err := r.ensureDatasetAssociations(ctx, dataset.ID, download.CreatorID); err != nil {
		return err
	}

	klog.Infof("Created dataset for %s %s (dataset ID: %d)", resourceLabel, download.Name, dataset.ID)
	return nil
}

func (r *ModelDownloadReconciler) ensureDatasetAssociations(
	ctx context.Context, datasetID, userID uint,
) error {
	qUserDataset := query.UserDataset
	if _, err := qUserDataset.WithContext(ctx).
		Where(qUserDataset.UserID.Eq(userID), qUserDataset.DatasetID.Eq(datasetID)).
		First(); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to query user-dataset association: %w", err)
		}
		if err := qUserDataset.WithContext(ctx).Create(&model.UserDataset{
			UserID: userID, DatasetID: datasetID,
		}); err != nil {
			return fmt.Errorf("failed to create user-dataset association: %w", err)
		}
	}

	qAccountDataset := query.AccountDataset
	if _, err := qAccountDataset.WithContext(ctx).
		Where(qAccountDataset.AccountID.Eq(model.DefaultAccountID), qAccountDataset.DatasetID.Eq(datasetID)).
		First(); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to query account-dataset association: %w", err)
		}
		if err := qAccountDataset.WithContext(ctx).Create(&model.AccountDataset{
			AccountID: model.DefaultAccountID, DatasetID: datasetID,
		}); err != nil {
			return fmt.Errorf("failed to create account-dataset association: %w", err)
		}
	}
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
