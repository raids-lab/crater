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
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
)

//nolint:gocyclo // The integration assertion covers links, local README, and personal inventory together.
func TestReconcilePublicLinksOnlyExactPhysicalPaths(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.ModelDatasetSource{},
		&model.Dataset{},
		&model.ModelDownload{},
		&model.ModelDatasetDiscovery{},
	); err != nil {
		t.Fatal(err)
	}
	download := model.ModelDownload{
		Name:      "owner/model",
		Source:    model.ModelSourceHuggingFace,
		Category:  model.DownloadCategoryModel,
		Path:      "public/Models/owner/model/huggingface/default",
		Status:    model.ModelDownloadStatusReady,
		CreatorID: 1,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	exact := model.Dataset{
		Name: "owner/model", URL: "shared/Models/owner/model/huggingface/default", Type: model.DataTypeModel,
	}
	mismatch := model.Dataset{
		Name: "owner/model", URL: "homes/user/model", Type: model.DataTypeModel,
	}
	if err := db.Create(&exact).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&mismatch).Error; err != nil {
		t.Fatal(err)
	}

	directory := t.TempDir()
	mustWriteFile(t, filepath.Join(directory, "README.md"), "# local README")
	now := time.Now().Round(time.Second)
	report, err := ReconcilePublic(context.Background(), db, []Candidate{{
		Path:         exact.URL,
		AbsolutePath: directory,
		Type:         model.DataTypeModel,
		Name:         "model",
		Evidence:     model.ModelDatasetDiscoveryEvidence{HasConfig: true, HasReadme: true, WeightFiles: 1},
	}}, &ReconcileOptions{
		Apply:                true,
		LogicalPublicPrefix:  "public",
		PhysicalPublicPrefix: "shared",
		MaxReadmeBytes:       1024,
		Now:                  now,
	})
	if err != nil {
		t.Fatalf("ReconcilePublic() error = %v", err)
	}
	if report.DatasetLinks != 1 || report.ReadmesFromStorage != 1 {
		t.Fatalf("report = %#v", report)
	}
	if err := db.First(&exact, exact.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&mismatch, mismatch.ID).Error; err != nil {
		t.Fatal(err)
	}
	if exact.ModelDatasetSourceID == nil {
		t.Fatal("exact path dataset was not linked")
	}
	if mismatch.ModelDatasetSourceID != nil {
		t.Fatal("same-name dataset at another path must not be linked")
	}
	var personalDiscovery model.ModelDatasetDiscovery
	if err := db.Where("path = ?", mismatch.URL).First(&personalDiscovery).Error; err != nil {
		t.Fatalf("personal dataset path was not inventoried: %v", err)
	}
	if personalDiscovery.Scope != model.ModelDatasetDiscoveryScopeUser ||
		personalDiscovery.DatasetID == nil || *personalDiscovery.DatasetID != mismatch.ID {
		t.Fatalf("personal discovery = %#v", personalDiscovery)
	}
	var source model.ModelDatasetSource
	if err := db.First(&source, *exact.ModelDatasetSourceID).Error; err != nil {
		t.Fatal(err)
	}
	if source.Readme != "# local README" {
		t.Fatalf("source.Readme = %q", source.Readme)
	}
}

func TestPhysicalStoragePathIsConfigurable(t *testing.T) {
	physical, public := PhysicalStoragePath("catalog/Models/a/b", "catalog", "shared-assets")
	if !public || physical != "shared-assets/Models/a/b" {
		t.Fatalf("PhysicalStoragePath() = %q, %v", physical, public)
	}
	physical, public = PhysicalStoragePath("user/alice/model", "catalog", "shared-assets")
	if public || physical != "user/alice/model" {
		t.Fatalf("personal path = %q, %v", physical, public)
	}
}

func TestReconcilePublicLinksOnePathlessHistoricalDatasetByIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:pathless_reconcile?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.ModelDatasetSource{},
		&model.Dataset{},
		&model.ModelDownload{},
		&model.ModelDatasetDiscovery{},
	); err != nil {
		t.Fatal(err)
	}
	download := model.ModelDownload{
		Name: "owner/pathless-model", Source: model.ModelSourceHuggingFace,
		Category: model.DownloadCategoryModel, Path: "public/Models/owner/pathless-model",
		Status: model.ModelDownloadStatusReady, CreatorID: 1,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	dataset := model.Dataset{Name: download.Name, URL: "", Type: model.DataTypeModel}
	if err := db.Create(&dataset).Error; err != nil {
		t.Fatal(err)
	}

	report, err := ReconcilePublic(context.Background(), db, nil, &ReconcileOptions{
		Apply: true, LogicalPublicPrefix: "public", PhysicalPublicPrefix: "shared",
		MaxReadmeBytes: 1024, Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("ReconcilePublic() error = %v", err)
	}
	if report.DatasetLinks != 1 {
		t.Fatalf("report.DatasetLinks = %d", report.DatasetLinks)
	}
	if report.PathlessDatasets != 1 {
		t.Fatalf("report.PathlessDatasets = %d", report.PathlessDatasets)
	}
	if err := db.First(&dataset, dataset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if dataset.ModelDatasetSourceID == nil {
		t.Fatal("pathless historical dataset was not linked to its unique source identity")
	}
	var discovery model.ModelDatasetDiscovery
	if err := db.Where("discovery_key = ?", "dataset:"+fmt.Sprint(dataset.ID)).First(&discovery).Error; err != nil {
		t.Fatalf("pathless discovery was not recorded: %v", err)
	}
	if discovery.Path != "" || discovery.Status != model.ModelDatasetDiscoveryStatusPathMissing {
		t.Fatalf("pathless discovery = %#v", discovery)
	}
}

//nolint:gocyclo // The integration assertion covers inferred source, README, ownership, and discovery linkage.
func TestReconcilePublicCreatesOnlyHighConfidenceSourceAndRecordsUniqueOwner(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:provenance_reconcile?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.ModelDatasetSource{}, &model.Dataset{}, &model.ModelDownload{},
		&model.ModelDatasetDiscovery{}, &model.User{},
	); err != nil {
		t.Fatal(err)
	}
	uid := "10042"
	user := model.User{
		Name: "historical-owner", Role: model.RoleUser, Status: model.StatusActive,
		Space: "user/historical-owner", Attributes: datatypes.NewJSONType(model.UserAttribute{UID: &uid}),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	mustWriteFile(t, filepath.Join(directory, "README.md"), "# local legacy model")
	candidate := Candidate{
		Path: "shared/LLM/falcon-40b-instruct", AbsolutePath: directory,
		Type: model.DataTypeModel, Name: "falcon-40b-instruct",
		Evidence: model.ModelDatasetDiscoveryEvidence{
			HasConfig: true, WeightFiles: 1, FilesystemUID: uid,
			Provider:     model.ModelDatasetProviderHuggingFace,
			RepositoryID: "tiiuae/falcon-40b-instruct", RepositoryURL: "https://huggingface.co/tiiuae/falcon-40b-instruct",
			ProvenanceSource: "git_remote", ProvenanceConfidence: provenanceConfidenceHigh,
		},
	}
	report, err := ReconcilePublic(context.Background(), db, []Candidate{candidate}, &ReconcileOptions{
		Apply: true, LogicalPublicPrefix: "public", PhysicalPublicPrefix: "shared",
		MaxReadmeBytes: 1024, Now: time.Now(),
	})
	if err != nil {
		t.Fatalf("ReconcilePublic() error = %v", err)
	}
	if report.InferredSources != 1 || report.OwnerMatches != 1 || report.ProvenanceHighConfidence != 1 {
		t.Fatalf("report = %#v", report)
	}
	var source model.ModelDatasetSource
	if err := db.Where("repository_id = ?", "tiiuae/falcon-40b-instruct").First(&source).Error; err != nil {
		t.Fatalf("inferred source not created: %v", err)
	}
	if source.Readme != "# local legacy model" {
		t.Fatalf("source.Readme = %q", source.Readme)
	}
	var discovery model.ModelDatasetDiscovery
	if err := db.Where("path = ?", candidate.Path).First(&discovery).Error; err != nil {
		t.Fatal(err)
	}
	evidence := discovery.Evidence.Data()
	if discovery.SourceID == nil || *discovery.SourceID != source.ID ||
		evidence.OwnerUserID == nil || *evidence.OwnerUserID != user.ID ||
		evidence.OwnerUsername != user.Name {
		t.Fatalf("discovery = %#v, evidence = %#v", discovery, evidence)
	}
}

func TestEnrichCandidateOwnersLeavesDuplicateUIDUnassigned(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:duplicate_uid?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	uid := "10042"
	for index, name := range []string{"first-owner", "second-owner"} {
		user := model.User{
			Name: name, Role: model.RoleUser, Status: model.StatusActive,
			Space:      fmt.Sprintf("user/%d", index),
			Attributes: datatypes.NewJSONType(model.UserAttribute{UID: &uid}),
		}
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	candidates := []Candidate{{Evidence: model.ModelDatasetDiscoveryEvidence{FilesystemUID: uid}}}
	matches, err := enrichCandidateOwners(context.Background(), db, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if matches != 0 || candidates[0].Evidence.OwnerUserID != nil || candidates[0].Evidence.OwnerUsername != "" {
		t.Fatalf("ambiguous UID was assigned: %#v", candidates[0].Evidence)
	}
}
