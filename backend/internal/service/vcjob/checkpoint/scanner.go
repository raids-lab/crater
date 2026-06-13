package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/storage"
)

const unknownCheckpointStep int64 = -1

var stepPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^checkpoint[-_](\d+)(?:$|[-_.])`),
	regexp.MustCompile(`(?i)^global_step[_-]?(\d+)(?:$|[-_.])`),
	regexp.MustCompile(`(?i)(?:^|[-_])step[-_]?(\d+)(?:$|[-_.])`),
}

type ScanResult struct {
	Items          []model.JobCheckpoint `json:"items"`
	Latest         *model.JobCheckpoint  `json:"latest,omitempty"`
	TotalSizeBytes int64                 `json:"totalSizeBytes"`
	ScannedAt      time.Time             `json:"scannedAt"`
	StoragePath    string                `json:"storagePath"`
}

func ScanJob(ctx context.Context, record *model.Job) (*ScanResult, error) {
	info, storagePath, err := prepareScan(record)
	if err != nil {
		return nil, err
	}

	root, err := storage.StatRelativePath(ctx, storagePath)
	if err != nil {
		return nil, fmt.Errorf("checkpoint directory is not accessible: %w", err)
	}

	candidates, err := discoverCheckpoints(ctx, record, info, storagePath, root)
	if err != nil {
		return nil, err
	}

	return finishScan(ctx, record, info, storagePath, candidates)
}

func prepareScan(record *model.Job) (*model.CheckpointInfo, string, error) {
	if record == nil {
		return nil, "", errors.New("job record is required")
	}
	info := jobCheckpointInfo(record)
	if info == nil || !info.Enabled {
		return nil, "", fmt.Errorf("checkpoint is not enabled for job %s", record.JobName)
	}

	storagePath, err := ResolveStoragePath(record, info.CheckpointDir)
	if err != nil {
		return nil, "", err
	}
	return info, storagePath, nil
}

func finishScan(
	ctx context.Context,
	record *model.Job,
	info *model.CheckpointInfo,
	storagePath string,
	candidates []model.JobCheckpoint,
) (*ScanResult, error) {
	latest := latestCheckpoint(candidates)
	for i := range candidates {
		candidates[i].Latest = latest != nil && candidates[i].Path == latest.Path
	}
	if latest != nil {
		latest.Latest = true
	}
	if err := persistScan(ctx, record, info, candidates, latest); err != nil {
		return nil, err
	}

	totalSize := int64(0)
	for i := range candidates {
		totalSize += candidates[i].SizeBytes
	}
	scannedAt := time.Now()
	result := &ScanResult{
		Items:          candidates,
		Latest:         latest,
		TotalSizeBytes: totalSize,
		ScannedAt:      scannedAt,
		StoragePath:    storagePath,
	}
	return result, nil
}

func ResolveStoragePath(record *model.Job, containerPath string) (string, error) {
	if record == nil || record.Attributes.Data() == nil {
		return "", errors.New("job record has no stored template")
	}
	containerPath = filepath.Clean(strings.TrimSpace(containerPath))
	if containerPath == "." || !filepath.IsAbs(containerPath) {
		return "", fmt.Errorf("checkpoint path %q must be absolute", containerPath)
	}

	bestMountPath, bestSubPath := bestWritableMountForCheckpoint(record, containerPath)
	if bestMountPath == "" {
		return "", fmt.Errorf("checkpoint path %s is not under a writable persistent mount", containerPath)
	}

	rel, err := filepath.Rel(bestMountPath, containerPath)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return filepath.ToSlash(filepath.Clean(bestSubPath)), nil
	}
	return filepath.ToSlash(filepath.Clean(filepath.Join(bestSubPath, rel))), nil
}

func bestWritableMountForCheckpoint(record *model.Job, containerPath string) (bestMountPath, bestSubPath string) {
	tasks := record.Attributes.Data().Spec.Tasks
	for taskIndex := range tasks {
		containers := tasks[taskIndex].Template.Spec.Containers
		for containerIndex := range containers {
			mountPath, subPath := bestWritableMountInContainer(
				containers[containerIndex].VolumeMounts,
				containerPath,
				bestMountPath,
			)
			if mountPath != "" {
				bestMountPath = mountPath
				bestSubPath = subPath
			}
		}
	}
	return bestMountPath, bestSubPath
}

func bestWritableMountInContainer(
	mounts []v1.VolumeMount,
	containerPath, currentBest string,
) (bestMountPath, bestSubPath string) {
	for mountIndex := range mounts {
		mount := &mounts[mountIndex]
		mountPath := filepath.Clean(strings.TrimSpace(mount.MountPath))
		if mountPath == "." || mount.SubPath == "" || mount.ReadOnly {
			continue
		}
		if !isPathUnderOrEqual(containerPath, mountPath) {
			continue
		}
		if len(mountPath) > len(currentBest) {
			currentBest = mountPath
			bestMountPath = mountPath
			bestSubPath = mount.SubPath
		}
	}
	return bestMountPath, bestSubPath
}

func discoverCheckpoints(
	ctx context.Context,
	record *model.Job,
	info *model.CheckpointInfo,
	storagePath string,
	root storage.Files,
) ([]model.JobCheckpoint, error) {
	if !root.IsDir {
		size, modTime, err := scanTree(ctx, storagePath)
		if err != nil {
			return nil, err
		}
		item := newCheckpointRecord(
			record,
			info,
			filepath.Base(storagePath),
			info.CheckpointDir,
			storagePath,
			size,
			modTime,
		)
		return []model.JobCheckpoint{item}, nil
	}

	children, err := storage.ListRelativePath(ctx, storagePath)
	if err != nil {
		return nil, err
	}

	items := make([]model.JobCheckpoint, 0, len(children))
	for _, child := range children {
		if shouldSkipCheckpointChild(child.Name) {
			continue
		}
		if !looksLikeCheckpoint(info.Framework, child) {
			continue
		}
		childStoragePath := filepath.ToSlash(filepath.Join(storagePath, child.Name))
		childContainerPath := filepath.ToSlash(filepath.Join(info.CheckpointDir, child.Name))
		size, modTime, err := scanTree(ctx, childStoragePath)
		if err != nil {
			return nil, err
		}
		if modTime.IsZero() {
			modTime = child.ModifyTime
		}
		items = append(items, newCheckpointRecord(record, info, child.Name, childContainerPath, childStoragePath, size, modTime))
	}
	return items, nil
}

func shouldSkipCheckpointChild(name string) bool {
	return name == "" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_tmp")
}

func looksLikeCheckpoint(framework string, file storage.Files) bool {
	if stepFromName(file.Name) >= 0 {
		return true
	}
	switch strings.ToLower(framework) {
	case FrameworkPytorch, FrameworkLightning:
		return !file.IsDir && isCheckpointFileName(file.Name)
	case FrameworkCustom:
		return file.IsDir || isCheckpointFileName(file.Name)
	default:
		return file.IsDir
	}
}

func isCheckpointFileName(name string) bool {
	return strings.HasSuffix(name, ".pt") ||
		strings.HasSuffix(name, ".pth") ||
		strings.HasSuffix(name, ".ckpt")
}

func scanTree(ctx context.Context, root string) (int64, time.Time, error) {
	stat, err := storage.StatRelativePath(ctx, root)
	if err != nil {
		return 0, time.Time{}, err
	}
	if !stat.IsDir {
		return stat.Size, stat.ModifyTime, nil
	}

	children, err := storage.ListRelativePath(ctx, root)
	if err != nil {
		return 0, time.Time{}, err
	}
	size := int64(0)
	modTime := stat.ModifyTime
	for _, child := range children {
		childPath := filepath.ToSlash(filepath.Join(root, child.Name))
		childSize, childModTime, err := scanTree(ctx, childPath)
		if err != nil {
			return 0, time.Time{}, err
		}
		size += childSize
		if childModTime.After(modTime) {
			modTime = childModTime
		}
	}
	return size, modTime, nil
}

func newCheckpointRecord(
	record *model.Job,
	info *model.CheckpointInfo,
	name string,
	path string,
	storagePath string,
	size int64,
	modTime time.Time,
) model.JobCheckpoint {
	return model.JobCheckpoint{
		JobID:       record.ID,
		JobName:     record.JobName,
		UserID:      record.UserID,
		AccountID:   record.AccountID,
		Framework:   info.Framework,
		Name:        name,
		Path:        filepath.ToSlash(filepath.Clean(path)),
		StoragePath: filepath.ToSlash(filepath.Clean(storagePath)),
		Step:        stepFromName(name),
		SizeBytes:   size,
		ModTime:     modTime,
		Status:      model.JobCheckpointStatusReady,
		Source:      "scan",
		Metadata: datatypes.JSONMap{
			"checkpointDir": info.CheckpointDir,
		},
	}
}

func stepFromName(name string) int64 {
	for _, pattern := range stepPatterns {
		matches := pattern.FindStringSubmatch(name)
		if len(matches) < 2 {
			continue
		}
		step, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil {
			return step
		}
	}
	return unknownCheckpointStep
}

func latestCheckpoint(items []model.JobCheckpoint) *model.JobCheckpoint {
	if len(items) == 0 {
		return nil
	}
	sorted := append([]model.JobCheckpoint(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Step >= 0 && sorted[j].Step >= 0 && sorted[i].Step != sorted[j].Step {
			return sorted[i].Step > sorted[j].Step
		}
		if sorted[i].Step >= 0 && sorted[j].Step < 0 {
			return true
		}
		if sorted[i].Step < 0 && sorted[j].Step >= 0 {
			return false
		}
		if !sorted[i].ModTime.Equal(sorted[j].ModTime) {
			return sorted[i].ModTime.After(sorted[j].ModTime)
		}
		return sorted[i].Name > sorted[j].Name
	})
	return &sorted[0]
}

func persistScan(
	ctx context.Context,
	record *model.Job,
	info *model.CheckpointInfo,
	items []model.JobCheckpoint,
	latest *model.JobCheckpoint,
) error {
	db := query.GetDB().WithContext(ctx)
	now := time.Now()
	seenPaths := make([]string, 0, len(items))
	for i := range items {
		seenPaths = append(seenPaths, items[i].Path)
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "job_id"}, {Name: "path"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"job_name",
				"user_id",
				"account_id",
				"framework",
				"name",
				"storage_path",
				"step",
				"size_bytes",
				"mod_time",
				"status",
				"latest",
				"source",
				"metadata",
				"updated_at",
			}),
		}).Create(&items[i]).Error; err != nil {
			return err
		}
	}

	missingQuery := db.Model(&model.JobCheckpoint{}).Where("job_id = ? AND status = ?", record.ID, model.JobCheckpointStatusReady)
	if len(seenPaths) > 0 {
		missingQuery = missingQuery.Where("path NOT IN ?", seenPaths)
	}
	if err := missingQuery.Updates(map[string]any{
		"status":     model.JobCheckpointStatusMissing,
		"latest":     false,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}

	info.LastScannedAt = now
	if latest != nil {
		info.LatestCheckpoint = latest.Path
	} else {
		info.LatestCheckpoint = ""
	}
	record.Checkpoint = ptrToJSON(info)
	return db.Model(&model.Job{}).Where("id = ?", record.ID).Update("checkpoint", datatypes.NewJSONType(info)).Error
}

func jobCheckpointInfo(record *model.Job) *model.CheckpointInfo {
	if record == nil || record.Checkpoint == nil {
		return nil
	}
	return record.Checkpoint.Data()
}

func ptrToJSON(info *model.CheckpointInfo) *datatypes.JSONType[*model.CheckpointInfo] {
	value := datatypes.NewJSONType(info)
	return &value
}
