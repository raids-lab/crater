// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modeldataset

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/raids-lab/crater/dao/model"
)

const maxEvidenceFiles = 16

type ScanOptions struct {
	StorageRoot           string
	PublicPrefix          string
	ModelsSubdirectory    string
	DatasetsSubdirectory  string
	MaxDepth              int
	ExcludedDirectories   []string
	WeightPatterns        []string
	DatasetMarkerPatterns []string
}

type Candidate struct {
	Path         string
	AbsolutePath string
	Type         model.DataType
	Name         string
	Evidence     model.ModelDatasetDiscoveryEvidence
	SizeBytes    int64
}

func ScanPublic(ctx context.Context, options *ScanOptions) ([]Candidate, error) {
	if options.StorageRoot == "" || options.PublicPrefix == "" {
		return nil, errors.New("storage root and public prefix are required")
	}
	if options.MaxDepth < 1 {
		return nil, errors.New("max depth must be positive")
	}

	publicRoot := filepath.Join(options.StorageRoot, filepath.FromSlash(options.PublicPrefix))
	excluded := makeSet(options.ExcludedDirectories)
	candidates, err := scanModels(
		ctx,
		filepath.Join(publicRoot, filepath.FromSlash(options.ModelsSubdirectory)),
		options.PublicPrefix,
		options.ModelsSubdirectory,
		options.MaxDepth,
		excluded,
		options.WeightPatterns,
	)
	if err != nil {
		return nil, err
	}

	if len(options.DatasetMarkerPatterns) > 0 {
		datasets, scanErr := scanDatasets(
			ctx,
			filepath.Join(publicRoot, filepath.FromSlash(options.DatasetsSubdirectory)),
			options.PublicPrefix,
			options.DatasetsSubdirectory,
			options.MaxDepth,
			excluded,
			options.DatasetMarkerPatterns,
		)
		if scanErr != nil {
			return nil, scanErr
		}
		candidates = append(candidates, datasets...)
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Path < candidates[j].Path })
	return candidates, nil
}

//nolint:gocyclo // Filesystem traversal keeps exclusion, depth, and candidate checks adjacent.
func scanModels(
	ctx context.Context,
	root, publicPrefix, subdirectory string,
	maxDepth int,
	excluded map[string]struct{},
	weightPatterns []string,
) ([]Candidate, error) {
	if len(weightPatterns) == 0 {
		return nil, errors.New("at least one model weight pattern is required")
	}
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return []Candidate{}, nil
	} else if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	candidates := make([]Candidate, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		depth, err := relativeDepth(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && isExcluded(entry.Name(), excluded) {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > maxDepth || entry.Name() != "config.json" {
			return nil
		}

		directory := filepath.Dir(path)
		if _, ok := seen[directory]; ok {
			return nil
		}
		candidate, ok, err := modelCandidate(directory, root, publicPrefix, subdirectory, weightPatterns)
		if err != nil {
			return err
		}
		if ok {
			seen[directory] = struct{}{}
			candidates = append(candidates, candidate)
		}
		return nil
	})
	return candidates, err
}

func modelCandidate(
	directory, root, publicPrefix, subdirectory string,
	weightPatterns []string,
) (Candidate, bool, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Candidate{}, false, err
	}
	evidence := model.ModelDatasetDiscoveryEvidence{HasConfig: true}
	var sizeBytes int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isReadme(name) {
			evidence.HasReadme = true
		}
		if !matchesAny(name, weightPatterns) {
			continue
		}
		evidence.WeightFiles++
		if len(evidence.MatchedFiles) < maxEvidenceFiles {
			evidence.MatchedFiles = append(evidence.MatchedFiles, name)
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return Candidate{}, false, infoErr
		}
		sizeBytes += info.Size()
	}
	if evidence.WeightFiles == 0 {
		return Candidate{}, false, nil
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return Candidate{}, false, err
	}
	return Candidate{
		Path:         normalizedPath(publicPrefix, subdirectory, relative),
		AbsolutePath: directory,
		Type:         model.DataTypeModel,
		Name:         filepath.Base(directory),
		Evidence:     evidence,
		SizeBytes:    sizeBytes,
	}, true, nil
}

func scanDatasets(
	ctx context.Context,
	root, publicPrefix, subdirectory string,
	maxDepth int,
	excluded map[string]struct{},
	markerPatterns []string,
) ([]Candidate, error) {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return []Candidate{}, nil
	} else if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	candidates := make([]Candidate, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		depth, err := relativeDepth(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && isExcluded(entry.Name(), excluded) {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if depth > maxDepth || !matchesAny(entry.Name(), markerPatterns) {
			return nil
		}
		directory := filepath.Dir(path)
		if _, ok := seen[directory]; ok {
			return nil
		}
		relative, err := filepath.Rel(root, directory)
		if err != nil {
			return err
		}
		seen[directory] = struct{}{}
		candidates = append(candidates, Candidate{
			Path:         normalizedPath(publicPrefix, subdirectory, relative),
			AbsolutePath: directory,
			Type:         model.DataTypeDataset,
			Name:         filepath.Base(directory),
			Evidence: model.ModelDatasetDiscoveryEvidence{
				HasReadme:    isReadme(entry.Name()),
				MatchedFiles: []string{entry.Name()},
			},
		})
		return nil
	})
	return candidates, err
}

func relativeDepth(root, path string) (int, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return 0, err
	}
	if relative == "." {
		return 0, nil
	}
	return len(strings.Split(filepath.ToSlash(relative), "/")), nil
}

func normalizedPath(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		cleaned = append(cleaned, filepath.ToSlash(part))
	}
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(filepath.Join(cleaned...))), "/")
}

func makeSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func isExcluded(name string, excluded map[string]struct{}) bool {
	_, ok := excluded[name]
	return ok
}

func matchesAny(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func isReadme(name string) bool {
	return strings.EqualFold(name, "README") || strings.EqualFold(name, "README.md")
}
