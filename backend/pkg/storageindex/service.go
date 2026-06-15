//nolint:dupl,funlen,gocritic,gocyclo,goconst,gosec,lll,mnd,unparam,unused // Metadata indexing orchestration intentionally keeps scanning, aggregation and redundancy detection in one service.
package storageindex

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	toolboxNamespace           = "rook-ceph"
	cephFSVolumeName           = "cephfs"
	insertBatchSize            = 500
	defaultOverviewLimit       = 10
	defaultRedundancyPageSize  = 50
	minimumRedundantFileBytes  = 10 * 1024 * 1024
	manualTriggerSource        = "manual"
	redundancyConfidenceHigh   = "high"
	redundancyConfidenceMedium = "medium"
	hashAlgorithmSHA256        = "sha256"
	hashAlgorithmSampledSHA256 = "sampled_sha256"
	compareModeOptimized       = "optimized"
	compareModeFullHash        = "full_hash"
	verificationModeMetadata   = "metadata"
	verificationModeFileName   = "file_name_size"
	// Keep this within storage_index_redundancy_hits.verification_mode varchar(32).
	verificationModeSafeTensorsHdrAndSampledSHA = "safetensors_hdr+sampled_sha256"
	findFieldSeparator                          = "\x1f"
	findRecordSeparator                         = "\x00"
	scanProgressLogEveryRecord                  = 1000
	sampledHashSegmentBytes                     = 64 * 1024
	safetensorsHeaderMaxBytes                   = 16 * 1024 * 1024
)

var publicBaselineTopLevelAllowList = map[string]struct{}{
	"models":   {},
	"dataset":  {},
	"datasets": {},
}

// Only exclude top-level directories that are very unlikely to contain
// user-managed copies of public models. Keep this list conservative.
var definitelyNotPublicModelCopyTopLevelDirSet = map[string]struct{}{
	"conda":              {},
	".conda":             {},
	".bun":               {},
	".claude":            {},
	".codex":             {},
	".config":            {},
	".copilot":           {},
	".craft-versions":    {},
	".cursor-server":     {},
	".dotnet":            {},
	".gongfeng-copilot":  {},
	".ipython":           {},
	".java":              {},
	".jupyter":           {},
	".lingma":            {},
	".local":             {},
	".marscode":          {},
	".mcp-auth":          {},
	".nvm":               {},
	".opencode":          {},
	".pip":               {},
	".pki":               {},
	".ray":               {},
	".redhat":            {},
	".rest-client":       {},
	".ssh":               {},
	".subversion":        {},
	".trae":              {},
	".trae-aicc":         {},
	".trae-server":       {},
	".nv":                {},
	".zed_server":        {},
	"miniconda3":         {},
	"anaconda3":          {},
	"mambaforge":         {},
	".mamba":             {},
	"micromamba":         {},
	"venv":               {},
	".venv":              {},
	".git":               {},
	".svn":               {},
	".hg":                {},
	".idea":              {},
	".codeverse":         {},
	".vscode":            {},
	".vscode-server":     {},
	".ipynb_checkpoints": {},
	".npm":               {},
	".yarn":              {},
	".pnpm-store":        {},
	".cargo":             {},
	".m2":                {},
	".gradle":            {},
	".pytest_cache":      {},
	".mypy_cache":        {},
	"__pycache__":        {},
}

// Keep `.cache` itself as a top-level candidate root, but only retain a
// conservative allowlist of second-level cache subtrees that may store
// user-managed copies of public models.
var selectiveTopLevelSubtreeAllowLists = map[string]map[string]struct{}{
	".cache": {
		"huggingface":  {},
		"modelscope":   {},
		"torch":        {},
		"transformers": {},
	},
}

// Recursively prune environment/runtime subtrees that are very unlikely to be
// useful for public model copy detection, even when they appear deep inside a
// user project directory.
var definitelyNotPublicModelCopyNestedDirSet = map[string]struct{}{
	"conda":              {},
	".conda":             {},
	"miniconda3":         {},
	"anaconda3":          {},
	"mambaforge":         {},
	".mamba":             {},
	"micromamba":         {},
	"venv":               {},
	".venv":              {},
	".ipynb_checkpoints": {},
	".pytest_cache":      {},
	".mypy_cache":        {},
	"__pycache__":        {},
}

type Service struct {
	kubeClient kubernetes.Interface
	kubeConfig *rest.Config
}

type StartScanRequest struct {
	WorkspaceType model.StorageIndexWorkspaceType
	WorkspaceName string
	TriggerSource string
	ScanMode      model.StorageIndexScanMode
}

type WorkspaceOverview struct {
	WorkspaceType   model.StorageIndexWorkspaceType `json:"workspace_type"`
	WorkspaceName   string                          `json:"workspace_name"`
	LogicalPath     string                          `json:"logical_path"`
	LastScanID      string                          `json:"last_scan_id"`
	LastScanStatus  model.StorageIndexScanStatus    `json:"last_scan_status"`
	LastScanAt      *time.Time                      `json:"last_scan_at,omitempty"`
	EntryCount      int64                           `json:"entry_count"`
	FileCount       int64                           `json:"file_count"`
	DirectoryCount  int64                           `json:"directory_count"`
	RedundancyCount int64                           `json:"redundancy_count"`
	RedundancyBytes int64                           `json:"redundancy_bytes"`
	TopDirectories  []DirectorySummary              `json:"top_directories"`
	LargestFiles    []FileSummary                   `json:"largest_files"`
}

type DirectorySummary struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	Depth          int    `json:"depth"`
	FileCount      int64  `json:"file_count"`
	DirectoryCount int64  `json:"directory_count"`
	TotalSizeBytes int64  `json:"total_size_bytes"`
	IsTopLevel     bool   `json:"is_top_level"`
}

type FileSummary struct {
	Path       string     `json:"path"`
	Name       string     `json:"name"`
	SizeBytes  int64      `json:"size_bytes"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

type DirectoryCompareTiming struct {
	ScanMs        int64 `json:"scan_ms"`
	PairingMs     int64 `json:"pairing_ms"`
	HeaderMs      int64 `json:"header_ms"`
	SampledHashMs int64 `json:"sampled_hash_ms"`
	FullHashMs    int64 `json:"full_hash_ms"`
	TotalMs       int64 `json:"total_ms"`
}

type DirectoryCompareFileResult struct {
	LeftRelativePath  string `json:"left_relative_path"`
	RightRelativePath string `json:"right_relative_path"`
	FileName          string `json:"file_name"`
	SizeBytes         int64  `json:"size_bytes"`
	VerificationMode  string `json:"verification_mode"`
	Same              bool   `json:"same"`
	Reason            string `json:"reason,omitempty"`
	HeaderMatched     *bool  `json:"header_matched,omitempty"`
	SampledHashMatch  *bool  `json:"sampled_hash_match,omitempty"`
}

type DirectoryCompareResult struct {
	CompareType        string                       `json:"compare_type"`
	CompareMode        string                       `json:"compare_mode"`
	LeftPath           string                       `json:"left_path"`
	RightPath          string                       `json:"right_path"`
	Same               bool                         `json:"same"`
	LeftKeyFileCount   int                          `json:"left_key_file_count"`
	RightKeyFileCount  int                          `json:"right_key_file_count"`
	ExactMatchCount    int                          `json:"exact_match_count"`
	FallbackMatchCount int                          `json:"fallback_match_count"`
	ComparedFileCount  int                          `json:"compared_file_count"`
	VerifiedFileCount  int                          `json:"verified_file_count"`
	MissingLeft        []string                     `json:"missing_left"`
	MissingRight       []string                     `json:"missing_right"`
	Files              []DirectoryCompareFileResult `json:"files"`
	Timing             DirectoryCompareTiming       `json:"timing"`
}

type DirectoryCompareJobStatus string

const (
	DirectoryCompareJobStatusPending DirectoryCompareJobStatus = "pending"
	DirectoryCompareJobStatusRunning DirectoryCompareJobStatus = "running"
	DirectoryCompareJobStatusDone    DirectoryCompareJobStatus = "done"
	DirectoryCompareJobStatusError   DirectoryCompareJobStatus = "error"
)

type DirectoryCompareJob struct {
	mu          sync.RWMutex
	JobID       string                    `json:"job_id"`
	Status      DirectoryCompareJobStatus `json:"status"`
	LeftPath    string                    `json:"left_path"`
	RightPath   string                    `json:"right_path"`
	CompareType string                    `json:"compare_type"`
	CompareMode string                    `json:"compare_mode"`
	Result      *DirectoryCompareResult   `json:"result,omitempty"`
	Error       string                    `json:"error,omitempty"`
	StartedAt   *time.Time                `json:"started_at,omitempty"`
	FinishedAt  *time.Time                `json:"finished_at,omitempty"`
}

type DirectoryCompareJobView struct {
	JobID       string                    `json:"job_id"`
	Status      DirectoryCompareJobStatus `json:"status"`
	LeftPath    string                    `json:"left_path"`
	RightPath   string                    `json:"right_path"`
	CompareType string                    `json:"compare_type"`
	CompareMode string                    `json:"compare_mode"`
	Result      *DirectoryCompareResult   `json:"result,omitempty"`
	Error       string                    `json:"error,omitempty"`
	StartedAt   *time.Time                `json:"started_at,omitempty"`
	FinishedAt  *time.Time                `json:"finished_at,omitempty"`
}

var directoryCompareJobs sync.Map

type resolvedWorkspace struct {
	WorkspaceType model.StorageIndexWorkspaceType
	WorkspaceName string
	LogicalPath   string
}

type topLevelSignature struct {
	Name              string
	LogicalPath       string
	ParentLogicalPath string
	ActualPath        string
	EntryType         model.StorageIndexEntryType
	SizeBytes         int64
	ModifiedAt        *time.Time
	ChangedAt         *time.Time
	OwnerUID          int64
	OwnerGID          int64
	Mode              string
	LinkCount         int64
}

type incrementalCollectResult struct {
	SnapshotName             string
	MaterializedSnapshotName string
	ScanRoot                 string
	DiffMethod               string
	ChangedPathCount         int64
	ChangedPrefixes          []string
	RemovedPrefixes          []string
	NewEntries               []model.StorageIndexEntry
	NewDirMetrics            []model.StorageIndexDirectoryMetric
	NewCandidates            []model.StorageIndexCandidate
	NewCandidateFiles        []model.StorageIndexCandidateFile
	NewHits                  []model.StorageIndexRedundancyHit
}

type incrementalPlan struct {
	RescanTargets   []topLevelSignature
	UpsertEntries   []model.StorageIndexEntry
	RemovedPrefixes []string
	ComparedNodes   int64
	PrunedDirs      int64
	NewNodes        int64
	UpdatedNodes    int64
	RemovedNodes    int64
	ReusedNodes     int64
}

type publicResourceRoot struct {
	Name        string
	LogicalPath string
	Category    string
}

type publicBaselineBuildResult struct {
	Roots []model.StorageIndexPublicRootBaseline
	Files []model.StorageIndexPublicFileBaseline
}

func (j *DirectoryCompareJob) setRunning(startedAt time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = DirectoryCompareJobStatusRunning
	j.StartedAt = &startedAt
}

func (j *DirectoryCompareJob) setDone(result *DirectoryCompareResult, finishedAt time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = DirectoryCompareJobStatusDone
	j.Result = result
	j.Error = ""
	j.FinishedAt = &finishedAt
}

func (j *DirectoryCompareJob) setError(message string, finishedAt time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = DirectoryCompareJobStatusError
	j.Result = nil
	j.Error = message
	j.FinishedAt = &finishedAt
}

func (j *DirectoryCompareJob) snapshot() *DirectoryCompareJobView {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return &DirectoryCompareJobView{
		JobID:       j.JobID,
		Status:      j.Status,
		LeftPath:    j.LeftPath,
		RightPath:   j.RightPath,
		CompareType: j.CompareType,
		CompareMode: j.CompareMode,
		Result:      j.Result,
		Error:       j.Error,
		StartedAt:   j.StartedAt,
		FinishedAt:  j.FinishedAt,
	}
}

func NewService(kubeClient kubernetes.Interface, kubeConfig *rest.Config) *Service {
	return &Service{
		kubeClient: kubeClient,
		kubeConfig: kubeConfig,
	}
}

func (s *Service) StartCompareDirectories(
	_ context.Context,
	leftPath string,
	rightPath string,
	compareType string,
	compareMode string,
) (string, error) {
	jobID := "cmp-" + uuid.NewString()
	job := &DirectoryCompareJob{
		JobID:       jobID,
		Status:      DirectoryCompareJobStatusPending,
		LeftPath:    normalizeCompareLogicalPath(leftPath),
		RightPath:   normalizeCompareLogicalPath(rightPath),
		CompareType: normalizeDirectoryCompareType(compareType),
		CompareMode: normalizeDirectoryCompareMode(compareMode),
	}
	directoryCompareJobs.Store(jobID, job)

	go func() {
		startedAt := time.Now()
		job.setRunning(startedAt)

		runCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		result, err := s.CompareDirectories(
			runCtx,
			leftPath,
			rightPath,
			compareType,
			compareMode,
		)
		finishedAt := time.Now()
		if err != nil {
			job.setError(err.Error(), finishedAt)
			return
		}
		job.setDone(result, finishedAt)
	}()

	return jobID, nil
}

func (s *Service) GetCompareDirectoryJob(jobID string) (*DirectoryCompareJobView, error) {
	value, ok := directoryCompareJobs.Load(strings.TrimSpace(jobID))
	if !ok {
		return nil, fmt.Errorf("compare job %s not found", jobID)
	}

	job, ok := value.(*DirectoryCompareJob)
	if !ok {
		return nil, fmt.Errorf("compare job %s has invalid type", jobID)
	}

	return job.snapshot(), nil
}

func (s *Service) StartFullScan(ctx context.Context, req StartScanRequest) (string, error) {
	workspace, err := s.resolveWorkspace(ctx, req.WorkspaceType, req.WorkspaceName)
	if err != nil {
		return "", err
	}

	triggerSource := strings.TrimSpace(req.TriggerSource)
	if triggerSource == "" {
		triggerSource = manualTriggerSource
	}
	scanMode := req.ScanMode
	if scanMode == "" {
		scanMode = model.StorageIndexScanModeFull
	}

	scanID := uuid.NewString()
	baseScanID, _ := s.findLatestCompletedScanID(ctx, workspace)
	job := &model.StorageIndexScanJob{
		ScanID:        scanID,
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		LogicalPath:   workspace.LogicalPath,
		TriggerSource: triggerSource,
		ScanMode:      scanMode,
		BaseScanID:    baseScanID,
		DiffMethod:    "db_diff",
		Status:        model.StorageIndexScanStatusPending,
	}

	if err := query.GetDB().WithContext(ctx).Create(job).Error; err != nil {
		return "", fmt.Errorf("create metadata scan job failed: %w", err)
	}

	klog.Infof("storageindex: 已创建扫描任务 scan_id=%s workspace_type=%s workspace_name=%s logical_path=%s",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath)

	go s.runFullScan(context.Background(), scanID, workspace)

	return scanID, nil
}

func (s *Service) RunFullScanNow(ctx context.Context, req StartScanRequest) (*model.StorageIndexScanJob, error) {
	workspace, err := s.resolveWorkspace(ctx, req.WorkspaceType, req.WorkspaceName)
	if err != nil {
		return nil, err
	}

	triggerSource := strings.TrimSpace(req.TriggerSource)
	if triggerSource == "" {
		triggerSource = manualTriggerSource
	}
	scanMode := req.ScanMode
	if scanMode == "" {
		scanMode = model.StorageIndexScanModeFull
	}

	scanID := uuid.NewString()
	baseScanID, _ := s.findLatestCompletedScanID(ctx, workspace)
	job := &model.StorageIndexScanJob{
		ScanID:        scanID,
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		LogicalPath:   workspace.LogicalPath,
		TriggerSource: triggerSource,
		ScanMode:      scanMode,
		BaseScanID:    baseScanID,
		DiffMethod:    "db_diff",
		Status:        model.StorageIndexScanStatusPending,
	}
	if err := query.GetDB().WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("create metadata scan job failed: %w", err)
	}

	s.runFullScan(ctx, scanID, workspace)
	return s.GetScanJob(ctx, scanID)
}

func (s *Service) RefreshPublicBaseline(ctx context.Context) (*model.StorageIndexScanJob, error) {
	workspace := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypePublic,
		WorkspaceName: "public",
		LogicalPath:   "/public",
	}
	scanID := uuid.NewString()
	startedAt := time.Now()

	job := &model.StorageIndexScanJob{
		ScanID:        scanID,
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		LogicalPath:   workspace.LogicalPath,
		TriggerSource: "patrol_daily_public_baseline",
		ScanMode:      model.StorageIndexScanModeDailyRefresh,
		DiffMethod:    "business_registry_baseline",
		Status:        model.StorageIndexScanStatusRunning,
		StartedAt:     &startedAt,
	}
	if err := query.GetDB().WithContext(ctx).Create(job).Error; err != nil {
		return nil, fmt.Errorf("create public baseline job failed: %w", err)
	}

	count, err := s.rebuildPublicFileBaseline(ctx, scanID, workspace)
	finishedAt := time.Now()
	updates := map[string]any{
		"finished_at": finishedAt,
		"latency_ms":  finishedAt.Sub(startedAt).Milliseconds(),
	}
	if err != nil {
		updates["status"] = model.StorageIndexScanStatusError
		updates["error_message"] = err.Error()
		_ = query.GetDB().WithContext(ctx).
			Model(&model.StorageIndexScanJob{}).
			Where("scan_id = ?", scanID).
			Updates(updates).Error
		return nil, err
	}

	updates["status"] = model.StorageIndexScanStatusDone
	updates["entry_count"] = count
	updates["file_count"] = count
	updates["directory_count"] = 0
	updates["total_size_bytes"] = 0
	updates["redundancy_count"] = 0
	updates["redundancy_bytes"] = 0
	updates["error_message"] = ""
	if err := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexScanJob{}).
		Where("scan_id = ?", scanID).
		Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("update public baseline job failed: %w", err)
	}

	return s.GetScanJob(ctx, scanID)
}

func (s *Service) RefreshAllUserWorkspaces(ctx context.Context) (map[string]any, error) {
	type userRow struct {
		Name  string `gorm:"column:name"`
		Space string `gorm:"column:space"`
	}
	var users []userRow
	if err := query.GetDB().WithContext(ctx).
		Raw("SELECT name, space FROM users WHERE deleted_at IS NULL AND space <> '' ORDER BY id ASC").
		Scan(&users).Error; err != nil {
		return nil, fmt.Errorf("query user workspaces failed: %w", err)
	}

	success := 0
	failed := 0
	results := make([]map[string]any, 0, len(users))
	for _, user := range users {
		job, err := s.RunFullScanNow(ctx, StartScanRequest{
			WorkspaceType: model.StorageIndexWorkspaceTypeUser,
			WorkspaceName: user.Name,
			TriggerSource: "patrol_daily_user_refresh",
			ScanMode:      model.StorageIndexScanModeDailyRefresh,
		})
		if err != nil {
			failed++
			results = append(results, map[string]any{
				"user":   user.Name,
				"status": "error",
				"error":  err.Error(),
			})
			klog.Warningf("storageindex: 用户空间每日刷新失败 user=%s err=%v", user.Name, err)
			continue
		}
		success++
		results = append(results, map[string]any{
			"user":    user.Name,
			"status":  job.Status,
			"scan_id": job.ScanID,
		})
	}

	return map[string]any{
		"total":   len(users),
		"success": success,
		"failed":  failed,
		"items":   results,
	}, nil
}

func (s *Service) GetScanJob(ctx context.Context, scanID string) (*model.StorageIndexScanJob, error) {
	var job model.StorageIndexScanJob
	if err := query.GetDB().WithContext(ctx).
		Where("scan_id = ?", scanID).
		First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("metadata scan job %s not found", scanID)
		}
		return nil, fmt.Errorf("query metadata scan job failed: %w", err)
	}
	return &job, nil
}

func (s *Service) GetWorkspaceOverview(
	ctx context.Context,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
) (*WorkspaceOverview, error) {
	workspace, err := s.resolveWorkspace(ctx, workspaceType, workspaceName)
	if err != nil {
		return nil, err
	}

	var latestJob model.StorageIndexScanJob
	if err := query.GetDB().WithContext(ctx).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		// Prefer the newest created scan job so an in-flight run remains visible
		// while directory metrics and candidate verification are still being built.
		Order("created_at DESC, updated_at DESC, id DESC").
		First(&latestJob).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &WorkspaceOverview{
				WorkspaceType:   workspace.WorkspaceType,
				WorkspaceName:   workspace.WorkspaceName,
				LogicalPath:     workspace.LogicalPath,
				LastScanID:      "",
				LastScanStatus:  "",
				LastScanAt:      nil,
				EntryCount:      0,
				FileCount:       0,
				DirectoryCount:  0,
				RedundancyCount: 0,
				RedundancyBytes: 0,
				TopDirectories:  []DirectorySummary{},
				LargestFiles:    []FileSummary{},
			}, nil
		}
		return nil, fmt.Errorf("query latest metadata scan job failed: %w", err)
	}
	lastScanAt := latestJob.FinishedAt
	if lastScanAt == nil {
		lastScanAt = latestJob.StartedAt
	}

	var topDirectoryRows []model.StorageIndexDirectoryMetric
	if err := query.GetDB().WithContext(ctx).
		Where("workspace_type = ? AND workspace_name = ? AND path <> ?", workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath).
		Order("total_size_bytes DESC, file_count DESC").
		Limit(defaultOverviewLimit).
		Find(&topDirectoryRows).Error; err != nil {
		return nil, fmt.Errorf("query directory overview failed: %w", err)
	}

	var largestFiles []model.StorageIndexCandidateFile
	if err := query.GetDB().WithContext(ctx).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Order("size_bytes DESC").
		Limit(defaultOverviewLimit).
		Find(&largestFiles).Error; err != nil {
		return nil, fmt.Errorf("query largest candidate files failed: %w", err)
	}

	var redundancySummary struct {
		Count int64 `gorm:"column:count"`
		Bytes int64 `gorm:"column:bytes"`
	}
	if err := query.GetDB().WithContext(ctx).
		Table(model.StorageIndexRedundancyHit{}.TableName()).
		Select("COUNT(*) AS count, COALESCE(SUM(estimated_bytes), 0) AS bytes").
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Scan(&redundancySummary).Error; err != nil {
		return nil, fmt.Errorf("query redundancy summary failed: %w", err)
	}

	overview := &WorkspaceOverview{
		WorkspaceType:   workspace.WorkspaceType,
		WorkspaceName:   workspace.WorkspaceName,
		LogicalPath:     workspace.LogicalPath,
		LastScanID:      latestJob.ScanID,
		LastScanStatus:  latestJob.Status,
		LastScanAt:      lastScanAt,
		EntryCount:      latestJob.EntryCount,
		FileCount:       latestJob.FileCount,
		DirectoryCount:  latestJob.DirectoryCount,
		RedundancyCount: redundancySummary.Count,
		RedundancyBytes: redundancySummary.Bytes,
		TopDirectories:  make([]DirectorySummary, 0, len(topDirectoryRows)),
		LargestFiles:    make([]FileSummary, 0, len(largestFiles)),
	}

	for _, item := range topDirectoryRows {
		overview.TopDirectories = append(overview.TopDirectories, DirectorySummary{
			Path:           item.Path,
			Name:           item.Name,
			Depth:          item.Depth,
			FileCount:      item.FileCount,
			DirectoryCount: item.DirectoryCount,
			TotalSizeBytes: item.TotalSizeBytes,
			IsTopLevel:     item.IsTopLevel,
		})
	}

	for _, item := range largestFiles {
		overview.LargestFiles = append(overview.LargestFiles, FileSummary{
			Path:       item.FilePath,
			Name:       item.FileName,
			SizeBytes:  item.SizeBytes,
			ModifiedAt: nil,
		})
	}

	return overview, nil
}

func (s *Service) ListRedundancyHits(
	ctx context.Context,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	page,
	pageSize int,
) ([]model.StorageIndexRedundancyHit, int64, error) {
	workspace, err := s.resolveWorkspace(ctx, workspaceType, workspaceName)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultRedundancyPageSize
	}

	db := query.GetDB().WithContext(ctx)
	var total int64
	if err := db.Model(&model.StorageIndexRedundancyHit{}).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count redundancy hits failed: %w", err)
	}

	var hits []model.StorageIndexRedundancyHit
	if err := db.
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Order("estimated_bytes DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&hits).Error; err != nil {
		return nil, 0, fmt.Errorf("list redundancy hits failed: %w", err)
	}

	return hits, total, nil
}

func (s *Service) ListCandidates(
	ctx context.Context,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	page int,
	pageSize int,
) ([]model.StorageIndexCandidate, int64, error) {
	workspace, err := s.resolveWorkspace(ctx, workspaceType, workspaceName)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultRedundancyPageSize
	}

	db := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexCandidate{}).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count candidates failed: %w", err)
	}

	var items []model.StorageIndexCandidate
	if err := db.
		Order("CASE status WHEN 'verified' THEN 0 WHEN 'suspected' THEN 1 ELSE 2 END ASC").
		Order("candidate_score DESC, target_path ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list candidates failed: %w", err)
	}

	return items, total, nil
}

func (s *Service) ListCandidateFiles(
	ctx context.Context,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	candidatePath string,
	page int,
	pageSize int,
) ([]model.StorageIndexCandidateFile, int64, error) {
	workspace, err := s.resolveWorkspace(ctx, workspaceType, workspaceName)
	if err != nil {
		return nil, 0, err
	}
	candidatePath = normalizeUnixPath(candidatePath)
	if candidatePath == "" {
		return nil, 0, fmt.Errorf("candidate path cannot be empty")
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 200
	}

	db := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexCandidateFile{}).
		Where(
			"workspace_type = ? AND workspace_name = ? AND candidate_path = ?",
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			candidatePath,
		)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count candidate files failed: %w", err)
	}

	var items []model.StorageIndexCandidateFile
	if err := db.
		Order("relative_path ASC, file_name ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list candidate files failed: %w", err)
	}

	return items, total, nil
}

func (s *Service) CompareDirectories(
	_ context.Context,
	leftPath string,
	rightPath string,
	compareType string,
	compareMode string,
) (*DirectoryCompareResult, error) {
	startedAt := time.Now()
	leftPath = normalizeCompareLogicalPath(leftPath)
	rightPath = normalizeCompareLogicalPath(rightPath)
	if leftPath == "" || rightPath == "" {
		return nil, fmt.Errorf("compare paths cannot be empty")
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	toolboxPod, err := ceph.FindCephToolboxPod(s.kubeClient, toolboxNamespace)
	if err != nil {
		return nil, fmt.Errorf("find ceph toolbox pod failed: %w", err)
	}

	leftActual, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, leftPath, prefixConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve left path failed: %w", err)
	}
	rightActual, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, rightPath, prefixConfig)
	if err != nil {
		return nil, fmt.Errorf("resolve right path failed: %w", err)
	}

	scanStartedAt := time.Now()
	resolvedCompareType, leftFiles, rightFiles, err := s.scanFilesForDirectoryCompare(toolboxPod, leftActual, rightActual, compareType)
	if err != nil {
		return nil, err
	}
	resolvedCompareMode := normalizeDirectoryCompareMode(compareMode)

	result := &DirectoryCompareResult{
		CompareType:       resolvedCompareType,
		CompareMode:       resolvedCompareMode,
		LeftPath:          leftPath,
		RightPath:         rightPath,
		LeftKeyFileCount:  len(leftFiles),
		RightKeyFileCount: len(rightFiles),
		MissingLeft:       make([]string, 0),
		MissingRight:      make([]string, 0),
		Files:             make([]DirectoryCompareFileResult, 0),
	}
	result.Timing.ScanMs = time.Since(scanStartedAt).Milliseconds()

	pairingStartedAt := time.Now()
	var (
		pairs              []struct{ left, right candidateFileProbe }
		exactMatchCount    int
		fallbackMatchCount int
		missingLeft        []string
		missingRight       []string
	)
	if resolvedCompareType == "dataset" {
		pairs, exactMatchCount, fallbackMatchCount, missingLeft, missingRight = pairDirectoryFilesByNameAndSize(leftFiles, rightFiles)
	} else {
		pairs, exactMatchCount, fallbackMatchCount, missingLeft, missingRight = pairCandidateFilesForComparison(leftFiles, rightFiles)
	}
	result.ExactMatchCount = exactMatchCount
	result.FallbackMatchCount = fallbackMatchCount
	result.MissingLeft = missingLeft
	result.MissingRight = missingRight
	result.Timing.PairingMs = time.Since(pairingStartedAt).Milliseconds()

	headerStartedAt := time.Now()
	type hashPair struct {
		left  candidateFileProbe
		right candidateFileProbe
		file  *DirectoryCompareFileResult
	}
	hashPairs := make([]hashPair, 0, len(pairs))
	for _, pair := range pairs {
		fileResult := DirectoryCompareFileResult{
			LeftRelativePath:  pair.left.RelativePath,
			RightRelativePath: pair.right.RelativePath,
			FileName:          pair.left.FileName,
			SizeBytes:         pair.left.SizeBytes,
			VerificationMode:  verificationModeFileName,
			Same:              false,
		}
		if resolvedCompareType == "dataset" {
			fileResult.Same = true
			result.VerifiedFileCount++
			result.Files = append(result.Files, fileResult)
			continue
		}
		if resolvedCompareMode == compareModeOptimized &&
			isSafeTensorsFile(pair.left.FileName) &&
			isSafeTensorsFile(pair.right.FileName) {
			headerMatched, headerErr := s.compareSafetensorsHeaders(toolboxPod, pair.left.ActualPath, pair.right.ActualPath)
			fileResult.HeaderMatched = &headerMatched
			if headerErr != nil {
				fileResult.Reason = "safetensors_header_error"
				result.Files = append(result.Files, fileResult)
				continue
			}
			if !headerMatched {
				fileResult.Reason = "safetensors_header_mismatch"
				result.Files = append(result.Files, fileResult)
				continue
			}
			fileResult.VerificationMode = verificationModeSafeTensorsHdrAndSampledSHA
		}
		if fileResult.VerificationMode == verificationModeFileName && resolvedCompareMode == compareModeOptimized {
			fileResult.VerificationMode = hashAlgorithmSampledSHA256
		}
		if fileResult.VerificationMode == verificationModeFileName && resolvedCompareMode == compareModeFullHash {
			fileResult.VerificationMode = hashAlgorithmSHA256
		}
		result.Files = append(result.Files, fileResult)
		hashPairs = append(hashPairs, hashPair{
			left:  pair.left,
			right: pair.right,
			file:  &result.Files[len(result.Files)-1],
		})
	}
	result.Timing.HeaderMs = time.Since(headerStartedAt).Milliseconds()

	if resolvedCompareType == "dataset" {
		result.ComparedFileCount = len(result.Files)
		result.Same = len(result.MissingLeft) == 0 &&
			len(result.MissingRight) == 0 &&
			result.ComparedFileCount > 0 &&
			result.VerifiedFileCount == result.ComparedFileCount
		result.Timing.TotalMs = time.Since(startedAt).Milliseconds()
		return result, nil
	}

	hashStartedAt := time.Now()
	leftActualPaths := make([]string, 0, len(hashPairs))
	rightActualPaths := make([]string, 0, len(hashPairs))
	leftSizes := make(map[string]int64, len(hashPairs))
	rightSizes := make(map[string]int64, len(hashPairs))
	for _, pair := range hashPairs {
		leftActualPaths = append(leftActualPaths, pair.left.ActualPath)
		rightActualPaths = append(rightActualPaths, pair.right.ActualPath)
		leftSizes[pair.left.ActualPath] = pair.left.SizeBytes
		rightSizes[pair.right.ActualPath] = pair.right.SizeBytes
	}

	leftHashes := map[string]string{}
	rightHashes := map[string]string{}
	if len(hashPairs) > 0 {
		switch resolvedCompareMode {
		case compareModeFullHash:
			leftHashes, err = s.computeActualFileFullHashesBatch(toolboxPod, leftActualPaths)
			if err != nil {
				return nil, fmt.Errorf("compute left full hashes failed: %w", err)
			}
			rightHashes, err = s.computeActualFileFullHashesBatch(toolboxPod, rightActualPaths)
			if err != nil {
				return nil, fmt.Errorf("compute right full hashes failed: %w", err)
			}
		default:
			leftHashes, err = s.computeActualFileHashesBatchWithSizes(toolboxPod, leftActualPaths, leftSizes)
			if err != nil {
				return nil, fmt.Errorf("compute left sampled hashes failed: %w", err)
			}
			rightHashes, err = s.computeActualFileHashesBatchWithSizes(toolboxPod, rightActualPaths, rightSizes)
			if err != nil {
				return nil, fmt.Errorf("compute right sampled hashes failed: %w", err)
			}
		}
	}
	for _, pair := range hashPairs {
		leftHash := leftHashes[pair.left.ActualPath]
		rightHash := rightHashes[pair.right.ActualPath]
		matched := leftHash != "" && rightHash != "" && leftHash == rightHash
		pair.file.SampledHashMatch = &matched
		pair.file.Same = matched
		if !matched {
			pair.file.Reason = "sampled_hash_mismatch"
			continue
		}
		result.VerifiedFileCount++
	}
	if resolvedCompareMode == compareModeFullHash {
		result.Timing.FullHashMs = time.Since(hashStartedAt).Milliseconds()
	} else {
		result.Timing.SampledHashMs = time.Since(hashStartedAt).Milliseconds()
	}

	result.ComparedFileCount = len(result.Files)
	result.Same = len(result.MissingLeft) == 0 &&
		len(result.MissingRight) == 0 &&
		result.ComparedFileCount > 0 &&
		result.VerifiedFileCount == result.ComparedFileCount
	result.Timing.TotalMs = time.Since(startedAt).Milliseconds()
	return result, nil
}

func (s *Service) runFullScan(ctx context.Context, scanID string, workspace resolvedWorkspace) {
	startedAt := time.Now()
	db := query.GetDB().WithContext(ctx)

	if err := db.Model(&model.StorageIndexScanJob{}).
		Where("scan_id = ?", scanID).
		Updates(map[string]any{
			"status":     model.StorageIndexScanStatusRunning,
			"started_at": startedAt,
		}).Error; err != nil {
		klog.Errorf("storageindex: 标记扫描任务运行中失败 scan_id=%s err=%v", scanID, err)
	}

	currentJob, _ := s.getScanJobByID(ctx, scanID)
	baseScanID := ""
	scanMode := model.StorageIndexScanModeFull
	if currentJob != nil {
		baseScanID = currentJob.BaseScanID
		if currentJob.ScanMode != "" {
			scanMode = currentJob.ScanMode
		}
	}

	klog.Infof(
		"storageindex: 开始扫描 scan_id=%s workspace_type=%s workspace_name=%s scan_mode=%s base_scan_id=%s",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, scanMode, baseScanID,
	)

	if scanMode == model.StorageIndexScanModeDailyRefresh && baseScanID != "" {
		klog.Infof(
			"storageindex: 尝试执行增量扫描 scan_id=%s workspace_type=%s workspace_name=%s base_scan_id=%s",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, baseScanID,
		)
		ok, err := s.runIncrementalScan(ctx, scanID, workspace, baseScanID, startedAt)
		if err != nil {
			klog.Warningf(
				"storageindex: 增量扫描失败，回退为全量扫描 scan_id=%s workspace_type=%s workspace_name=%s base_scan_id=%s err=%v",
				scanID, workspace.WorkspaceType, workspace.WorkspaceName, baseScanID, err,
			)
		}
		if ok {
			return
		}
	}

	if scanMode == model.StorageIndexScanModeDailyRefresh && baseScanID == "" {
		klog.Infof(
			"storageindex: 没有可用基线，改为全量扫描 scan_id=%s workspace_type=%s workspace_name=%s",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName,
		)
	}

	if baseScanID == "" {
		clearedRows, err := cleanupWorkspaceStateBeforeInitialScan(db, workspace)
		if err != nil {
			s.markScanError(ctx, scanID, startedAt, fmt.Errorf("cleanup workspace state before initial scan failed: %w", err))
			return
		}
		klog.Infof(
			"storageindex: 首次扫描前已清理残留数据 scan_id=%s workspace_type=%s workspace_name=%s cleared_rows=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, clearedRows,
		)
	}

	if err := s.runFullSnapshotScan(ctx, scanID, workspace, baseScanID, startedAt); err != nil {
		s.markScanError(ctx, scanID, startedAt, err)
	}
}

func (s *Service) markScanError(ctx context.Context, scanID string, startedAt time.Time, runErr error) {
	finishedAt := time.Now()
	if err := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexScanJob{}).
		Where("scan_id = ?", scanID).
		Updates(map[string]any{
			"status":        model.StorageIndexScanStatusError,
			"error_message": runErr.Error(),
			"finished_at":   finishedAt,
			"latency_ms":    finishedAt.Sub(startedAt).Milliseconds(),
		}).Error; err != nil {
		klog.Errorf("storageindex: 标记扫描任务失败失败 scan_id=%s err=%v original=%v", scanID, err, runErr)
	}
}

func (s *Service) runFullSnapshotScan(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
	baseScanID string,
	startedAt time.Time,
) error {
	snapshotName, materializedSnapshotName, scanRoot, entries, dirMetrics, changedPathCount, hits, totalSize, err := s.collectWorkspaceSnapshot(ctx, scanID, workspace, baseScanID)
	if err != nil {
		return err
	}

	fileCount := int64(0)
	dirCount := int64(0)
	for _, entry := range entries {
		switch entry.EntryType {
		case model.StorageIndexEntryTypeFile:
			fileCount++
		case model.StorageIndexEntryTypeDir:
			dirCount++
		}
	}

	redundancyBytes := int64(0)
	for _, hit := range hits {
		redundancyBytes += hit.EstimatedBytes
	}
	klog.Infof(
		"storageindex: 开始保存全量扫描结果 scan_id=%s workspace_type=%s workspace_name=%s entries=%d metrics=%d hits=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(entries), len(dirMetrics), len(hits),
	)

	db := query.GetDB().WithContext(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Delete(&model.StorageIndexEntry{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Delete(&model.StorageIndexDirectoryMetric{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Delete(&model.StorageIndexRedundancyHit{}).Error; err != nil {
			return err
		}

		if len(entries) > 0 {
			if err := insertEntriesInChunks(tx, scanID, workspace, entries, insertBatchSize); err != nil {
				return err
			}
		}
		if len(dirMetrics) > 0 {
			if err := insertDirectoryMetricsInChunks(tx, scanID, workspace, dirMetrics, insertBatchSize); err != nil {
				return err
			}
		}
		if len(hits) > 0 {
			if err := insertRedundancyHitsInChunks(tx, scanID, workspace, hits, insertBatchSize); err != nil {
				return err
			}
		}

		return tx.Model(&model.StorageIndexScanJob{}).
			Where("scan_id = ?", scanID).
			Updates(map[string]any{
				"snapshot_name":              snapshotName,
				"materialized_snapshot_name": materializedSnapshotName,
				"scan_root":                  scanRoot,
				"status":                     model.StorageIndexScanStatusRunning,
				"entry_count":                len(entries),
				"file_count":                 fileCount,
				"directory_count":            dirCount,
				"total_size_bytes":           totalSize,
				"base_scan_id":               baseScanID,
				"diff_method":                "db_diff",
				"changed_path_count":         changedPathCount,
				"redundancy_count":           len(hits),
				"redundancy_bytes":           redundancyBytes,
				"finished_at":                nil,
				"latency_ms":                 0,
				"error_message":              "",
			}).Error
	})
	if err != nil {
		return fmt.Errorf("persist full metadata scan result failed: %w", err)
	}

	if err := s.rebuildCandidates(ctx, scanID, workspace); err != nil {
		return fmt.Errorf("rebuild candidates after full scan failed: %w", err)
	}
	if workspace.WorkspaceType == model.StorageIndexWorkspaceTypePublic {
		if _, err := s.rebuildPublicFileBaseline(ctx, scanID, workspace); err != nil {
			return fmt.Errorf("rebuild public file baseline failed: %w", err)
		}
	}

	finalCounts, err := s.queryWorkspaceCounts(ctx, workspace)
	if err != nil {
		return fmt.Errorf("query final workspace counts failed: %w", err)
	}
	finishedAt := time.Now()
	if err := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexScanJob{}).
		Where("scan_id = ?", scanID).
		Updates(map[string]any{
			"status":           model.StorageIndexScanStatusDone,
			"entry_count":      finalCounts.EntryCount,
			"file_count":       finalCounts.FileCount,
			"directory_count":  finalCounts.DirectoryCount,
			"total_size_bytes": finalCounts.TotalSizeBytes,
			"redundancy_count": finalCounts.RedundancyCount,
			"redundancy_bytes": finalCounts.RedundancyBytes,
			"finished_at":      finishedAt,
			"latency_ms":       finishedAt.Sub(startedAt).Milliseconds(),
		}).Error; err != nil {
		return fmt.Errorf("finalize full metadata scan job failed: %w", err)
	}

	klog.Infof(
		"storageindex: 全量扫描完成 scan_id=%s workspace_type=%s workspace_name=%s mode=full entries=%d dirs=%d files=%d changed_paths=%d redundancy_hits=%d redundancy_bytes=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName,
		finalCounts.EntryCount, finalCounts.DirectoryCount, finalCounts.FileCount, changedPathCount, finalCounts.RedundancyCount, finalCounts.RedundancyBytes,
	)
	return nil
}

func (s *Service) runIncrementalScan(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
	baseScanID string,
	startedAt time.Time,
) (bool, error) {
	result, err := s.collectWorkspaceIncremental(ctx, scanID, workspace, baseScanID)
	if err != nil {
		return false, err
	}

	db := query.GetDB().WithContext(ctx)
	finalCounts := workspaceCounts{}
	err = db.Transaction(func(tx *gorm.DB) error {
		prefixesToDelete := append([]string{}, result.ChangedPrefixes...)
		prefixesToDelete = append(prefixesToDelete, result.RemovedPrefixes...)

		for _, prefix := range prefixesToDelete {
			if err := deleteWorkspacePathPrefix(tx, workspace, prefix); err != nil {
				return err
			}
		}
		for _, entry := range result.NewEntries {
			if isCoveredByPrefixes(entry.LogicalPath, result.ChangedPrefixes) {
				continue
			}
			if err := deleteWorkspaceExactPath(tx, workspace, entry.LogicalPath); err != nil {
				return err
			}
		}

		if len(result.NewEntries) > 0 {
			if err := insertEntriesInChunks(tx, scanID, workspace, result.NewEntries, insertBatchSize); err != nil {
				return err
			}
		}
		if len(result.NewDirMetrics) > 0 {
			if err := insertDirectoryMetricsInChunks(tx, scanID, workspace, result.NewDirMetrics, insertBatchSize); err != nil {
				return err
			}
		}
		if err := rebuildWorkspaceRootMetric(tx, workspace, scanID); err != nil {
			return err
		}

		affectedMetricPaths := collectAffectedMetricPaths(workspace.LogicalPath, result.ChangedPrefixes, result.RemovedPrefixes, result.NewEntries)
		for _, metricPath := range affectedMetricPaths {
			if isCoveredByPrefixes(metricPath, result.ChangedPrefixes) {
				continue
			}
			if err := deleteWorkspaceMetricExactPath(tx, workspace, metricPath); err != nil {
				return err
			}
			if err := rebuildDirectoryMetric(tx, workspace, scanID, metricPath); err != nil {
				return err
			}
		}

		counts, err := s.queryWorkspaceCountsWithDB(tx, workspace)
		if err != nil {
			return err
		}
		finalCounts = counts
		return tx.Model(&model.StorageIndexScanJob{}).
			Where("scan_id = ?", scanID).
			Updates(map[string]any{
				"snapshot_name":              result.SnapshotName,
				"materialized_snapshot_name": result.MaterializedSnapshotName,
				"scan_root":                  result.ScanRoot,
				"status":                     model.StorageIndexScanStatusRunning,
				"entry_count":                counts.EntryCount,
				"file_count":                 counts.FileCount,
				"directory_count":            counts.DirectoryCount,
				"total_size_bytes":           counts.TotalSizeBytes,
				"base_scan_id":               baseScanID,
				"diff_method":                result.DiffMethod,
				"changed_path_count":         result.ChangedPathCount,
				"redundancy_count":           counts.RedundancyCount,
				"redundancy_bytes":           counts.RedundancyBytes,
				"finished_at":                nil,
				"latency_ms":                 0,
				"error_message":              "",
			}).Error
	})
	if err != nil {
		return false, fmt.Errorf("persist incremental metadata scan result failed: %w", err)
	}

	if err := s.refreshIncrementalDerivedState(ctx, scanID, workspace, result); err != nil {
		return false, fmt.Errorf("refresh incremental derived state failed: %w", err)
	}
	if workspace.WorkspaceType == model.StorageIndexWorkspaceTypePublic {
		if _, err := s.rebuildPublicFileBaseline(ctx, scanID, workspace); err != nil {
			return false, fmt.Errorf("rebuild public file baseline failed: %w", err)
		}
	}

	finalCounts, err = s.queryWorkspaceCounts(ctx, workspace)
	if err != nil {
		return false, fmt.Errorf("query final workspace counts failed: %w", err)
	}
	finishedAt := time.Now()
	if err := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexScanJob{}).
		Where("scan_id = ?", scanID).
		Updates(map[string]any{
			"status":           model.StorageIndexScanStatusDone,
			"entry_count":      finalCounts.EntryCount,
			"file_count":       finalCounts.FileCount,
			"directory_count":  finalCounts.DirectoryCount,
			"total_size_bytes": finalCounts.TotalSizeBytes,
			"redundancy_count": finalCounts.RedundancyCount,
			"redundancy_bytes": finalCounts.RedundancyBytes,
			"finished_at":      finishedAt,
			"latency_ms":       finishedAt.Sub(startedAt).Milliseconds(),
		}).Error; err != nil {
		return false, fmt.Errorf("finalize incremental metadata scan job failed: %w", err)
	}

	klog.Infof(
		"storageindex: 增量扫描完成 scan_id=%s workspace_type=%s workspace_name=%s mode=incremental diff_method=%s changed_paths=%d rescan_targets=%d upsert_entries=%d removed_prefixes=%d entry_count=%d file_count=%d dir_count=%d redundancy_hits=%d redundancy_bytes=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, result.DiffMethod, result.ChangedPathCount,
		len(result.ChangedPrefixes), len(result.NewEntries), len(result.RemovedPrefixes),
		finalCounts.EntryCount, finalCounts.FileCount, finalCounts.DirectoryCount, finalCounts.RedundancyCount, finalCounts.RedundancyBytes,
	)
	return true, nil
}

func (s *Service) collectWorkspaceSnapshot(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
	baseScanID string,
) (string, string, string, []model.StorageIndexEntry, []model.StorageIndexDirectoryMetric, int64, []model.StorageIndexRedundancyHit, int64, error) {
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	toolboxPod, err := ceph.FindCephToolboxPod(s.kubeClient, toolboxNamespace)
	if err != nil {
		return "", "", "", nil, nil, 0, nil, 0, fmt.Errorf("find ceph toolbox pod failed: %w", err)
	}

	rootPath, err := ceph.ResolveCephFSPath(
		s.kubeClient,
		s.kubeConfig,
		toolboxNamespace,
		workspace.LogicalPath,
		prefixConfig,
	)
	if err != nil {
		return "", "", "", nil, nil, 0, nil, 0, fmt.Errorf("resolve workspace path failed: %w", err)
	}

	subvolumeRoot, err := ceph.GetCephMountRoot(s.kubeClient, s.kubeConfig, toolboxNamespace)
	if err != nil {
		return "", "", "", nil, nil, 0, nil, 0, fmt.Errorf("resolve ceph subvolume root failed: %w", err)
	}

	klog.Infof(
		"storageindex: 正在准备扫描路径 scan_id=%s workspace_type=%s workspace_name=%s root_path=%s subvolume_root=%s",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, rootPath, subvolumeRoot,
	)

	scanPath, snapshotName, materializedSnapshotName, cleanup, err := s.prepareScanPath(toolboxPod, subvolumeRoot, rootPath, scanID)
	if err != nil {
		return "", "", "", nil, nil, 0, nil, 0, err
	}

	klog.Infof(
		"storageindex: 开始扫描工作空间条目 scan_id=%s workspace_type=%s workspace_name=%s scan_path=%s snapshot_name=%s",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, scanPath, snapshotName,
	)

	if workspace.WorkspaceType == model.StorageIndexWorkspaceTypePublic {
		return s.collectPublicWorkspaceSnapshot(ctx, toolboxPod, prefixConfig, scanID, workspace, scanPath, snapshotName, materializedSnapshotName, baseScanID)
	}
	if shouldApplyTopLevelModelCopyPrefilter(workspace) {
		return s.collectFilteredUserWorkspaceSnapshot(ctx, toolboxPod, prefixConfig, scanID, workspace, scanPath, snapshotName, materializedSnapshotName, baseScanID)
	}

	entries, err := s.scanWorkspaceEntries(toolboxPod, scanID, workspace, scanPath)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}
	klog.Infof(
		"storageindex: 条目解析完成 scan_id=%s workspace_type=%s workspace_name=%s entry_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(entries),
	)

	dirMetrics, totalSize := buildDirectoryMetrics(scanID, workspace, entries)
	klog.Infof(
		"storageindex: 目录聚合完成 scan_id=%s workspace_type=%s workspace_name=%s metric_count=%d total_size_bytes=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(dirMetrics), totalSize,
	)
	changedPathCount, err := s.applyGrowthFromPreviousScan(ctx, workspace, baseScanID, dirMetrics)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}
	klog.Infof(
		"storageindex: 目录增长差异计算完成 scan_id=%s workspace_type=%s workspace_name=%s base_scan_id=%s changed_path_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, baseScanID, changedPathCount,
	)

	klog.Infof(
		"storageindex: 开始执行冗余检测 scan_id=%s workspace_type=%s workspace_name=%s",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName,
	)
	hits, err := s.detectRedundancy(ctx, toolboxPod, prefixConfig, scanID, workspace, entries, dirMetrics)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}
	klog.Infof(
		"storageindex: 冗余检测完成 scan_id=%s workspace_type=%s workspace_name=%s hit_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(hits),
	)

	return snapshotName, materializedSnapshotName, scanPath, entries, dirMetrics, changedPathCount, hits, totalSize, nil
}

func (s *Service) collectFilteredUserWorkspaceSnapshot(
	ctx context.Context,
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	scanPath string,
	snapshotName string,
	materializedSnapshotName string,
	baseScanID string,
) (string, string, string, []model.StorageIndexEntry, []model.StorageIndexDirectoryMetric, int64, []model.StorageIndexRedundancyHit, int64, error) {
	signatures, err := s.listImmediateSignatures(toolboxPod, scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, scanPath)
	if err != nil {
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}

	selected, skipped := filterTopLevelSignaturesForModelCopyScan(signatures)
	selectedNames := make([]string, 0, len(selected))
	for _, item := range selected {
		selectedNames = append(selectedNames, item.Name)
	}
	skippedNames := make([]string, 0, len(skipped))
	for _, item := range skipped {
		skippedNames = append(skippedNames, item.Name)
	}
	klog.Infof(
		"storageindex: 用户空间顶层目录过滤完成 scan_id=%s workspace_name=%s selected_top_level=%v skipped_top_level=%v total_candidates=%d",
		scanID, workspace.WorkspaceName, selectedNames, skippedNames, len(signatures),
	)

	entries := make([]model.StorageIndexEntry, 0)
	dirMetrics := make([]model.StorageIndexDirectoryMetric, 0)
	for _, sig := range selected {
		if sig.EntryType != model.StorageIndexEntryTypeDir {
			continue
		}

		klog.Infof(
			"storageindex: 开始扫描用户空间候选顶层子树 scan_id=%s subtree=%s actual_path=%s",
			scanID, sig.LogicalPath, sig.ActualPath,
		)

		subtreeEntries, scanErr := s.scanUserWorkspaceSubtree(toolboxPod, scanID, workspace, sig)
		if scanErr != nil {
			return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, scanErr
		}
		entries = append(entries, subtreeEntries...)

		subtreeMetrics, _ := buildDirectoryMetrics(
			scanID,
			resolvedWorkspace{
				WorkspaceType: workspace.WorkspaceType,
				WorkspaceName: workspace.WorkspaceName,
				LogicalPath:   sig.LogicalPath,
			},
			subtreeEntries,
		)
		dirMetrics = append(dirMetrics, subtreeMetrics...)
	}

	rootMetric := model.StorageIndexDirectoryMetric{
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		ScanID:        scanID,
		Path:          workspace.LogicalPath,
		Name:          path.Base(workspace.LogicalPath),
		Depth:         0,
		IsTopLevel:    false,
	}
	for _, entry := range entries {
		switch entry.EntryType {
		case model.StorageIndexEntryTypeFile:
			rootMetric.FileCount++
			rootMetric.TotalSizeBytes += entry.SizeBytes
		case model.StorageIndexEntryTypeDir:
			if entry.LogicalPath != workspace.LogicalPath {
				rootMetric.DirectoryCount++
			}
		}
	}
	dirMetrics = append(dirMetrics, rootMetric)

	changedPathCount, err := s.applyGrowthFromPreviousScan(ctx, workspace, baseScanID, dirMetrics)
	if err != nil {
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}

	hits, err := s.detectRedundancy(ctx, toolboxPod, prefixConfig, scanID, workspace, entries, dirMetrics)
	if err != nil {
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}

	klog.Infof(
		"storageindex: 用户空间候选顶层子树扫描完成 scan_id=%s workspace_name=%s selected_subtrees=%d skipped_subtrees=%d entry_count=%d metric_count=%d hit_count=%d",
		scanID, workspace.WorkspaceName, len(selected), len(skipped), len(entries), len(dirMetrics), len(hits),
	)

	return snapshotName, materializedSnapshotName, scanPath, entries, dirMetrics, changedPathCount, hits, rootMetric.TotalSizeBytes, nil
}

func (s *Service) scanUserWorkspaceSubtree(
	toolboxPod *corev1.Pod,
	scanID string,
	workspace resolvedWorkspace,
	sig topLevelSignature,
) ([]model.StorageIndexEntry, error) {
	if path.Clean(sig.ParentLogicalPath) != path.Clean(workspace.LogicalPath) {
		return s.scanPathDirectories(
			toolboxPod,
			scanID,
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			sig.LogicalPath,
			sig.ActualPath,
		)
	}

	allowList, ok := immediateSubtreeAllowListForTopLevel(sig.Name)
	if !ok {
		return s.scanPathDirectories(
			toolboxPod,
			scanID,
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			sig.LogicalPath,
			sig.ActualPath,
		)
	}

	children, err := s.listImmediateSignatures(
		toolboxPod,
		scanID,
		workspace.WorkspaceType,
		workspace.WorkspaceName,
		sig.LogicalPath,
		sig.ActualPath,
	)
	if err != nil {
		return nil, err
	}

	selectedChildren, skippedChildren := filterImmediateSubtreesByAllowList(children, allowList)
	selectedNames := make([]string, 0, len(selectedChildren))
	skippedNames := make([]string, 0, len(skippedChildren))
	pruneActualPaths := make([]string, 0, len(skippedChildren))
	for _, child := range selectedChildren {
		selectedNames = append(selectedNames, child.Name)
	}
	for _, child := range skippedChildren {
		skippedNames = append(skippedNames, child.Name)
		pruneActualPaths = append(pruneActualPaths, child.ActualPath)
	}

	klog.Infof(
		"storageindex: 顶层子树二级白名单过滤完成 scan_id=%s workspace_name=%s subtree=%s selected_children=%v skipped_children=%v total_children=%d",
		scanID, workspace.WorkspaceName, sig.LogicalPath, selectedNames, skippedNames, len(children),
	)

	return s.scanPathDirectoriesWithPrunedChildren(
		toolboxPod,
		scanID,
		workspace.WorkspaceType,
		workspace.WorkspaceName,
		sig.LogicalPath,
		sig.ActualPath,
		pruneActualPaths,
	)
}

func (s *Service) collectPublicWorkspaceSnapshot(
	ctx context.Context,
	toolboxPod *corev1.Pod,
	_ ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	scanPath string,
	snapshotName string,
	materializedSnapshotName string,
	baseScanID string,
) (string, string, string, []model.StorageIndexEntry, []model.StorageIndexDirectoryMetric, int64, []model.StorageIndexRedundancyHit, int64, error) {
	signatures, err := s.listImmediateSignatures(toolboxPod, scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, scanPath)
	if err != nil {
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}

	selected := make([]topLevelSignature, 0)
	for _, sig := range signatures {
		if _, ok := publicBaselineTopLevelAllowList[strings.ToLower(sig.Name)]; ok {
			selected = append(selected, sig)
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].LogicalPath < selected[j].LogicalPath
	})

	selectedNames := make([]string, 0, len(selected))
	for _, item := range selected {
		selectedNames = append(selectedNames, item.Name)
	}
	klog.Infof(
		"storageindex: 公共基线目录过滤完成 scan_id=%s workspace_name=%s selected_top_level=%v total_candidates=%d",
		scanID, workspace.WorkspaceName, selectedNames, len(signatures),
	)

	entries := make([]model.StorageIndexEntry, 0)
	dirMetrics := make([]model.StorageIndexDirectoryMetric, 0)
	for _, sig := range selected {
		klog.Infof(
			"storageindex: 开始扫描公共空间顶层子树 scan_id=%s subtree=%s actual_path=%s",
			scanID, sig.LogicalPath, sig.ActualPath,
		)

		subtreeEntries, scanErr := s.scanPathDirectories(toolboxPod, scanID, workspace.WorkspaceType, workspace.WorkspaceName, sig.LogicalPath, sig.ActualPath)
		if scanErr != nil {
			return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, scanErr
		}
		entries = append(entries, subtreeEntries...)

		if sig.EntryType == model.StorageIndexEntryTypeDir {
			subtreeMetrics, _ := buildDirectoryMetrics(
				scanID,
				resolvedWorkspace{
					WorkspaceType: workspace.WorkspaceType,
					WorkspaceName: workspace.WorkspaceName,
					LogicalPath:   sig.LogicalPath,
				},
				subtreeEntries,
			)
			dirMetrics = append(dirMetrics, subtreeMetrics...)
		}
	}

	rootMetric := model.StorageIndexDirectoryMetric{
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		ScanID:        scanID,
		Path:          workspace.LogicalPath,
		Name:          path.Base(workspace.LogicalPath),
		Depth:         0,
		IsTopLevel:    false,
	}
	for _, entry := range entries {
		switch entry.EntryType {
		case model.StorageIndexEntryTypeFile:
			rootMetric.FileCount++
			rootMetric.TotalSizeBytes += entry.SizeBytes
		case model.StorageIndexEntryTypeDir:
			if entry.LogicalPath != workspace.LogicalPath {
				rootMetric.DirectoryCount++
			}
		}
	}
	dirMetrics = append(dirMetrics, rootMetric)

	changedPathCount, err := s.applyGrowthFromPreviousScan(ctx, workspace, baseScanID, dirMetrics)
	if err != nil {
		return snapshotName, materializedSnapshotName, scanPath, nil, nil, 0, nil, 0, err
	}

	klog.Infof(
		"storageindex: 公共基线子树扫描完成 scan_id=%s workspace_name=%s selected_subtrees=%d entry_count=%d metric_count=%d total_size_bytes=%d",
		scanID, workspace.WorkspaceName, len(selected), len(entries), len(dirMetrics), rootMetric.TotalSizeBytes,
	)

	return snapshotName, materializedSnapshotName, scanPath, entries, dirMetrics, changedPathCount, nil, rootMetric.TotalSizeBytes, nil
}

type workspaceCounts struct {
	EntryCount      int64
	FileCount       int64
	DirectoryCount  int64
	TotalSizeBytes  int64
	RedundancyCount int64
	RedundancyBytes int64
}

func (s *Service) collectWorkspaceIncremental(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
	baseScanID string,
) (*incrementalCollectResult, error) {
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}

	toolboxPod, err := ceph.FindCephToolboxPod(s.kubeClient, toolboxNamespace)
	if err != nil {
		return nil, fmt.Errorf("find ceph toolbox pod failed: %w", err)
	}

	rootPath, err := ceph.ResolveCephFSPath(
		s.kubeClient,
		s.kubeConfig,
		toolboxNamespace,
		workspace.LogicalPath,
		prefixConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace path failed: %w", err)
	}

	subvolumeRoot, err := ceph.GetCephMountRoot(s.kubeClient, s.kubeConfig, toolboxNamespace)
	if err != nil {
		return nil, fmt.Errorf("resolve ceph subvolume root failed: %w", err)
	}

	currentScanPath, snapshotName, materializedSnapshotName, cleanup, err := s.prepareScanPath(toolboxPod, subvolumeRoot, rootPath, scanID)
	if err != nil {
		return nil, err
	}

	baseJob, err := s.getScanJobByID(ctx, baseScanID)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	if baseJob == nil || strings.TrimSpace(baseJob.SnapshotName) == "" {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("base scan %s has no retained snapshot", baseScanID)
	}

	previousScanPath := strings.TrimSpace(baseJob.ScanRoot)
	if previousScanPath == "" {
		previousScanPath, err = s.resolveExistingSnapshotScanPath(toolboxPod, subvolumeRoot, rootPath, baseJob.SnapshotName)
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			return nil, fmt.Errorf("resolve previous snapshot path failed: %w", err)
		}
	}

	klog.Infof(
		"storageindex: 开始比较快照差异 scan_id=%s workspace_type=%s workspace_name=%s base_scan_id=%s current_scan_root=%s previous_scan_root=%s",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, baseScanID, currentScanPath, previousScanPath,
	)

	previousRecordedChangedAt, err := loadCurrentWorkspaceDirectoryChangedAtMap(ctx, workspace)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("query recorded directory rctime from latest workspace state failed: %w", err)
	}
	plan, changedCount, err := s.buildIncrementalPlan(
		toolboxPod,
		scanID,
		workspace,
		workspace.LogicalPath,
		currentScanPath,
		previousScanPath,
		previousRecordedChangedAt,
	)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	existingCandidates, err := listWorkspaceCandidates(ctx, workspace)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("query workspace candidates before incremental candidate-root diff failed: %w", err)
	}
	plan, changedCount, err = s.augmentIncrementalPlanWithCandidateRootDiffs(
		toolboxPod,
		scanID,
		workspace,
		currentScanPath,
		previousScanPath,
		plan,
		changedCount,
		existingCandidates,
		previousRecordedChangedAt,
	)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	klog.Infof(
		"storageindex: 增量子树差异统计 scan_id=%s workspace_type=%s workspace_name=%s compared_nodes=%d reused_nodes=%d pruned_dirs=%d new_nodes=%d updated_nodes=%d removed_nodes=%d rescan_targets=%d upsert_entries=%d removed_prefixes=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName,
		plan.ComparedNodes, plan.ReusedNodes, plan.PrunedDirs, plan.NewNodes, plan.UpdatedNodes, plan.RemovedNodes,
		len(plan.RescanTargets), len(plan.UpsertEntries), len(plan.RemovedPrefixes),
	)

	if changedCount > 0 {
		plan = expandIncrementalPlanWithCandidateRoots(scanID, workspace, currentScanPath, plan, existingCandidates)
	}

	if changedCount == 0 {
		return &incrementalCollectResult{
			SnapshotName:             snapshotName,
			MaterializedSnapshotName: materializedSnapshotName,
			ScanRoot:                 currentScanPath,
			DiffMethod:               "recursive_snapshot_diff",
			ChangedPathCount:         0,
		}, nil
	}

	newEntries := make([]model.StorageIndexEntry, 0, len(plan.UpsertEntries))
	newMetrics := make([]model.StorageIndexDirectoryMetric, 0)
	changedPrefixes := make([]string, 0, len(plan.RescanTargets))

	for _, entry := range plan.UpsertEntries {
		newEntries = appendUniqueEntry(newEntries, entry)
	}
	if changedCount > 0 {
		rootSignature, rootErr := s.loadDirectorySignatureAtPath(
			toolboxPod,
			scanID,
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			workspace.LogicalPath,
			"",
			currentScanPath,
		)
		if rootErr != nil {
			if cleanup != nil {
				cleanup()
			}
			return nil, fmt.Errorf("load workspace root signature failed: %w", rootErr)
		}
		newEntries = appendUniqueEntry(
			newEntries,
			signatureToEntry(scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, *rootSignature),
		)
	}

	for _, sig := range plan.RescanTargets {
		klog.Infof(
			"storageindex: 开始重扫变化子树 scan_id=%s workspace_type=%s workspace_name=%s subtree=%s entry_type=%s size_bytes=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, sig.LogicalPath, sig.EntryType, sig.SizeBytes,
		)

		entries, scanErr := s.scanUserWorkspaceSubtree(toolboxPod, scanID, workspace, sig)
		if scanErr != nil {
			if cleanup != nil {
				cleanup()
			}
			return nil, fmt.Errorf("scan changed subtree %s failed: %w", sig.LogicalPath, scanErr)
		}
		newEntries = append(newEntries, entries...)
		changedPrefixes = append(changedPrefixes, sig.LogicalPath)

		if sig.EntryType == model.StorageIndexEntryTypeDir {
			subtreeMetrics, _ := buildDirectoryMetrics(
				scanID,
				resolvedWorkspace{
					WorkspaceType: workspace.WorkspaceType,
					WorkspaceName: workspace.WorkspaceName,
					LogicalPath:   sig.LogicalPath,
				},
				entries,
			)
			newMetrics = append(newMetrics, subtreeMetrics...)
		}
	}
	affectedEntryPaths := collectAffectedMetricPaths(workspace.LogicalPath, changedPrefixes, plan.RemovedPrefixes, newEntries)
	for _, entryPath := range affectedEntryPaths {
		if entryPath == workspace.LogicalPath || isCoveredByPrefixes(entryPath, changedPrefixes) {
			continue
		}
		signature, signatureErr := s.loadDirectorySignatureAtPath(
			toolboxPod,
			scanID,
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			entryPath,
			parentForDirectory(entryPath, workspace.LogicalPath),
			logicalPathToActualPath(currentScanPath, workspace.LogicalPath, entryPath),
		)
		if signatureErr != nil {
			if cleanup != nil {
				cleanup()
			}
			return nil, fmt.Errorf("load ancestor directory signature failed for %s: %w", entryPath, signatureErr)
		}
		newEntries = appendUniqueEntry(
			newEntries,
			signatureToEntry(scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, *signature),
		)
	}

	return &incrementalCollectResult{
		SnapshotName:             snapshotName,
		MaterializedSnapshotName: materializedSnapshotName,
		ScanRoot:                 currentScanPath,
		DiffMethod:               "recursive_snapshot_diff",
		ChangedPathCount:         changedCount,
		ChangedPrefixes:          changedPrefixes,
		RemovedPrefixes:          plan.RemovedPrefixes,
		NewEntries:               newEntries,
		NewDirMetrics:            newMetrics,
	}, nil
}

func (s *Service) queryWorkspaceCounts(ctx context.Context, workspace resolvedWorkspace) (workspaceCounts, error) {
	return s.queryWorkspaceCountsWithDB(query.GetDB().WithContext(ctx), workspace)
}

func (s *Service) queryWorkspaceCountsWithDB(db *gorm.DB, workspace resolvedWorkspace) (workspaceCounts, error) {
	counts := workspaceCounts{}

	if err := db.Model(&model.StorageIndexEntry{}).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Count(&counts.EntryCount).Error; err != nil {
		return counts, fmt.Errorf("count workspace entries failed: %w", err)
	}
	if err := db.Model(&model.StorageIndexEntry{}).
		Where("workspace_type = ? AND workspace_name = ? AND entry_type = ? AND logical_path <> ?", workspace.WorkspaceType, workspace.WorkspaceName, model.StorageIndexEntryTypeDir, workspace.LogicalPath).
		Count(&counts.DirectoryCount).Error; err != nil {
		return counts, fmt.Errorf("count workspace directories failed: %w", err)
	}
	if err := db.Model(&model.StorageIndexCandidate{}).
		Where("workspace_type = ? AND workspace_name = ? AND status = ?", workspace.WorkspaceType, workspace.WorkspaceName, model.StorageIndexCandidateStatusVerified).
		Count(&counts.FileCount).Error; err != nil {
		return counts, fmt.Errorf("count candidates failed: %w", err)
	}
	if err := db.Model(&model.StorageIndexDirectoryMetric{}).
		Select("COALESCE(total_size_bytes, 0)").
		Where("workspace_type = ? AND workspace_name = ? AND path = ?", workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath).
		Scan(&counts.TotalSizeBytes).Error; err != nil {
		return counts, fmt.Errorf("query workspace root total bytes failed: %w", err)
	}
	if err := db.Model(&model.StorageIndexRedundancyHit{}).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Count(&counts.RedundancyCount).Error; err != nil {
		return counts, fmt.Errorf("count workspace redundancy hits failed: %w", err)
	}
	if err := db.Model(&model.StorageIndexRedundancyHit{}).
		Select("COALESCE(SUM(estimated_bytes), 0)").
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Scan(&counts.RedundancyBytes).Error; err != nil {
		return counts, fmt.Errorf("sum workspace redundancy bytes failed: %w", err)
	}

	return counts, nil
}

func listWorkspaceCandidates(ctx context.Context, workspace resolvedWorkspace) ([]model.StorageIndexCandidate, error) {
	items := make([]model.StorageIndexCandidate, 0)
	if err := query.GetDB().WithContext(ctx).
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Order("target_path ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("query workspace candidates failed: %w", err)
	}
	return items, nil
}

func loadCurrentWorkspaceDirectoryChangedAtMap(
	ctx context.Context,
	workspace resolvedWorkspace,
) (map[string]*time.Time, error) {
	result := make(map[string]*time.Time)

	type row struct {
		LogicalPath string     `gorm:"column:logical_path"`
		ChangedAt   *time.Time `gorm:"column:changed_at"`
	}
	rows := make([]row, 0)
	if err := query.GetDB().WithContext(ctx).
		Model(&model.StorageIndexEntry{}).
		Select("logical_path, changed_at").
		Where(
			"workspace_type = ? AND workspace_name = ? AND entry_type = ?",
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			model.StorageIndexEntryTypeDir,
		).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[normalizeUnixPath(row.LogicalPath)] = row.ChangedAt
	}
	return result, nil
}

func cleanupWorkspaceStateBeforeInitialScan(db *gorm.DB, workspace resolvedWorkspace) (int64, error) {
	clearedRows := int64(0)
	err := db.Transaction(func(tx *gorm.DB) error {
		deleteScoped := func(value any) error {
			result := tx.
				Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
				Delete(value)
			if result.Error != nil {
				return result.Error
			}
			clearedRows += result.RowsAffected
			return nil
		}

		if err := deleteScoped(&model.StorageIndexEntry{}); err != nil {
			return err
		}
		if err := deleteScoped(&model.StorageIndexDirectoryMetric{}); err != nil {
			return err
		}
		if err := deleteScoped(&model.StorageIndexRedundancyHit{}); err != nil {
			return err
		}
		if err := deleteScoped(&model.StorageIndexCandidate{}); err != nil {
			return err
		}
		if err := deleteScoped(&model.StorageIndexCandidateFile{}); err != nil {
			return err
		}
		if workspace.WorkspaceType != model.StorageIndexWorkspaceTypePublic {
			return nil
		}

		rootResult := tx.Exec("DELETE FROM " + (&model.StorageIndexPublicRootBaseline{}).TableName())
		if rootResult.Error != nil {
			return rootResult.Error
		}
		clearedRows += rootResult.RowsAffected

		fileResult := tx.Exec("DELETE FROM " + (&model.StorageIndexPublicFileBaseline{}).TableName())
		if fileResult.Error != nil {
			return fileResult.Error
		}
		clearedRows += fileResult.RowsAffected
		return nil
	})
	if err != nil {
		return 0, err
	}
	return clearedRows, nil
}

func deleteWorkspacePathPrefix(tx *gorm.DB, workspace resolvedWorkspace, prefix string) error {
	entryWhere := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName)
	if err := entryWhere.Where("logical_path = ? OR logical_path LIKE ?", prefix, prefix+"/%").
		Delete(&model.StorageIndexEntry{}).Error; err != nil {
		return err
	}

	metricWhere := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName)
	if err := metricWhere.Where("path = ? OR path LIKE ?", prefix, prefix+"/%").
		Delete(&model.StorageIndexDirectoryMetric{}).Error; err != nil {
		return err
	}

	hitWhere := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName)
	if err := hitWhere.Where("target_path = ? OR target_path LIKE ?", prefix, prefix+"/%").
		Delete(&model.StorageIndexRedundancyHit{}).Error; err != nil {
		return err
	}

	return nil
}

func deleteWorkspaceExactPath(tx *gorm.DB, workspace resolvedWorkspace, targetPath string) error {
	if err := tx.Where("workspace_type = ? AND workspace_name = ? AND logical_path = ?", workspace.WorkspaceType, workspace.WorkspaceName, targetPath).
		Delete(&model.StorageIndexEntry{}).Error; err != nil {
		return err
	}
	if err := tx.Where("workspace_type = ? AND workspace_name = ? AND target_path = ?", workspace.WorkspaceType, workspace.WorkspaceName, targetPath).
		Delete(&model.StorageIndexRedundancyHit{}).Error; err != nil {
		return err
	}
	return nil
}

func deleteWorkspaceMetricExactPath(tx *gorm.DB, workspace resolvedWorkspace, targetPath string) error {
	return tx.Where("workspace_type = ? AND workspace_name = ? AND path = ?", workspace.WorkspaceType, workspace.WorkspaceName, targetPath).
		Delete(&model.StorageIndexDirectoryMetric{}).Error
}

func deleteWorkspaceRedundancyHitsForExactPaths(
	tx *gorm.DB,
	workspace resolvedWorkspace,
	targetPaths []string,
) error {
	normalized := make([]string, 0, len(targetPaths))
	for _, targetPath := range targetPaths {
		cleaned := normalizeUnixPath(targetPath)
		if cleaned == "" {
			continue
		}
		normalized = append(normalized, cleaned)
	}
	if len(normalized) == 0 {
		return nil
	}
	return tx.
		Where("workspace_type = ? AND workspace_name = ? AND target_path IN ?", workspace.WorkspaceType, workspace.WorkspaceName, normalized).
		Delete(&model.StorageIndexRedundancyHit{}).Error
}

func deleteWorkspaceCandidateStateByPaths(
	tx *gorm.DB,
	workspace resolvedWorkspace,
	candidatePaths []string,
) error {
	for _, candidatePath := range candidatePaths {
		cleaned := normalizeUnixPath(candidatePath)
		if cleaned == "" {
			continue
		}
		if err := tx.
			Where("workspace_type = ? AND workspace_name = ? AND target_path = ?", workspace.WorkspaceType, workspace.WorkspaceName, cleaned).
			Delete(&model.StorageIndexCandidate{}).Error; err != nil {
			return err
		}
		if err := tx.
			Where("workspace_type = ? AND workspace_name = ? AND candidate_path = ?", workspace.WorkspaceType, workspace.WorkspaceName, cleaned).
			Delete(&model.StorageIndexCandidateFile{}).Error; err != nil {
			return err
		}
		if err := tx.
			Where(
				"workspace_type = ? AND workspace_name = ? AND (target_path = ? OR target_path LIKE ?)",
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				cleaned,
				cleaned+"/%",
			).
			Delete(&model.StorageIndexRedundancyHit{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func pruneCandidateDescendants(
	tx *gorm.DB,
	workspace resolvedWorkspace,
	candidates []model.StorageIndexCandidate,
) error {
	pruned := 0
	for _, candidate := range candidates {
		if candidate.TargetPath == "" {
			continue
		}
		likePrefix := candidate.TargetPath + "/%"
		dirResult := tx.
			Where(
				"workspace_type = ? AND workspace_name = ? AND path LIKE ?",
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				likePrefix,
			).
			Delete(&model.StorageIndexDirectoryMetric{})
		if dirResult.Error != nil {
			return dirResult.Error
		}
		pruned += int(dirResult.RowsAffected)
		entryResult := tx.
			Where(
				"workspace_type = ? AND workspace_name = ? AND logical_path LIKE ?",
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				likePrefix,
			).
			Delete(&model.StorageIndexEntry{})
		if entryResult.Error != nil {
			return entryResult.Error
		}
		pruned += int(entryResult.RowsAffected)
	}
	klog.Infof(
		"storageindex: 已按候选目录裁剪子目录骨架 workspace_type=%s workspace_name=%s candidate_count=%d pruned_rows=%d",
		workspace.WorkspaceType, workspace.WorkspaceName, len(candidates), pruned,
	)
	return nil
}

func rebuildWorkspaceRootMetric(tx *gorm.DB, workspace resolvedWorkspace, scanID string) error {
	var fileCount int64
	var directoryCount int64
	var totalSize int64

	if err := tx.Model(&model.StorageIndexEntry{}).
		Where("workspace_type = ? AND workspace_name = ? AND entry_type = ?", workspace.WorkspaceType, workspace.WorkspaceName, model.StorageIndexEntryTypeFile).
		Count(&fileCount).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.StorageIndexEntry{}).
		Where("workspace_type = ? AND workspace_name = ? AND entry_type = ? AND logical_path <> ?", workspace.WorkspaceType, workspace.WorkspaceName, model.StorageIndexEntryTypeDir, workspace.LogicalPath).
		Count(&directoryCount).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.StorageIndexEntry{}).
		Select("COALESCE(SUM(size_bytes), 0)").
		Where("workspace_type = ? AND workspace_name = ? AND entry_type = ?", workspace.WorkspaceType, workspace.WorkspaceName, model.StorageIndexEntryTypeFile).
		Scan(&totalSize).Error; err != nil {
		return err
	}

	if err := tx.Where("workspace_type = ? AND workspace_name = ? AND path = ?", workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath).
		Delete(&model.StorageIndexDirectoryMetric{}).Error; err != nil {
		return err
	}

	rootMetric := &model.StorageIndexDirectoryMetric{
		WorkspaceType:  workspace.WorkspaceType,
		WorkspaceName:  workspace.WorkspaceName,
		ScanID:         scanID,
		Path:           workspace.LogicalPath,
		ParentPath:     "",
		Name:           path.Base(workspace.LogicalPath),
		Depth:          0,
		IsTopLevel:     false,
		FileCount:      fileCount,
		DirectoryCount: directoryCount,
		TotalSizeBytes: totalSize,
		LatestGrowth:   0,
	}
	return tx.Create(rootMetric).Error
}

func collectAffectedMetricPaths(
	workspaceRoot string,
	changedPrefixes []string,
	removedPrefixes []string,
	newEntries []model.StorageIndexEntry,
) []string {
	set := make(map[string]struct{})
	addPathAndAncestors := func(target string, includeSelf bool) {
		current := normalizeUnixPath(target)
		root := normalizeUnixPath(workspaceRoot)
		for current != "" {
			if includeSelf || current != target {
				set[current] = struct{}{}
			}
			if current == root {
				break
			}
			next := path.Dir(current)
			if next == current || next == "." || next == "/" {
				break
			}
			current = next
		}
	}

	for _, prefix := range changedPrefixes {
		addPathAndAncestors(prefix, false)
	}
	for _, prefix := range removedPrefixes {
		addPathAndAncestors(path.Dir(prefix), true)
	}
	for _, entry := range newEntries {
		if entry.EntryType == model.StorageIndexEntryTypeDir {
			addPathAndAncestors(entry.LogicalPath, true)
		} else {
			addPathAndAncestors(entry.ParentPath, true)
		}
	}

	result := make([]string, 0, len(set))
	for item := range set {
		if item != "" {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return depthFromRoot(result[i], workspaceRoot) > depthFromRoot(result[j], workspaceRoot)
	})
	return result
}

func expandIncrementalPlanWithCandidateRoots(
	scanID string,
	workspace resolvedWorkspace,
	currentScanRoot string,
	plan *incrementalPlan,
	existingCandidates []model.StorageIndexCandidate,
) *incrementalPlan {
	if plan == nil || len(existingCandidates) == 0 {
		return plan
	}

	impactedRoots := collectImpactedCandidateRoots(plan, existingCandidates)
	if len(impactedRoots) == 0 {
		return plan
	}

	rescanRoots := make([]string, 0, len(impactedRoots))
	for _, root := range impactedRoots {
		if isCoveredByPrefixes(root, plan.RemovedPrefixes) {
			continue
		}
		rescanRoots = appendUniquePrefix(rescanRoots, root)
	}
	if len(rescanRoots) == 0 {
		return plan
	}

	expandedRoots := collapsePathPrefixes(impactedRoots)
	rescanRoots = collapsePathPrefixes(rescanRoots)

	expanded := &incrementalPlan{
		RescanTargets:   filterRescanTargetsOutsidePrefixes(plan.RescanTargets, expandedRoots),
		UpsertEntries:   filterEntriesOutsidePrefixes(plan.UpsertEntries, expandedRoots),
		RemovedPrefixes: append([]string{}, plan.RemovedPrefixes...),
		ComparedNodes:   plan.ComparedNodes,
		PrunedDirs:      plan.PrunedDirs,
		NewNodes:        plan.NewNodes,
		UpdatedNodes:    plan.UpdatedNodes,
		RemovedNodes:    plan.RemovedNodes,
		ReusedNodes:     plan.ReusedNodes,
	}
	for _, root := range rescanRoots {
		expanded.RescanTargets = appendUniqueSignature(
			expanded.RescanTargets,
			buildCandidateRootRescanSignature(scanID, workspace, currentScanRoot, root),
		)
	}
	return expanded
}

func collectImpactedCandidateRoots(
	plan *incrementalPlan,
	existingCandidates []model.StorageIndexCandidate,
) []string {
	if plan == nil {
		return nil
	}

	relatedPaths := make([]string, 0, len(plan.RescanTargets)+len(plan.UpsertEntries)+len(plan.RemovedPrefixes))
	for _, sig := range plan.RescanTargets {
		relatedPaths = appendUniquePrefix(relatedPaths, sig.LogicalPath)
	}
	for _, entry := range plan.UpsertEntries {
		relatedPaths = appendUniquePrefix(relatedPaths, entry.LogicalPath)
	}
	for _, prefix := range plan.RemovedPrefixes {
		relatedPaths = appendUniquePrefix(relatedPaths, prefix)
	}

	impacted := make([]string, 0)
	for _, candidate := range existingCandidates {
		targetPath := normalizeUnixPath(candidate.TargetPath)
		if targetPath == "" {
			continue
		}
		for _, changedPath := range relatedPaths {
			if pathOverlaps(targetPath, changedPath) {
				impacted = appendUniquePrefix(impacted, targetPath)
				break
			}
		}
	}
	return collapsePathPrefixes(impacted)
}

func collectAffectedCandidatePaths(
	existingCandidates []model.StorageIndexCandidate,
	affectedMetricPaths []string,
	prefixesToDelete []string,
) []string {
	affectedSet := make(map[string]struct{}, len(affectedMetricPaths))
	for _, metricPath := range affectedMetricPaths {
		cleaned := normalizeUnixPath(metricPath)
		if cleaned == "" {
			continue
		}
		affectedSet[cleaned] = struct{}{}
	}

	paths := make([]string, 0)
	for _, candidate := range existingCandidates {
		targetPath := normalizeUnixPath(candidate.TargetPath)
		if targetPath == "" {
			continue
		}
		if _, ok := affectedSet[targetPath]; ok || isCoveredByPrefixes(targetPath, prefixesToDelete) {
			paths = appendUniquePrefix(paths, targetPath)
		}
	}
	return paths
}

func (s *Service) augmentIncrementalPlanWithCandidateRootDiffs(
	toolboxPod *corev1.Pod,
	scanID string,
	workspace resolvedWorkspace,
	currentScanRoot string,
	previousScanRoot string,
	plan *incrementalPlan,
	changedCount int64,
	existingCandidates []model.StorageIndexCandidate,
	previousRecordedChangedAt map[string]*time.Time,
) (*incrementalPlan, int64, error) {
	if plan == nil || len(existingCandidates) == 0 {
		return plan, changedCount, nil
	}

	candidateRoots := make([]string, 0, len(existingCandidates))
	for _, candidate := range existingCandidates {
		if candidate.Status != model.StorageIndexCandidateStatusVerified {
			continue
		}
		targetPath := normalizeUnixPath(candidate.TargetPath)
		if targetPath == "" || strings.TrimSpace(candidate.PublicPath) == "" {
			continue
		}
		candidateRoots = appendUniquePrefix(candidateRoots, targetPath)
	}

	for _, candidateRoot := range collapsePathPrefixes(candidateRoots) {
		if planTouchesPath(plan, candidateRoot) {
			continue
		}

		currentExists, err := s.directoryExistsInSnapshot(toolboxPod, logicalPathToActualPath(currentScanRoot, workspace.LogicalPath, candidateRoot))
		if err != nil {
			return nil, 0, err
		}
		previousExists, err := s.directoryExistsInSnapshot(toolboxPod, logicalPathToActualPath(previousScanRoot, workspace.LogicalPath, candidateRoot))
		if err != nil {
			return nil, 0, err
		}
		if !currentExists && !previousExists {
			continue
		}

		switch {
		case currentExists && !previousExists:
			plan.RescanTargets = appendUniqueSignature(plan.RescanTargets, buildCandidateRootRescanSignature(scanID, workspace, currentScanRoot, candidateRoot))
			plan.NewNodes++
			changedCount++
		case !currentExists && previousExists:
			plan.RemovedPrefixes = appendUniquePrefix(plan.RemovedPrefixes, candidateRoot)
			plan.RemovedNodes++
			changedCount++
		default:
			currentSignature, err := s.loadDirectorySignatureAtPath(
				toolboxPod,
				scanID,
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				candidateRoot,
				parentForDirectory(candidateRoot, workspace.LogicalPath),
				logicalPathToActualPath(currentScanRoot, workspace.LogicalPath, candidateRoot),
			)
			if err != nil {
				return nil, 0, err
			}
			previousSignature, err := s.loadDirectorySignatureAtPath(
				toolboxPod,
				scanID,
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				candidateRoot,
				parentForDirectory(candidateRoot, workspace.LogicalPath),
				logicalPathToActualPath(previousScanRoot, workspace.LogicalPath, candidateRoot),
			)
			if err != nil {
				return nil, 0, err
			}
			recordedChangedAt, ok := previousRecordedChangedAt[normalizeUnixPath(candidateRoot)]
			if !ok || recordedChangedAt == nil {
				return nil, 0, fmt.Errorf("missing recorded directory rctime for %s", candidateRoot)
			}
			previousSignature.ChangedAt = recordedChangedAt
			if !timestampsDifferent(currentSignature.ChangedAt, previousSignature.ChangedAt) {
				continue
			}
			plan.RescanTargets = appendUniqueSignature(plan.RescanTargets, *currentSignature)
			plan.UpdatedNodes++
			changedCount++
		}
	}

	return plan, changedCount, nil
}

func mergeIncrementalPlans(dst *incrementalPlan, src *incrementalPlan) {
	if dst == nil || src == nil {
		return
	}
	for _, target := range src.RescanTargets {
		dst.RescanTargets = appendUniqueSignature(dst.RescanTargets, target)
	}
	for _, entry := range src.UpsertEntries {
		dst.UpsertEntries = appendUniqueEntry(dst.UpsertEntries, entry)
	}
	for _, prefix := range src.RemovedPrefixes {
		dst.RemovedPrefixes = appendUniquePrefix(dst.RemovedPrefixes, prefix)
	}
	dst.ComparedNodes += src.ComparedNodes
	dst.PrunedDirs += src.PrunedDirs
	dst.NewNodes += src.NewNodes
	dst.UpdatedNodes += src.UpdatedNodes
	dst.RemovedNodes += src.RemovedNodes
	dst.ReusedNodes += src.ReusedNodes
}

func planTouchesPath(plan *incrementalPlan, targetPath string) bool {
	if plan == nil {
		return false
	}
	for _, target := range plan.RescanTargets {
		if pathOverlaps(target.LogicalPath, targetPath) {
			return true
		}
	}
	for _, entry := range plan.UpsertEntries {
		if pathOverlaps(entry.LogicalPath, targetPath) {
			return true
		}
	}
	for _, prefix := range plan.RemovedPrefixes {
		if pathOverlaps(prefix, targetPath) {
			return true
		}
	}
	return false
}

func (s *Service) directoryExistsInSnapshot(toolboxPod *corev1.Pod, actualPath string) (bool, error) {
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", fmt.Sprintf("if [ -d %s ]; then echo 1; else echo 0; fi", shellQuote(actualPath))},
	)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "1", nil
}

func filterRescanTargetsOutsidePrefixes(
	targets []topLevelSignature,
	prefixes []string,
) []topLevelSignature {
	filtered := make([]topLevelSignature, 0, len(targets))
	for _, target := range targets {
		if isCoveredByPrefixes(target.LogicalPath, prefixes) {
			continue
		}
		filtered = appendUniqueSignature(filtered, target)
	}
	return filtered
}

func filterEntriesOutsidePrefixes(
	entries []model.StorageIndexEntry,
	prefixes []string,
) []model.StorageIndexEntry {
	filtered := make([]model.StorageIndexEntry, 0, len(entries))
	for _, entry := range entries {
		if isCoveredByPrefixes(entry.LogicalPath, prefixes) {
			continue
		}
		filtered = appendUniqueEntry(filtered, entry)
	}
	return filtered
}

func buildCandidateRootRescanSignature(
	_ string,
	workspace resolvedWorkspace,
	currentScanRoot string,
	candidatePath string,
) topLevelSignature {
	cleanedPath := normalizeUnixPath(candidatePath)
	parentPath := path.Dir(cleanedPath)
	if parentPath == "." || parentPath == "/" {
		parentPath = workspace.LogicalPath
	}
	return topLevelSignature{
		Name:              path.Base(cleanedPath),
		LogicalPath:       cleanedPath,
		ParentLogicalPath: parentPath,
		ActualPath:        logicalPathToActualPath(currentScanRoot, workspace.LogicalPath, cleanedPath),
		EntryType:         model.StorageIndexEntryTypeDir,
	}
}

func logicalPathToActualPath(scanRoot, workspaceRoot, logicalPath string) string {
	cleanedLogical := normalizeUnixPath(logicalPath)
	cleanedRoot := normalizeUnixPath(workspaceRoot)
	relative := strings.TrimPrefix(cleanedLogical, cleanedRoot)
	relative = strings.TrimPrefix(relative, "/")
	if relative == "" {
		return normalizeUnixPath(scanRoot)
	}
	return normalizeUnixPath(path.Join(scanRoot, relative))
}

func isCoveredByPrefixes(target string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return true
		}
	}
	return false
}

func rebuildDirectoryMetric(tx *gorm.DB, workspace resolvedWorkspace, scanID string, metricPath string) error {
	metricPath = normalizeUnixPath(metricPath)
	rootPath := normalizeUnixPath(workspace.LogicalPath)

	if metricPath != rootPath {
		var exists int64
		if err := tx.Model(&model.StorageIndexEntry{}).
			Where(
				"workspace_type = ? AND workspace_name = ? AND logical_path = ? AND entry_type = ?",
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				metricPath,
				model.StorageIndexEntryTypeDir,
			).
			Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return nil
		}
	}

	likePrefix := metricPath + "/%"
	var fileCount int64
	var directoryCount int64
	var totalSize int64

	if err := tx.Model(&model.StorageIndexEntry{}).
		Where(
			"workspace_type = ? AND workspace_name = ? AND entry_type = ? AND (logical_path = ? OR logical_path LIKE ?)",
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			model.StorageIndexEntryTypeFile,
			metricPath,
			likePrefix,
		).
		Count(&fileCount).Error; err != nil {
		return err
	}

	if err := tx.Model(&model.StorageIndexEntry{}).
		Where(
			"workspace_type = ? AND workspace_name = ? AND entry_type = ? AND logical_path LIKE ?",
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			model.StorageIndexEntryTypeDir,
			likePrefix,
		).
		Count(&directoryCount).Error; err != nil {
		return err
	}

	if err := tx.Model(&model.StorageIndexEntry{}).
		Select("COALESCE(SUM(size_bytes), 0)").
		Where(
			"workspace_type = ? AND workspace_name = ? AND entry_type = ? AND (logical_path = ? OR logical_path LIKE ?)",
			workspace.WorkspaceType,
			workspace.WorkspaceName,
			model.StorageIndexEntryTypeFile,
			metricPath,
			likePrefix,
		).
		Scan(&totalSize).Error; err != nil {
		return err
	}

	metric := &model.StorageIndexDirectoryMetric{
		WorkspaceType:  workspace.WorkspaceType,
		WorkspaceName:  workspace.WorkspaceName,
		ScanID:         scanID,
		Path:           metricPath,
		ParentPath:     parentForDirectory(metricPath, rootPath),
		Name:           path.Base(metricPath),
		Depth:          depthFromRoot(metricPath, rootPath),
		IsTopLevel:     isTopLevelPath(metricPath, rootPath),
		FileCount:      fileCount,
		DirectoryCount: directoryCount,
		TotalSizeBytes: totalSize,
		LatestGrowth:   0,
	}
	return tx.Create(metric).Error
}

func (s *Service) prepareScanPath(
	toolboxPod *corev1.Pod,
	subvolumeRoot string,
	rootPath string,
	scanID string,
) (string, string, string, func(), error) {
	relativeSuffix := relativeUnixPath(subvolumeRoot, rootPath)
	if csi, ok := parseCephCSIWorkspacePath(subvolumeRoot, relativeSuffix); ok {
		return s.prepareSubvolumeSnapshotScanPath(toolboxPod, csi, scanID)
	}

	snapshotName := "index-" + time.Now().Format("20060102150405") + "-" + strings.ReplaceAll(scanID[:8], "-", "")
	snapshotPath := path.Join(rootPath, ".snap", snapshotName)

	if _, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"mkdir", snapshotPath},
	); err != nil {
		klog.Warningf("storageindex: 创建目录快照失败，回退为 live scan path=%s err=%v", rootPath, err)
		return rootPath, "", "", nil, nil
	}

	cleanup := func() {
		if _, err := ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			[]string{"rmdir", snapshotPath},
		); err != nil {
			klog.Warningf("storageindex: 清理目录快照失败 snapshot_path=%s err=%v", snapshotPath, err)
		}
	}

	return snapshotPath, snapshotName, snapshotName, cleanup, nil
}

func (s *Service) prepareSubvolumeSnapshotScanPath(
	toolboxPod *corev1.Pod,
	csi cephCSIWorkspacePath,
	scanID string,
) (string, string, string, func(), error) {
	snapshotName := "index-" + time.Now().Format("20060102150405") + "-" + strings.ReplaceAll(scanID[:8], "-", "")

	if _, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"ceph", "fs", "subvolume", "snapshot", "create", cephFSVolumeName, csi.SubvolumeName, snapshotName, "--group_name", csi.GroupName},
	); err != nil {
		klog.Warningf(
			"storageindex: 创建 subvolume 快照失败，回退为 live scan group=%s subvolume=%s root=%s err=%v",
			csi.GroupName, csi.SubvolumeName, csi.WorkspaceRoot, err,
		)
		return csi.WorkspaceRoot, "", "", nil, nil
	}

	out, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"ceph", "fs", "subvolume", "getpath", cephFSVolumeName, csi.SubvolumeName, "--group_name", csi.GroupName},
	)
	if err != nil {
		_, _ = ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			[]string{"ceph", "fs", "subvolume", "snapshot", "rm", cephFSVolumeName, csi.SubvolumeName, snapshotName, "--group_name", csi.GroupName, "--force"},
		)
		klog.Warningf(
			"storageindex: 获取 subvolume 路径失败，回退为 live scan group=%s subvolume=%s snapshot=%s err=%v",
			csi.GroupName, csi.SubvolumeName, snapshotName, err,
		)
		return csi.WorkspaceRoot, "", "", nil, nil
	}

	subvolumeRoot := strings.TrimSpace(out)
	if subvolumeRoot == "" {
		_, _ = ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			[]string{"ceph", "fs", "subvolume", "snapshot", "rm", cephFSVolumeName, csi.SubvolumeName, snapshotName, "--group_name", csi.GroupName, "--force"},
		)
		klog.Warningf(
			"storageindex: subvolume 路径为空，回退为 live scan group=%s subvolume=%s snapshot=%s",
			csi.GroupName, csi.SubvolumeName, snapshotName,
		)
		return csi.WorkspaceRoot, "", "", nil, nil
	}

	materializedSnapshotName, err := s.resolveMaterializedSnapshotName(toolboxPod, csi.MountRoot, subvolumeRoot, snapshotName)
	if err != nil {
		_, _ = ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			[]string{"ceph", "fs", "subvolume", "snapshot", "rm", cephFSVolumeName, csi.SubvolumeName, snapshotName, "--group_name", csi.GroupName, "--force"},
		)
		klog.Warningf(
			"storageindex: 解析物化快照目录名失败，回退为 live scan group=%s subvolume=%s snapshot=%s err=%v",
			csi.GroupName, csi.SubvolumeName, snapshotName, err,
		)
		return csi.WorkspaceRoot, "", "", nil, nil
	}

	scanRoot := normalizeUnixPath(path.Join(csi.MountRoot, subvolumeRoot, ".snap", materializedSnapshotName, csi.RelativeSuffix))
	cleanup := func() {
		if _, err := ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			[]string{"ceph", "fs", "subvolume", "snapshot", "rm", cephFSVolumeName, csi.SubvolumeName, snapshotName, "--group_name", csi.GroupName, "--force"},
		); err != nil {
			klog.Warningf(
				"storageindex: 清理 subvolume 快照失败 group=%s subvolume=%s snapshot=%s err=%v",
				csi.GroupName, csi.SubvolumeName, snapshotName, err,
			)
		}
	}

	klog.Infof(
		"storageindex: 使用 subvolume 快照扫描 group=%s subvolume=%s snapshot=%s materialized_snapshot=%s subvolume_root=%s scan_root=%s",
		csi.GroupName, csi.SubvolumeName, snapshotName, materializedSnapshotName, subvolumeRoot, scanRoot,
	)

	return scanRoot, snapshotName, materializedSnapshotName, cleanup, nil
}

func (s *Service) resolveWorkspace(
	ctx context.Context,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
) (resolvedWorkspace, error) {
	name := strings.TrimSpace(workspaceName)
	switch workspaceType {
	case model.StorageIndexWorkspaceTypeUser:
		if name == "" {
			return resolvedWorkspace{}, fmt.Errorf("workspace_name is required for user workspace")
		}
		var user struct {
			Name  string `gorm:"column:name"`
			Space string `gorm:"column:space"`
		}
		if err := query.GetDB().WithContext(ctx).
			Raw("SELECT name, space FROM users WHERE name = ? AND deleted_at IS NULL", name).
			Scan(&user).Error; err != nil {
			return resolvedWorkspace{}, fmt.Errorf("query user workspace failed: %w", err)
		}
		if user.Name == "" {
			return resolvedWorkspace{}, fmt.Errorf("user workspace %s not found", name)
		}
		return resolvedWorkspace{
			WorkspaceType: workspaceType,
			WorkspaceName: user.Name,
			LogicalPath:   normalizeUserWorkspacePath(user.Space),
		}, nil
	case model.StorageIndexWorkspaceTypeAccount:
		if name == "" {
			return resolvedWorkspace{}, fmt.Errorf("workspace_name is required for account workspace")
		}
		var account struct {
			Name  string `gorm:"column:name"`
			Space string `gorm:"column:space"`
		}
		if err := query.GetDB().WithContext(ctx).
			Raw("SELECT name, space FROM accounts WHERE name = ? AND deleted_at IS NULL", name).
			Scan(&account).Error; err != nil {
			return resolvedWorkspace{}, fmt.Errorf("query account workspace failed: %w", err)
		}
		if account.Name == "" {
			return resolvedWorkspace{}, fmt.Errorf("account workspace %s not found", name)
		}
		return resolvedWorkspace{
			WorkspaceType: workspaceType,
			WorkspaceName: account.Name,
			LogicalPath:   normalizeAccountWorkspacePath(account.Space),
		}, nil
	case model.StorageIndexWorkspaceTypePublic:
		if name == "" {
			name = "public"
		}
		return resolvedWorkspace{
			WorkspaceType: workspaceType,
			WorkspaceName: name,
			LogicalPath:   "/public",
		}, nil
	default:
		return resolvedWorkspace{}, fmt.Errorf("unsupported workspace type: %s", workspaceType)
	}
}

func normalizeUserWorkspacePath(space string) string {
	cleaned := strings.TrimSpace(space)
	if cleaned == "" {
		return ""
	}
	if strings.HasPrefix(cleaned, "/user/") {
		return path.Clean(cleaned)
	}
	return path.Clean("/user/" + strings.TrimPrefix(cleaned, "/"))
}

func normalizeAccountWorkspacePath(space string) string {
	cleaned := strings.TrimSpace(space)
	if cleaned == "" {
		return ""
	}
	if strings.HasPrefix(cleaned, "/") {
		return path.Clean(cleaned)
	}
	return path.Clean("/account/" + strings.TrimPrefix(cleaned, "/"))
}

type cephCSIWorkspacePath struct {
	MountRoot      string
	GroupName      string
	SubvolumeName  string
	RelativeSuffix string
	WorkspaceRoot  string
}

func parseCephCSIWorkspacePath(subvolumeRoot string, relativeSuffix string) (cephCSIWorkspacePath, bool) {
	normalized := normalizeUnixPath(subvolumeRoot)
	marker := "/volumes/"
	idx := strings.Index(normalized, marker)
	if idx < 0 {
		return cephCSIWorkspacePath{}, false
	}

	mountRoot := normalized[:idx]
	suffix := strings.TrimPrefix(normalized[idx+len(marker):], "/")
	parts := strings.Split(suffix, "/")
	if len(parts) < 2 {
		return cephCSIWorkspacePath{}, false
	}

	return cephCSIWorkspacePath{
		MountRoot:      mountRoot,
		GroupName:      parts[0],
		SubvolumeName:  parts[1],
		RelativeSuffix: relativeSuffix,
		WorkspaceRoot:  path.Join(normalized, relativeSuffix),
	}, true
}

func (s *Service) resolveMaterializedSnapshotName(
	toolboxPod *corev1.Pod,
	mountRoot string,
	subvolumeRoot string,
	requestedSnapshotName string,
) (string, error) {
	snapRoot := normalizeUnixPath(path.Join(mountRoot, subvolumeRoot, ".snap"))
	out, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"ls", "-1", snapRoot},
	)
	if err != nil {
		return "", fmt.Errorf("list snapshot directory failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || (len(lines) == 1 && strings.TrimSpace(lines[0]) == "") {
		return "", fmt.Errorf("snapshot directory %s is empty", snapRoot)
	}

	exact := ""
	prefixed := ""
	contains := ""
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" || name == "." || name == ".." {
			continue
		}
		if name == requestedSnapshotName {
			exact = name
			break
		}
		if strings.HasPrefix(name, "_"+requestedSnapshotName+"_") {
			prefixed = name
		}
		if contains == "" && strings.Contains(name, requestedSnapshotName) {
			contains = name
		}
	}

	switch {
	case exact != "":
		return exact, nil
	case prefixed != "":
		return prefixed, nil
	case contains != "":
		return contains, nil
	default:
		return "", fmt.Errorf("requested snapshot %s not materialized under %s (entries=%v)", requestedSnapshotName, snapRoot, lines)
	}
}

func (s *Service) findLatestCompletedScanID(ctx context.Context, workspace resolvedWorkspace) (string, error) {
	var job model.StorageIndexScanJob
	if err := query.GetDB().WithContext(ctx).
		Where("workspace_type = ? AND workspace_name = ? AND status = ?", workspace.WorkspaceType, workspace.WorkspaceName, model.StorageIndexScanStatusDone).
		Order("finished_at DESC, updated_at DESC").
		First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("query latest completed metadata scan failed: %w", err)
	}
	return job.ScanID, nil
}

func (s *Service) applyGrowthFromPreviousScan(
	ctx context.Context,
	workspace resolvedWorkspace,
	baseScanID string,
	dirMetrics []model.StorageIndexDirectoryMetric,
) (int64, error) {
	if baseScanID == "" || len(dirMetrics) == 0 {
		return 0, nil
	}

	var previousMetrics []model.StorageIndexDirectoryMetric
	if err := query.GetDB().WithContext(ctx).
		Where("workspace_type = ? AND workspace_name = ? AND scan_id = ?", workspace.WorkspaceType, workspace.WorkspaceName, baseScanID).
		Find(&previousMetrics).Error; err != nil {
		return 0, fmt.Errorf("query previous directory metrics failed: %w", err)
	}

	previousByPath := make(map[string]model.StorageIndexDirectoryMetric, len(previousMetrics))
	for _, metric := range previousMetrics {
		previousByPath[metric.Path] = metric
	}

	changedCount := int64(0)
	for i := range dirMetrics {
		current := &dirMetrics[i]
		previous, ok := previousByPath[current.Path]
		if !ok {
			current.LatestGrowth = current.TotalSizeBytes
			if current.TotalSizeBytes > 0 {
				changedCount++
			}
			continue
		}

		current.LatestGrowth = current.TotalSizeBytes - previous.TotalSizeBytes
		if current.LatestGrowth != 0 {
			changedCount++
		}
	}

	return changedCount, nil
}

func (s *Service) getScanJobByID(ctx context.Context, scanID string) (*model.StorageIndexScanJob, error) {
	var job model.StorageIndexScanJob
	if err := query.GetDB().WithContext(ctx).Where("scan_id = ?", scanID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query metadata scan job %s failed: %w", scanID, err)
	}
	return &job, nil
}

func (s *Service) resolveExistingSnapshotScanPath(
	toolboxPod *corev1.Pod,
	subvolumeRoot string,
	rootPath string,
	snapshotName string,
) (string, error) {
	if snapshotName == "" {
		return "", fmt.Errorf("snapshot name is empty")
	}

	relativeSuffix := relativeUnixPath(subvolumeRoot, rootPath)
	if csi, ok := parseCephCSIWorkspacePath(subvolumeRoot, relativeSuffix); ok {
		out, err := ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			[]string{"ceph", "fs", "subvolume", "getpath", cephFSVolumeName, csi.SubvolumeName, "--group_name", csi.GroupName},
		)
		if err != nil {
			return "", fmt.Errorf("getpath for existing subvolume snapshot failed: %w", err)
		}

		subvolumePath := strings.TrimSpace(out)
		if subvolumePath == "" {
			return "", fmt.Errorf("subvolume getpath returned empty path")
		}

		materializedSnapshotName, err := s.resolveMaterializedSnapshotName(toolboxPod, csi.MountRoot, subvolumePath, snapshotName)
		if err != nil {
			return "", err
		}
		return normalizeUnixPath(path.Join(csi.MountRoot, subvolumePath, ".snap", materializedSnapshotName, csi.RelativeSuffix)), nil
	}

	return normalizeUnixPath(path.Join(rootPath, ".snap", snapshotName)), nil
}

func (s *Service) listImmediateSignatures(
	toolboxPod *corev1.Pod,
	scanID string,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	logicalBase string,
	scanRoot string,
) (map[string]topLevelSignature, error) {
	script := fmt.Sprintf(
		"find %s -mindepth 1 -maxdepth 1 -printf %s",
		shellQuote(scanRoot),
		shellQuote(`%y\037%s\037%T@\037%C@\037%U\037%G\037%m\037%n\037%p\0`),
	)

	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return nil, fmt.Errorf("list top-level signatures failed: %w", err)
	}

	signatures := make(map[string]topLevelSignature)
	records := strings.Split(output, findRecordSeparator)
	for _, record := range records {
		if record == "" {
			continue
		}
		fields := strings.Split(record, findFieldSeparator)
		if len(fields) != 9 {
			continue
		}

		actualPath := normalizeUnixPath(fields[8])
		relativePath := relativeUnixPath(scanRoot, actualPath)
		if relativePath == "." || relativePath == "" || strings.Contains(relativePath, "/") {
			continue
		}

		entryType := parseEntryType(fields[0])
		sizeBytes := parseInt64(fields[1])
		changedAt := parseUnixTimestamp(fields[3])
		if entryType == model.StorageIndexEntryTypeDir {
			sizeBytes, err = s.getDirectoryRBytes(toolboxPod, actualPath)
			if err != nil {
				klog.Warningf("storageindex: 获取目录 ceph.dir.rbytes 失败 scan_id=%s path=%s err=%v", scanID, actualPath, err)
			}
			recursiveChangedAt, rctimeErr := s.getDirectoryRecursiveChangedAt(toolboxPod, actualPath)
			if rctimeErr != nil {
				klog.Warningf("storageindex: 获取目录 ceph.dir.rctime 失败，回退为 stat ctime scan_id=%s path=%s err=%v", scanID, actualPath, rctimeErr)
			} else if recursiveChangedAt != nil {
				changedAt = recursiveChangedAt
			}
		}

		signatures[relativePath] = topLevelSignature{
			Name:              relativePath,
			LogicalPath:       path.Join(logicalBase, relativePath),
			ParentLogicalPath: logicalBase,
			ActualPath:        actualPath,
			EntryType:         entryType,
			SizeBytes:         sizeBytes,
			ModifiedAt:        parseUnixTimestamp(fields[2]),
			ChangedAt:         changedAt,
			OwnerUID:          parseInt64(fields[4]),
			OwnerGID:          parseInt64(fields[5]),
			Mode:              strings.TrimSpace(fields[6]),
			LinkCount:         parseInt64(fields[7]),
		}
	}

	klog.Infof(
		"storageindex: 直接子项签名采集完成 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s signature_count=%d",
		scanID, workspaceType, workspaceName, logicalBase, len(signatures),
	)
	return signatures, nil
}

func (s *Service) getDirectoryRBytes(toolboxPod *corev1.Pod, actualPath string) (int64, error) {
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"getfattr", "-n", "ceph.dir.rbytes", actualPath},
	)
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ceph.dir.rbytes=") {
			sizeStr := strings.Trim(strings.TrimPrefix(line, "ceph.dir.rbytes="), "\"")
			size, parseErr := strconv.ParseInt(sizeStr, 10, 64)
			if parseErr != nil {
				return 0, fmt.Errorf("parse ceph.dir.rbytes failed: %w", parseErr)
			}
			return size, nil
		}
	}

	return 0, fmt.Errorf("ceph.dir.rbytes not found for %s", actualPath)
}

func (s *Service) getDirectoryRecursiveChangedAt(toolboxPod *corev1.Pod, actualPath string) (*time.Time, error) {
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"getfattr", "-n", "ceph.dir.rctime", actualPath},
	)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ceph.dir.rctime=") {
			value := strings.Trim(strings.TrimPrefix(line, "ceph.dir.rctime="), "\"")
			parsed := parseUnixTimestamp(value)
			if parsed == nil {
				return nil, fmt.Errorf("parse ceph.dir.rctime failed for %s: %q", actualPath, value)
			}
			return parsed, nil
		}
	}

	return nil, fmt.Errorf("ceph.dir.rctime not found for %s", actualPath)
}

func diffTopLevelSignatures(
	current map[string]topLevelSignature,
	previous map[string]topLevelSignature,
) ([]topLevelSignature, []string, int64) {
	changedCurrent := make([]topLevelSignature, 0)
	removedPrefixes := make([]string, 0)
	changedCount := int64(0)

	keys := make(map[string]struct{}, len(current)+len(previous))
	for key := range current {
		keys[key] = struct{}{}
	}
	for key := range previous {
		keys[key] = struct{}{}
	}

	allKeys := make([]string, 0, len(keys))
	for key := range keys {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)

	for _, key := range allKeys {
		cur, curOK := current[key]
		prev, prevOK := previous[key]
		switch {
		case curOK && !prevOK:
			changedCurrent = append(changedCurrent, cur)
			changedCount++
		case !curOK && prevOK:
			removedPrefixes = append(removedPrefixes, prev.LogicalPath)
			changedCount++
		case curOK && prevOK:
			if cur.EntryType != prev.EntryType {
				changedCurrent = append(changedCurrent, cur)
				changedCount++
				continue
			}
			if cur.EntryType == model.StorageIndexEntryTypeDir {
				if timestampsDifferent(cur.ChangedAt, prev.ChangedAt) {
					changedCurrent = append(changedCurrent, cur)
					changedCount++
				}
				continue
			}
			if cur.SizeBytes != prev.SizeBytes ||
				timestampsDifferent(cur.ModifiedAt, prev.ModifiedAt) ||
				timestampsDifferent(cur.ChangedAt, prev.ChangedAt) ||
				cur.Mode != prev.Mode ||
				cur.LinkCount != prev.LinkCount {
				changedCurrent = append(changedCurrent, cur)
				changedCount++
			}
		}
	}

	return changedCurrent, removedPrefixes, changedCount
}

func signatureToEntry(
	scanID string,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	workspaceRoot string,
	sig topLevelSignature,
) model.StorageIndexEntry {
	relativePath := relativeLogicalPath(workspaceRoot, sig.LogicalPath)
	isTopLevel := path.Clean(sig.ParentLogicalPath) == path.Clean(workspaceRoot)
	return model.StorageIndexEntry{
		WorkspaceType: workspaceType,
		WorkspaceName: workspaceName,
		ScanID:        scanID,
		LogicalPath:   sig.LogicalPath,
		RelativePath:  relativePath,
		ParentPath:    sig.ParentLogicalPath,
		Name:          sig.Name,
		EntryType:     sig.EntryType,
		SizeBytes:     sig.SizeBytes,
		OwnerUID:      sig.OwnerUID,
		OwnerGID:      sig.OwnerGID,
		Mode:          sig.Mode,
		LinkCount:     sig.LinkCount,
		ModifiedAt:    sig.ModifiedAt,
		ChangedAt:     sig.ChangedAt,
		IsTopLevel:    isTopLevel,
	}
}

func appendUniquePrefix(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func appendUniqueEntry(items []model.StorageIndexEntry, entry model.StorageIndexEntry) []model.StorageIndexEntry {
	for i, item := range items {
		if item.LogicalPath == entry.LogicalPath {
			items[i] = entry
			return items
		}
	}
	return append(items, entry)
}

func appendUniqueSignature(items []topLevelSignature, sig topLevelSignature) []topLevelSignature {
	for i, item := range items {
		if item.LogicalPath == sig.LogicalPath {
			items[i] = sig
			return items
		}
	}
	return append(items, sig)
}

func (s *Service) buildIncrementalPlan(
	toolboxPod *corev1.Pod,
	scanID string,
	workspace resolvedWorkspace,
	currentLogicalBase string,
	currentActualDir string,
	previousActualDir string,
	previousRecordedChangedAt map[string]*time.Time,
) (*incrementalPlan, int64, error) {
	currentSigns, err := s.listImmediateSignatures(toolboxPod, scanID, workspace.WorkspaceType, workspace.WorkspaceName, currentLogicalBase, currentActualDir)
	if err != nil {
		return nil, 0, err
	}
	previousSigns, err := s.listImmediateSignatures(toolboxPod, scanID, workspace.WorkspaceType, workspace.WorkspaceName, currentLogicalBase, previousActualDir)
	if err != nil {
		return nil, 0, err
	}
	if shouldApplyTopLevelModelCopyPrefilter(workspace) && normalizeUnixPath(currentLogicalBase) == normalizeUnixPath(workspace.LogicalPath) {
		filteredCurrent, skippedCurrent := filterTopLevelSignaturesForModelCopyScan(currentSigns)
		filteredPrevious, skippedPrevious := filterTopLevelSignaturesForModelCopyScan(previousSigns)
		currentSigns = make(map[string]topLevelSignature, len(filteredCurrent))
		for _, sig := range filteredCurrent {
			currentSigns[path.Base(sig.LogicalPath)] = sig
		}
		previousSigns = make(map[string]topLevelSignature, len(filteredPrevious))
		for _, sig := range filteredPrevious {
			previousSigns[path.Base(sig.LogicalPath)] = sig
		}

		skippedNames := make([]string, 0, len(skippedCurrent)+len(skippedPrevious))
		seenSkipped := make(map[string]struct{}, len(skippedCurrent)+len(skippedPrevious))
		for _, sig := range skippedCurrent {
			if _, ok := seenSkipped[sig.Name]; ok {
				continue
			}
			seenSkipped[sig.Name] = struct{}{}
			skippedNames = append(skippedNames, sig.Name)
		}
		for _, sig := range skippedPrevious {
			if _, ok := seenSkipped[sig.Name]; ok {
				continue
			}
			seenSkipped[sig.Name] = struct{}{}
			skippedNames = append(skippedNames, sig.Name)
		}
		sort.Strings(skippedNames)
		if len(skippedNames) > 0 {
			klog.Infof(
				"storageindex: 增量对比顶层目录过滤完成 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s skipped_top_level=%v",
				scanID, workspace.WorkspaceType, workspace.WorkspaceName, currentLogicalBase, skippedNames,
			)
		}
	}
	plan := &incrementalPlan{
		RescanTargets:   make([]topLevelSignature, 0),
		UpsertEntries:   make([]model.StorageIndexEntry, 0),
		RemovedPrefixes: make([]string, 0),
	}

	changedCount := int64(0)
	keys := make(map[string]struct{}, len(currentSigns)+len(previousSigns))
	for key := range currentSigns {
		keys[key] = struct{}{}
	}
	for key := range previousSigns {
		keys[key] = struct{}{}
	}

	allKeys := make([]string, 0, len(keys))
	for key := range keys {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)
	plan.ComparedNodes += int64(len(allKeys))

	for _, key := range allKeys {
		cur, curOK := currentSigns[key]
		prev, prevOK := previousSigns[key]
		switch {
		case curOK && !prevOK:
			changedCount++
			plan.NewNodes++
			if cur.EntryType == model.StorageIndexEntryTypeDir {
				plan.RescanTargets = appendUniqueSignature(plan.RescanTargets, cur)
			} else {
				plan.UpsertEntries = appendUniqueEntry(plan.UpsertEntries, signatureToEntry(scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, cur))
			}
		case !curOK && prevOK:
			changedCount++
			plan.RemovedNodes++
			plan.RemovedPrefixes = appendUniquePrefix(plan.RemovedPrefixes, prev.LogicalPath)
		case curOK && prevOK:
			if cur.EntryType != prev.EntryType {
				changedCount++
				plan.UpdatedNodes++
				plan.RemovedPrefixes = appendUniquePrefix(plan.RemovedPrefixes, prev.LogicalPath)
				if cur.EntryType == model.StorageIndexEntryTypeDir {
					plan.RescanTargets = appendUniqueSignature(plan.RescanTargets, cur)
				} else {
					plan.UpsertEntries = appendUniqueEntry(plan.UpsertEntries, signatureToEntry(scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, cur))
				}
				continue
			}

			if cur.EntryType != model.StorageIndexEntryTypeDir {
				if cur.SizeBytes != prev.SizeBytes || timestampsDifferent(cur.ModifiedAt, prev.ModifiedAt) || timestampsDifferent(cur.ChangedAt, prev.ChangedAt) || cur.Mode != prev.Mode || cur.LinkCount != prev.LinkCount {
					changedCount++
					plan.UpdatedNodes++
					plan.UpsertEntries = appendUniqueEntry(plan.UpsertEntries, signatureToEntry(scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, cur))
				} else {
					plan.ReusedNodes++
				}
				continue
			}

			recorded, ok := previousRecordedChangedAt[normalizeUnixPath(prev.LogicalPath)]
			if !ok || recorded == nil {
				return nil, 0, fmt.Errorf("missing recorded directory rctime for %s", prev.LogicalPath)
			}
			prev.ChangedAt = recorded

			dirChanged := timestampsDifferent(cur.ChangedAt, prev.ChangedAt)
			klog.Infof(
				"storageindex: 目录签名比较 scan_id=%s workspace_type=%s workspace_name=%s path=%s current_rbytes=%d previous_rbytes=%d current_rctime=%s previous_rctime=%s previous_rctime_source=db recurse=%t",
				scanID,
				workspace.WorkspaceType,
				workspace.WorkspaceName,
				cur.LogicalPath,
				cur.SizeBytes,
				prev.SizeBytes,
				formatTimeForLog(cur.ChangedAt),
				formatTimeForLog(prev.ChangedAt),
				dirChanged,
			)

			if !dirChanged {
				plan.ReusedNodes++
				plan.PrunedDirs++
				continue
			}

			childPlan, childChanged, childErr := s.buildIncrementalPlan(
				toolboxPod,
				scanID,
				workspace,
				cur.LogicalPath,
				cur.ActualPath,
				prev.ActualPath,
				previousRecordedChangedAt,
			)
			if childErr != nil {
				return nil, 0, childErr
			}

			changedCount += childChanged
			plan.ComparedNodes += childPlan.ComparedNodes
			plan.PrunedDirs += childPlan.PrunedDirs
			plan.NewNodes += childPlan.NewNodes
			plan.UpdatedNodes += childPlan.UpdatedNodes
			plan.RemovedNodes += childPlan.RemovedNodes
			plan.ReusedNodes += childPlan.ReusedNodes
			if childChanged == 0 {
				plan.ReusedNodes++
				continue
			}
			plan.UpdatedNodes++
			plan.UpsertEntries = appendUniqueEntry(plan.UpsertEntries, signatureToEntry(scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, cur))
			for _, target := range childPlan.RescanTargets {
				plan.RescanTargets = appendUniqueSignature(plan.RescanTargets, target)
			}
			for _, entry := range childPlan.UpsertEntries {
				plan.UpsertEntries = appendUniqueEntry(plan.UpsertEntries, entry)
			}
			for _, removed := range childPlan.RemovedPrefixes {
				plan.RemovedPrefixes = appendUniquePrefix(plan.RemovedPrefixes, removed)
			}
		}
	}

	return plan, changedCount, nil
}

func timestampsDifferent(a, b *time.Time) bool {
	switch {
	case a == nil && b == nil:
		return false
	case a == nil || b == nil:
		return true
	default:
		return !a.Equal(*b)
	}
}

func (s *Service) scanWorkspaceEntries(
	toolboxPod *corev1.Pod,
	scanID string,
	workspace resolvedWorkspace,
	scanRoot string,
) ([]model.StorageIndexEntry, error) {
	return s.scanPathDirectories(toolboxPod, scanID, workspace.WorkspaceType, workspace.WorkspaceName, workspace.LogicalPath, scanRoot)
}

func (s *Service) loadDirectorySignatureAtPath(
	toolboxPod *corev1.Pod,
	_ string,
	_ model.StorageIndexWorkspaceType,
	_ string,
	logicalPath string,
	parentLogicalPath string,
	actualPath string,
) (*topLevelSignature, error) {
	sizeBytes, err := s.getDirectoryRBytes(toolboxPod, actualPath)
	if err != nil {
		return nil, err
	}
	changedAt, err := s.getDirectoryRecursiveChangedAt(toolboxPod, actualPath)
	if err != nil {
		return nil, err
	}

	script := fmt.Sprintf(
		"find %s -mindepth 0 -maxdepth 0 -printf %s",
		shellQuote(actualPath),
		shellQuote(`%T@\037%U\037%G\037%m\037%n\037%p\0`),
	)
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return nil, fmt.Errorf("load directory signature at path failed: %w", err)
	}

	records := strings.Split(output, findRecordSeparator)
	for _, record := range records {
		if record == "" {
			continue
		}
		fields := strings.Split(record, findFieldSeparator)
		if len(fields) != 6 {
			continue
		}
		return &topLevelSignature{
			Name:              path.Base(logicalPath),
			LogicalPath:       normalizeUnixPath(logicalPath),
			ParentLogicalPath: normalizeUnixPath(parentLogicalPath),
			ActualPath:        normalizeUnixPath(actualPath),
			EntryType:         model.StorageIndexEntryTypeDir,
			SizeBytes:         sizeBytes,
			ModifiedAt:        parseUnixTimestamp(fields[0]),
			ChangedAt:         changedAt,
			OwnerUID:          parseInt64(fields[1]),
			OwnerGID:          parseInt64(fields[2]),
			Mode:              strings.TrimSpace(fields[3]),
			LinkCount:         parseInt64(fields[4]),
		}, nil
	}

	return nil, fmt.Errorf("directory signature output missing for %s", actualPath)
}

func buildDirectoryScanScript(scanRoot string, pruneActualPaths []string) string {
	pruneMatchers := []string{
		fmt.Sprintf("-path %s", shellQuote(path.Join(scanRoot, ".snap"))),
	}
	for _, name := range sortedPrunableNestedDirNames() {
		pruneMatchers = append(pruneMatchers, fmt.Sprintf("-name %s", shellQuote(name)))
	}
	for _, prunePath := range pruneActualPaths {
		normalized := normalizeUnixPath(prunePath)
		if normalized == "" || normalized == "." {
			continue
		}
		pruneMatchers = append(pruneMatchers, fmt.Sprintf("-path %s", shellQuote(normalized)))
	}

	return fmt.Sprintf(
		`find %s \( %s \) -prune -o -type d -print0 | xargs -0 -r -n 32 sh -c 'for d in "$@"; do m=$(stat -c %%Y "$d" 2>/dev/null || echo 0); s=$(getfattr --only-values -n ceph.dir.rbytes "$d" 2>/dev/null || echo 0); rc=$(getfattr --only-values -n ceph.dir.rctime "$d" 2>/dev/null || stat -c %%Z "$d" 2>/dev/null || echo 0); printf "%%s\037%%s\037%%s\037%%s\0" "$m" "$s" "$rc" "$d"; done' sh`,
		shellQuote(scanRoot),
		strings.Join(pruneMatchers, " -o "),
	)
}

func sortedPrunableNestedDirNames() []string {
	names := make([]string, 0, len(definitelyNotPublicModelCopyNestedDirSet))
	for name := range definitelyNotPublicModelCopyNestedDirSet {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *Service) scanPathDirectories(
	toolboxPod *corev1.Pod,
	scanID string,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	logicalBase string,
	scanRoot string,
) ([]model.StorageIndexEntry, error) {
	return s.scanPathDirectoriesWithPrunedChildren(
		toolboxPod,
		scanID,
		workspaceType,
		workspaceName,
		logicalBase,
		scanRoot,
		nil,
	)
}

func (s *Service) scanPathDirectoriesWithPrunedChildren(
	toolboxPod *corev1.Pod,
	scanID string,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	logicalBase string,
	scanRoot string,
	pruneActualPaths []string,
) ([]model.StorageIndexEntry, error) {
	klog.Infof(
		"storageindex: 正在执行目录骨架扫描 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s scan_root=%s",
		scanID, workspaceType, workspaceName, logicalBase, scanRoot,
	)

	script := buildDirectoryScanScript(scanRoot, pruneActualPaths)

	var stdout strings.Builder
	var stderr strings.Builder
	progress := &findProgressWriter{
		scanID:        scanID,
		workspaceType: workspaceType,
		workspaceName: workspaceName,
		everyRecords:  scanProgressLogEveryRecord,
	}

	err := ceph.ExecInPodStream(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
		io.MultiWriter(&stdout, progress),
		&stderr,
	)
	if err != nil {
		return nil, fmt.Errorf("scan workspace directories failed: %w, stderr: %s", err, stderr.String())
	}

	records := strings.Split(stdout.String(), findRecordSeparator)
	klog.Infof(
		"storageindex: 目录骨架扫描完成 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s raw_record_count=%d stderr_len=%d",
		scanID, workspaceType, workspaceName, logicalBase, len(records), stderr.Len(),
	)

	entries := make([]model.StorageIndexEntry, 0, len(records))
	for _, record := range records {
		if record == "" {
			continue
		}
		fields := strings.Split(record, findFieldSeparator)
		if len(fields) != 4 {
			continue
		}

		fullPath := normalizeUnixPath(fields[3])
		relativePath := relativeUnixPath(scanRoot, fullPath)
		logicalPath := logicalBase
		if relativePath != "." {
			logicalPath = path.Join(logicalBase, relativePath)
		}
		parentPath := ""
		if logicalPath != logicalBase {
			parentPath = path.Dir(logicalPath)
			if parentPath == "." {
				parentPath = logicalBase
			}
		}

		entry := model.StorageIndexEntry{
			WorkspaceType: workspaceType,
			WorkspaceName: workspaceName,
			ScanID:        scanID,
			LogicalPath:   logicalPath,
			RelativePath:  relativeLogicalPath(logicalBase, logicalPath),
			ParentPath:    parentPath,
			Name:          path.Base(logicalPath),
			EntryType:     model.StorageIndexEntryTypeDir,
			SizeBytes:     parseInt64(fields[1]),
			ModifiedAt:    parseUnixTimestamp(fields[0]),
			ChangedAt:     parseUnixTimestamp(fields[2]),
			IsTopLevel:    isTopLevelRelative(relativeLogicalPath(logicalBase, logicalPath)),
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LogicalPath < entries[j].LogicalPath
	})

	klog.Infof(
		"storageindex: 目录骨架规范化完成 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s dir_count=%d",
		scanID, workspaceType, workspaceName, logicalBase, len(entries),
	)
	return entries, nil
}

func (s *Service) scanPathEntries(
	toolboxPod *corev1.Pod,
	scanID string,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
	logicalBase string,
	scanRoot string,
) ([]model.StorageIndexEntry, error) {
	klog.Infof(
		"storageindex: 正在执行 find 扫描 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s scan_root=%s",
		scanID, workspaceType, workspaceName, logicalBase, scanRoot,
	)

	script := fmt.Sprintf(
		"find %s \\( %s \\) -prune -o -printf %s",
		shellQuote(scanRoot),
		strings.Join(append([]string{fmt.Sprintf("-path %s", shellQuote(path.Join(scanRoot, ".snap")))}, func() []string {
			items := make([]string, 0, len(sortedPrunableNestedDirNames()))
			for _, name := range sortedPrunableNestedDirNames() {
				items = append(items, fmt.Sprintf("-name %s", shellQuote(name)))
			}
			return items
		}()...), " -o "),
		shellQuote(`%y\037%i\037%s\037%T@\037%C@\037%A@\037%U\037%G\037%m\037%n\037%p\0`),
	)

	var stdout strings.Builder
	var stderr strings.Builder
	progress := &findProgressWriter{
		scanID:        scanID,
		workspaceType: workspaceType,
		workspaceName: workspaceName,
		everyRecords:  scanProgressLogEveryRecord,
	}

	err := ceph.ExecInPodStream(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
		io.MultiWriter(&stdout, progress),
		&stderr,
	)
	if err != nil {
		return nil, fmt.Errorf("scan workspace entries failed: %w, stderr: %s", err, stderr.String())
	}

	output := stdout.String()
	records := strings.Split(output, findRecordSeparator)
	klog.Infof(
		"storageindex: find 扫描完成 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s raw_record_count=%d stderr_len=%d",
		scanID, workspaceType, workspaceName, logicalBase, len(records), stderr.Len(),
	)
	entries := make([]model.StorageIndexEntry, 0, len(records))

	for _, record := range records {
		if record == "" {
			continue
		}
		fields := strings.Split(record, findFieldSeparator)
		if len(fields) != 11 {
			continue
		}

		fullPath := normalizeUnixPath(fields[10])
		relativePath := relativeUnixPath(scanRoot, fullPath)
		logicalPath := logicalBase
		if relativePath != "." {
			logicalPath = path.Join(logicalBase, relativePath)
		}

		parentPath := ""
		if logicalPath != logicalBase {
			parentPath = path.Dir(logicalPath)
			if parentPath == "." {
				parentPath = logicalBase
			}
		}

		entry := model.StorageIndexEntry{
			WorkspaceType: workspaceType,
			WorkspaceName: workspaceName,
			ScanID:        scanID,
			LogicalPath:   logicalPath,
			RelativePath:  relativePath,
			ParentPath:    parentPath,
			Name:          path.Base(logicalPath),
			EntryType:     parseEntryType(fields[0]),
			SizeBytes:     parseInt64(fields[2]),
			OwnerUID:      parseInt64(fields[6]),
			OwnerGID:      parseInt64(fields[7]),
			Mode:          strings.TrimSpace(fields[8]),
			LinkCount:     parseInt64(fields[9]),
			ModifiedAt:    parseUnixTimestamp(fields[3]),
			ChangedAt:     parseUnixTimestamp(fields[4]),
			AccessedAt:    parseUnixTimestamp(fields[5]),
			IsTopLevel:    isTopLevelRelative(relativePath),
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LogicalPath < entries[j].LogicalPath
	})

	klog.Infof(
		"storageindex: 规范化条目完成 scan_id=%s workspace_type=%s workspace_name=%s logical_base=%s normalized_entry_count=%d",
		scanID, workspaceType, workspaceName, logicalBase, len(entries),
	)

	return entries, nil
}

func buildDirectoryMetrics(
	scanID string,
	workspace resolvedWorkspace,
	entries []model.StorageIndexEntry,
) ([]model.StorageIndexDirectoryMetric, int64) {
	metricsByPath := make(map[string]*model.StorageIndexDirectoryMetric)
	immediateChildDirs := make(map[string][]string)
	immediateChildFileCount := make(map[string]int64)
	latestModifiedByPath := make(map[string]*time.Time)

	ensureMetric := func(entryPath, parentPath, name string, depth int, isTopLevel bool) *model.StorageIndexDirectoryMetric {
		if metric, ok := metricsByPath[entryPath]; ok {
			return metric
		}
		metric := &model.StorageIndexDirectoryMetric{
			WorkspaceType: workspace.WorkspaceType,
			WorkspaceName: workspace.WorkspaceName,
			ScanID:        scanID,
			Path:          entryPath,
			ParentPath:    parentPath,
			Name:          name,
			Depth:         depth,
			IsTopLevel:    isTopLevel,
		}
		metricsByPath[entryPath] = metric
		return metric
	}

	ensureMetric(workspace.LogicalPath, "", path.Base(workspace.LogicalPath), 0, false)

	for _, entry := range entries {
		if entry.EntryType == model.StorageIndexEntryTypeDir {
			metric := ensureMetric(
				entry.LogicalPath,
				entry.ParentPath,
				entry.Name,
				depthFromRelative(entry.RelativePath),
				entry.IsTopLevel,
			)
			metric.TotalSizeBytes = entry.SizeBytes
			if entry.ParentPath != "" {
				immediateChildDirs[entry.ParentPath] = append(immediateChildDirs[entry.ParentPath], entry.Name)
			}
			if entry.ModifiedAt != nil {
				latestModifiedByPath[entry.LogicalPath] = entry.ModifiedAt
			}
		}
	}

	for _, entry := range entries {
		switch entry.EntryType {
		case model.StorageIndexEntryTypeFile:
			startPath := entry.ParentPath
			if startPath == "" {
				startPath = workspace.LogicalPath
			}
			for _, ancestor := range ancestorDirectories(startPath, workspace.LogicalPath) {
				metric := ensureMetric(ancestor, parentForDirectory(ancestor, workspace.LogicalPath), path.Base(ancestor), depthFromRoot(ancestor, workspace.LogicalPath), isTopLevelPath(ancestor, workspace.LogicalPath))
				metric.TotalSizeBytes += entry.SizeBytes
				metric.FileCount++
			}
		case model.StorageIndexEntryTypeDir:
			if entry.LogicalPath == workspace.LogicalPath {
				continue
			}
			for _, ancestor := range ancestorDirectories(path.Dir(entry.LogicalPath), workspace.LogicalPath) {
				metric := ensureMetric(ancestor, parentForDirectory(ancestor, workspace.LogicalPath), path.Base(ancestor), depthFromRoot(ancestor, workspace.LogicalPath), isTopLevelPath(ancestor, workspace.LogicalPath))
				metric.DirectoryCount++
			}
		}
	}

	metrics := make([]model.StorageIndexDirectoryMetric, 0, len(metricsByPath))
	rootTotal := int64(0)
	for _, metric := range metricsByPath {
		childDirs := immediateChildDirs[metric.Path]
		sort.Strings(childDirs)
		metric.ImmediateChildDirCount = int64(len(childDirs))
		metric.ImmediateChildFileCount = immediateChildFileCount[metric.Path]
		metric.LatestModifiedAt = latestModifiedByPath[metric.Path]
		metric.Signature = buildDirectorySignature(metric, childDirs)
		metric.CategoryHint = classifyDirectory(metric.Path)
		metric.CandidateScore = computeCandidateScore(metric)
		metrics = append(metrics, *metric)
		if metric.Path == workspace.LogicalPath {
			rootTotal = metric.TotalSizeBytes
		}
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Path < metrics[j].Path
	})

	return metrics, rootTotal
}

func (s *Service) detectRedundancy(
	ctx context.Context,
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	entries []model.StorageIndexEntry,
	dirMetrics []model.StorageIndexDirectoryMetric,
) ([]model.StorageIndexRedundancyHit, error) {
	_ = toolboxPod
	_ = prefixConfig
	_ = entries
	if workspace.WorkspaceType == model.StorageIndexWorkspaceTypePublic {
		return nil, nil
	}

	db := query.GetDB().WithContext(ctx)

	var publicDirs []model.StorageIndexDirectoryMetric
	if err := db.
		Where("workspace_type = ? AND is_top_level = ?", model.StorageIndexWorkspaceTypePublic, true).
		Find(&publicDirs).Error; err != nil {
		return nil, fmt.Errorf("query public directory baseline failed: %w", err)
	}
	if len(publicDirs) == 0 {
		return nil, nil
	}

	dirBaseline := make(map[string][]model.StorageIndexDirectoryMetric)
	for _, item := range publicDirs {
		key := redundancyDirectoryKey(item.Name, item.TotalSizeBytes)
		dirBaseline[key] = append(dirBaseline[key], item)
	}

	hits := make([]model.StorageIndexRedundancyHit, 0)
	seen := make(map[string]struct{})

	for _, metric := range dirMetrics {
		if !metric.IsTopLevel || metric.TotalSizeBytes <= 0 || metric.Path == workspace.LogicalPath {
			continue
		}
		key := redundancyDirectoryKey(metric.Name, metric.TotalSizeBytes)
		candidates := dirBaseline[key]
		if len(candidates) == 0 {
			continue
		}
		publicMetric := candidates[0]
		hitKey := metric.Path + "->" + publicMetric.Path
		if _, ok := seen[hitKey]; ok {
			continue
		}
		seen[hitKey] = struct{}{}
		hits = append(hits, model.StorageIndexRedundancyHit{
			WorkspaceType:      workspace.WorkspaceType,
			WorkspaceName:      workspace.WorkspaceName,
			ScanID:             scanID,
			TargetType:         model.StorageIndexRedundancyTargetTypeDirectory,
			TargetPath:         metric.Path,
			PublicPath:         publicMetric.Path,
			MatchKey:           key,
			Evidence:           "目录名与目录总大小匹配公共空间基线，疑似重复保存的模型或数据集目录",
			Confidence:         redundancyConfidenceHigh,
			VerificationStatus: model.StorageIndexVerificationStatusSuspected,
			VerificationMode:   verificationModeMetadata,
			EstimatedBytes:     metric.TotalSizeBytes,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].EstimatedBytes == hits[j].EstimatedBytes {
			return hits[i].TargetPath < hits[j].TargetPath
		}
		return hits[i].EstimatedBytes > hits[j].EstimatedBytes
	})

	return hits, nil
}

func (s *Service) buildCandidates(
	ctx context.Context,
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	dirMetrics []model.StorageIndexDirectoryMetric,
	hits []model.StorageIndexRedundancyHit,
	existingCandidates []model.StorageIndexCandidate,
) ([]model.StorageIndexCandidate, []model.StorageIndexCandidateFile, []model.StorageIndexRedundancyHit, error) {
	if workspace.WorkspaceType == model.StorageIndexWorkspaceTypePublic {
		return nil, nil, nil, nil
	}

	var publicRoots []model.StorageIndexPublicRootBaseline
	if err := query.GetDB().WithContext(ctx).
		Where("category <> ''").
		Find(&publicRoots).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("query public root baseline failed: %w", err)
	}
	if len(publicRoots) == 0 {
		return nil, nil, nil, nil
	}
	publicRootLookup := buildPublicRootLookup(publicRoots)

	candidates := make([]model.StorageIndexCandidate, 0)
	sort.Slice(dirMetrics, func(i, j int) bool {
		if dirMetrics[i].Depth == dirMetrics[j].Depth {
			return dirMetrics[i].Path < dirMetrics[j].Path
		}
		return dirMetrics[i].Depth < dirMetrics[j].Depth
	})
	matchedCandidateRoots := make([]string, 0)
	skippedCandidateRoots := make([]string, 0)
	for _, metric := range dirMetrics {
		if metric.Path == workspace.LogicalPath {
			continue
		}
		if isCoveredByPrefixes(metric.Path, skippedCandidateRoots) {
			continue
		}
		if shouldSkipTopLevelModelCopyCandidate(metric) {
			skippedCandidateRoots = appendUniquePrefix(skippedCandidateRoots, metric.Path)
			continue
		}
		if isCoveredByPrefixes(metric.Path, matchedCandidateRoots) {
			continue
		}

		matchedPublicPath := ""
		evidence := ""
		score := metric.CandidateScore
		for _, publicRoot := range publicRootLookup[strings.ToLower(strings.TrimSpace(metric.Name))] {
			if metric.TotalSizeBytes <= 0 {
				continue
			}
			if !roughlySameSize(metric.TotalSizeBytes, publicRoot.TotalSizeBytes) {
				continue
			}
			if publicRoot.Category == metric.CategoryHint || metric.CategoryHint == "" {
				matchedPublicPath = publicRoot.LogicalPath
				evidence = "目录名称与大小接近公共空间基线，疑似为冗余目录"
				if score < 80 {
					score = 80
				}
				break
			}
		}

		if matchedPublicPath == "" {
			continue
		}
		if score < 10 {
			continue
		}
		if evidence == "" {
			evidence = "目录类别提示命中，作为冗余候选目录保留待验证"
		}

		candidate := model.StorageIndexCandidate{
			WorkspaceType:  workspace.WorkspaceType,
			WorkspaceName:  workspace.WorkspaceName,
			ScanID:         scanID,
			CandidateType:  loCoalesce(metric.CategoryHint, inferCandidateTypeFromPublicPath(matchedPublicPath)),
			TargetPath:     metric.Path,
			PublicPath:     matchedPublicPath,
			Evidence:       evidence,
			CandidateScore: score,
			Status:         model.StorageIndexCandidateStatusSuspected,
		}
		candidates = append(candidates, candidate)
		if matchedPublicPath != "" {
			matchedCandidateRoots = appendUniquePrefix(matchedCandidateRoots, candidate.TargetPath)
		}
	}
	candidates = mergeExistingCandidateBindings(scanID, workspace, candidates, existingCandidates)

	candidateFiles := make([]model.StorageIndexCandidateFile, 0)
	verifiedFileHits := make([]model.StorageIndexRedundancyHit, 0)
	verifyEligibleCount := 0
	filteredCandidates := make([]model.StorageIndexCandidate, 0, len(candidates))
	for i := range candidates {
		if candidates[i].PublicPath == "" {
			continue
		}
		verifyEligibleCount++
		files, verifiedHits, err := s.verifyCandidateDirectory(
			toolboxPod,
			prefixConfig,
			scanID,
			workspace,
			candidates[i].TargetPath,
			candidates[i].PublicPath,
			candidates[i].CandidateType,
		)
		if err != nil {
			klog.Warningf(
				"storageindex: 候选目录关键文件校验失败 scan_id=%s candidate=%s public=%s err=%v",
				scanID, candidates[i].TargetPath, candidates[i].PublicPath, err,
			)
			continue
		}
		candidateFiles = append(candidateFiles, files...)
		if !allCandidateFilesVerified(files) {
			klog.Infof(
				"storageindex: 候选目录校验未通过，保留为疑似候选 scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s matched_file_count=%d verified_hit_count=%d",
				scanID, workspace.WorkspaceType, workspace.WorkspaceName, candidates[i].TargetPath, candidates[i].PublicPath, len(files), len(verifiedHits),
			)
			if len(files) == 0 {
				candidates[i].Evidence = "候选目录命中公共基线，但当前目录内容与公共目录不再完全一致"
			} else {
				candidates[i].Evidence = "候选目录命中公共基线，但关键文件未全部校验通过"
			}
			filteredCandidates = append(filteredCandidates, candidates[i])
			continue
		}
		verifiedFileHits = append(verifiedFileHits, verifiedHits...)
		candidates[i].Status = model.StorageIndexCandidateStatusVerified
		candidates[i].Evidence = "候选目录中的关键文件与公共空间资源哈希一致，确认存在冗余"
		if candidates[i].CandidateScore < 100 {
			candidates[i].CandidateScore = 100
		}
		hits = append(hits, verifiedHits...)
		filteredCandidates = append(filteredCandidates, candidates[i])
	}
	candidates = filteredCandidates

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].CandidateScore == candidates[j].CandidateScore {
			return candidates[i].TargetPath < candidates[j].TargetPath
		}
		return candidates[i].CandidateScore > candidates[j].CandidateScore
	})

	klog.Infof(
		"storageindex: 候选目录识别完成 scan_id=%s workspace_type=%s workspace_name=%s candidate_count=%d candidate_file_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(candidates), len(candidateFiles),
	)
	klog.Infof(
		"storageindex: candidate rebuild stats scan_id=%s workspace_type=%s workspace_name=%s candidate_count=%d verify_eligible_count=%d candidate_file_count=%d verified_hit_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(candidates), verifyEligibleCount, len(candidateFiles), len(verifiedFileHits),
	)
	return candidates, candidateFiles, verifiedFileHits, nil
}

func mergeExistingCandidateBindings(
	scanID string,
	workspace resolvedWorkspace,
	candidates []model.StorageIndexCandidate,
	existingCandidates []model.StorageIndexCandidate,
) []model.StorageIndexCandidate {
	if len(existingCandidates) == 0 {
		return candidates
	}

	indexByPath := make(map[string]int, len(candidates))
	for i := range candidates {
		indexByPath[normalizeUnixPath(candidates[i].TargetPath)] = i
	}

	for _, existing := range existingCandidates {
		targetPath := normalizeUnixPath(existing.TargetPath)
		publicPath := normalizeUnixPath(existing.PublicPath)
		if targetPath == "" || publicPath == "" {
			continue
		}

		if idx, ok := indexByPath[targetPath]; ok {
			if strings.TrimSpace(candidates[idx].PublicPath) == "" {
				candidates[idx].PublicPath = publicPath
			}
			if strings.TrimSpace(candidates[idx].CandidateType) == "" {
				candidates[idx].CandidateType = existing.CandidateType
			}
			if strings.TrimSpace(candidates[idx].Evidence) == "" {
				candidates[idx].Evidence = existing.Evidence
			}
			if candidates[idx].CandidateScore < existing.CandidateScore {
				candidates[idx].CandidateScore = existing.CandidateScore
			}
			continue
		}

		evidence := strings.TrimSpace(existing.Evidence)
		if evidence == "" {
			evidence = "沿用上一次候选命中的公共基线，执行增量重校验"
		}
		candidateType := strings.TrimSpace(existing.CandidateType)
		if candidateType == "" {
			candidateType = inferCandidateTypeFromPublicPath(publicPath)
		}
		candidates = append(candidates, model.StorageIndexCandidate{
			WorkspaceType:  workspace.WorkspaceType,
			WorkspaceName:  workspace.WorkspaceName,
			ScanID:         scanID,
			CandidateType:  candidateType,
			TargetPath:     targetPath,
			PublicPath:     publicPath,
			Evidence:       evidence,
			CandidateScore: existing.CandidateScore,
			Status:         model.StorageIndexCandidateStatusSuspected,
		})
		indexByPath[targetPath] = len(candidates) - 1
	}

	return candidates
}

func allCandidateFilesVerified(files []model.StorageIndexCandidateFile) bool {
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if file.VerificationStatus != model.StorageIndexVerificationStatusVerified {
			return false
		}
	}
	return true
}

func (s *Service) rebuildCandidates(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
) error {
	db := query.GetDB().WithContext(ctx)
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	toolboxPod, err := ceph.FindCephToolboxPod(s.kubeClient, toolboxNamespace)
	if err != nil {
		return fmt.Errorf("find ceph toolbox pod for candidate rebuild failed: %w", err)
	}

	var dirMetrics []model.StorageIndexDirectoryMetric
	if err := db.
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Find(&dirMetrics).Error; err != nil {
		return fmt.Errorf("query workspace directory metrics failed: %w", err)
	}

	var hits []model.StorageIndexRedundancyHit
	if err := db.
		Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
		Find(&hits).Error; err != nil {
		return fmt.Errorf("query workspace redundancy hits failed: %w", err)
	}

	candidates, candidateFiles, verifiedHits, err := s.buildCandidates(ctx, toolboxPod, prefixConfig, scanID, workspace, dirMetrics, hits, nil)
	if err != nil {
		return err
	}
	allHits := cloneRedundancyHitsForScan(scanID, hits)
	allHits = append(allHits, verifiedHits...)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Delete(&model.StorageIndexRedundancyHit{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Delete(&model.StorageIndexCandidate{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Delete(&model.StorageIndexCandidateFile{}).Error; err != nil {
			return err
		}
		if len(allHits) > 0 {
			if err := insertRedundancyHitsInChunks(tx, scanID, workspace, allHits, insertBatchSize); err != nil {
				return err
			}
		}
		if len(candidates) > 0 {
			if err := insertCandidatesInChunks(tx, scanID, workspace, candidates, insertBatchSize); err != nil {
				return err
			}
		}
		if len(candidateFiles) > 0 {
			if err := insertCandidateFilesInChunks(tx, scanID, workspace, candidateFiles, insertBatchSize); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) refreshIncrementalDerivedState(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
	result *incrementalCollectResult,
) error {
	if workspace.WorkspaceType == model.StorageIndexWorkspaceTypePublic {
		return nil
	}

	affectedMetricPaths := collectAffectedMetricPaths(workspace.LogicalPath, result.ChangedPrefixes, result.RemovedPrefixes, result.NewEntries)
	prefixesToDelete := append([]string{}, result.ChangedPrefixes...)
	prefixesToDelete = append(prefixesToDelete, result.RemovedPrefixes...)

	affectedCandidates, err := listWorkspaceCandidates(ctx, workspace)
	if err != nil {
		return err
	}
	affectedCandidatePaths := collectAffectedCandidatePaths(affectedCandidates, affectedMetricPaths, prefixesToDelete)

	dirMetrics := make([]model.StorageIndexDirectoryMetric, 0, len(affectedMetricPaths))
	if len(affectedMetricPaths) > 0 {
		if err := query.GetDB().WithContext(ctx).
			Where("workspace_type = ? AND workspace_name = ? AND path IN ?", workspace.WorkspaceType, workspace.WorkspaceName, affectedMetricPaths).
			Find(&dirMetrics).Error; err != nil {
			return fmt.Errorf("query affected directory metrics failed: %w", err)
		}
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	toolboxPod, err := ceph.FindCephToolboxPod(s.kubeClient, toolboxNamespace)
	if err != nil {
		return fmt.Errorf("find ceph toolbox pod for incremental derived state failed: %w", err)
	}

	directoryHits, err := s.detectRedundancy(ctx, toolboxPod, prefixConfig, scanID, workspace, nil, dirMetrics)
	if err != nil {
		return err
	}
	candidates, candidateFiles, verifiedHits, err := s.buildCandidates(ctx, toolboxPod, prefixConfig, scanID, workspace, dirMetrics, nil, affectedCandidates)
	if err != nil {
		return err
	}

	allHits := append([]model.StorageIndexRedundancyHit{}, directoryHits...)
	allHits = append(allHits, verifiedHits...)

	klog.Infof(
		"storageindex: 增量派生结果刷新准备完成 scan_id=%s workspace_type=%s workspace_name=%s metric_count=%d affected_candidate_count=%d new_candidate_count=%d new_candidate_file_count=%d new_hit_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, len(dirMetrics), len(affectedCandidatePaths), len(candidates), len(candidateFiles), len(allHits),
	)

	return query.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := deleteWorkspaceRedundancyHitsForExactPaths(tx, workspace, affectedMetricPaths); err != nil {
			return err
		}
		if err := deleteWorkspaceCandidateStateByPaths(tx, workspace, affectedCandidatePaths); err != nil {
			return err
		}
		if len(allHits) > 0 {
			if err := insertRedundancyHitsInChunks(tx, scanID, workspace, allHits, insertBatchSize); err != nil {
				return err
			}
		}
		if len(candidates) > 0 {
			if err := insertCandidatesInChunks(tx, scanID, workspace, candidates, insertBatchSize); err != nil {
				return err
			}
		}
		if len(candidateFiles) > 0 {
			if err := insertCandidateFilesInChunks(tx, scanID, workspace, candidateFiles, insertBatchSize); err != nil {
				return err
			}
		}
		return nil
	})
}

func cloneRedundancyHitsForScan(
	scanID string,
	hits []model.StorageIndexRedundancyHit,
) []model.StorageIndexRedundancyHit {
	if len(hits) == 0 {
		return nil
	}

	cloned := make([]model.StorageIndexRedundancyHit, 0, len(hits))
	for _, hit := range hits {
		hit.ID = 0
		hit.ScanID = scanID
		hit.CreatedAt = time.Time{}
		hit.UpdatedAt = time.Time{}
		cloned = append(cloned, hit)
	}
	return cloned
}

func (s *Service) rebuildPublicFileBaseline(
	ctx context.Context,
	scanID string,
	workspace resolvedWorkspace,
) (int, error) {
	if workspace.WorkspaceType != model.StorageIndexWorkspaceTypePublic {
		return 0, nil
	}

	db := query.GetDB().WithContext(ctx)
	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	toolboxPod, err := ceph.FindCephToolboxPod(s.kubeClient, toolboxNamespace)
	if err != nil {
		return 0, fmt.Errorf("find ceph toolbox pod for public baseline failed: %w", err)
	}

	resourceRoots, err := s.listPublicResourceRoots(ctx, prefixConfig)
	if err != nil {
		return 0, err
	}
	klog.Infof("storageindex: 公共资源登记表加载完成 scan_id=%s root_count=%d", scanID, len(resourceRoots))

	buildResult := &publicBaselineBuildResult{
		Roots: make([]model.StorageIndexPublicRootBaseline, 0, len(resourceRoots)),
		Files: make([]model.StorageIndexPublicFileBaseline, 0),
	}

	for _, root := range resourceRoots {
		actualDir, pathErr := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, root.LogicalPath, prefixConfig)
		if pathErr != nil {
			klog.Warningf("storageindex: 解析公共基线路径失败 scan_id=%s path=%s err=%v", scanID, root.LogicalPath, pathErr)
			continue
		}
		totalSize, sizeErr := ceph.GetCephDirectorySize(s.kubeClient, s.kubeConfig, toolboxNamespace, root.LogicalPath, prefixConfig)
		if sizeErr != nil {
			klog.Warningf("storageindex: 获取公共基线目录大小失败 scan_id=%s path=%s err=%v", scanID, root.LogicalPath, sizeErr)
			continue
		}
		files, scanErr := s.scanComparableFilesByCategory(toolboxPod, actualDir, root.Category)
		if scanErr != nil {
			klog.Warningf("storageindex: 扫描公共基线关键文件失败 scan_id=%s path=%s err=%v", scanID, root.LogicalPath, scanErr)
			continue
		}
		buildResult.Roots = append(buildResult.Roots, model.StorageIndexPublicRootBaseline{
			ScanID:         scanID,
			ResourceName:   root.Name,
			LogicalPath:    root.LogicalPath,
			RootHash:       hashString(root.LogicalPath),
			Category:       root.Category,
			TotalSizeBytes: totalSize,
			KeyFileCount:   int64(len(files)),
			Signature:      hashString(strings.ToLower(root.Name) + "|" + root.Category + "|" + strconv.FormatInt(totalSize, 10) + "|" + strconv.Itoa(len(files))),
		})
		for _, file := range files {
			matchKey := file.RelativePath + "|" + strconv.FormatInt(file.SizeBytes, 10)
			buildResult.Files = append(buildResult.Files, model.StorageIndexPublicFileBaseline{
				ScanID:         scanID,
				PublicRootPath: root.LogicalPath,
				PublicRootHash: hashString(root.LogicalPath),
				FilePath:       path.Join(root.LogicalPath, file.RelativePath),
				FileName:       file.FileName,
				RelativePath:   file.RelativePath,
				SizeBytes:      file.SizeBytes,
				MatchKey:       matchKey,
				MatchKeyHash:   hashString(matchKey),
				HashAlgorithm:  "",
				FileHash:       "",
			})
		}
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM " + (&model.StorageIndexPublicRootBaseline{}).TableName()).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM " + (&model.StorageIndexPublicFileBaseline{}).TableName()).Error; err != nil {
			return err
		}
		if len(buildResult.Roots) > 0 {
			if err := insertPublicRootBaselinesInChunks(tx, scanID, buildResult.Roots, insertBatchSize); err != nil {
				return err
			}
		}
		if len(buildResult.Files) == 0 {
			return nil
		}
		return insertPublicBaselineFilesInChunks(tx, scanID, buildResult.Files, insertBatchSize)
	})
	if err != nil {
		return 0, err
	}

	klog.Infof(
		"storageindex: 公共资源基线重建完成 scan_id=%s root_count=%d keyfile_count=%d",
		scanID, len(buildResult.Roots), len(buildResult.Files),
	)
	return len(buildResult.Files), nil
}

func longestCandidatePrefix(targetPath string, candidates []model.StorageIndexCandidate) string {
	longest := ""
	for _, candidate := range candidates {
		if targetPath == candidate.TargetPath || strings.HasPrefix(targetPath, candidate.TargetPath+"/") {
			if len(candidate.TargetPath) > len(longest) {
				longest = candidate.TargetPath
			}
		}
	}
	return longest
}

func roughlySameSize(a, b int64) bool {
	if a == b {
		return true
	}
	if a <= 0 || b <= 0 {
		return false
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	threshold := int64(float64(maxInt64(a, b)) * 0.05)
	if threshold < 100*1024*1024 {
		threshold = 100 * 1024 * 1024
	}
	return diff <= threshold
}

func buildPublicRootLookup(
	publicRoots []model.StorageIndexPublicRootBaseline,
) map[string][]model.StorageIndexPublicRootBaseline {
	lookup := make(map[string][]model.StorageIndexPublicRootBaseline)
	seen := make(map[string]map[string]struct{})
	appendKey := func(key string, item model.StorageIndexPublicRootBaseline) {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			return
		}
		if _, ok := seen[normalized]; !ok {
			seen[normalized] = make(map[string]struct{})
		}
		identity := normalizeUnixPath(item.LogicalPath)
		if _, ok := seen[normalized][identity]; ok {
			return
		}
		seen[normalized][identity] = struct{}{}
		lookup[normalized] = append(lookup[normalized], item)
	}
	for _, item := range publicRoots {
		appendKey(item.ResourceName, item)
		appendKey(path.Base(item.LogicalPath), item)
	}
	return lookup
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var candidateFileNameAllowList = []string{
	"config.json",
	"tokenizer.json",
	"tokenizer_config.json",
	"generation_config.json",
	"dataset_info.json",
}

var candidateFileSuffixAllowList = []string{
	".safetensors",
	".bin",
	".pt",
	".pth",
	".ckpt",
	".index.json",
}

type candidateFileProbe struct {
	RelativePath string
	FileName     string
	ActualPath   string
	SizeBytes    int64
}

func (s *Service) verifyCandidateDirectory(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	targetPath string,
	publicPath string,
	candidateType string,
) ([]model.StorageIndexCandidateFile, []model.StorageIndexRedundancyHit, error) {
	if toolboxPod != nil {
		if strings.TrimSpace(candidateType) == "dataset_dir" {
			return s.verifyCandidateDirectoryByNameSize(toolboxPod, prefixConfig, scanID, workspace, targetPath, publicPath, candidateType)
		}
		return s.verifyCandidateDirectoryOptimized(toolboxPod, prefixConfig, scanID, workspace, targetPath, publicPath)
	}

	targetActual, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, targetPath, prefixConfig)
	if err != nil {
		return nil, nil, err
	}
	missingTargetFiles, missingPublicFiles, err := s.ensureDirectoryFileSetMatches(toolboxPod, prefixConfig, targetActual, publicPath)
	if err != nil {
		return nil, nil, err
	}
	if len(missingTargetFiles) > 0 || len(missingPublicFiles) > 0 {
		klog.Infof(
			"storageindex: candidate hash compare rejected due to file set drift scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s missing_target=%v missing_public=%v",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, missingTargetFiles, missingPublicFiles,
		)
		return nil, nil, nil
	}

	targetFiles, err := s.scanCandidateKeyFiles(toolboxPod, targetActual)
	if err != nil {
		return nil, nil, err
	}

	publicRootHash := hashString(publicPath)
	matchKeyHashes := make([]string, 0, len(targetFiles))
	keyByHash := make(map[string]string, len(targetFiles))
	for _, target := range targetFiles {
		matchKey := target.RelativePath + "|" + strconv.FormatInt(target.SizeBytes, 10)
		matchKeyHash := hashString(matchKey)
		matchKeyHashes = append(matchKeyHashes, matchKeyHash)
		keyByHash[matchKeyHash] = matchKey
	}

	var publicFiles []model.StorageIndexPublicFileBaseline
	if len(matchKeyHashes) > 0 {
		if err := query.GetDB().WithContext(context.Background()).
			Where("public_root_hash = ? AND match_key_hash IN ?", publicRootHash, matchKeyHashes).
			Find(&publicFiles).Error; err != nil {
			return nil, nil, fmt.Errorf("query public file baseline failed: %w", err)
		}
	}
	publicByKey := make(map[string]model.StorageIndexPublicFileBaseline, len(publicFiles))
	for _, item := range publicFiles {
		publicByKey[item.MatchKeyHash] = item
	}

	candidateFiles := make([]model.StorageIndexCandidateFile, 0)
	verifiedHits := make([]model.StorageIndexRedundancyHit, 0)
	for _, target := range targetFiles {
		key := target.RelativePath + "|" + strconv.FormatInt(target.SizeBytes, 10)
		keyHash := hashString(key)
		publicCandidate, ok := publicByKey[keyHash]
		if !ok {
			continue
		}

		targetHash, publicHash, hashErr := s.computePublicBaselineAwareHashes(toolboxPod, target.ActualPath, publicCandidate)
		status := model.StorageIndexVerificationStatusSuspected
		if hashErr == nil && targetHash == publicHash {
			status = model.StorageIndexVerificationStatusVerified
		}

		targetLogicalPath := path.Join(targetPath, target.RelativePath)
		publicLogicalPath := publicCandidate.FilePath
		candidateFiles = append(candidateFiles, model.StorageIndexCandidateFile{
			WorkspaceType:      workspace.WorkspaceType,
			WorkspaceName:      workspace.WorkspaceName,
			ScanID:             scanID,
			CandidatePath:      targetPath,
			FilePath:           targetLogicalPath,
			FileName:           target.FileName,
			RelativePath:       target.RelativePath,
			SizeBytes:          target.SizeBytes,
			MatchedPublicFile:  publicLogicalPath,
			HashAlgorithm:      hashAlgorithmSHA256,
			FileHash:           targetHash,
			VerificationStatus: status,
		})

		if status == model.StorageIndexVerificationStatusVerified {
			verifiedHits = append(verifiedHits, model.StorageIndexRedundancyHit{
				WorkspaceType:      workspace.WorkspaceType,
				WorkspaceName:      workspace.WorkspaceName,
				ScanID:             scanID,
				TargetType:         model.StorageIndexRedundancyTargetTypeFile,
				TargetPath:         targetLogicalPath,
				PublicPath:         publicLogicalPath,
				MatchKey:           key,
				Evidence:           "候选目录中的关键文件相对路径、大小与公共空间一致，且 SHA256 校验一致",
				Confidence:         redundancyConfidenceHigh,
				VerificationStatus: model.StorageIndexVerificationStatusVerified,
				VerificationMode:   hashAlgorithmSHA256,
				HashAlgorithm:      hashAlgorithmSampledSHA256,
				TargetHash:         targetHash,
				PublicHash:         publicHash,
				EstimatedBytes:     target.SizeBytes,
			})
		}
	}

	return candidateFiles, verifiedHits, nil
}

func (s *Service) verifyCandidateDirectoryOptimized(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	targetPath string,
	publicPath string,
) ([]model.StorageIndexCandidateFile, []model.StorageIndexRedundancyHit, error) {
	targetActual, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, targetPath, prefixConfig)
	if err != nil {
		return nil, nil, err
	}
	missingTargetFiles, missingPublicFiles, err := s.ensureDirectoryFileSetMatches(
		toolboxPod,
		prefixConfig,
		targetActual,
		publicPath,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(missingTargetFiles) > 0 || len(missingPublicFiles) > 0 {
		klog.Infof(
			"storageindex: candidate hash compare rejected due to file set drift scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s missing_target=%v missing_public=%v",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, missingTargetFiles, missingPublicFiles,
		)
		return nil, nil, nil
	}

	targetFiles, err := s.scanCandidateKeyFiles(toolboxPod, targetActual)
	if err != nil {
		return nil, nil, err
	}

	publicRootHash := hashString(publicPath)
	matchKeyHashes := make([]string, 0, len(targetFiles))
	for _, target := range targetFiles {
		matchKey := target.RelativePath + "|" + strconv.FormatInt(target.SizeBytes, 10)
		matchKeyHashes = append(matchKeyHashes, hashString(matchKey))
	}

	var publicFiles []model.StorageIndexPublicFileBaseline
	if len(matchKeyHashes) > 0 {
		if err := query.GetDB().WithContext(context.Background()).
			Where("public_root_hash = ? AND match_key_hash IN ?", publicRootHash, matchKeyHashes).
			Find(&publicFiles).Error; err != nil {
			return nil, nil, fmt.Errorf("query public file baseline failed: %w", err)
		}
	}
	publicByKey := make(map[string]model.StorageIndexPublicFileBaseline, len(publicFiles))
	for _, item := range publicFiles {
		publicByKey[item.MatchKeyHash] = item
	}

	type matchedCandidateFile struct {
		target          candidateFileProbe
		publicCandidate model.StorageIndexPublicFileBaseline
		key             string
		publicActual    string
	}

	matches := make([]matchedCandidateFile, 0)
	publicRootActual := ""
	for _, target := range targetFiles {
		key := target.RelativePath + "|" + strconv.FormatInt(target.SizeBytes, 10)
		keyHash := hashString(key)
		publicCandidate, ok := publicByKey[keyHash]
		if !ok {
			continue
		}

		publicActual := ""
		needPublicActual := strings.TrimSpace(publicCandidate.FileHash) == "" || strings.TrimSpace(publicCandidate.HashAlgorithm) != hashAlgorithmSampledSHA256 || isSafeTensorsFile(target.FileName)
		if needPublicActual {
			if publicRootActual == "" {
				publicRootActual, err = ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, publicPath, prefixConfig)
				if err != nil {
					return nil, nil, err
				}
			}
			publicActual = normalizeUnixPath(path.Join(publicRootActual, publicCandidate.RelativePath))
		}
		matches = append(matches, matchedCandidateFile{
			target:          target,
			publicCandidate: publicCandidate,
			key:             key,
			publicActual:    publicActual,
		})
	}
	fallbackMatchCount := 0
	if len(matches) == 0 {
		fallbackMatches, fallbackPublicRootActual, err := buildFallbackCandidateMatches(targetFiles, publicFiles)
		if err != nil {
			return nil, nil, err
		}
		if len(fallbackMatches) > 0 {
			if publicRootActual == "" {
				publicRootActual = fallbackPublicRootActual
			}
			for _, match := range fallbackMatches {
				if match.publicActual == "" && (strings.TrimSpace(match.publicCandidate.FileHash) == "" || isSafeTensorsFile(match.target.FileName)) {
					if publicRootActual == "" {
						publicRootActual, err = ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, publicPath, prefixConfig)
						if err != nil {
							return nil, nil, err
						}
					}
					match.publicActual = normalizeUnixPath(path.Join(publicRootActual, match.publicCandidate.RelativePath))
				}
				matches = append(matches, match)
			}
			fallbackMatchCount = len(fallbackMatches)
		}
	}
	if len(matches) == 0 {
		klog.Infof(
			"storageindex: candidate hash compare skipped scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s target_key_file_count=%d matched_key_file_count=0 fallback_match_count=0",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, len(targetFiles),
		)
		return nil, nil, nil
	}

	targetActualPaths := make([]string, 0, len(matches))
	targetActualSizes := make(map[string]int64, len(matches))
	missingPublicActualPaths := make([]string, 0)
	publicActualSizes := make(map[string]int64)
	for _, match := range matches {
		targetActualPaths = append(targetActualPaths, match.target.ActualPath)
		targetActualSizes[match.target.ActualPath] = match.target.SizeBytes
		if match.publicActual != "" {
			missingPublicActualPaths = append(missingPublicActualPaths, match.publicActual)
			publicActualSizes[match.publicActual] = match.publicCandidate.SizeBytes
		}
	}

	targetHashes, err := s.computeActualFileHashesBatchWithSizes(toolboxPod, targetActualPaths, targetActualSizes)
	if err != nil {
		return nil, nil, err
	}
	publicHashes := make(map[string]string)
	if len(missingPublicActualPaths) > 0 {
		publicHashes, err = s.computeActualFileHashesBatchWithSizes(toolboxPod, missingPublicActualPaths, publicActualSizes)
		if err != nil {
			return nil, nil, err
		}
		for _, match := range matches {
			if match.publicCandidate.ID == 0 || match.publicActual == "" {
				continue
			}
			publicHash := publicHashes[match.publicActual]
			if publicHash == "" {
				continue
			}
			_ = query.GetDB().
				Model(&model.StorageIndexPublicFileBaseline{}).
				Where("id = ?", match.publicCandidate.ID).
				Updates(map[string]any{
					"hash_algorithm": hashAlgorithmSampledSHA256,
					"file_hash":      publicHash,
				}).Error
		}
	}

	candidateFiles := make([]model.StorageIndexCandidateFile, 0, len(matches))
	verifiedHits := make([]model.StorageIndexRedundancyHit, 0)
	for _, match := range matches {
		targetHash := targetHashes[match.target.ActualPath]
		publicHash := strings.TrimSpace(match.publicCandidate.FileHash)
		if (publicHash == "" || strings.TrimSpace(match.publicCandidate.HashAlgorithm) != hashAlgorithmSampledSHA256) && match.publicActual != "" {
			publicHash = publicHashes[match.publicActual]
		}
		status := model.StorageIndexVerificationStatusSuspected
		verificationMode := hashAlgorithmSampledSHA256
		if isSafeTensorsFile(match.target.FileName) {
			headersMatch, headerErr := s.compareSafetensorsHeaders(toolboxPod, match.target.ActualPath, match.publicActual)
			if headerErr == nil && headersMatch {
				verificationMode = verificationModeSafeTensorsHdrAndSampledSHA
			} else {
				targetHash = ""
				publicHash = ""
			}
		}
		if targetHash != "" && publicHash != "" && targetHash == publicHash {
			status = model.StorageIndexVerificationStatusVerified
		}

		targetLogicalPath := path.Join(targetPath, match.target.RelativePath)
		publicLogicalPath := match.publicCandidate.FilePath
		candidateFiles = append(candidateFiles, model.StorageIndexCandidateFile{
			WorkspaceType:      workspace.WorkspaceType,
			WorkspaceName:      workspace.WorkspaceName,
			ScanID:             scanID,
			CandidatePath:      targetPath,
			FilePath:           targetLogicalPath,
			FileName:           match.target.FileName,
			RelativePath:       match.target.RelativePath,
			SizeBytes:          match.target.SizeBytes,
			MatchedPublicFile:  publicLogicalPath,
			HashAlgorithm:      hashAlgorithmSampledSHA256,
			FileHash:           targetHash,
			VerificationStatus: status,
		})

		if status == model.StorageIndexVerificationStatusVerified {
			verifiedHits = append(verifiedHits, model.StorageIndexRedundancyHit{
				WorkspaceType:      workspace.WorkspaceType,
				WorkspaceName:      workspace.WorkspaceName,
				ScanID:             scanID,
				TargetType:         model.StorageIndexRedundancyTargetTypeFile,
				TargetPath:         targetLogicalPath,
				PublicPath:         publicLogicalPath,
				MatchKey:           match.key,
				Evidence:           "候选目录中的关键文件相对路径、大小与公共空间一致，且 SHA256 校验一致",
				Confidence:         redundancyConfidenceHigh,
				VerificationStatus: model.StorageIndexVerificationStatusVerified,
				VerificationMode:   verificationMode,
				HashAlgorithm:      hashAlgorithmSampledSHA256,
				TargetHash:         targetHash,
				PublicHash:         publicHash,
				EstimatedBytes:     match.target.SizeBytes,
			})
		}
	}

	klog.Infof(
		"storageindex: candidate hash compare finished scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s target_key_file_count=%d matched_key_file_count=%d fallback_match_count=%d verified_hit_count=%d missing_public_hash_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, len(targetFiles), len(matches), fallbackMatchCount, len(verifiedHits), len(missingPublicActualPaths),
	)
	return candidateFiles, verifiedHits, nil
}

func (s *Service) verifyCandidateDirectoryByNameSize(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	targetPath string,
	publicPath string,
	candidateType string,
) ([]model.StorageIndexCandidateFile, []model.StorageIndexRedundancyHit, error) {
	targetActual, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, targetPath, prefixConfig)
	if err != nil {
		return nil, nil, err
	}
	missingTargetFiles, missingPublicFiles, err := s.ensureDirectoryFileSetMatches(toolboxPod, prefixConfig, targetActual, publicPath)
	if err != nil {
		return nil, nil, err
	}
	if len(missingTargetFiles) > 0 || len(missingPublicFiles) > 0 {
		klog.Infof(
			"storageindex: candidate keyfile compare rejected due to file set drift scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s missing_target=%v missing_public=%v",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, missingTargetFiles, missingPublicFiles,
		)
		return nil, nil, nil
	}

	targetFiles, err := s.scanComparableFilesByCategory(toolboxPod, targetActual, candidateType)
	if err != nil {
		return nil, nil, err
	}

	publicRootHash := hashString(publicPath)
	matchKeyHashes := make([]string, 0, len(targetFiles))
	for _, target := range targetFiles {
		matchKey := target.RelativePath + "|" + strconv.FormatInt(target.SizeBytes, 10)
		matchKeyHashes = append(matchKeyHashes, hashString(matchKey))
	}

	var publicFiles []model.StorageIndexPublicFileBaseline
	if len(matchKeyHashes) > 0 {
		if err := query.GetDB().WithContext(context.Background()).
			Where("public_root_hash = ? AND match_key_hash IN ?", publicRootHash, matchKeyHashes).
			Find(&publicFiles).Error; err != nil {
			return nil, nil, fmt.Errorf("query public file baseline failed: %w", err)
		}
	}

	type matchedCandidateFile struct {
		target          candidateFileProbe
		publicCandidate model.StorageIndexPublicFileBaseline
		key             string
	}

	publicByKey := make(map[string]model.StorageIndexPublicFileBaseline, len(publicFiles))
	for _, item := range publicFiles {
		publicByKey[item.MatchKeyHash] = item
	}

	matches := make([]matchedCandidateFile, 0)
	for _, target := range targetFiles {
		key := target.RelativePath + "|" + strconv.FormatInt(target.SizeBytes, 10)
		keyHash := hashString(key)
		publicCandidate, ok := publicByKey[keyHash]
		if !ok {
			continue
		}
		matches = append(matches, matchedCandidateFile{
			target:          target,
			publicCandidate: publicCandidate,
			key:             key,
		})
	}

	fallbackMatchCount := 0
	if len(matches) == 0 {
		fallbackMatches := buildFallbackCandidateMatchesByNameSize(targetFiles, publicFiles)
		fallbackMatchCount = len(fallbackMatches)
		for _, match := range fallbackMatches {
			matches = append(matches, matchedCandidateFile{
				target:          match.target,
				publicCandidate: match.publicCandidate,
				key:             match.key,
			})
		}
	}
	if len(matches) == 0 {
		klog.Infof(
			"storageindex: candidate keyfile compare skipped scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s target_key_file_count=%d matched_key_file_count=0 fallback_match_count=0",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, len(targetFiles),
		)
		return nil, nil, nil
	}

	candidateFiles := make([]model.StorageIndexCandidateFile, 0, len(matches))
	verifiedHits := make([]model.StorageIndexRedundancyHit, 0, len(matches))
	for _, match := range matches {
		targetLogicalPath := path.Join(targetPath, match.target.RelativePath)
		publicLogicalPath := match.publicCandidate.FilePath
		candidateFiles = append(candidateFiles, model.StorageIndexCandidateFile{
			WorkspaceType:      workspace.WorkspaceType,
			WorkspaceName:      workspace.WorkspaceName,
			ScanID:             scanID,
			CandidatePath:      targetPath,
			FilePath:           targetLogicalPath,
			FileName:           match.target.FileName,
			RelativePath:       match.target.RelativePath,
			SizeBytes:          match.target.SizeBytes,
			MatchedPublicFile:  publicLogicalPath,
			HashAlgorithm:      "",
			FileHash:           "",
			VerificationStatus: model.StorageIndexVerificationStatusVerified,
		})

		verifiedHits = append(verifiedHits, model.StorageIndexRedundancyHit{
			WorkspaceType:      workspace.WorkspaceType,
			WorkspaceName:      workspace.WorkspaceName,
			ScanID:             scanID,
			TargetType:         model.StorageIndexRedundancyTargetTypeFile,
			TargetPath:         targetLogicalPath,
			PublicPath:         publicLogicalPath,
			MatchKey:           match.key,
			Evidence:           "候选目录中的关键文件名与大小和公共空间资源一致，确认存在冗余",
			Confidence:         redundancyConfidenceHigh,
			VerificationStatus: model.StorageIndexVerificationStatusVerified,
			VerificationMode:   verificationModeFileName,
			HashAlgorithm:      "",
			TargetHash:         "",
			PublicHash:         "",
			EstimatedBytes:     match.target.SizeBytes,
		})
	}

	klog.Infof(
		"storageindex: candidate keyfile compare finished scan_id=%s workspace_type=%s workspace_name=%s candidate=%s public=%s target_key_file_count=%d matched_key_file_count=%d fallback_match_count=%d verified_hit_count=%d",
		scanID, workspace.WorkspaceType, workspace.WorkspaceName, targetPath, publicPath, len(targetFiles), len(matches), fallbackMatchCount, len(verifiedHits),
	)
	return candidateFiles, verifiedHits, nil
}

func (s *Service) scanCandidateKeyFiles(
	toolboxPod *corev1.Pod,
	actualDir string,
) ([]candidateFileProbe, error) {
	script := fmt.Sprintf(
		`find %s -type f \( -name '*.safetensors' -o -name '*.bin' -o -name '*.pt' -o -name '*.pth' -o -name '*.ckpt' -o -name '*.index.json' -o -name 'config.json' -o -name 'tokenizer.json' -o -name 'tokenizer_config.json' -o -name 'generation_config.json' -o -name 'dataset_info.json' \) -printf '%%P\037%%f\037%%s\037%%p\0'`,
		shellQuote(actualDir),
	)
	return s.scanCandidateFilesWithScript(toolboxPod, script, "scan candidate key files failed")
}

func (s *Service) scanModelComparableFiles(
	toolboxPod *corev1.Pod,
	actualDir string,
) ([]candidateFileProbe, error) {
	return s.scanCandidateKeyFiles(toolboxPod, actualDir)
}

func (s *Service) scanDatasetComparableFiles(
	toolboxPod *corev1.Pod,
	actualDir string,
) ([]candidateFileProbe, error) {
	script := fmt.Sprintf(
		`find %s -name .snap -prune -o -type f -printf '%%P\037%%f\037%%s\037%%p\0'`,
		shellQuote(actualDir),
	)
	return s.scanCandidateFilesWithScript(toolboxPod, script, "scan dataset comparable files failed")
}

func (s *Service) scanCandidateFilesWithScript(
	toolboxPod *corev1.Pod,
	script string,
	errorMessage string,
) ([]candidateFileProbe, error) {
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errorMessage, err)
	}

	records := strings.Split(output, findRecordSeparator)
	result := make([]candidateFileProbe, 0, len(records))
	for _, record := range records {
		if record == "" {
			continue
		}
		fields := strings.Split(record, findFieldSeparator)
		if len(fields) != 4 {
			continue
		}
		result = append(result, candidateFileProbe{
			RelativePath: normalizeUnixPath(fields[0]),
			FileName:     fields[1],
			SizeBytes:    parseInt64(fields[2]),
			ActualPath:   normalizeUnixPath(fields[3]),
		})
	}
	return result, nil
}

func (s *Service) ensureDirectoryFileSetMatches(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	targetActual string,
	publicPath string,
) ([]string, []string, error) {
	publicActual, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, publicPath, prefixConfig)
	if err != nil {
		return nil, nil, err
	}
	targetFiles, err := s.scanDatasetComparableFiles(toolboxPod, targetActual)
	if err != nil {
		return nil, nil, err
	}
	publicFiles, err := s.scanDatasetComparableFiles(toolboxPod, publicActual)
	if err != nil {
		return nil, nil, err
	}
	missingTarget, missingPublic := compareStrictDirectoryFileSets(targetFiles, publicFiles)
	return missingTarget, missingPublic, nil
}

func (s *Service) scanComparableFilesByCategory(
	toolboxPod *corev1.Pod,
	actualDir string,
	category string,
) ([]candidateFileProbe, error) {
	if strings.TrimSpace(category) == "dataset_dir" {
		return s.scanDatasetComparableFiles(toolboxPod, actualDir)
	}
	return s.scanCandidateKeyFiles(toolboxPod, actualDir)
}

func (s *Service) computeActualFileHashes(
	toolboxPod *corev1.Pod,
	targetActualPath string,
	publicActualPath string,
) (string, string, error) {
	targetHash, err := s.computeActualFileHash(toolboxPod, targetActualPath)
	if err != nil {
		return "", "", err
	}
	publicHash, err := s.computeActualFileHash(toolboxPod, publicActualPath)
	if err != nil {
		return "", "", err
	}
	return targetHash, publicHash, nil
}

func (s *Service) computePublicBaselineAwareHashes(
	toolboxPod *corev1.Pod,
	targetActualPath string,
	publicBaseline model.StorageIndexPublicFileBaseline,
) (string, string, error) {
	targetHash, err := s.computeActualFileHash(toolboxPod, targetActualPath)
	if err != nil {
		return "", "", err
	}

	publicHash := strings.TrimSpace(publicBaseline.FileHash)
	if publicHash != "" {
		return targetHash, publicHash, nil
	}

	cfg := config.GetConfig()
	prefixConfig := ceph.StoragePrefixConfig{
		User:    cfg.Storage.Prefix.User,
		Account: cfg.Storage.Prefix.Account,
		Public:  cfg.Storage.Prefix.Public,
	}
	publicActualPath, err := ceph.ResolveCephFSPath(s.kubeClient, s.kubeConfig, toolboxNamespace, publicBaseline.FilePath, prefixConfig)
	if err != nil {
		return "", "", err
	}
	publicHash, err = s.computeActualFileHash(toolboxPod, publicActualPath)
	if err != nil {
		return "", "", err
	}

	if publicBaseline.ID != 0 {
		_ = query.GetDB().
			Model(&model.StorageIndexPublicFileBaseline{}).
			Where("id = ?", publicBaseline.ID).
			Updates(map[string]any{
				"hash_algorithm": hashAlgorithmSHA256,
				"file_hash":      publicHash,
			}).Error
	}

	return targetHash, publicHash, nil
}

func (s *Service) computeActualFileHash(
	toolboxPod *corev1.Pod,
	actualPath string,
) (string, error) {
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sha256sum", actualPath},
	)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty sha256sum output for %s", actualPath)
	}
	return fields[0], nil
}

func (s *Service) computeActualFileHashesBatch(
	toolboxPod *corev1.Pod,
	actualPaths []string,
) (map[string]string, error) {
	return s.computeActualFileHashesBatchWithSizes(toolboxPod, actualPaths, nil)
}

func (s *Service) computeActualFileHashesBatchWithSizes(
	toolboxPod *corev1.Pod,
	actualPaths []string,
	sizeByPath map[string]int64,
) (map[string]string, error) {
	uniquePaths := make([]string, 0, len(actualPaths))
	seen := make(map[string]struct{}, len(actualPaths))
	for _, actualPath := range actualPaths {
		normalized := normalizeUnixPath(actualPath)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		uniquePaths = append(uniquePaths, normalized)
	}
	if len(uniquePaths) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(uniquePaths))
	for _, actualPath := range uniquePaths {
		sizeBytes := int64(0)
		if sizeByPath != nil {
			sizeBytes = sizeByPath[actualPath]
		}
		hashValue, err := s.computeActualFileSampledHash(toolboxPod, actualPath, sizeBytes)
		if err != nil {
			return nil, err
		}
		result[actualPath] = hashValue
	}

	return result, nil
}

func (s *Service) computeActualFileFullHashesBatch(
	toolboxPod *corev1.Pod,
	actualPaths []string,
) (map[string]string, error) {
	uniquePaths := make([]string, 0, len(actualPaths))
	seen := make(map[string]struct{}, len(actualPaths))
	for _, actualPath := range actualPaths {
		normalized := normalizeUnixPath(actualPath)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		uniquePaths = append(uniquePaths, normalized)
	}
	if len(uniquePaths) == 0 {
		return map[string]string{}, nil
	}

	result := make(map[string]string, len(uniquePaths))
	for _, chunk := range chunkStrings(uniquePaths, 128) {
		args := make([]string, 0, len(chunk)+2)
		args = append(args, "sha256sum", "--")
		args = append(args, chunk...)
		output, err := ceph.ExecInPod(
			s.kubeClient,
			s.kubeConfig,
			toolboxPod,
			args,
		)
		if err != nil {
			return nil, err
		}
		for path, hashValue := range parseSha256sumOutput(output) {
			result[path] = hashValue
		}
	}

	for _, actualPath := range uniquePaths {
		if strings.TrimSpace(result[actualPath]) == "" {
			return nil, fmt.Errorf("missing full sha256 output for %s", actualPath)
		}
	}

	return result, nil
}

func chunkStrings(items []string, size int) [][]string {
	if size <= 0 || len(items) == 0 {
		return nil
	}

	chunks := make([][]string, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

func parseSha256sumOutput(output string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len(trimmed) < 64 {
			continue
		}
		hashValue := trimmed[:64]
		actualPath := strings.TrimSpace(trimmed[64:])
		actualPath = strings.TrimPrefix(actualPath, "*")
		actualPath = normalizeUnixPath(actualPath)
		if actualPath == "" {
			continue
		}
		result[actualPath] = hashValue
	}
	return result
}

func isSafeTensorsFile(fileName string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(fileName)), ".safetensors")
}

type sampledRegion struct {
	Offset int64
	Count  int64
}

func buildSampledRegions(sizeBytes int64) []sampledRegion {
	if sizeBytes <= 0 {
		return nil
	}
	if sizeBytes <= sampledHashSegmentBytes {
		return []sampledRegion{{Offset: 0, Count: sizeBytes}}
	}

	offsets := []int64{
		sizeBytes / 4,
		sizeBytes / 2,
		(sizeBytes * 3) / 4,
	}
	regions := make([]sampledRegion, 0, len(offsets))
	seen := make(map[int64]struct{}, len(offsets))
	for _, offset := range offsets {
		if offset > sizeBytes-sampledHashSegmentBytes {
			offset = sizeBytes - sampledHashSegmentBytes
		}
		if offset < 0 {
			offset = 0
		}
		if _, ok := seen[offset]; ok {
			continue
		}
		seen[offset] = struct{}{}
		regions = append(regions, sampledRegion{
			Offset: offset,
			Count:  sampledHashSegmentBytes,
		})
	}
	return regions
}

func (s *Service) computeActualFileSampledHash(
	toolboxPod *corev1.Pod,
	actualPath string,
	sizeBytes int64,
) (string, error) {
	regions := buildSampledRegions(sizeBytes)
	if len(regions) == 0 {
		return "", fmt.Errorf("invalid file size for sampled hash: %d", sizeBytes)
	}

	commands := make([]string, 0, len(regions))
	for _, region := range regions {
		commands = append(
			commands,
			fmt.Sprintf(
				"dd if=%s skip=%d count=%d iflag=skip_bytes,count_bytes 2>/dev/null",
				shellQuote(actualPath),
				region.Offset,
				region.Count,
			),
		)
	}
	script := "{ " + strings.Join(commands, "; ") + "; } | sha256sum"
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty sampled sha256sum output for %s", actualPath)
	}
	return fields[0], nil
}

func (s *Service) compareSafetensorsHeaders(
	toolboxPod *corev1.Pod,
	targetActualPath string,
	publicActualPath string,
) (bool, error) {
	if targetActualPath == "" || publicActualPath == "" {
		return false, fmt.Errorf("empty safetensors path for header compare")
	}
	targetSkeleton, err := s.readSafeTensorsHeaderSkeleton(toolboxPod, targetActualPath)
	if err != nil {
		return false, err
	}
	publicSkeleton, err := s.readSafeTensorsHeaderSkeleton(toolboxPod, publicActualPath)
	if err != nil {
		return false, err
	}
	if len(targetSkeleton) != len(publicSkeleton) {
		return false, nil
	}
	for i := range targetSkeleton {
		if targetSkeleton[i] != publicSkeleton[i] {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) readSafeTensorsHeaderSkeleton(
	toolboxPod *corev1.Pod,
	actualPath string,
) ([]string, error) {
	headerLength, err := s.readSafeTensorsHeaderLength(toolboxPod, actualPath)
	if err != nil {
		return nil, err
	}
	if headerLength <= 0 || headerLength > safetensorsHeaderMaxBytes {
		return nil, fmt.Errorf("invalid safetensors header length %d for %s", headerLength, actualPath)
	}

	script := fmt.Sprintf(
		"dd if=%s skip=8 count=%d iflag=skip_bytes,count_bytes 2>/dev/null",
		shellQuote(actualPath),
		headerLength,
	)
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return nil, err
	}
	return parseSafeTensorsHeaderSkeleton(output)
}

func (s *Service) readSafeTensorsHeaderLength(
	toolboxPod *corev1.Pod,
	actualPath string,
) (int64, error) {
	script := fmt.Sprintf(
		"dd if=%s count=8 iflag=count_bytes 2>/dev/null | od -An -v -t x1",
		shellQuote(actualPath),
	)
	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) != 8 {
		return 0, fmt.Errorf("unexpected safetensors header length bytes for %s: %q", actualPath, output)
	}
	headerBytes := make([]byte, 8)
	for i, field := range fields {
		value, parseErr := strconv.ParseUint(field, 16, 8)
		if parseErr != nil {
			return 0, fmt.Errorf("parse safetensors header length byte failed: %w", parseErr)
		}
		headerBytes[i] = byte(value)
	}
	return int64(binary.LittleEndian.Uint64(headerBytes)), nil
}

func parseSafeTensorsHeaderSkeleton(headerJSON string) ([]string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(headerJSON), &payload); err != nil {
		return nil, err
	}

	type tensorShapePayload struct {
		Shape []int64 `json:"shape"`
	}

	skeleton := make([]string, 0, len(payload))
	for name, raw := range payload {
		if name == "__metadata__" {
			continue
		}
		var tensor tensorShapePayload
		if err := json.Unmarshal(raw, &tensor); err != nil {
			return nil, err
		}
		shapeParts := make([]string, 0, len(tensor.Shape))
		for _, dim := range tensor.Shape {
			shapeParts = append(shapeParts, strconv.FormatInt(dim, 10))
		}
		skeleton = append(skeleton, name+"="+strings.Join(shapeParts, ","))
	}
	sort.Strings(skeleton)
	return skeleton, nil
}

func buildFallbackCandidateMatches(
	targetFiles []candidateFileProbe,
	publicFiles []model.StorageIndexPublicFileBaseline,
) ([]struct {
	target          candidateFileProbe
	publicCandidate model.StorageIndexPublicFileBaseline
	key             string
	publicActual    string
}, string, error) {
	type matchedCandidateFile struct {
		target          candidateFileProbe
		publicCandidate model.StorageIndexPublicFileBaseline
		key             string
		publicActual    string
	}

	targetByNameSize := make(map[string][]candidateFileProbe)
	for _, target := range targetFiles {
		key := strings.ToLower(strings.TrimSpace(target.FileName)) + "|" + strconv.FormatInt(target.SizeBytes, 10)
		targetByNameSize[key] = append(targetByNameSize[key], target)
	}
	publicByNameSize := make(map[string][]model.StorageIndexPublicFileBaseline)
	for _, item := range publicFiles {
		key := strings.ToLower(strings.TrimSpace(item.FileName)) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		publicByNameSize[key] = append(publicByNameSize[key], item)
	}

	matches := make([]matchedCandidateFile, 0)
	for key, targets := range targetByNameSize {
		if len(targets) != 1 {
			continue
		}
		publicCandidates := publicByNameSize[key]
		if len(publicCandidates) != 1 {
			continue
		}
		matchKey := targets[0].RelativePath + "|" + strconv.FormatInt(targets[0].SizeBytes, 10)
		matches = append(matches, matchedCandidateFile{
			target:          targets[0],
			publicCandidate: publicCandidates[0],
			key:             matchKey,
			publicActual:    "",
		})
	}

	result := make([]struct {
		target          candidateFileProbe
		publicCandidate model.StorageIndexPublicFileBaseline
		key             string
		publicActual    string
	}, 0, len(matches))
	for _, match := range matches {
		result = append(result, struct {
			target          candidateFileProbe
			publicCandidate model.StorageIndexPublicFileBaseline
			key             string
			publicActual    string
		}{
			target:          match.target,
			publicCandidate: match.publicCandidate,
			key:             match.key,
			publicActual:    match.publicActual,
		})
	}
	return result, "", nil
}

func buildFallbackCandidateMatchesByNameSize(
	targetFiles []candidateFileProbe,
	publicFiles []model.StorageIndexPublicFileBaseline,
) []struct {
	target          candidateFileProbe
	publicCandidate model.StorageIndexPublicFileBaseline
	key             string
} {
	targetByNameSize := make(map[string][]candidateFileProbe)
	for _, target := range targetFiles {
		key := strings.ToLower(strings.TrimSpace(target.FileName)) + "|" + strconv.FormatInt(target.SizeBytes, 10)
		targetByNameSize[key] = append(targetByNameSize[key], target)
	}
	publicByNameSize := make(map[string][]model.StorageIndexPublicFileBaseline)
	for _, item := range publicFiles {
		key := strings.ToLower(strings.TrimSpace(item.FileName)) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		publicByNameSize[key] = append(publicByNameSize[key], item)
	}

	matches := make([]struct {
		target          candidateFileProbe
		publicCandidate model.StorageIndexPublicFileBaseline
		key             string
	}, 0)
	for key, targets := range targetByNameSize {
		if len(targets) != 1 {
			continue
		}
		publicCandidates := publicByNameSize[key]
		if len(publicCandidates) != 1 {
			continue
		}
		matchKey := targets[0].RelativePath + "|" + strconv.FormatInt(targets[0].SizeBytes, 10)
		matches = append(matches, struct {
			target          candidateFileProbe
			publicCandidate model.StorageIndexPublicFileBaseline
			key             string
		}{
			target:          targets[0],
			publicCandidate: publicCandidates[0],
			key:             matchKey,
		})
	}
	return matches
}

func compareStrictDirectoryFileSets(
	targetFiles []candidateFileProbe,
	publicFiles []candidateFileProbe,
) ([]string, []string) {
	targetByKey := make(map[string]int, len(targetFiles))
	for _, item := range targetFiles {
		key := normalizeUnixPath(item.RelativePath) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		targetByKey[key]++
	}
	publicByKey := make(map[string]int, len(publicFiles))
	for _, item := range publicFiles {
		key := normalizeUnixPath(item.RelativePath) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		publicByKey[key]++
	}

	keys := make(map[string]struct{}, len(targetByKey)+len(publicByKey))
	for key := range targetByKey {
		keys[key] = struct{}{}
	}
	for key := range publicByKey {
		keys[key] = struct{}{}
	}

	missingTarget := make([]string, 0)
	missingPublic := make([]string, 0)
	for key := range keys {
		targetCount := targetByKey[key]
		publicCount := publicByKey[key]
		relativePath := strings.SplitN(key, "|", 2)[0]
		if targetCount > publicCount {
			for i := 0; i < targetCount-publicCount; i++ {
				missingPublic = append(missingPublic, relativePath)
			}
		}
		if publicCount > targetCount {
			for i := 0; i < publicCount-targetCount; i++ {
				missingTarget = append(missingTarget, relativePath)
			}
		}
	}
	sort.Strings(missingTarget)
	sort.Strings(missingPublic)
	return missingTarget, missingPublic
}

func pairCandidateFilesForComparison(
	leftFiles []candidateFileProbe,
	rightFiles []candidateFileProbe,
) ([]struct {
	left  candidateFileProbe
	right candidateFileProbe
}, int, int, []string, []string) {
	type comparePair struct {
		left  candidateFileProbe
		right candidateFileProbe
	}

	leftByExact := make(map[string]candidateFileProbe, len(leftFiles))
	rightByExact := make(map[string]candidateFileProbe, len(rightFiles))
	for _, item := range leftFiles {
		leftByExact[item.RelativePath+"|"+strconv.FormatInt(item.SizeBytes, 10)] = item
	}
	for _, item := range rightFiles {
		rightByExact[item.RelativePath+"|"+strconv.FormatInt(item.SizeBytes, 10)] = item
	}

	pairs := make([]comparePair, 0)
	matchedLeft := make(map[string]struct{}, len(leftFiles))
	matchedRight := make(map[string]struct{}, len(rightFiles))
	exactMatchCount := 0
	for key, left := range leftByExact {
		right, ok := rightByExact[key]
		if !ok {
			continue
		}
		pairs = append(pairs, comparePair{left: left, right: right})
		matchedLeft[left.RelativePath] = struct{}{}
		matchedRight[right.RelativePath] = struct{}{}
		exactMatchCount++
	}

	unmatchedLeft := make([]candidateFileProbe, 0)
	for _, item := range leftFiles {
		if _, ok := matchedLeft[item.RelativePath]; ok {
			continue
		}
		unmatchedLeft = append(unmatchedLeft, item)
	}
	unmatchedRight := make([]candidateFileProbe, 0)
	for _, item := range rightFiles {
		if _, ok := matchedRight[item.RelativePath]; ok {
			continue
		}
		unmatchedRight = append(unmatchedRight, item)
	}

	leftFallback := make(map[string][]candidateFileProbe)
	for _, item := range unmatchedLeft {
		key := strings.ToLower(strings.TrimSpace(item.FileName)) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		leftFallback[key] = append(leftFallback[key], item)
	}
	rightFallback := make(map[string][]candidateFileProbe)
	for _, item := range unmatchedRight {
		key := strings.ToLower(strings.TrimSpace(item.FileName)) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		rightFallback[key] = append(rightFallback[key], item)
	}

	fallbackMatchCount := 0
	for key, leftItems := range leftFallback {
		if len(leftItems) != 1 {
			continue
		}
		rightItems := rightFallback[key]
		if len(rightItems) != 1 {
			continue
		}
		left := leftItems[0]
		right := rightItems[0]
		pairs = append(pairs, comparePair{left: left, right: right})
		matchedLeft[left.RelativePath] = struct{}{}
		matchedRight[right.RelativePath] = struct{}{}
		fallbackMatchCount++
	}

	missingLeft := make([]string, 0)
	for _, item := range leftFiles {
		if _, ok := matchedLeft[item.RelativePath]; ok {
			continue
		}
		missingLeft = append(missingLeft, item.RelativePath)
	}
	sort.Strings(missingLeft)
	missingRight := make([]string, 0)
	for _, item := range rightFiles {
		if _, ok := matchedRight[item.RelativePath]; ok {
			continue
		}
		missingRight = append(missingRight, item.RelativePath)
	}
	sort.Strings(missingRight)

	result := make([]struct {
		left  candidateFileProbe
		right candidateFileProbe
	}, 0, len(pairs))
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].left.RelativePath == pairs[j].left.RelativePath {
			return pairs[i].right.RelativePath < pairs[j].right.RelativePath
		}
		return pairs[i].left.RelativePath < pairs[j].left.RelativePath
	})
	for _, pair := range pairs {
		result = append(result, struct {
			left  candidateFileProbe
			right candidateFileProbe
		}{
			left:  pair.left,
			right: pair.right,
		})
	}

	return result, exactMatchCount, fallbackMatchCount, missingLeft, missingRight
}

func pairDirectoryFilesByNameAndSize(
	leftFiles []candidateFileProbe,
	rightFiles []candidateFileProbe,
) ([]struct {
	left  candidateFileProbe
	right candidateFileProbe
}, int, int, []string, []string) {
	type comparePair struct {
		left  candidateFileProbe
		right candidateFileProbe
	}

	leftByNameSize := make(map[string][]candidateFileProbe)
	for _, item := range leftFiles {
		key := strings.ToLower(strings.TrimSpace(item.FileName)) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		leftByNameSize[key] = append(leftByNameSize[key], item)
	}
	rightByNameSize := make(map[string][]candidateFileProbe)
	for _, item := range rightFiles {
		key := strings.ToLower(strings.TrimSpace(item.FileName)) + "|" + strconv.FormatInt(item.SizeBytes, 10)
		rightByNameSize[key] = append(rightByNameSize[key], item)
	}

	keys := make(map[string]struct{}, len(leftByNameSize)+len(rightByNameSize))
	for key := range leftByNameSize {
		keys[key] = struct{}{}
	}
	for key := range rightByNameSize {
		keys[key] = struct{}{}
	}

	pairs := make([]comparePair, 0)
	missingLeft := make([]string, 0)
	missingRight := make([]string, 0)
	fallbackMatchCount := 0
	for key := range keys {
		leftItems := leftByNameSize[key]
		rightItems := rightByNameSize[key]
		sort.Slice(leftItems, func(i, j int) bool {
			return leftItems[i].RelativePath < leftItems[j].RelativePath
		})
		sort.Slice(rightItems, func(i, j int) bool {
			return rightItems[i].RelativePath < rightItems[j].RelativePath
		})
		limit := minInt(len(leftItems), len(rightItems))
		for i := 0; i < limit; i++ {
			pairs = append(pairs, comparePair{
				left:  leftItems[i],
				right: rightItems[i],
			})
			fallbackMatchCount++
		}
		for i := limit; i < len(leftItems); i++ {
			missingLeft = append(missingLeft, leftItems[i].RelativePath)
		}
		for i := limit; i < len(rightItems); i++ {
			missingRight = append(missingRight, rightItems[i].RelativePath)
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].left.FileName == pairs[j].left.FileName {
			return pairs[i].left.RelativePath < pairs[j].left.RelativePath
		}
		return pairs[i].left.FileName < pairs[j].left.FileName
	})
	sort.Strings(missingLeft)
	sort.Strings(missingRight)

	result := make([]struct {
		left  candidateFileProbe
		right candidateFileProbe
	}, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, struct {
			left  candidateFileProbe
			right candidateFileProbe
		}{
			left:  pair.left,
			right: pair.right,
		})
	}

	return result, 0, fallbackMatchCount, missingLeft, missingRight
}

func (s *Service) buildFileRedundancyHit(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	scanID string,
	workspace resolvedWorkspace,
	entry model.StorageIndexEntry,
	publicEntry model.StorageIndexEntry,
	matchKey string,
) (*model.StorageIndexRedundancyHit, bool, error) {
	baseHit := &model.StorageIndexRedundancyHit{
		WorkspaceType:      workspace.WorkspaceType,
		WorkspaceName:      workspace.WorkspaceName,
		ScanID:             scanID,
		TargetType:         model.StorageIndexRedundancyTargetTypeFile,
		TargetPath:         entry.LogicalPath,
		PublicPath:         publicEntry.LogicalPath,
		MatchKey:           matchKey,
		EstimatedBytes:     entry.SizeBytes,
		VerificationStatus: model.StorageIndexVerificationStatusSuspected,
		VerificationMode:   verificationModeMetadata,
		Evidence:           "文件名与文件大小匹配公共空间基线，疑似重复保存的公共资源文件",
		Confidence:         redundancyConfidenceMedium,
	}

	targetHash, publicHash, err := s.computeCandidateHashes(toolboxPod, prefixConfig, entry.LogicalPath, publicEntry.LogicalPath)
	if err != nil {
		baseHit.VerificationMode = hashAlgorithmSHA256
		baseHit.HashAlgorithm = hashAlgorithmSHA256
		baseHit.Evidence = "文件名与文件大小匹配公共空间基线；已尝试进行 SHA256 校验，但校验过程失败，当前仍为疑似命中"
		return baseHit, false, err
	}

	if targetHash != publicHash {
		return nil, false, nil
	}

	baseHit.VerificationStatus = model.StorageIndexVerificationStatusVerified
	baseHit.VerificationMode = hashAlgorithmSHA256
	baseHit.HashAlgorithm = hashAlgorithmSHA256
	baseHit.TargetHash = targetHash
	baseHit.PublicHash = publicHash
	baseHit.Evidence = "文件名与文件大小匹配公共空间基线，且 SHA256 校验一致，确认与公共空间资源重复"
	baseHit.Confidence = redundancyConfidenceHigh

	return baseHit, true, nil
}

func (s *Service) computeCandidateHashes(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	targetLogicalPath string,
	publicLogicalPath string,
) (string, string, error) {
	targetHash, err := s.computeFileHash(toolboxPod, prefixConfig, targetLogicalPath)
	if err != nil {
		return "", "", err
	}
	publicHash, err := s.computeFileHash(toolboxPod, prefixConfig, publicLogicalPath)
	if err != nil {
		return "", "", err
	}
	return targetHash, publicHash, nil
}

func (s *Service) computeFileHash(
	toolboxPod *corev1.Pod,
	prefixConfig ceph.StoragePrefixConfig,
	logicalPath string,
) (string, error) {
	actualPath, err := ceph.ResolveCephFSPath(
		s.kubeClient,
		s.kubeConfig,
		toolboxNamespace,
		logicalPath,
		prefixConfig,
	)
	if err != nil {
		return "", fmt.Errorf("resolve logical path %s failed: %w", logicalPath, err)
	}

	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sha256sum", actualPath},
	)
	if err != nil {
		return "", fmt.Errorf("sha256sum %s failed: %w", logicalPath, err)
	}

	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return "", fmt.Errorf("empty sha256sum output for %s", logicalPath)
	}
	return fields[0], nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func parseEntryType(value string) model.StorageIndexEntryType {
	switch strings.TrimSpace(value) {
	case "f":
		return model.StorageIndexEntryTypeFile
	case "d":
		return model.StorageIndexEntryTypeDir
	case "l":
		return model.StorageIndexEntryTypeSymlink
	default:
		return model.StorageIndexEntryTypeOther
	}
}

func parseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseUnixTimestamp(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	floatValue, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	seconds := int64(floatValue)
	nanos := int64((floatValue - float64(seconds)) * float64(time.Second))
	timestamp := time.Unix(seconds, nanos).UTC().Truncate(time.Microsecond)
	return &timestamp
}

func formatTimeForLog(value *time.Time) string {
	if value == nil {
		return "nil"
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func normalizeUnixPath(value string) string {
	cleaned := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if cleaned == "" {
		return ""
	}
	return path.Clean(cleaned)
}

func normalizeCompareLogicalPath(value string) string {
	cleaned := normalizeUnixPath(value)
	if cleaned == "" {
		return ""
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	switch {
	case strings.HasPrefix(cleaned, "/admin-user/"):
		cleaned = "/user/" + strings.TrimPrefix(cleaned, "/admin-user/")
	case cleaned == "/admin-user":
		cleaned = "/user"
	case strings.HasPrefix(cleaned, "/admin-account/"):
		cleaned = "/account/" + strings.TrimPrefix(cleaned, "/admin-account/")
	case cleaned == "/admin-account":
		cleaned = "/account"
	case strings.HasPrefix(cleaned, "/admin-public/"):
		cleaned = "/public/" + strings.TrimPrefix(cleaned, "/admin-public/")
	case cleaned == "/admin-public":
		cleaned = "/public"
	}
	return normalizeUnixPath(cleaned)
}

func normalizeDirectoryCompareType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return "auto"
	case "dataset", "dataset_dir":
		return "dataset"
	default:
		return "model"
	}
}

func normalizeDirectoryCompareMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", compareModeOptimized:
		return compareModeOptimized
	case "baseline", "full-hash", compareModeFullHash:
		return compareModeFullHash
	default:
		return compareModeOptimized
	}
}

func inferDirectoryCompareTypeFromFiles(leftFiles []candidateFileProbe, rightFiles []candidateFileProbe) string {
	allFiles := append([]candidateFileProbe{}, leftFiles...)
	allFiles = append(allFiles, rightFiles...)
	for _, file := range allFiles {
		lower := strings.ToLower(strings.TrimSpace(file.FileName))
		if isSafeTensorsFile(lower) ||
			strings.HasSuffix(lower, ".bin") ||
			strings.HasSuffix(lower, ".pt") ||
			strings.HasSuffix(lower, ".pth") ||
			strings.HasSuffix(lower, ".ckpt") ||
			strings.HasSuffix(lower, ".gguf") ||
			strings.HasSuffix(lower, ".index.json") ||
			lower == "config.json" ||
			lower == "tokenizer.json" ||
			lower == "tokenizer.model" ||
			lower == "tokenizer_config.json" ||
			lower == "generation_config.json" ||
			lower == "special_tokens_map.json" ||
			lower == "processor_config.json" ||
			lower == "preprocessor_config.json" ||
			lower == "adapter_config.json" ||
			lower == "model_index.json" {
			return "model"
		}
	}
	return "dataset"
}

func (s *Service) scanFilesForDirectoryCompare(
	toolboxPod *corev1.Pod,
	leftActual string,
	rightActual string,
	compareType string,
) (string, []candidateFileProbe, []candidateFileProbe, error) {
	normalizedType := normalizeDirectoryCompareType(compareType)
	if normalizedType == "model" {
		leftFiles, err := s.scanModelComparableFiles(toolboxPod, leftActual)
		if err != nil {
			return "", nil, nil, fmt.Errorf("scan left model files failed: %w", err)
		}
		rightFiles, err := s.scanModelComparableFiles(toolboxPod, rightActual)
		if err != nil {
			return "", nil, nil, fmt.Errorf("scan right model files failed: %w", err)
		}
		return normalizedType, leftFiles, rightFiles, nil
	}
	if normalizedType == "dataset" {
		leftFiles, err := s.scanDatasetComparableFiles(toolboxPod, leftActual)
		if err != nil {
			return "", nil, nil, fmt.Errorf("scan left dataset files failed: %w", err)
		}
		rightFiles, err := s.scanDatasetComparableFiles(toolboxPod, rightActual)
		if err != nil {
			return "", nil, nil, fmt.Errorf("scan right dataset files failed: %w", err)
		}
		return normalizedType, leftFiles, rightFiles, nil
	}

	leftModelFiles, err := s.scanModelComparableFiles(toolboxPod, leftActual)
	if err != nil {
		return "", nil, nil, fmt.Errorf("scan left model probe files failed: %w", err)
	}
	rightModelFiles, err := s.scanModelComparableFiles(toolboxPod, rightActual)
	if err != nil {
		return "", nil, nil, fmt.Errorf("scan right model probe files failed: %w", err)
	}
	inferredType := inferDirectoryCompareTypeFromFiles(leftModelFiles, rightModelFiles)
	if inferredType == "model" {
		return inferredType, leftModelFiles, rightModelFiles, nil
	}

	leftFiles, err := s.scanDatasetComparableFiles(toolboxPod, leftActual)
	if err != nil {
		return "", nil, nil, fmt.Errorf("scan left dataset files failed: %w", err)
	}
	rightFiles, err := s.scanDatasetComparableFiles(toolboxPod, rightActual)
	if err != nil {
		return "", nil, nil, fmt.Errorf("scan right dataset files failed: %w", err)
	}
	return inferredType, leftFiles, rightFiles, nil
}

func relativeUnixPath(rootPath, fullPath string) string {
	root := strings.TrimSuffix(normalizeUnixPath(rootPath), "/")
	full := normalizeUnixPath(fullPath)
	if full == root {
		return "."
	}
	return strings.TrimPrefix(full, root+"/")
}

func relativeLogicalPath(rootPath, fullPath string) string {
	return relativeUnixPath(rootPath, fullPath)
}

func isTopLevelRelative(relativePath string) bool {
	if relativePath == "" || relativePath == "." {
		return false
	}
	return !strings.Contains(relativePath, "/")
}

func depthFromRelative(relativePath string) int {
	if relativePath == "" || relativePath == "." {
		return 0
	}
	return len(strings.Split(relativePath, "/"))
}

func depthFromRoot(targetPath, rootPath string) int {
	if normalizeUnixPath(targetPath) == normalizeUnixPath(rootPath) {
		return 0
	}
	relative := strings.TrimPrefix(normalizeUnixPath(targetPath), strings.TrimSuffix(normalizeUnixPath(rootPath), "/")+"/")
	return depthFromRelative(relative)
}

func isTopLevelPath(targetPath, rootPath string) bool {
	return depthFromRoot(targetPath, rootPath) == 1
}

func parentForDirectory(targetPath, rootPath string) string {
	cleanedTarget := normalizeUnixPath(targetPath)
	cleanedRoot := normalizeUnixPath(rootPath)
	if cleanedTarget == cleanedRoot {
		return ""
	}
	parentPath := path.Dir(cleanedTarget)
	if parentPath == "." {
		return cleanedRoot
	}
	return parentPath
}

func ancestorDirectories(targetPath, rootPath string) []string {
	current := normalizeUnixPath(targetPath)
	root := normalizeUnixPath(rootPath)
	if current == "" || root == "" {
		return nil
	}

	result := make([]string, 0)
	for {
		result = append(result, current)
		if current == root {
			break
		}
		next := path.Dir(current)
		if next == current || next == "." || next == "/" {
			break
		}
		current = next
	}

	return result
}

func collapsePathPrefixes(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	sorted := append([]string{}, paths...)
	sort.Slice(sorted, func(i, j int) bool {
		depthI := strings.Count(strings.TrimPrefix(normalizeUnixPath(sorted[i]), "/"), "/")
		depthJ := strings.Count(strings.TrimPrefix(normalizeUnixPath(sorted[j]), "/"), "/")
		if depthI == depthJ {
			return sorted[i] < sorted[j]
		}
		return depthI < depthJ
	})

	collapsed := make([]string, 0, len(sorted))
	for _, item := range sorted {
		cleaned := normalizeUnixPath(item)
		if cleaned == "" || isCoveredByPrefixes(cleaned, collapsed) {
			continue
		}
		collapsed = append(collapsed, cleaned)
	}
	return collapsed
}

func pathOverlaps(leftPath, rightPath string) bool {
	left := normalizeUnixPath(leftPath)
	right := normalizeUnixPath(rightPath)
	if left == "" || right == "" {
		return false
	}
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func redundancyDirectoryKey(name string, size int64) string {
	return strings.ToLower(strings.TrimSpace(name)) + "|" + strconv.FormatInt(size, 10)
}

func redundancyFileKey(name string, size int64) string {
	return strings.ToLower(strings.TrimSpace(name)) + "|" + strconv.FormatInt(size, 10)
}

func coveredByDirectoryHit(targetPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if targetPath == prefix || strings.HasPrefix(targetPath, prefix+"/") {
			return true
		}
	}
	return false
}

func buildDirectorySignature(metric *model.StorageIndexDirectoryMetric, childDirs []string) string {
	mtime := "0"
	if metric.LatestModifiedAt != nil {
		mtime = metric.LatestModifiedAt.UTC().Format(time.RFC3339)
	}
	childSet := strings.Join(childDirs, ",")
	raw := fmt.Sprintf(
		"%s|%d|%d|%d|%s|%s",
		strings.ToLower(metric.Name),
		metric.TotalSizeBytes,
		metric.ImmediateChildDirCount,
		metric.ImmediateChildFileCount,
		mtime,
		childSet,
	)
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func hashString(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func shouldApplyTopLevelModelCopyPrefilter(workspace resolvedWorkspace) bool {
	return workspace.WorkspaceType == model.StorageIndexWorkspaceTypeUser
}

func shouldSkipTopLevelModelCopyName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	_, blocked := definitelyNotPublicModelCopyTopLevelDirSet[lower]
	return blocked
}

func shouldSkipTopLevelModelCopySignature(sig topLevelSignature) bool {
	if sig.EntryType != model.StorageIndexEntryTypeDir {
		return false
	}
	return shouldSkipTopLevelModelCopyName(sig.Name)
}

func filterTopLevelSignaturesForModelCopyScan(
	signatures map[string]topLevelSignature,
) ([]topLevelSignature, []topLevelSignature) {
	selected := make([]topLevelSignature, 0, len(signatures))
	skipped := make([]topLevelSignature, 0)
	for _, sig := range signatures {
		if shouldSkipTopLevelModelCopySignature(sig) {
			skipped = append(skipped, sig)
			continue
		}
		selected = append(selected, sig)
	}
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].LogicalPath < selected[j].LogicalPath
	})
	sort.Slice(skipped, func(i, j int) bool {
		return skipped[i].LogicalPath < skipped[j].LogicalPath
	})
	return selected, skipped
}

func shouldSkipTopLevelModelCopyCandidate(metric model.StorageIndexDirectoryMetric) bool {
	if !metric.IsTopLevel {
		return false
	}
	return shouldSkipTopLevelModelCopyName(metric.Name)
}

func immediateSubtreeAllowListForTopLevel(name string) (map[string]struct{}, bool) {
	allowList, ok := selectiveTopLevelSubtreeAllowLists[strings.ToLower(strings.TrimSpace(name))]
	return allowList, ok
}

func filterImmediateSubtreesByAllowList(
	signatures map[string]topLevelSignature,
	allowList map[string]struct{},
) ([]topLevelSignature, []topLevelSignature) {
	selected := make([]topLevelSignature, 0, len(signatures))
	skipped := make([]topLevelSignature, 0)
	for _, sig := range signatures {
		if sig.EntryType != model.StorageIndexEntryTypeDir {
			continue
		}
		if _, ok := allowList[strings.ToLower(strings.TrimSpace(sig.Name))]; ok {
			selected = append(selected, sig)
			continue
		}
		skipped = append(skipped, sig)
	}
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].LogicalPath < selected[j].LogicalPath
	})
	sort.Slice(skipped, func(i, j int) bool {
		return skipped[i].LogicalPath < skipped[j].LogicalPath
	})
	return selected, skipped
}

func classifyDirectory(targetPath string) string {
	lower := strings.ToLower(targetPath)
	switch {
	case strings.Contains(lower, "models"),
		strings.Contains(lower, "model"),
		strings.Contains(lower, "checkpoint"),
		strings.Contains(lower, "checkpoints"),
		strings.Contains(lower, "huggingface"),
		strings.Contains(lower, "transformers"):
		return "model_dir"
	case strings.Contains(lower, "datasets"),
		strings.Contains(lower, "dataset"),
		strings.Contains(lower, "/data"):
		return "dataset_dir"
	default:
		return ""
	}
}

func computeCandidateScore(metric *model.StorageIndexDirectoryMetric) float64 {
	score := 0.0
	if metric.CategoryHint != "" {
		score += 10
	}
	if metric.TotalSizeBytes > 0 {
		score += float64(metric.TotalSizeBytes) / float64(1024*1024*1024)
	}
	if metric.ImmediateChildDirCount > 0 {
		score += float64(metric.ImmediateChildDirCount) * 0.5
	}
	return score
}

func inferCandidateTypeFromPublicPath(publicPath string) string {
	return classifyDirectory(publicPath)
}

func datasetTypeToCandidateType(dataType model.DataType) string {
	switch dataType {
	case model.DataTypeModel:
		return "model_dir"
	case model.DataTypeDataset:
		return "dataset_dir"
	default:
		return ""
	}
}

func logicalPublicPathFromDatasetURL(prefixConfig ceph.StoragePrefixConfig, rawURL string) string {
	normalized := normalizeUnixPath(rawURL)
	publicPrefix := normalizeUnixPath(prefixConfig.Public)
	publicRoot := normalizeUnixPath("/public")

	if normalized == publicPrefix {
		return publicRoot
	}
	if strings.HasPrefix(normalized, publicPrefix+"/") {
		return normalizeUnixPath(publicRoot + strings.TrimPrefix(normalized, publicPrefix))
	}
	if strings.HasPrefix(normalized, "/"+publicPrefix+"/") {
		return normalizeUnixPath(publicRoot + strings.TrimPrefix(normalized, "/"+publicPrefix))
	}
	if strings.HasPrefix(normalized, "/public/") || normalized == "/public" {
		return normalized
	}
	return ""
}

func (s *Service) listPublicResourceRoots(
	ctx context.Context,
	prefixConfig ceph.StoragePrefixConfig,
) ([]publicResourceRoot, error) {
	var datasets []model.Dataset
	if err := query.GetDB().WithContext(ctx).
		Where("type IN ? AND deleted_at IS NULL", []model.DataType{model.DataTypeModel, model.DataTypeDataset}).
		Find(&datasets).Error; err != nil {
		return nil, fmt.Errorf("query public resource registry failed: %w", err)
	}

	roots := make([]publicResourceRoot, 0)
	seen := make(map[string]struct{})
	for _, dataset := range datasets {
		logicalPath := logicalPublicPathFromDatasetURL(prefixConfig, dataset.URL)
		if logicalPath == "" || logicalPath == "/public" {
			continue
		}
		if _, ok := seen[logicalPath]; ok {
			continue
		}
		seen[logicalPath] = struct{}{}
		roots = append(roots, publicResourceRoot{
			Name:        loCoalesce(strings.TrimSpace(dataset.Name), path.Base(logicalPath)),
			LogicalPath: logicalPath,
			Category:    datasetTypeToCandidateType(dataset.Type),
		})
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].LogicalPath < roots[j].LogicalPath
	})
	return roots, nil
}

func loCoalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func insertEntriesInChunks(
	tx *gorm.DB,
	scanID string,
	workspace resolvedWorkspace,
	items []model.StorageIndexEntry,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入条目 scan_id=%s workspace_type=%s workspace_name=%s start=%d end=%d total=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertDirectoryMetricsInChunks(
	tx *gorm.DB,
	scanID string,
	workspace resolvedWorkspace,
	items []model.StorageIndexDirectoryMetric,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入目录聚合 scan_id=%s workspace_type=%s workspace_name=%s start=%d end=%d total=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertRedundancyHitsInChunks(
	tx *gorm.DB,
	scanID string,
	workspace resolvedWorkspace,
	items []model.StorageIndexRedundancyHit,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入冗余命中 scan_id=%s workspace_type=%s workspace_name=%s start=%d end=%d total=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertCandidatesInChunks(
	tx *gorm.DB,
	scanID string,
	workspace resolvedWorkspace,
	items []model.StorageIndexCandidate,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入候选目录 scan_id=%s workspace_type=%s workspace_name=%s start=%d end=%d total=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertCandidateFilesInChunks(
	tx *gorm.DB,
	scanID string,
	workspace resolvedWorkspace,
	items []model.StorageIndexCandidateFile,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入候选关键文件 scan_id=%s workspace_type=%s workspace_name=%s start=%d end=%d total=%d",
			scanID, workspace.WorkspaceType, workspace.WorkspaceName, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertPublicBaselineFilesInChunks(
	tx *gorm.DB,
	scanID string,
	items []model.StorageIndexPublicFileBaseline,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入公共关键文件基线 scan_id=%s start=%d end=%d total=%d",
			scanID, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertPublicRootBaselinesInChunks(
	tx *gorm.DB,
	scanID string,
	items []model.StorageIndexPublicRootBaseline,
	chunkSize int,
) error {
	return insertInChunks(tx, items, chunkSize, func(start, end int) error {
		klog.Infof(
			"storageindex: 分批写入公共资源根基线 scan_id=%s start=%d end=%d total=%d",
			scanID, start, end, len(items),
		)
		batch := items[start:end]
		return tx.Create(&batch).Error
	})
}

func insertInChunks[T any](_ *gorm.DB, items []T, chunkSize int, insertFn func(start, end int) error) error {
	if chunkSize <= 0 {
		chunkSize = insertBatchSize
	}
	for start := 0; start < len(items); start += chunkSize {
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}
		if err := insertFn(start, end); err != nil {
			return err
		}
	}
	return nil
}

type findProgressWriter struct {
	scanID        string
	workspaceType model.StorageIndexWorkspaceType
	workspaceName string
	everyRecords  int64
	recordCount   int64
	nextLogAt     int64
}

func (w *findProgressWriter) Write(p []byte) (int, error) {
	if w.everyRecords <= 0 {
		w.everyRecords = scanProgressLogEveryRecord
	}
	if w.nextLogAt == 0 {
		w.nextLogAt = w.everyRecords
	}

	count := int64(bytes.Count(p, []byte(findRecordSeparator)))
	if count == 0 {
		return len(p), nil
	}

	w.recordCount += count
	for w.recordCount >= w.nextLogAt {
		klog.Infof(
			"storageindex: find 流式扫描进度 scan_id=%s workspace_type=%s workspace_name=%s discovered_records=%d",
			w.scanID, w.workspaceType, w.workspaceName, w.recordCount,
		)
		w.nextLogAt += w.everyRecords
	}

	return len(p), nil
}
