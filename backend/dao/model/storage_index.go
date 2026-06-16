//nolint:lll // GORM tags encode storage index schema metadata inline with each field.
package model

import "time"

type StorageIndexWorkspaceType string

const (
	StorageIndexWorkspaceTypeUser    StorageIndexWorkspaceType = "user"
	StorageIndexWorkspaceTypeAccount StorageIndexWorkspaceType = "account"
	StorageIndexWorkspaceTypePublic  StorageIndexWorkspaceType = "public"
)

type StorageIndexScanStatus string

const (
	StorageIndexScanStatusPending StorageIndexScanStatus = "pending"
	StorageIndexScanStatusRunning StorageIndexScanStatus = "running"
	StorageIndexScanStatusDone    StorageIndexScanStatus = "done"
	StorageIndexScanStatusError   StorageIndexScanStatus = "error"
)

type StorageIndexScanMode string

const (
	StorageIndexScanModeFull         StorageIndexScanMode = "full"
	StorageIndexScanModeDailyRefresh StorageIndexScanMode = "daily_refresh"
)

type StorageIndexEntryType string

const (
	StorageIndexEntryTypeFile    StorageIndexEntryType = "file"
	StorageIndexEntryTypeDir     StorageIndexEntryType = "dir"
	StorageIndexEntryTypeSymlink StorageIndexEntryType = "symlink"
	StorageIndexEntryTypeOther   StorageIndexEntryType = "other"
)

type StorageIndexRedundancyTargetType string

const (
	StorageIndexRedundancyTargetTypeFile      StorageIndexRedundancyTargetType = "file"
	StorageIndexRedundancyTargetTypeDirectory StorageIndexRedundancyTargetType = "directory"
)

type StorageIndexVerificationStatus string

const (
	StorageIndexVerificationStatusSuspected StorageIndexVerificationStatus = "suspected"
	StorageIndexVerificationStatusVerified  StorageIndexVerificationStatus = "verified"
)

// StorageIndexScanJob records a metadata indexing run for a workspace.
type StorageIndexScanJob struct {
	ID                       uint                      `gorm:"primaryKey" json:"id"`
	ScanID                   string                    `gorm:"type:varchar(64);not null;uniqueIndex;comment:扫描任务ID" json:"scanId"`
	WorkspaceType            StorageIndexWorkspaceType `gorm:"type:varchar(16);not null;index;comment:工作空间类型" json:"workspaceType"`
	WorkspaceName            string                    `gorm:"type:varchar(128);not null;index;comment:工作空间名称" json:"workspaceName"`
	LogicalPath              string                    `gorm:"type:text;not null;comment:逻辑扫描路径" json:"logicalPath"`
	SnapshotName             string                    `gorm:"type:varchar(128);comment:扫描使用的快照名称" json:"snapshotName"`
	MaterializedSnapshotName string                    `gorm:"type:varchar(160);comment:实际物化快照目录名称" json:"materializedSnapshotName"`
	ScanRoot                 string                    `gorm:"type:text;comment:实际扫描根路径" json:"scanRoot"`
	TriggerSource            string                    `gorm:"type:varchar(32);not null;default:manual;comment:触发来源" json:"triggerSource"`
	ScanMode                 StorageIndexScanMode      `gorm:"type:varchar(32);not null;default:full;comment:扫描模式" json:"scanMode"`
	BaseScanID               string                    `gorm:"type:varchar(64);index;comment:差异比对基线扫描ID" json:"baseScanId"`
	DiffMethod               string                    `gorm:"type:varchar(32);comment:差异计算方式" json:"diffMethod"`
	Status                   StorageIndexScanStatus    `gorm:"type:varchar(16);not null;index;default:pending;comment:扫描状态" json:"status"`
	EntryCount               int64                     `gorm:"not null;default:0;comment:入库条目数" json:"entryCount"`
	FileCount                int64                     `gorm:"not null;default:0;comment:文件条目数" json:"fileCount"`
	DirectoryCount           int64                     `gorm:"not null;default:0;comment:目录条目数" json:"directoryCount"`
	TotalSizeBytes           int64                     `gorm:"not null;default:0;comment:工作空间总大小" json:"totalSizeBytes"`
	ChangedPathCount         int64                     `gorm:"not null;default:0;comment:与基线相比的变化目录数" json:"changedPathCount"`
	RedundancyCount          int64                     `gorm:"not null;default:0;comment:冗余命中数" json:"redundancyCount"`
	RedundancyBytes          int64                     `gorm:"not null;default:0;comment:冗余空间估算字节数" json:"redundancyBytes"`
	ErrorMessage             string                    `gorm:"type:text;comment:错误信息" json:"errorMessage"`
	StartedAt                *time.Time                `gorm:"comment:开始时间" json:"startedAt"`
	FinishedAt               *time.Time                `gorm:"comment:结束时间" json:"finishedAt"`
	LatencyMs                int64                     `gorm:"not null;default:0;comment:耗时毫秒" json:"latencyMs"`
	CreatedAt                time.Time                 `json:"createdAt"`
	UpdatedAt                time.Time                 `json:"updatedAt"`
}

func (StorageIndexScanJob) TableName() string {
	return "storage_index_scan_jobs"
}

// StorageIndexEntry stores the latest indexed path metadata for a workspace.
type StorageIndexEntry struct {
	ID            uint                      `gorm:"primaryKey" json:"id"`
	WorkspaceType StorageIndexWorkspaceType `gorm:"type:varchar(16);not null;index;comment:工作空间类型" json:"workspaceType"`
	WorkspaceName string                    `gorm:"type:varchar(128);not null;index;comment:工作空间名称" json:"workspaceName"`
	ScanID        string                    `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	LogicalPath   string                    `gorm:"type:text;not null;comment:逻辑路径" json:"logicalPath"`
	RelativePath  string                    `gorm:"type:text;not null;comment:相对路径" json:"relativePath"`
	ParentPath    string                    `gorm:"type:text;comment:父路径" json:"parentPath"`
	Name          string                    `gorm:"type:varchar(512);not null;index;comment:对象名称" json:"name"`
	EntryType     StorageIndexEntryType     `gorm:"type:varchar(16);not null;index;comment:对象类型" json:"entryType"`
	SizeBytes     int64                     `gorm:"not null;default:0;comment:对象大小" json:"sizeBytes"`
	OwnerUID      int64                     `gorm:"not null;default:0;comment:属主UID" json:"ownerUid"`
	OwnerGID      int64                     `gorm:"not null;default:0;comment:属组GID" json:"ownerGid"`
	Mode          string                    `gorm:"type:varchar(16);comment:权限位" json:"mode"`
	LinkCount     int64                     `gorm:"not null;default:0;comment:链接数" json:"linkCount"`
	ModifiedAt    *time.Time                `gorm:"comment:mtime" json:"modifiedAt"`
	ChangedAt     *time.Time                `gorm:"comment:ctime" json:"changedAt"`
	AccessedAt    *time.Time                `gorm:"comment:atime" json:"accessedAt"`
	IsTopLevel    bool                      `gorm:"not null;default:false;index;comment:是否根目录直系子节点" json:"isTopLevel"`
	CreatedAt     time.Time                 `json:"createdAt"`
	UpdatedAt     time.Time                 `json:"updatedAt"`
}

func (StorageIndexEntry) TableName() string {
	return "storage_index_entries"
}

// StorageIndexDirectoryMetric stores aggregated directory metrics for a workspace.
type StorageIndexDirectoryMetric struct {
	ID                      uint                      `gorm:"primaryKey" json:"id"`
	WorkspaceType           StorageIndexWorkspaceType `gorm:"type:varchar(16);not null;index;comment:工作空间类型" json:"workspaceType"`
	WorkspaceName           string                    `gorm:"type:varchar(128);not null;index;comment:工作空间名称" json:"workspaceName"`
	ScanID                  string                    `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	Path                    string                    `gorm:"type:text;not null;comment:目录路径" json:"path"`
	ParentPath              string                    `gorm:"type:text;comment:父目录路径" json:"parentPath"`
	Name                    string                    `gorm:"type:varchar(512);not null;index;comment:目录名" json:"name"`
	Depth                   int                       `gorm:"not null;default:0;comment:目录深度" json:"depth"`
	IsTopLevel              bool                      `gorm:"not null;default:false;index;comment:是否根目录直系子目录" json:"isTopLevel"`
	FileCount               int64                     `gorm:"not null;default:0;comment:子树文件数" json:"fileCount"`
	DirectoryCount          int64                     `gorm:"not null;default:0;comment:子树目录数" json:"directoryCount"`
	TotalSizeBytes          int64                     `gorm:"not null;default:0;comment:子树累计大小" json:"totalSizeBytes"`
	LatestGrowth            int64                     `gorm:"not null;default:0;comment:最近增量字节数" json:"latestGrowth"`
	ImmediateChildDirCount  int64                     `gorm:"not null;default:0;comment:直接子目录数" json:"immediateChildDirCount"`
	ImmediateChildFileCount int64                     `gorm:"not null;default:0;comment:直接子文件数" json:"immediateChildFileCount"`
	LatestModifiedAt        *time.Time                `gorm:"comment:目录最近修改时间" json:"latestModifiedAt"`
	Signature               string                    `gorm:"type:varchar(128);index;comment:目录签名" json:"signature"`
	CategoryHint            string                    `gorm:"type:varchar(64);index;comment:目录类别提示" json:"categoryHint"`
	CandidateScore          float64                   `gorm:"not null;default:0;comment:候选目录评分" json:"candidateScore"`
	CreatedAt               time.Time                 `json:"createdAt"`
	UpdatedAt               time.Time                 `json:"updatedAt"`
}

func (StorageIndexDirectoryMetric) TableName() string {
	return "storage_index_directory_metrics"
}

type StorageIndexCandidateStatus string

const (
	StorageIndexCandidateStatusSuspected StorageIndexCandidateStatus = "suspected"
	StorageIndexCandidateStatusVerified  StorageIndexCandidateStatus = "verified"
	StorageIndexCandidateStatusRejected  StorageIndexCandidateStatus = "rejected"
)

type StorageIndexCandidate struct {
	ID             uint                        `gorm:"primaryKey" json:"id"`
	WorkspaceType  StorageIndexWorkspaceType   `gorm:"type:varchar(16);not null;index;comment:工作空间类型" json:"workspaceType"`
	WorkspaceName  string                      `gorm:"type:varchar(128);not null;index;comment:工作空间名称" json:"workspaceName"`
	ScanID         string                      `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	CandidateType  string                      `gorm:"type:varchar(32);index;comment:候选目录类型" json:"candidateType"`
	TargetPath     string                      `gorm:"type:text;not null;comment:候选目录路径" json:"targetPath"`
	PublicPath     string                      `gorm:"type:text;comment:匹配的公共空间目录路径" json:"publicPath"`
	Evidence       string                      `gorm:"type:text;comment:候选依据" json:"evidence"`
	CandidateScore float64                     `gorm:"not null;default:0;comment:候选评分" json:"candidateScore"`
	Status         StorageIndexCandidateStatus `gorm:"type:varchar(32);not null;default:suspected;index;comment:候选状态" json:"status"`
	CreatedAt      time.Time                   `json:"createdAt"`
	UpdatedAt      time.Time                   `json:"updatedAt"`
}

func (StorageIndexCandidate) TableName() string {
	return "storage_index_candidates"
}

type StorageIndexCandidateFile struct {
	ID                 uint                           `gorm:"primaryKey" json:"id"`
	WorkspaceType      StorageIndexWorkspaceType      `gorm:"type:varchar(16);not null;index;comment:工作空间类型" json:"workspaceType"`
	WorkspaceName      string                         `gorm:"type:varchar(128);not null;index;comment:工作空间名称" json:"workspaceName"`
	ScanID             string                         `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	CandidatePath      string                         `gorm:"type:text;not null;comment:候选目录路径" json:"candidatePath"`
	FilePath           string                         `gorm:"type:text;not null;comment:候选文件路径" json:"filePath"`
	FileName           string                         `gorm:"type:varchar(512);index;comment:候选文件名" json:"fileName"`
	RelativePath       string                         `gorm:"type:text;comment:候选文件相对路径" json:"relativePath"`
	SizeBytes          int64                          `gorm:"not null;default:0;comment:候选文件大小" json:"sizeBytes"`
	MatchedPublicFile  string                         `gorm:"type:text;comment:匹配的公共文件路径" json:"matchedPublicFile"`
	HashAlgorithm      string                         `gorm:"type:varchar(32);comment:哈希算法" json:"hashAlgorithm"`
	FileHash           string                         `gorm:"type:varchar(128);comment:候选文件哈希" json:"fileHash"`
	VerificationStatus StorageIndexVerificationStatus `gorm:"type:varchar(32);not null;default:suspected;index;comment:校验状态" json:"verificationStatus"`
	CreatedAt          time.Time                      `json:"createdAt"`
	UpdatedAt          time.Time                      `json:"updatedAt"`
}

func (StorageIndexCandidateFile) TableName() string {
	return "storage_index_candidate_files"
}

type StorageIndexPublicRootBaseline struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ScanID         string    `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	ResourceName   string    `gorm:"type:varchar(512);not null;index;comment:公共资源名称" json:"resourceName"`
	LogicalPath    string    `gorm:"type:text;not null;comment:公共资源逻辑路径" json:"logicalPath"`
	RootHash       string    `gorm:"type:varchar(128);index;comment:公共资源根目录哈希" json:"rootHash"`
	Category       string    `gorm:"type:varchar(64);index;comment:公共资源类别" json:"category"`
	TotalSizeBytes int64     `gorm:"not null;default:0;comment:资源总大小" json:"totalSizeBytes"`
	KeyFileCount   int64     `gorm:"not null;default:0;comment:关键文件数量" json:"keyFileCount"`
	Signature      string    `gorm:"type:varchar(128);index;comment:资源目录签名" json:"signature"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (StorageIndexPublicRootBaseline) TableName() string {
	return "storage_index_public_root_baseline"
}

type StorageIndexPublicFileBaseline struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ScanID         string    `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	PublicRootPath string    `gorm:"type:text;not null;index;comment:公共基线根目录" json:"publicRootPath"`
	PublicRootHash string    `gorm:"type:varchar(128);index;comment:公共基线根目录哈希" json:"publicRootHash"`
	FilePath       string    `gorm:"type:text;not null;comment:公共文件路径" json:"filePath"`
	FileName       string    `gorm:"type:varchar(512);index;comment:公共文件名" json:"fileName"`
	RelativePath   string    `gorm:"type:text;index;comment:相对公共根目录的路径" json:"relativePath"`
	SizeBytes      int64     `gorm:"not null;default:0;comment:文件大小" json:"sizeBytes"`
	MatchKey       string    `gorm:"type:varchar(1024);index;comment:快速匹配键" json:"matchKey"`
	MatchKeyHash   string    `gorm:"type:varchar(128);index;comment:快速匹配键哈希" json:"matchKeyHash"`
	HashAlgorithm  string    `gorm:"type:varchar(32);comment:哈希算法" json:"hashAlgorithm"`
	FileHash       string    `gorm:"type:varchar(128);comment:文件哈希" json:"fileHash"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (StorageIndexPublicFileBaseline) TableName() string {
	return "storage_index_public_file_baseline"
}

// StorageIndexRedundancyHit stores redundancy findings against the public baseline.
type StorageIndexRedundancyHit struct {
	ID                 uint                             `gorm:"primaryKey" json:"id"`
	WorkspaceType      StorageIndexWorkspaceType        `gorm:"type:varchar(16);not null;index;comment:工作空间类型" json:"workspaceType"`
	WorkspaceName      string                           `gorm:"type:varchar(128);not null;index;comment:工作空间名称" json:"workspaceName"`
	ScanID             string                           `gorm:"type:varchar(64);not null;index;comment:来源扫描任务ID" json:"scanId"`
	TargetType         StorageIndexRedundancyTargetType `gorm:"type:varchar(16);not null;index;comment:冗余对象类型" json:"targetType"`
	TargetPath         string                           `gorm:"type:text;not null;comment:工作空间中的冗余路径" json:"targetPath"`
	PublicPath         string                           `gorm:"type:text;not null;comment:公共空间基线路径" json:"publicPath"`
	MatchKey           string                           `gorm:"type:varchar(512);index;comment:匹配键" json:"matchKey"`
	Evidence           string                           `gorm:"type:text;comment:检测依据" json:"evidence"`
	Confidence         string                           `gorm:"type:varchar(32);comment:置信级别" json:"confidence"`
	VerificationStatus StorageIndexVerificationStatus   `gorm:"type:varchar(32);not null;default:suspected;index;comment:校验状态" json:"verificationStatus"`
	VerificationMode   string                           `gorm:"type:varchar(32);comment:校验方式" json:"verificationMode"`
	HashAlgorithm      string                           `gorm:"type:varchar(32);comment:哈希算法" json:"hashAlgorithm"`
	TargetHash         string                           `gorm:"type:varchar(128);comment:工作空间对象哈希" json:"targetHash"`
	PublicHash         string                           `gorm:"type:varchar(128);comment:公共空间对象哈希" json:"publicHash"`
	EstimatedBytes     int64                            `gorm:"not null;default:0;comment:估算冗余空间大小" json:"estimatedBytes"`
	CreatedAt          time.Time                        `json:"createdAt"`
	UpdatedAt          time.Time                        `json:"updatedAt"`
}

func (StorageIndexRedundancyHit) TableName() string {
	return "storage_index_redundancy_hits"
}
