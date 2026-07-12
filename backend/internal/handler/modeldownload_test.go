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
	"path/filepath"
	"strings"
	"testing"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
)

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

func TestModelDownloadStoragePathSeparatesSourceAndRevision(t *testing.T) {
	base := "public/Models"
	name := "Qwen/Qwen3-32B"
	defaultPath := modelDownloadStoragePath(base, name, model.ModelSourceModelScope, "")
	huggingFacePath := modelDownloadStoragePath(base, name, model.ModelSourceHuggingFace, "")
	revisionPath := modelDownloadStoragePath(base, name, model.ModelSourceModelScope, "v2")

	if defaultPath != filepath.Join(base, name, "modelscope", "default") {
		t.Fatalf("unexpected default storage path: %s", defaultPath)
	}
	if defaultPath == huggingFacePath || defaultPath == revisionPath || huggingFacePath == revisionPath {
		t.Fatalf("source/revision paths must be distinct: %q %q %q", defaultPath, huggingFacePath, revisionPath)
	}
	if revisionPath != modelDownloadStoragePath(base, name, model.ModelSourceModelScope, "v2") {
		t.Fatal("revision storage path must be deterministic")
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
