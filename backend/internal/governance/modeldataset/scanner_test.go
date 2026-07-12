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
	"os"
	"path/filepath"
	"testing"

	"github.com/raids-lab/crater/dao/model"
)

func TestScanPublicFindsCompleteModelsAndSkipsExcludedTrees(t *testing.T) {
	root := t.TempDir()
	modelDir := filepath.Join(root, "shared", "Models", "owner", "model", "huggingface", "default")
	mustWriteFile(t, filepath.Join(modelDir, "config.json"), "{}")
	mustWriteFile(t, filepath.Join(modelDir, "README.md"), "# model")
	mustWriteFile(t, filepath.Join(modelDir, "model-00001-of-00002.safetensors"), "weights")
	incompleteDir := filepath.Join(root, "shared", "Models", "owner", "incomplete")
	mustWriteFile(t, filepath.Join(incompleteDir, "config.json"), "{}")
	excludedDir := filepath.Join(root, "shared", "Models", "tests", "fixture")
	mustWriteFile(t, filepath.Join(excludedDir, "config.json"), "{}")
	mustWriteFile(t, filepath.Join(excludedDir, "model.safetensors"), "weights")

	candidates, err := ScanPublic(context.Background(), &ScanOptions{
		StorageRoot:          root,
		PublicPrefix:         "shared",
		ModelsSubdirectory:   "Models",
		DatasetsSubdirectory: "Datasets",
		MaxDepth:             8,
		ExcludedDirectories:  []string{"tests"},
		WeightPatterns:       []string{"*.safetensors"},
	})
	if err != nil {
		t.Fatalf("ScanPublic() error = %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("ScanPublic() candidates = %#v, want one", candidates)
	}
	candidate := candidates[0]
	if candidate.Path != "shared/Models/owner/model/huggingface/default" {
		t.Fatalf("candidate.Path = %q", candidate.Path)
	}
	if candidate.Type != model.DataTypeModel || !candidate.Evidence.HasReadme || candidate.Evidence.WeightFiles != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
}

func TestScanPublicDatasetDiscoveryIsOptIn(t *testing.T) {
	root := t.TempDir()
	datasetDir := filepath.Join(root, "shared", "Datasets", "owner", "dataset")
	mustWriteFile(t, filepath.Join(datasetDir, "dataset_info.json"), "{}")

	withoutMarkers, err := ScanPublic(context.Background(), &ScanOptions{
		StorageRoot:          root,
		PublicPrefix:         "shared",
		ModelsSubdirectory:   "Models",
		DatasetsSubdirectory: "Datasets",
		MaxDepth:             8,
		WeightPatterns:       []string{"*.safetensors"},
	})
	if err != nil {
		t.Fatalf("ScanPublic() without markers error = %v", err)
	}
	if len(withoutMarkers) != 0 {
		t.Fatalf("dataset discovery must be disabled without explicit markers: %#v", withoutMarkers)
	}

	withMarkers, err := ScanPublic(context.Background(), &ScanOptions{
		StorageRoot:           root,
		PublicPrefix:          "shared",
		ModelsSubdirectory:    "Models",
		DatasetsSubdirectory:  "Datasets",
		MaxDepth:              8,
		WeightPatterns:        []string{"*.safetensors"},
		DatasetMarkerPatterns: []string{"dataset_info.json"},
	})
	if err != nil {
		t.Fatalf("ScanPublic() with markers error = %v", err)
	}
	if len(withMarkers) != 1 || withMarkers[0].Type != model.DataTypeDataset {
		t.Fatalf("dataset candidates = %#v", withMarkers)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
