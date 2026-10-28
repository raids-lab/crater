//go:build cli_compatibility

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

package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type releasedCompatibilityCLI struct{ name, path string }

const releasedV4 = "v4"

func (binary releasedCompatibilityCLI) command(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	if !filepath.IsAbs(binary.path) {
		t.Fatal("compatibility binaries must use absolute paths")
	}
	info, err := os.Stat(binary.path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("compatibility binary is not a regular file: %s; %v", binary.path, err)
	}
	//nolint:gosec // G204: the operator explicitly supplies trusted test binaries; no shell or HTTP input is used.
	return exec.CommandContext(t.Context(), binary.path, args...)
}

type releasedCompatibilityCase struct {
	name     string
	args     []string
	key      string
	requests [2]int
}

type releasedCLIFixture struct {
	db           *gorm.DB
	mode         atomic.Int32
	traceMu      sync.Mutex
	trace        []string
	server       *httptest.Server
	humanHeaders map[string]string
	legacyOutput map[string]map[string]map[string]any
}

// TestReleasedCLICompatibility runs real binaries against actual handlers and
// isolated SQLite/Kubernetes fixtures. It never reads user credentials or calls
// a cluster. The explicit target requires both binaries instead of skipping.
func TestReleasedCLICompatibility(t *testing.T) {
	binaries := []releasedCompatibilityCLI{
		{releasedV4, os.Getenv("OLD_CLI_BIN")}, {"v5", os.Getenv("NEW_CLI_BIN")},
	}
	for _, binary := range binaries {
		if binary.path == "" {
			t.Fatal("OLD_CLI_BIN and NEW_CLI_BIN are required")
		}
		output, err := binary.command(t, "version", "--json").Output()
		if err != nil {
			t.Fatalf("read %s version: %v", binary.name, err)
		}
		assertReleasedCLIVersion(t, binary.name, output)
		t.Logf("%s binary: %s", binary.name, output)
	}
	fixture := newReleasedCLIFixture(t)
	defer fixture.server.Close()

	for _, binary := range binaries {
		t.Run(binary.name+"/compatibility", func(t *testing.T) {
			command := binary.command(t, "compatibility", "--platform", fixture.server.URL, "--json", "--no-interactive")
			output, err := command.Output()
			if err != nil || !bytes.Contains(output, []byte(`"status": "compatible"`)) {
				t.Fatalf("compatibility diagnostic disagrees with supported clients: %s; %v", output, err)
			}
		})
	}
	cases := []releasedCompatibilityCase{
		{"templates", []string{"template", "ls"}, "templates", [2]int{1, 2}},
		{"builds", []string{"image", "build", "ls", "--all-pages"}, "builds", [2]int{1, 2}},
		{"images", []string{"image", "ls", "--all-pages"}, "images", [2]int{1, 2}},
		{"downloads", []string{"model-download", "ls", "--all-pages", "--page-size", "100"}, "downloads", [2]int{3, 3}},
		{"pods", []string{"node", "pods", "node-a", "--all-namespaces", "--all-pages"}, "pods", [2]int{1, 2}},
	}
	for _, scenario := range []int32{0, 3, 1, 2} {
		fixture.mode.Store(scenario)
		if scenario == 1 {
			for _, entity := range []any{&model.Jobtemplate{}, &model.Kaniko{}, &model.Image{}, &model.ModelDownload{}} {
				if err := fixture.db.Where("1 = 1").Delete(entity).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
		for binaryIndex, binary := range binaries {
			for _, tc := range cases {
				t.Run(fmt.Sprintf("%s/%s/scenario-%d", binary.name, tc.name, scenario), func(t *testing.T) {
					fixture.checkRead(t, binary, binaryIndex, tc, scenario)
				})
			}
		}
	}
}

func cliCompatibilityResourceID(kind string, row map[string]any) string {
	switch kind {
	case "images", "builds":
		return fmt.Sprint(row["ID"])
	case "pods":
		return fmt.Sprint(row["namespace"], "/", row["name"])
	default:
		return fmt.Sprint(row["id"])
	}
}

func assertReleasedCLIVersion(t *testing.T, name string, output []byte) {
	t.Helper()
	var response struct {
		Data struct {
			Version struct {
				API int `json:"api_version"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatal(err)
	}
	want := 5
	if name == releasedV4 {
		want = 4
	}
	if response.Data.Version.API != want {
		t.Fatalf("%s binary implements API %d, want %d", name, response.Data.Version.API, want)
	}
}

func newReleasedCLIFixture(t *testing.T) *releasedCLIFixture {
	t.Helper()
	fixture := &releasedCLIFixture{db: setupKanikoListTestDB(t), humanHeaders: make(map[string]string), legacyOutput: make(map[string]map[string]map[string]any)}
	if err := fixture.db.AutoMigrate(&model.Jobtemplate{}, &model.Image{}, &model.ImageAccount{}, &model.ImageUser{}, &model.ModelDownload{}, &model.UserModelDownload{}); err != nil {
		t.Fatal(err)
	}
	createKanikoListTestUsers(t, fixture.db, "alice", "bob")
	for id := uint(1); id <= 201; id++ {
		if err := fixture.db.Create(&model.Jobtemplate{Model: gorm.Model{ID: id}, Name: fmt.Sprintf("compat-template-%03d", id), Template: "{}", UserID: 1}).Error; err != nil {
			t.Fatal(err)
		}
		createKanikoListTestRecords(t, fixture.db, model.Kaniko{Model: gorm.Model{ID: id}, UserID: 1})
		for _, row := range []any{
			&model.Image{Model: gorm.Model{ID: id}, UserID: 1, ImageLink: fmt.Sprintf("test.invalid/compat-image-%03d", id)},
			&model.ModelDownload{Model: gorm.Model{ID: id}, Name: fmt.Sprintf("compat-download-%03d", id), CreatorID: 1, Category: model.DownloadCategoryModel, Status: model.ModelDownloadStatusReady},
		} {
			if err := fixture.db.Create(row).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	createKanikoListTestRecords(t, fixture.db, model.Kaniko{Model: gorm.Model{ID: 202}, UserID: 2})
	if err := fixture.db.Create(&model.Image{Model: gorm.Model{ID: 202}, UserID: 2, ImageLink: "test.invalid/foreign-private-image"}).Error; err != nil {
		t.Fatal(err)
	}
	kubeClient := releasedCLICompatibilityKube(&fixture.mode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		fixture.traceMu.Lock()
		fixture.trace = append(fixture.trace, c.Request.URL.String())
		fixture.traceMu.Unlock()
		util.SetJWTContext(c, util.JWTMessage{UserID: 1, AccountID: 1, Username: "alice", RolePlatform: model.RoleUser, RoleAccount: model.RoleUser})
		if fixture.mode.Load() == 2 || (fixture.mode.Load() == 3 && (c.Query("page") == "2" || c.Query("cursor") != "")) {
			resputil.HandleError(c, bizerr.Internal.DatabaseError.New("fixture list failed"))
			c.Abort()
			return
		}
		c.Next()
	})
	router.GET("/api/v1/jobtemplate/list", (&handler.JobTemplateMgr{}).UserListJobTemplates)
	router.GET("/api/v1/images/kaniko", func(c *gin.Context) {
		owner := util.GetToken(c).UserID
		(&ImagePackMgr{}).listKanikoPageWithDB(c, fixture.db, &owner)
	})
	router.GET("/api/v1/images/image", (&ImagePackMgr{}).UserListImage)
	router.GET("/api/v1/model-download/models/downloads", (&handler.ModelDownloadMgr{}).ListDownloads)
	router.GET("/api/cli/compatibility", (&handler.CLICompatibilityMgr{}).GetCompatibility)
	handler.NewNodeMgr(&handler.RegisterConfig{KubeClient: kubeClient}).RegisterProtected(router.Group("/api/v1/nodes"))
	fixture.server = httptest.NewServer(router)
	return fixture
}

func releasedCLICompatibilityKube(mode *atomic.Int32) *fake.Clientset {
	kubeClient := fake.NewSimpleClientset()
	kubeClient.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		options := action.(clienttesting.ListActionImpl).GetListOptions()
		start := 0
		if options.Continue != "" {
			var err error
			start, err = strconv.Atoi(options.Continue)
			if err != nil {
				return true, nil, err
			}
		}
		end, next := 201, ""
		if mode.Load() == 1 {
			end = 0
		}
		if options.Limit > 0 && start+int(options.Limit) < end {
			end = start + int(options.Limit)
			next = strconv.Itoa(end)
		}
		pods := make([]corev1.Pod, 0, end-start)
		for i := start; i < end; i++ {
			pods = append(pods, corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("compat-pod-%03d", i+1), Namespace: "test"}})
		}
		return true, &corev1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "100", Continue: next}, Items: pods}, nil
	})
	return kubeClient
}

func (fixture *releasedCLIFixture) checkRead(t *testing.T, binary releasedCompatibilityCLI, binaryIndex int, tc releasedCompatibilityCase, scenario int32) {
	t.Helper()
	fixture.traceMu.Lock()
	fixture.trace = nil
	fixture.traceMu.Unlock()
	args := append(append([]string{}, tc.args...), "--json", "--no-interactive")
	command := binary.command(t, args...)
	home := t.TempDir()
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "XDG_CONFIG_HOME=" + home, "CRATER_LANG=en", "CRATER_TEST_SANDBOX=1", "CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL=" + fixture.server.URL}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if scenario == 2 || (scenario == 3 && tc.requests[binaryIndex] > 1) {
		if err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), "fixture list failed") {
			t.Fatalf("failure must not produce a successful partial list: %v; %s; %s", err, stdout.String(), stderr.String())
		}
		return
	}
	if err != nil || stderr.Len() != 0 {
		t.Fatalf("CLI read failed: %v; %s", err, stderr.String())
	}
	want := 201
	if scenario == 1 {
		want = 0
	}
	items := assertReleasedCLIJSON(t, stdout.Bytes(), tc.key, want)
	if scenario == 0 {
		fixture.assertCompleteOutput(t, binary.name, tc.name, items)
	}
	fixture.assertRequests(t, binary, binaryIndex, tc, scenario, len(items))
	if scenario == 0 {
		fixture.assertHumanOutput(t, binary, tc, home)
	}
	if directory := os.Getenv("CLI_COMPATIBILITY_OUTPUT"); directory != "" {
		name := fmt.Sprintf("%s-%s-%d.json", binary.name, tc.name, scenario)
		if err := os.WriteFile(filepath.Join(directory, name), stdout.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func (fixture *releasedCLIFixture) assertCompleteOutput(t *testing.T, cliName, kind string, items []json.RawMessage) {
	t.Helper()
	want := 201
	rows := make(map[string]map[string]any, len(items))
	for _, raw := range items {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			t.Fatal(err)
		}
		rows[cliCompatibilityResourceID(kind, row)] = row
	}
	if len(rows) != want {
		t.Fatalf("duplicate or missing resource identity: got %d unique rows, want %d", len(rows), want)
	}
	if cliName == releasedV4 {
		fixture.legacyOutput[kind] = rows
	} else if !reflect.DeepEqual(rows, fixture.legacyOutput[kind]) {
		t.Error("complete JSON resource fields or values differ between old and new CLI")
	}
}

func (fixture *releasedCLIFixture) assertHumanOutput(t *testing.T, binary releasedCompatibilityCLI, tc releasedCompatibilityCase, home string) {
	t.Helper()
	command := binary.command(t, append(append([]string{}, tc.args...), "--no-interactive")...)
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "XDG_CONFIG_HOME=" + home, "CRATER_LANG=en", "CRATER_TEST_SANDBOX=1", "CRATER_TEST_SANDBOX_HTTP=passthrough", "CRATER_TEST_SANDBOX_PLATFORM_URL=" + fixture.server.URL}
	humanOutput, err := command.Output()
	lines := strings.Split(strings.TrimSpace(string(humanOutput)), "\n")
	if err != nil || len(lines) < 202 || bytes.Contains(humanOutput, []byte("foreign-private-image")) {
		t.Fatalf("human output lost complete rows: lines=%d; %v", len(lines), err)
	}
	if binary.name == releasedV4 {
		fixture.humanHeaders[tc.name] = strings.Join(strings.Fields(lines[0]), " ")
	} else if strings.Join(strings.Fields(lines[0]), " ") != fixture.humanHeaders[tc.name] {
		t.Errorf("table header changed: %q, want %q", lines[0], fixture.humanHeaders[tc.name])
	}
	if directory := os.Getenv("CLI_COMPATIBILITY_OUTPUT"); directory != "" {
		if err := os.WriteFile(filepath.Join(directory, binary.name+"-"+tc.name+"-table.txt"), humanOutput, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func (fixture *releasedCLIFixture) assertRequests(t *testing.T, binary releasedCompatibilityCLI, binaryIndex int, tc releasedCompatibilityCase, scenario int32, count int) {
	t.Helper()
	fixture.traceMu.Lock()
	t.Logf("%s %s: count=%d requests=%v", binary.name, strings.Join(tc.args, " "), count, fixture.trace)
	if (scenario == 0 || scenario == 3) && len(fixture.trace) != tc.requests[binaryIndex] {
		t.Errorf("expected %d real list requests, got %v", tc.requests[binaryIndex], fixture.trace)
	}
	fixture.traceMu.Unlock()
}

func assertReleasedCLIJSON(t *testing.T, output []byte, key string, want int) []json.RawMessage {
	t.Helper()
	var response struct{ Data map[string]json.RawMessage }
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatal(err)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(response.Data[key], &items); err != nil || items == nil {
		t.Fatalf("CLI resource array changed: %s; %v", string(output), err)
	}
	if len(items) != want || strings.Contains(string(output), "bob") {
		t.Fatalf("incomplete result or foreign-owner leak: got %d, want %d", len(items), want)
	}
	return items
}
