//nolint:gocritic,lll // Experiment helpers assemble shell probes and aggregate directory signatures.
package storageindex

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/ceph"
	"github.com/raids-lab/crater/pkg/config"
)

type WorkspaceExperimentStats struct {
	WorkspaceType             model.StorageIndexWorkspaceType `json:"workspace_type"`
	WorkspaceName             string                          `json:"workspace_name"`
	LogicalPath               string                          `json:"logical_path"`
	ActualPath                string                          `json:"actual_path"`
	TotalFileCount            int64                           `json:"total_file_count"`
	TotalDirectoryCount       int64                           `json:"total_directory_count"`
	FilteredFileCount         int64                           `json:"filtered_file_count"`
	FilteredDirectoryCount    int64                           `json:"filtered_directory_count"`
	TopLevelCandidateDirCount int                             `json:"top_level_candidate_dir_count"`
	SelectedTopLevelDirCount  int                             `json:"selected_top_level_dir_count"`
	SkippedTopLevelDirCount   int                             `json:"skipped_top_level_dir_count"`
	SelectedTopLevelDirNames  []string                        `json:"selected_top_level_dir_names,omitempty"`
	SkippedTopLevelDirNames   []string                        `json:"skipped_top_level_dir_names,omitempty"`
}

func (s *Service) CollectWorkspaceExperimentStats(
	ctx context.Context,
	workspaceType model.StorageIndexWorkspaceType,
	workspaceName string,
) (*WorkspaceExperimentStats, error) {
	workspace, err := s.resolveWorkspace(ctx, workspaceType, workspaceName)
	if err != nil {
		return nil, err
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

	actualPath, err := ceph.ResolveCephFSPath(
		s.kubeClient,
		s.kubeConfig,
		toolboxNamespace,
		workspace.LogicalPath,
		prefixConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace path failed: %w", err)
	}

	signatures, err := s.listImmediateSignatures(
		toolboxPod,
		"experiment-stats",
		workspace.WorkspaceType,
		workspace.WorkspaceName,
		workspace.LogicalPath,
		actualPath,
	)
	if err != nil {
		return nil, err
	}

	selected, skipped := filterTopLevelSignaturesForModelCopyScan(signatures)
	selectedDirNames, selectedDirPaths := collectTopLevelDirectories(selected)
	skippedDirNames, _ := collectTopLevelDirectories(skipped)

	totalFiles, totalDirs, err := s.countDirectoryTree(toolboxPod, actualPath, false)
	if err != nil {
		return nil, err
	}

	filteredFiles, filteredDirs, err := s.countDirectoryForest(toolboxPod, selectedDirPaths, true)
	if err != nil {
		return nil, err
	}

	return &WorkspaceExperimentStats{
		WorkspaceType:             workspace.WorkspaceType,
		WorkspaceName:             workspace.WorkspaceName,
		LogicalPath:               workspace.LogicalPath,
		ActualPath:                actualPath,
		TotalFileCount:            totalFiles,
		TotalDirectoryCount:       totalDirs,
		FilteredFileCount:         filteredFiles,
		FilteredDirectoryCount:    filteredDirs,
		TopLevelCandidateDirCount: len(selectedDirNames) + len(skippedDirNames),
		SelectedTopLevelDirCount:  len(selectedDirNames),
		SkippedTopLevelDirCount:   len(skippedDirNames),
		SelectedTopLevelDirNames:  selectedDirNames,
		SkippedTopLevelDirNames:   skippedDirNames,
	}, nil
}

func collectTopLevelDirectories(items []topLevelSignature) ([]string, []string) {
	names := make([]string, 0)
	paths := make([]string, 0)
	for _, item := range items {
		if item.EntryType != model.StorageIndexEntryTypeDir {
			continue
		}
		names = append(names, item.Name)
		paths = append(paths, item.ActualPath)
	}
	return names, paths
}

func (s *Service) countDirectoryTree(
	toolboxPod *corev1.Pod,
	actualPath string,
	includeRootDir bool,
) (int64, int64, error) {
	if strings.TrimSpace(actualPath) == "" {
		return 0, 0, nil
	}
	return s.countDirectoryForest(toolboxPod, []string{actualPath}, includeRootDir)
}

func (s *Service) countDirectoryForest(
	toolboxPod *corev1.Pod,
	actualPaths []string,
	includeRootDir bool,
) (int64, int64, error) {
	quoted := make([]string, 0, len(actualPaths))
	for _, actualPath := range actualPaths {
		normalized := normalizeUnixPath(actualPath)
		if normalized == "" {
			continue
		}
		quoted = append(quoted, shellQuote(normalized))
	}
	if len(quoted) == 0 {
		return 0, 0, nil
	}

	dirClause := "-mindepth 1 -type d -print"
	if includeRootDir {
		dirClause = "-type d -print"
	}

	script := fmt.Sprintf(
		`files=0; dirs=0; for root in %s; do if [ ! -d "$root" ]; then continue; fi; f=$(find "$root" -path '*/.snap' -prune -o -type f -print | wc -l | tr -d ' '); d=$(find "$root" -path '*/.snap' -prune -o %s | wc -l | tr -d ' '); files=$((files + f)); dirs=$((dirs + d)); done; printf '%%s%s%%s' "$files" "$dirs"`,
		strings.Join(quoted, " "),
		dirClause,
		findFieldSeparator,
	)

	output, err := ceph.ExecInPod(
		s.kubeClient,
		s.kubeConfig,
		toolboxPod,
		[]string{"sh", "-c", script},
	)
	if err != nil {
		return 0, 0, fmt.Errorf("count directory forest failed: %w", err)
	}

	fields := strings.Split(strings.TrimSpace(output), findFieldSeparator)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected directory count output: %q", output)
	}

	fileCount, err := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse file count failed: %w", err)
	}
	dirCount, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse directory count failed: %w", err)
	}

	return fileCount, dirCount, nil
}
