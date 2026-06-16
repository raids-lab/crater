//nolint:goconst,gosec // Tests repeat literal fixture paths and guard slice lengths before fixed-index checks.
package storageindex

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

const aliceModelsPath = "/user/alice/models"

func TestVerificationModesFitStorageIndexColumns(t *testing.T) {
	t.Parallel()

	modes := map[string]string{
		"metadata":                  verificationModeMetadata,
		"file_name_size":            verificationModeFileName,
		"sha256":                    hashAlgorithmSHA256,
		"sampled_sha256":            hashAlgorithmSampledSHA256,
		"safetensors header + hash": verificationModeSafeTensorsHdrAndSampledSHA,
	}

	for name, mode := range modes {
		if len(mode) > 32 {
			t.Fatalf("%s exceeds varchar(32): %q (%d)", name, mode, len(mode))
		}
	}
}

func TestBuildDirectoryMetricsAggregatesFilesToAncestorDirectories(t *testing.T) {
	workspace := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "alice",
		LogicalPath:   "/user/alice-space",
	}

	entries := []model.StorageIndexEntry{
		{
			WorkspaceType: workspace.WorkspaceType,
			WorkspaceName: workspace.WorkspaceName,
			LogicalPath:   workspace.LogicalPath,
			RelativePath:  ".",
			Name:          "alice-space",
			EntryType:     model.StorageIndexEntryTypeDir,
		},
		{
			WorkspaceType: workspace.WorkspaceType,
			WorkspaceName: workspace.WorkspaceName,
			LogicalPath:   "/user/alice-space/models",
			RelativePath:  "models",
			ParentPath:    workspace.LogicalPath,
			Name:          "models",
			EntryType:     model.StorageIndexEntryTypeDir,
			IsTopLevel:    true,
		},
		{
			WorkspaceType: workspace.WorkspaceType,
			WorkspaceName: workspace.WorkspaceName,
			LogicalPath:   "/user/alice-space/models/model.bin",
			RelativePath:  "models/model.bin",
			ParentPath:    "/user/alice-space/models",
			Name:          "model.bin",
			EntryType:     model.StorageIndexEntryTypeFile,
			SizeBytes:     128,
		},
	}

	metrics, totalSize := buildDirectoryMetrics("scan-1", workspace, entries)
	if totalSize != 128 {
		t.Fatalf("expected root total size 128, got %d", totalSize)
	}

	metricByPath := make(map[string]model.StorageIndexDirectoryMetric, len(metrics))
	for _, item := range metrics {
		metricByPath[item.Path] = item
	}

	rootMetric, ok := metricByPath["/user/alice-space"]
	if !ok {
		t.Fatalf("root metric not found")
	}
	if rootMetric.FileCount != 1 {
		t.Fatalf("expected root file count 1, got %d", rootMetric.FileCount)
	}
	if rootMetric.DirectoryCount != 1 {
		t.Fatalf("expected root directory count 1, got %d", rootMetric.DirectoryCount)
	}
	if rootMetric.TotalSizeBytes != 128 {
		t.Fatalf("expected root total size 128, got %d", rootMetric.TotalSizeBytes)
	}

	modelsMetric, ok := metricByPath["/user/alice-space/models"]
	if !ok {
		t.Fatalf("models metric not found")
	}
	if modelsMetric.FileCount != 1 {
		t.Fatalf("expected models file count 1, got %d", modelsMetric.FileCount)
	}
	if modelsMetric.TotalSizeBytes != 128 {
		t.Fatalf("expected models total size 128, got %d", modelsMetric.TotalSizeBytes)
	}

	if _, exists := metricByPath["/user/alice-space/models/model.bin"]; exists {
		t.Fatalf("file path should not be materialized as directory metric")
	}
}

func TestCoveredByDirectoryHit(t *testing.T) {
	prefixes := []string{"/user/alice-space/models"}
	if !coveredByDirectoryHit("/user/alice-space/models/model.bin", prefixes) {
		t.Fatalf("expected file to be covered by redundant directory prefix")
	}
	if coveredByDirectoryHit("/user/alice-space/logs/train.log", prefixes) {
		t.Fatalf("did not expect unrelated path to be covered by redundant directory prefix")
	}
}

func TestShouldSkipTopLevelModelCopyCandidate(t *testing.T) {
	tests := []struct {
		name   string
		metric model.StorageIndexDirectoryMetric
		want   bool
	}{
		{
			name: "skip top level conda root",
			metric: model.StorageIndexDirectoryMetric{
				Name:       "conda",
				IsTopLevel: true,
			},
			want: true,
		},
		{
			name: "skip top level codex workspace",
			metric: model.StorageIndexDirectoryMetric{
				Name:       ".codex",
				IsTopLevel: true,
			},
			want: true,
		},
		{
			name: "skip top level local runtime dir",
			metric: model.StorageIndexDirectoryMetric{
				Name:       ".local",
				IsTopLevel: true,
			},
			want: true,
		},
		{
			name: "skip top level pip cache dir",
			metric: model.StorageIndexDirectoryMetric{
				Name:       ".pip",
				IsTopLevel: true,
			},
			want: true,
		},
		{
			name: "skip top level nvidia cache dir",
			metric: model.StorageIndexDirectoryMetric{
				Name:       ".nv",
				IsTopLevel: true,
			},
			want: true,
		},
		{
			name: "keep huggingface cache because it may store duplicated public models",
			metric: model.StorageIndexDirectoryMetric{
				Name:       ".cache",
				IsTopLevel: true,
			},
			want: false,
		},
		{
			name: "keep outputs because some jobs write checkpoints there",
			metric: model.StorageIndexDirectoryMetric{
				Name:       "outputs",
				IsTopLevel: true,
			},
			want: false,
		},
		{
			name: "do not skip nested env-style names",
			metric: model.StorageIndexDirectoryMetric{
				Name:       ".venv",
				IsTopLevel: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		if got := shouldSkipTopLevelModelCopyCandidate(tt.metric); got != tt.want {
			t.Fatalf("%s: expected %t, got %t", tt.name, tt.want, got)
		}
	}
}

func TestSkippedTopLevelModelCopyRootPrunesDescendants(t *testing.T) {
	skippedRoots := appendUniquePrefix(nil, "/user/alice-space/conda")
	if !isCoveredByPrefixes("/user/alice-space/conda/envs/base/lib/python3.10/site-packages", skippedRoots) {
		t.Fatalf("expected descendants under skipped top-level root to be pruned")
	}
	if isCoveredByPrefixes("/user/alice-space/models/llama-7b", skippedRoots) {
		t.Fatalf("did not expect unrelated model directory to be pruned")
	}
}

func TestFilterTopLevelSignaturesForModelCopyScan(t *testing.T) {
	signatures := map[string]topLevelSignature{
		"conda": {
			Name:        "conda",
			LogicalPath: "/user/alice-space/conda",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
		"models": {
			Name:        "models",
			LogicalPath: "/user/alice-space/models",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
		".cache": {
			Name:        ".cache",
			LogicalPath: "/user/alice-space/.cache",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
		".codex": {
			Name:        ".codex",
			LogicalPath: "/user/alice-space/.codex",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
	}

	selected, skipped := filterTopLevelSignaturesForModelCopyScan(signatures)
	if len(selected) != 2 {
		t.Fatalf("expected 2 selected top-level signatures, got %d", len(selected))
	}
	if len(skipped) != 2 {
		t.Fatalf("expected 2 skipped top-level signatures, got %d", len(skipped))
	}
	skippedNames := []string{skipped[0].Name, skipped[1].Name}
	if skippedNames[0] != ".codex" || skippedNames[1] != "conda" {
		t.Fatalf("unexpected skipped names: %v", skippedNames)
	}
	selectedNames := []string{selected[0].Name, selected[1].Name}
	if selectedNames[0] != ".cache" || selectedNames[1] != "models" {
		t.Fatalf("unexpected selected names: %v", selectedNames)
	}
}

func TestImmediateSubtreeAllowListForCache(t *testing.T) {
	allowList, ok := immediateSubtreeAllowListForTopLevel(".cache")
	if !ok {
		t.Fatalf("expected .cache allowlist to exist")
	}
	if _, ok := allowList["huggingface"]; !ok {
		t.Fatalf("expected huggingface to be retained under .cache")
	}
	if _, ok := allowList["pip"]; ok {
		t.Fatalf("did not expect pip to be retained under .cache")
	}
}

func TestFilterImmediateSubtreesByAllowList(t *testing.T) {
	signatures := map[string]topLevelSignature{
		"huggingface": {
			Name:        "huggingface",
			LogicalPath: "/user/alice-space/.cache/huggingface",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
		"modelscope": {
			Name:        "modelscope",
			LogicalPath: "/user/alice-space/.cache/modelscope",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
		"pip": {
			Name:        "pip",
			LogicalPath: "/user/alice-space/.cache/pip",
			EntryType:   model.StorageIndexEntryTypeDir,
		},
		"readme.txt": {
			Name:        "readme.txt",
			LogicalPath: "/user/alice-space/.cache/readme.txt",
			EntryType:   model.StorageIndexEntryTypeFile,
		},
	}

	allowList, _ := immediateSubtreeAllowListForTopLevel(".cache")
	selected, skipped := filterImmediateSubtreesByAllowList(signatures, allowList)
	if len(selected) != 2 {
		t.Fatalf("expected 2 selected cache subtrees, got %d", len(selected))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped cache subtree, got %d", len(skipped))
	}

	selectedNames := []string{selected[0].Name, selected[1].Name}
	if selectedNames[0] != "huggingface" || selectedNames[1] != "modelscope" {
		t.Fatalf("unexpected selected cache subtree names: %v", selectedNames)
	}
	if skipped[0].Name != "pip" {
		t.Fatalf("expected pip to be skipped, got %s", skipped[0].Name)
	}
}

func TestSortedPrunableNestedDirNamesIncludesVenv(t *testing.T) {
	names := sortedPrunableNestedDirNames()
	foundVenv := false
	foundPycache := false
	for _, name := range names {
		if name == ".venv" {
			foundVenv = true
		}
		if name == "__pycache__" {
			foundPycache = true
		}
	}
	if !foundVenv || !foundPycache {
		t.Fatalf("expected recursive prune names to include .venv and __pycache__, got %v", names)
	}
}

func TestBuildDirectoryScanScriptPrunesNestedEnvDirs(t *testing.T) {
	script := buildDirectoryScanScript("/user/alice-space/models", nil)
	if !strings.Contains(script, "-name '.venv'") {
		t.Fatalf("expected script to prune nested .venv directories, got %s", script)
	}
	if !strings.Contains(script, "-name '__pycache__'") {
		t.Fatalf("expected script to prune nested __pycache__ directories, got %s", script)
	}
}

func TestBuildPublicRootLookupIndexesResourceAndPathBase(t *testing.T) {
	publicRoots := []model.StorageIndexPublicRootBaseline{
		{
			ResourceName: "Qwen2.5-7B",
			LogicalPath:  "/public/models/Qwen2.5-7B",
		},
	}

	lookup := buildPublicRootLookup(publicRoots)
	if len(lookup["qwen2.5-7b"]) != 1 {
		t.Fatalf("expected deduplicated root lookup entry, got %d entries", len(lookup["qwen2.5-7b"]))
	}
}

func TestParseSha256sumOutput(t *testing.T) {
	output := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  /mnt/mycephfs/models/model.bin\n" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  /mnt/mycephfs/models/path with spaces/config.json\n"

	parsed := parseSha256sumOutput(output)
	if parsed["/mnt/mycephfs/models/model.bin"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("failed to parse normal sha256sum output")
	}
	if parsed["/mnt/mycephfs/models/path with spaces/config.json"] != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("failed to parse sha256sum output with spaces in path")
	}
}

func TestParseSafeTensorsHeaderSkeleton(t *testing.T) {
	header := `{
		"__metadata__": {"format":"pt"},
		"model.layers.1.weight": {"dtype":"F16","shape":[3,4],"data_offsets":[100,200]},
		"model.layers.0.weight": {"dtype":"F16","shape":[1,2],"data_offsets":[0,99]}
	}`

	skeleton, err := parseSafeTensorsHeaderSkeleton(header)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(skeleton) != 2 {
		t.Fatalf("expected 2 tensor entries, got %d", len(skeleton))
	}
	if skeleton[0] != "model.layers.0.weight=1,2" {
		t.Fatalf("unexpected first skeleton entry: %s", skeleton[0])
	}
	if skeleton[1] != "model.layers.1.weight=3,4" {
		t.Fatalf("unexpected second skeleton entry: %s", skeleton[1])
	}
}

func TestBuildSampledRegions(t *testing.T) {
	regions := buildSampledRegions(1024 * 1024)
	if len(regions) != 3 {
		t.Fatalf("expected 3 sampled regions, got %d", len(regions))
	}
	if regions[0].Offset != 262144 {
		t.Fatalf("unexpected 25%% offset: %d", regions[0].Offset)
	}
	if regions[1].Offset != 524288 {
		t.Fatalf("unexpected 50%% offset: %d", regions[1].Offset)
	}
	if regions[2].Offset != 786432 {
		t.Fatalf("unexpected 75%% offset: %d", regions[2].Offset)
	}

	small := buildSampledRegions(1024)
	if len(small) != 1 || small[0].Offset != 0 || small[0].Count != 1024 {
		t.Fatalf("unexpected sampled region for small file: %+v", small)
	}
}

func TestCloneRedundancyHitsForScanRetagsExistingRows(t *testing.T) {
	now := time.Unix(1714377600, 0)
	hits := []model.StorageIndexRedundancyHit{
		{
			ID:                 42,
			WorkspaceType:      model.StorageIndexWorkspaceTypeUser,
			WorkspaceName:      "alice",
			ScanID:             "scan-old",
			TargetType:         model.StorageIndexRedundancyTargetTypeDirectory,
			TargetPath:         "/user/alice/models",
			PublicPath:         "/public/models/foo",
			MatchKey:           "models|123",
			VerificationStatus: model.StorageIndexVerificationStatusSuspected,
			EstimatedBytes:     123,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}

	cloned := cloneRedundancyHitsForScan("scan-new", hits)
	if len(cloned) != 1 {
		t.Fatalf("expected 1 cloned hit, got %d", len(cloned))
	}
	if cloned[0].ID != 0 {
		t.Fatalf("expected cloned hit id reset to 0, got %d", cloned[0].ID)
	}
	if cloned[0].ScanID != "scan-new" {
		t.Fatalf("expected cloned hit scan id to be retagged, got %q", cloned[0].ScanID)
	}
	if !cloned[0].CreatedAt.IsZero() || !cloned[0].UpdatedAt.IsZero() {
		t.Fatalf("expected cloned hit timestamps to be cleared, got created=%v updated=%v", cloned[0].CreatedAt, cloned[0].UpdatedAt)
	}
	if cloned[0].TargetPath != hits[0].TargetPath || cloned[0].PublicPath != hits[0].PublicPath {
		t.Fatalf("expected cloned hit payload to be preserved, got %+v", cloned[0])
	}
	if hits[0].ID != 42 || hits[0].ScanID != "scan-old" {
		t.Fatalf("expected original hit to remain unchanged, got %+v", hits[0])
	}
}

func TestExpandIncrementalPlanWithCandidateRootsPromotesCandidateAncestor(t *testing.T) {
	workspace := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "alice",
		LogicalPath:   "/user/alice",
	}
	plan := &incrementalPlan{
		RescanTargets: []topLevelSignature{
			{
				Name:              "subdir",
				LogicalPath:       "/user/alice/models/subdir",
				ParentLogicalPath: "/user/alice/models",
				ActualPath:        "/snap/alice/models/subdir",
				EntryType:         model.StorageIndexEntryTypeDir,
			},
		},
		UpsertEntries: []model.StorageIndexEntry{
			{
				LogicalPath: "/user/alice/models/subdir/file.bin",
				EntryType:   model.StorageIndexEntryTypeFile,
			},
		},
		RemovedPrefixes: []string{},
	}
	existingCandidates := []model.StorageIndexCandidate{
		{TargetPath: aliceModelsPath},
		{TargetPath: "/user/alice/unrelated"},
	}

	expanded := expandIncrementalPlanWithCandidateRoots("scan-1", workspace, "/snap/alice", plan, existingCandidates)
	if len(expanded.RescanTargets) != 1 {
		t.Fatalf("expected 1 expanded rescan target, got %d", len(expanded.RescanTargets))
	}
	if expanded.RescanTargets[0].LogicalPath != aliceModelsPath {
		t.Fatalf("expected candidate ancestor to become rescan target, got %s", expanded.RescanTargets[0].LogicalPath)
	}
	if expanded.RescanTargets[0].ActualPath != "/snap/alice/models" {
		t.Fatalf("expected candidate ancestor actual path to be projected from snapshot root, got %s", expanded.RescanTargets[0].ActualPath)
	}
	if len(expanded.UpsertEntries) != 0 {
		t.Fatalf("expected nested upsert entries to be subsumed by candidate root rescan, got %d entries", len(expanded.UpsertEntries))
	}
}

func TestCollectAffectedCandidatePathsIncludesExactAndRemovedPrefixes(t *testing.T) {
	existingCandidates := []model.StorageIndexCandidate{
		{TargetPath: "/user/alice/models"},
		{TargetPath: "/user/alice/removed/old-copy"},
		{TargetPath: "/user/alice/unrelated"},
	}

	affected := collectAffectedCandidatePaths(
		existingCandidates,
		[]string{"/user/alice/models", "/user/alice"},
		[]string{"/user/alice/removed"},
	)

	if len(affected) != 2 {
		t.Fatalf("expected 2 affected candidate paths, got %d: %v", len(affected), affected)
	}
	if affected[0] != "/user/alice/models" && affected[1] != "/user/alice/models" {
		t.Fatalf("expected exact affected candidate path to be included, got %v", affected)
	}
	if affected[0] != "/user/alice/removed/old-copy" && affected[1] != "/user/alice/removed/old-copy" {
		t.Fatalf("expected removed-prefix candidate path to be included, got %v", affected)
	}
}

func TestMergeExistingCandidateBindingsPreservesPublicPathAfterSizeDrift(t *testing.T) {
	workspace := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "alice",
		LogicalPath:   "/user/alice",
	}
	candidates := []model.StorageIndexCandidate{
		{
			WorkspaceType:  workspace.WorkspaceType,
			WorkspaceName:  workspace.WorkspaceName,
			ScanID:         "scan-new",
			CandidateType:  "model_dir",
			TargetPath:     "/user/alice/models/qwen",
			PublicPath:     "",
			Evidence:       "category hint only",
			CandidateScore: 48,
			Status:         model.StorageIndexCandidateStatusSuspected,
		},
	}
	existing := []model.StorageIndexCandidate{
		{
			WorkspaceType:  workspace.WorkspaceType,
			WorkspaceName:  workspace.WorkspaceName,
			ScanID:         "scan-old",
			CandidateType:  "model_dir",
			TargetPath:     "/user/alice/models/qwen",
			PublicPath:     "/public/models/qwen",
			Evidence:       "previous verified match",
			CandidateScore: 100,
			Status:         model.StorageIndexCandidateStatusVerified,
		},
	}

	merged := mergeExistingCandidateBindings("scan-new", workspace, candidates, existing)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged candidate, got %d", len(merged))
	}
	if merged[0].PublicPath != "/public/models/qwen" {
		t.Fatalf("expected public path binding to be preserved, got %q", merged[0].PublicPath)
	}
	if merged[0].CandidateScore != 100 {
		t.Fatalf("expected candidate score to keep stronger historical score, got %v", merged[0].CandidateScore)
	}
}

func TestMergeExistingCandidateBindingsAppendsHistoricalCandidateForRevalidation(t *testing.T) {
	workspace := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "alice",
		LogicalPath:   "/user/alice",
	}
	existing := []model.StorageIndexCandidate{
		{
			WorkspaceType:  workspace.WorkspaceType,
			WorkspaceName:  workspace.WorkspaceName,
			ScanID:         "scan-old",
			CandidateType:  "model_dir",
			TargetPath:     "/user/alice/models/llama",
			PublicPath:     "/public/models/llama",
			Evidence:       "previous verified match",
			CandidateScore: 100,
			Status:         model.StorageIndexCandidateStatusVerified,
		},
	}

	merged := mergeExistingCandidateBindings("scan-new", workspace, nil, existing)
	if len(merged) != 1 {
		t.Fatalf("expected historical candidate to be appended for revalidation, got %d", len(merged))
	}
	if merged[0].TargetPath != "/user/alice/models/llama" || merged[0].PublicPath != "/public/models/llama" {
		t.Fatalf("expected appended candidate to keep previous target/public binding, got %+v", merged[0])
	}
}

func TestCompareStrictDirectoryFileSetsDetectsAddedFiles(t *testing.T) {
	targetFiles := []candidateFileProbe{
		{RelativePath: "config.json", SizeBytes: 100},
		{RelativePath: "weights.bin", SizeBytes: 200},
		{RelativePath: "extra.txt", SizeBytes: 50},
	}
	publicFiles := []candidateFileProbe{
		{RelativePath: "config.json", SizeBytes: 100},
		{RelativePath: "weights.bin", SizeBytes: 200},
	}

	missingTarget, missingPublic := compareStrictDirectoryFileSets(targetFiles, publicFiles)
	if len(missingTarget) != 0 {
		t.Fatalf("expected public baseline not to miss target-required files, got %v", missingTarget)
	}
	if len(missingPublic) != 1 || missingPublic[0] != "extra.txt" {
		t.Fatalf("expected added target file to appear in missing public set, got %v", missingPublic)
	}
}

func TestAllCandidateFilesVerifiedRequiresEveryFileVerified(t *testing.T) {
	files := []model.StorageIndexCandidateFile{
		{VerificationStatus: model.StorageIndexVerificationStatusVerified},
		{VerificationStatus: model.StorageIndexVerificationStatusSuspected},
	}
	if allCandidateFilesVerified(files) {
		t.Fatalf("expected mixed verification states to fail full candidate verification")
	}
	if !allCandidateFilesVerified([]model.StorageIndexCandidateFile{{VerificationStatus: model.StorageIndexVerificationStatusVerified}}) {
		t.Fatalf("expected fully verified file list to pass")
	}
}

func TestDiffTopLevelSignaturesDetectsRecursiveChangedAtUpdate(t *testing.T) {
	currentChangedAt := time.Unix(1714480000, 0)
	previousChangedAt := currentChangedAt.Add(-time.Second)
	current := map[string]topLevelSignature{
		"models": {
			Name:        "models",
			LogicalPath: "/user/alice/models",
			EntryType:   model.StorageIndexEntryTypeDir,
			SizeBytes:   1024,
			ChangedAt:   &currentChangedAt,
		},
	}
	previous := map[string]topLevelSignature{
		"models": {
			Name:        "models",
			LogicalPath: "/user/alice/models",
			EntryType:   model.StorageIndexEntryTypeDir,
			SizeBytes:   1024,
			ChangedAt:   &previousChangedAt,
		},
	}

	changed, removed, changedCount := diffTopLevelSignatures(current, previous)
	if len(removed) != 0 {
		t.Fatalf("expected no removed prefixes, got %v", removed)
	}
	if changedCount != 1 || len(changed) != 1 || changed[0].LogicalPath != "/user/alice/models" {
		t.Fatalf("expected recursive changed time drift to be detected, got changed=%v count=%d", changed, changedCount)
	}
}

func TestParseUnixTimestampTruncatesToMicrosecond(t *testing.T) {
	parsed := parseUnixTimestamp("1714416536.118177413")
	if parsed == nil {
		t.Fatalf("expected timestamp to parse")
	}
	if parsed.UTC().Format(time.RFC3339Nano) != "2024-04-29T18:48:56.118177Z" {
		t.Fatalf("expected parsed timestamp to be truncated to microsecond precision, got %s", parsed.UTC().Format(time.RFC3339Nano))
	}
}

func TestCleanupWorkspaceStateBeforeInitialScanClearsOnlyTargetWorkspace(t *testing.T) {
	db := openStorageIndexTestDB(t)
	target := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "alice",
		LogicalPath:   "/user/alice",
	}
	other := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "bob",
		LogicalPath:   "/user/bob",
	}

	mustCreateStorageIndexState(t, db, target, "scan-target")
	mustCreateStorageIndexState(t, db, other, "scan-other")

	clearedRows, err := cleanupWorkspaceStateBeforeInitialScan(db, target)
	if err != nil {
		t.Fatalf("cleanup workspace state failed: %v", err)
	}
	if clearedRows != 5 {
		t.Fatalf("expected 5 cleared rows for target workspace, got %d", clearedRows)
	}

	assertWorkspaceStateRowCount(t, db, target, 0)
	assertWorkspaceStateRowCount(t, db, other, 5)
}

func TestCleanupWorkspaceStateBeforeInitialScanClearsPublicBaselines(t *testing.T) {
	db := openStorageIndexTestDB(t)
	publicWorkspace := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypePublic,
		WorkspaceName: "public",
		LogicalPath:   "/public",
	}
	other := resolvedWorkspace{
		WorkspaceType: model.StorageIndexWorkspaceTypeUser,
		WorkspaceName: "alice",
		LogicalPath:   "/user/alice",
	}

	mustCreateStorageIndexState(t, db, publicWorkspace, "scan-public")
	mustCreatePublicBaselineState(t, db, "scan-public")
	mustCreateStorageIndexState(t, db, other, "scan-other")

	clearedRows, err := cleanupWorkspaceStateBeforeInitialScan(db, publicWorkspace)
	if err != nil {
		t.Fatalf("cleanup public workspace state failed: %v", err)
	}
	if clearedRows != 7 {
		t.Fatalf("expected 7 cleared rows for public workspace, got %d", clearedRows)
	}

	assertWorkspaceStateRowCount(t, db, publicWorkspace, 0)
	assertWorkspaceStateRowCount(t, db, other, 5)
	assertTableRowCount(t, db, &model.StorageIndexPublicRootBaseline{}, 0)
	assertTableRowCount(t, db, &model.StorageIndexPublicFileBaseline{}, 0)
}

func openStorageIndexTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "storageindex-test.sqlite")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite sql db failed: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(
		&model.StorageIndexEntry{},
		&model.StorageIndexDirectoryMetric{},
		&model.StorageIndexRedundancyHit{},
		&model.StorageIndexCandidate{},
		&model.StorageIndexCandidateFile{},
		&model.StorageIndexPublicRootBaseline{},
		&model.StorageIndexPublicFileBaseline{},
	); err != nil {
		t.Fatalf("auto migrate sqlite db failed: %v", err)
	}
	return db
}

func mustCreateStorageIndexState(t *testing.T, db *gorm.DB, workspace resolvedWorkspace, scanID string) {
	t.Helper()

	entry := model.StorageIndexEntry{
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		ScanID:        scanID,
		LogicalPath:   workspace.LogicalPath,
		RelativePath:  ".",
		Name:          filepath.Base(workspace.LogicalPath),
		EntryType:     model.StorageIndexEntryTypeDir,
	}
	metric := model.StorageIndexDirectoryMetric{
		WorkspaceType: workspace.WorkspaceType,
		WorkspaceName: workspace.WorkspaceName,
		ScanID:        scanID,
		Path:          workspace.LogicalPath,
		Name:          filepath.Base(workspace.LogicalPath),
	}
	hit := model.StorageIndexRedundancyHit{
		WorkspaceType:      workspace.WorkspaceType,
		WorkspaceName:      workspace.WorkspaceName,
		ScanID:             scanID,
		TargetType:         model.StorageIndexRedundancyTargetTypeDirectory,
		TargetPath:         workspace.LogicalPath + "/dup",
		PublicPath:         "/public/dup",
		MatchKey:           "dup|1",
		VerificationStatus: model.StorageIndexVerificationStatusSuspected,
	}
	candidate := model.StorageIndexCandidate{
		WorkspaceType:  workspace.WorkspaceType,
		WorkspaceName:  workspace.WorkspaceName,
		ScanID:         scanID,
		CandidateType:  "model_dir",
		TargetPath:     workspace.LogicalPath + "/candidate",
		PublicPath:     "/public/candidate",
		CandidateScore: 80,
		Status:         model.StorageIndexCandidateStatusSuspected,
	}
	candidateFile := model.StorageIndexCandidateFile{
		WorkspaceType:      workspace.WorkspaceType,
		WorkspaceName:      workspace.WorkspaceName,
		ScanID:             scanID,
		CandidatePath:      candidate.TargetPath,
		FilePath:           candidate.TargetPath + "/weights.bin",
		FileName:           "weights.bin",
		RelativePath:       "weights.bin",
		VerificationStatus: model.StorageIndexVerificationStatusSuspected,
	}

	for _, item := range []any{&entry, &metric, &hit, &candidate, &candidateFile} {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("seed storage index state failed: %v", err)
		}
	}
}

func mustCreatePublicBaselineState(t *testing.T, db *gorm.DB, scanID string) {
	t.Helper()

	root := model.StorageIndexPublicRootBaseline{
		ScanID:       scanID,
		ResourceName: "qwen",
		LogicalPath:  "/public/models/qwen",
		Category:     "model_dir",
	}
	file := model.StorageIndexPublicFileBaseline{
		ScanID:         scanID,
		PublicRootPath: root.LogicalPath,
		PublicRootHash: hashString(root.LogicalPath),
		FilePath:       root.LogicalPath + "/weights.bin",
		FileName:       "weights.bin",
		RelativePath:   "weights.bin",
		MatchKey:       "weights.bin|1",
		MatchKeyHash:   hashString("weights.bin|1"),
	}

	if err := db.Create(&root).Error; err != nil {
		t.Fatalf("seed public root baseline failed: %v", err)
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("seed public file baseline failed: %v", err)
	}
}

func assertWorkspaceStateRowCount(t *testing.T, db *gorm.DB, workspace resolvedWorkspace, want int64) {
	t.Helper()

	models := []any{
		&model.StorageIndexEntry{},
		&model.StorageIndexDirectoryMetric{},
		&model.StorageIndexRedundancyHit{},
		&model.StorageIndexCandidate{},
		&model.StorageIndexCandidateFile{},
	}
	total := int64(0)
	for _, table := range models {
		var count int64
		if err := db.Model(table).
			Where("workspace_type = ? AND workspace_name = ?", workspace.WorkspaceType, workspace.WorkspaceName).
			Count(&count).Error; err != nil {
			t.Fatalf("count workspace state rows failed: %v", err)
		}
		total += count
	}
	if total != want {
		t.Fatalf("expected workspace %s/%s to have %d state rows, got %d", workspace.WorkspaceType, workspace.WorkspaceName, want, total)
	}
}

func assertTableRowCount(t *testing.T, db *gorm.DB, table any, want int64) {
	t.Helper()

	var count int64
	if err := db.Model(table).Count(&count).Error; err != nil {
		t.Fatalf("count table rows failed: %v", err)
	}
	if count != want {
		t.Fatalf("expected %T to have %d rows, got %d", table, want, count)
	}
}
