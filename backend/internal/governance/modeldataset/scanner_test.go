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

func TestScanPublicSupportsMultipleModelRootsAndGitProvenance(t *testing.T) {
	root := t.TempDir()
	legacyDir := filepath.Join(root, "shared", "LLM", "falcon-40b-instruct")
	mustWriteFile(t, filepath.Join(legacyDir, "config.json"), `{}`)
	mustWriteFile(t, filepath.Join(legacyDir, "model.safetensors"), "weights")
	mustWriteFile(t, filepath.Join(legacyDir, ".git", "config"), "url = git@hf.co:tiiuae/falcon-40b-instruct\n")
	managedDir := filepath.Join(root, "shared", "Models", "owner", "managed")
	mustWriteFile(t, filepath.Join(managedDir, "config.json"), `{}`)
	mustWriteFile(t, filepath.Join(managedDir, "model.safetensors"), "weights")

	candidates, err := ScanPublic(context.Background(), &ScanOptions{
		StorageRoot: root, PublicPrefix: "shared",
		ModelsSubdirectory:   "ignored-when-list-is-set",
		ModelsSubdirectories: []string{"Models", "LLM", "LLM"},
		MaxDepth:             8, WeightPatterns: []string{"*.safetensors"},
	})
	if err != nil {
		t.Fatalf("ScanPublic() error = %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %#v", candidates)
	}
	legacy := candidates[0]
	if legacy.Path != "shared/LLM/falcon-40b-instruct" {
		t.Fatalf("legacy.Path = %q", legacy.Path)
	}
	if legacy.Evidence.Provider != model.ModelDatasetProviderHuggingFace ||
		legacy.Evidence.RepositoryID != "tiiuae/falcon-40b-instruct" ||
		legacy.Evidence.ProvenanceSource != "git_remote" ||
		legacy.Evidence.ProvenanceConfidence != provenanceConfidenceHigh {
		t.Fatalf("legacy evidence = %#v", legacy.Evidence)
	}
	if legacy.Evidence.FilesystemUID == "" || legacy.Evidence.FilesystemGID == "" ||
		legacy.Evidence.ModifiedAt == nil {
		t.Fatalf("filesystem evidence = %#v", legacy.Evidence)
	}
}

func TestScanPublicTreatsConfigAndAmbiguousReadmeAsHints(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "shared", "Models", "llama2-7b")
	mustWriteFile(t, filepath.Join(directory, "config.json"), `{"_name_or_path":"meta-llama/Llama-2-7b-hf"}`)
	mustWriteFile(t, filepath.Join(directory, "model.safetensors"), "weights")
	mustWriteFile(t, filepath.Join(directory, "README.md"),
		"Derived from https://huggingface.co/other/base and https://modelscope.cn/models/other/tokenizer")

	candidates, err := ScanPublic(context.Background(), &ScanOptions{
		StorageRoot: root, PublicPrefix: "shared", ModelsSubdirectory: "Models",
		MaxDepth: 8, WeightPatterns: []string{"*.safetensors"},
	})
	if err != nil {
		t.Fatalf("ScanPublic() error = %v", err)
	}
	evidence := candidates[0].Evidence
	if evidence.Provider != "" || evidence.RepositoryID != "meta-llama/Llama-2-7b-hf" ||
		evidence.ProvenanceSource != "config_name_or_path" ||
		evidence.ProvenanceConfidence != provenanceConfidenceMedium {
		t.Fatalf("evidence = %#v", evidence)
	}
	if len(evidence.CandidateURLs) != 2 {
		t.Fatalf("candidate URLs = %#v", evidence.CandidateURLs)
	}
}

func TestScanPublicRecognizesMatchingModelScopeReadme(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "shared", "Models", "Qwen2.5-7B-Instruct")
	mustWriteFile(t, filepath.Join(directory, "config.json"), `{}`)
	mustWriteFile(t, filepath.Join(directory, "model.safetensors"), "weights")
	mustWriteFile(t, filepath.Join(directory, "README.md"),
		"Model card: https://modelscope.cn/models/Qwen/Qwen2.5-7B-Instruct")

	candidates, err := ScanPublic(context.Background(), &ScanOptions{
		StorageRoot: root, PublicPrefix: "shared", ModelsSubdirectory: "Models",
		MaxDepth: 8, WeightPatterns: []string{"*.safetensors"},
	})
	if err != nil {
		t.Fatalf("ScanPublic() error = %v", err)
	}
	evidence := candidates[0].Evidence
	if evidence.Provider != model.ModelDatasetProviderModelScope ||
		evidence.RepositoryID != "Qwen/Qwen2.5-7B-Instruct" ||
		evidence.ProvenanceSource != "readme_url" ||
		evidence.ProvenanceConfidence != provenanceConfidenceHigh {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestScanPublicRejectsModelRootsOutsidePublicPrefix(t *testing.T) {
	for _, modelRoot := range []string{"../private", "/private", `..\private`} {
		_, err := ScanPublic(context.Background(), &ScanOptions{
			StorageRoot: t.TempDir(), PublicPrefix: "shared",
			ModelsSubdirectories: []string{modelRoot}, MaxDepth: 8,
			WeightPatterns: []string{"*.safetensors"},
		})
		if err == nil {
			t.Fatalf("ScanPublic() accepted model root %q outside the public prefix", modelRoot)
		}
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
