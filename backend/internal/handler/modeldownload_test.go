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

package handler

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/util"
)

type testSQLStateError struct {
	state string
}

func (err testSQLStateError) Error() string {
	return "database error with SQLSTATE " + err.state
}

func (err testSQLStateError) SQLState() string {
	return err.state
}

func TestDownloadActionRevisionDistinguishesOmittedAndDefault(t *testing.T) {
	var omitted DownloadActionReq
	if err := json.Unmarshal([]byte(`{"token":"temporary"}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if omitted.Revision != nil {
		t.Fatalf("omitted revision should preserve the failed record revision: %#v", omitted.Revision)
	}

	var defaultBranch DownloadActionReq
	if err := json.Unmarshal([]byte(`{"revision":""}`), &defaultBranch); err != nil {
		t.Fatal(err)
	}
	if defaultBranch.Revision == nil || *defaultBranch.Revision != "" {
		t.Fatalf("explicit empty revision should select the source default: %#v", defaultBranch.Revision)
	}
}

func TestNormalizeRetryRevision(t *testing.T) {
	revision := "  master  "
	action := DownloadActionReq{Revision: &revision}
	if err := normalizeRetryRevision(&action); err != nil {
		t.Fatal(err)
	}
	if action.Revision == nil || *action.Revision != "master" {
		t.Fatalf("retry revision should be trimmed: %#v", action.Revision)
	}

	tooLong := strings.Repeat("x", maxDownloadRevisionLength+1)
	if err := normalizeRetryRevision(&DownloadActionReq{Revision: &tooLong}); err == nil {
		t.Fatal("overlong retry revision should be rejected")
	}
}

func TestDownloadRelationshipDefaults(t *testing.T) {
	creator := util.JWTMessage{UserID: 7}
	download := &model.ModelDownload{CreatorID: creator.UserID}
	if got := convertDownloadToResp(download, creator).Relation; got != ModelDownloadRelationCreator {
		t.Fatalf("creator relation = %q", got)
	}
	if got := convertDownloadToResp(download, util.JWTMessage{UserID: 8}).Relation; got != ModelDownloadRelationNone {
		t.Fatalf("unassociated user relation = %q", got)
	}
}

func TestModelDownloadResponseKeepsReferenceCountCompatibilityAlias(t *testing.T) {
	download := &model.ModelDownload{CreatorID: 7, ReferenceCount: 3}
	response := convertDownloadToResp(download, util.JWTMessage{UserID: 7})
	if response.ReferenceCount != response.RequesterCount || response.RequesterCount != 3 {
		t.Fatalf("referenceCount/requesterCount = %d/%d, want 3/3",
			response.ReferenceCount, response.RequesterCount)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"referenceCount":3`, `"requesterCount":3`} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("response JSON %s does not contain compatibility field %s", encoded, field)
		}
	}
}

func TestApplyDownloadRequestersRecordsDemandWithoutChangingPublicAccess(t *testing.T) {
	responses := []ModelDownloadResp{
		{ID: 10, Relation: ModelDownloadRelationCreator},
		{ID: 11, Relation: ModelDownloadRelationNone},
	}
	associations := []*model.UserModelDownload{
		{ModelDownloadID: 10, UserID: 1, User: model.User{Name: "alice", Nickname: "Alice"}},
		{ModelDownloadID: 10, UserID: 2, User: model.User{Name: "bob"}},
		{ModelDownloadID: 10, UserID: 3}, // Keep deleted users in the demand count.
		{ModelDownloadID: 11, UserID: 1, User: model.User{Name: "alice", Nickname: "Alice"}},
	}

	applyDownloadRequesters(responses, associations, util.JWTMessage{UserID: 1})

	if len(responses) != 2 {
		t.Fatalf("response count = %d, want 2", len(responses))
	}
	creatorResponse := responses[0]
	if creatorResponse.Relation != ModelDownloadRelationCreator {
		t.Fatalf("creator relation changed to %q", creatorResponse.Relation)
	}
	if creatorResponse.RequesterCount != 3 || len(creatorResponse.Requesters) != 2 {
		t.Fatalf("requesters = %d/%d, want 3 recorded and 2 visible", creatorResponse.RequesterCount,
			len(creatorResponse.Requesters))
	}
	visibleRequesters := creatorResponse.Requesters
	if len(visibleRequesters) != 2 {
		t.Fatalf("visible requester count = %d, want 2", len(visibleRequesters))
	}
	if visibleRequesters[1] != (model.UserInfo{Username: "bob", Nickname: "bob"}) {
		t.Fatalf("empty nickname should fall back to username: %#v", visibleRequesters[1])
	}
	requesterResponse := responses[1]
	if requesterResponse.Relation != ModelDownloadRelationSubmitted || !requesterResponse.CanViewLogs {
		t.Fatalf("requesting user context not applied: %#v", requesterResponse)
	}
}

func TestCheckRetryRevisionConflictIncludesSoftDeletedRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:retry_revision_conflict?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}

	active := model.ModelDownload{
		Name: "owner/model", Source: model.ModelSourceHuggingFace,
		Category: model.DownloadCategoryModel, Revision: "old", Path: "public/Models/owner/model",
		Status: model.ModelDownloadStatusFailed, CreatorID: 1,
	}
	softDeleted := active
	softDeleted.Revision = "new"
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&softDeleted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&softDeleted).Error; err != nil {
		t.Fatal(err)
	}

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	revision := "new"
	err = checkRetryRevisionConflict(ginContext, query.Use(db), &active, &revision)
	if !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("expected a conflict for soft-deleted revision, got %v", err)
	}
}

func TestRetryUpdateDuplicateIsConflict(t *testing.T) {
	if err := retryUpdateError(gorm.ErrDuplicatedKey); !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("duplicate retry update should be a conflict: %v", err)
	}
	if err := retryUpdateError(testSQLStateError{state: "23505"}); !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("PostgreSQL unique violation should be a conflict: %v", err)
	}
	if err := retryUpdateError(gorm.ErrInvalidData); !errors.Is(err, bizerr.Internal.Base) {
		t.Fatalf("non-duplicate retry update should remain internal: %v", err)
	}
}

func TestRestoredReadyDownloadDoesNotSubmitJob(t *testing.T) {
	for _, testCase := range []struct {
		status model.ModelDownloadStatus
		want   bool
	}{
		{status: model.ModelDownloadStatusReady, want: false},
		{status: model.ModelDownloadStatusPending, want: true},
		{status: model.ModelDownloadStatusDownloading, want: true},
		{status: model.ModelDownloadStatusPaused, want: true},
		{status: model.ModelDownloadStatusFailed, want: true},
	} {
		download := &model.ModelDownload{Status: testCase.status}
		if got := shouldSubmitRestoredDownload(download); got != testCase.want {
			t.Fatalf("status %s: shouldSubmitRestoredDownload() = %v, want %v", testCase.status, got, testCase.want)
		}
	}
}

func TestModelScopeDownloadCommandUsesArgumentArray(t *testing.T) {
	download := &model.ModelDownload{
		Name:     "Qwen/Qwen3-32B",
		Source:   model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel,
		Revision: `main; touch /tmp/injected; $(id)`,
	}

	command := (&ModelDownloadMgr{}).buildDownloadCommand(download, "Qwen3-32B")

	for _, expected := range []string{
		`args = ["modelscope", "download", resource_flag, repo_id]`,
		`args.extend(["--revision", revision])`,
		"subprocess.run(args, check=True)",
		"revision_not_found",
		"available revisions",
		"modelscope==" + modelScopeVersion,
		"modelscope-hub==" + modelScopeHubVersion,
	} {
		if !strings.Contains(command, expected) {
			t.Fatalf("download command does not contain %q", expected)
		}
	}
	if strings.Contains(command, "modelscope download --model Qwen/Qwen3-32B --revision") {
		t.Fatal("download command interpolates arguments into a shell command")
	}
	if strings.Contains(command, "pip install -q modelscope") {
		t.Fatal("download command performs an unpinned runtime installation")
	}
	if strings.Contains(command, "%!") {
		t.Fatalf("download command contains an unresolved format directive: %s", command)
	}
}

func TestShouldDisableHuggingFaceXet(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		source   model.ModelSource
		endpoint string
		want     bool
	}{
		{
			name:     "hugging face mirror",
			source:   model.ModelSourceHuggingFace,
			endpoint: "https://hf-mirror.com",
			want:     true,
		},
		{
			name:     "official hugging face endpoint",
			source:   model.ModelSourceHuggingFace,
			endpoint: "https://huggingface.co/",
			want:     false,
		},
		{
			name:     "modelscope download",
			source:   model.ModelSourceModelScope,
			endpoint: "https://hf-mirror.com",
			want:     false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := shouldDisableHuggingFaceXet(testCase.source, testCase.endpoint); got != testCase.want {
				t.Fatalf("shouldDisableHuggingFaceXet(%q, %q) = %v, want %v",
					testCase.source, testCase.endpoint, got, testCase.want)
			}
		})
	}
}

func TestModelDownloadStoragePathUsesCanonicalShortPath(t *testing.T) {
	base := "public/Models"
	name := "Qwen/Qwen3-32B"
	path := modelDownloadStoragePath(base, name)

	if path != filepath.Join(base, name) {
		t.Fatalf("unexpected canonical storage path: %s", path)
	}
}

func TestFindReadyLogicalDownloadReusesHistoricalPath(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:logical_download_reuse?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}

	longPath := model.ModelDownload{
		Name: "Qwen/Qwen3-32B", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "master",
		Path: "public/Models/Qwen/Qwen3-32B/modelscope/fc613b4dfd67", Status: model.ModelDownloadStatusReady,
		CreatorID: 1,
	}
	shortPath := model.ModelDownload{
		Name: "Qwen/Qwen3-32B", Source: model.ModelSourceHuggingFace,
		Category: model.DownloadCategoryModel, Revision: "main",
		Path: "public/Models/Qwen/Qwen3-32B", Status: model.ModelDownloadStatusReady,
		CreatorID: 2,
	}
	if err := db.Create(&longPath).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&shortPath).Error; err != nil {
		t.Fatal(err)
	}

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	got, err := (&ModelDownloadMgr{}).findReadyOrOngoingDownload(
		ginContext, query.Use(db), "Qwen/Qwen3-32B", model.ModelSourceModelScope,
		model.DownloadCategoryModel, "master",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != longPath.ID {
		t.Fatalf("findReadyOrOngoingDownload() = %#v, want exact historical record %d", got, longPath.ID)
	}
}

func TestFindOngoingLogicalDownloadDoesNotReuseReadyOtherSource(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:logical_download_exact_ongoing?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}

	readyOtherSource := model.ModelDownload{
		Name: "Qwen/Qwen3-32B", Source: model.ModelSourceHuggingFace,
		Category: model.DownloadCategoryModel, Revision: "main",
		Path: "public/Models/Qwen/Qwen3-32B", Status: model.ModelDownloadStatusReady,
		CreatorID: 1,
	}
	ongoingExact := model.ModelDownload{
		Name: "Qwen/Qwen3-32B", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "master",
		Path: "public/Models/Qwen/Qwen3-32B/modelscope/fc613b4dfd67", Status: model.ModelDownloadStatusDownloading,
		CreatorID: 2,
	}
	if err := db.Create(&readyOtherSource).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&ongoingExact).Error; err != nil {
		t.Fatal(err)
	}

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	got, err := (&ModelDownloadMgr{}).findReadyOrOngoingDownload(
		ginContext, query.Use(db), "Qwen/Qwen3-32B", model.ModelSourceModelScope,
		model.DownloadCategoryModel, "master",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != ongoingExact.ID {
		t.Fatalf("findReadyOrOngoingDownload() = %#v, want exact ongoing record %d", got, ongoingExact.ID)
	}

	got, err = (&ModelDownloadMgr{}).findReadyOrOngoingDownload(
		ginContext, query.Use(db), "Qwen/Qwen3-32B", model.ModelSourceModelScope,
		model.DownloadCategoryModel, "v2",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("cross-revision request reused download %#v", got)
	}
	if err := checkLogicalDownloadConflict(
		ginContext, query.Use(db), "Qwen/Qwen3-32B", model.ModelSourceModelScope,
		model.DownloadCategoryModel, "v2",
	); !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("cross-revision request should conflict with canonical storage, got %v", err)
	}
}

func TestAssociateUserWithLogicalDownloadIncrementsReferenceOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:logical_download_reference?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}, &model.UserModelDownload{}); err != nil {
		t.Fatal(err)
	}

	download := model.ModelDownload{
		Name: "Qwen/Qwen3-32B", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "master",
		Path: "public/Models/Qwen/Qwen3-32B/modelscope/fc613b4dfd67", Status: model.ModelDownloadStatusReady,
		CreatorID: 1, ReferenceCount: 1,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserModelDownload{UserID: 1, ModelDownloadID: download.ID}).Error; err != nil {
		t.Fatal(err)
	}

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	mgr := &ModelDownloadMgr{}
	for range 2 {
		if err := mgr.associateUserWithDownload(ginContext, query.Use(db), 2, download.ID); err != nil {
			t.Fatal(err)
		}
	}

	var updated model.ModelDownload
	if err := db.First(&updated, download.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.ReferenceCount != 2 {
		t.Fatalf("reference count = %d, want 2 unique users", updated.ReferenceCount)
	}
	var associations int64
	if err := db.Model(&model.UserModelDownload{}).
		Where("model_download_id = ? AND user_id = ?", download.ID, 2).
		Count(&associations).Error; err != nil {
		t.Fatal(err)
	}
	if associations != 1 {
		t.Fatalf("user association count = %d, want 1", associations)
	}
}

func TestLogicalDownloadConflictIncludesOtherSourceAndSoftDeletedRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:logical_download_conflict?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ModelDownload{}); err != nil {
		t.Fatal(err)
	}

	download := model.ModelDownload{
		Name: "Qwen/Qwen3-32B", Source: model.ModelSourceModelScope,
		Category: model.DownloadCategoryModel, Revision: "master",
		Path: "public/Models/Qwen/Qwen3-32B/modelscope/fc613b4dfd67", Status: model.ModelDownloadStatusFailed,
		CreatorID: 1,
	}
	if err := db.Create(&download).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&download).Error; err != nil {
		t.Fatal(err)
	}

	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	err = checkLogicalDownloadConflict(
		ginContext, query.Use(db), "Qwen/Qwen3-32B", model.ModelSourceHuggingFace,
		model.DownloadCategoryModel, "main",
	)
	if !errors.Is(err, bizerr.Conflict.Base) {
		t.Fatalf("expected cross-source soft-deleted conflict, got %v", err)
	}

	if err := checkLogicalDownloadConflict(
		ginContext, query.Use(db), "Qwen/Qwen3-32B", model.ModelSourceModelScope,
		model.DownloadCategoryModel, "master",
	); err != nil {
		t.Fatalf("exact historical identity should be retried in place, got %v", err)
	}
}

func TestDownloadImagePullSecrets(t *testing.T) {
	if got := downloadImagePullSecrets(""); got != nil {
		t.Fatalf("public downloader image should not require pull secrets: %#v", got)
	}

	got := downloadImagePullSecrets("internal-registry")
	if len(got) != 1 || got[0].Name != "internal-registry" {
		t.Fatalf("private downloader image should use the configured pull secret: %#v", got)
	}
}

func TestDownloadTokenEnvIsSourceSpecificAndEphemeral(t *testing.T) {
	hfEnv := downloadTokenEnv(model.ModelSourceHuggingFace, "secret")
	if len(hfEnv) != 2 || hfEnv[0].Value != "secret" {
		t.Fatalf("unexpected HuggingFace token environment: %#v", hfEnv)
	}

	modelScopeEnv := downloadTokenEnv(model.ModelSourceModelScope, "secret")
	if len(modelScopeEnv) != 1 || modelScopeEnv[0].Name != "MODELSCOPE_API_TOKEN" || modelScopeEnv[0].Value != "secret" {
		t.Fatalf("unexpected ModelScope token environment: %#v", modelScopeEnv)
	}

	if env := downloadTokenEnv(model.ModelSourceModelScope, ""); env != nil {
		t.Fatalf("empty token should not create environment variables: %#v", env)
	}
}

func TestTruncateDownloadLogTail(t *testing.T) {
	logs := strings.Repeat("old line\n", 20) + "last line\n"
	truncated := truncateDownloadLogTail(logs, 32)

	if len(truncated) > 32 || strings.Contains(truncated, "old line\nold line\nold line\nold line") || !strings.HasSuffix(truncated, "last line\n") {
		t.Fatalf("unexpected truncated log tail: %q", truncated)
	}
}

func TestDownloadRecordDeletionIsAdminOnly(t *testing.T) {
	download := &model.ModelDownload{CreatorID: 42}
	creator := util.JWTMessage{UserID: 42, RolePlatform: model.RoleUser}
	creatorResponse := convertDownloadToResp(download, creator)
	if !creatorResponse.CanManage || creatorResponse.CanDelete || canDeleteDownload(creator) {
		t.Fatalf("creator permissions = %#v", creatorResponse)
	}

	admin := util.JWTMessage{UserID: 7, RolePlatform: model.RoleAdmin}
	adminResponse := convertDownloadToResp(download, admin)
	if !adminResponse.CanManage || !adminResponse.CanDelete || !canDeleteDownload(admin) {
		t.Fatalf("admin permissions = %#v", adminResponse)
	}
}
