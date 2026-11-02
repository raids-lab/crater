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

//go:build darwin || linux

package cmd

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"syscall"
	"testing"

	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
)

func TestAccountUseWriteFailurePreservesCredentials(t *testing.T) {
	const helperEnv = "CRATER_ACCOUNT_WRITE_FAILURE_HELPER"
	if os.Getenv(helperEnv) != "1" {
		child := exec.Command(os.Args[0], "-test.run=^TestAccountUseWriteFailurePreservesCredentials$")
		child.Env = append(os.Environ(), helperEnv+"=1")
		if output, err := child.CombinedOutput(); err != nil {
			t.Fatalf("write-failure subprocess: %v\n%s", err, output)
		}
		return
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"code":0,"data":{"accessToken":"scoped-access","context":{"queue":"research"}}}`)
	}))
	t.Cleanup(server.Close)
	want := setupAccountSession(t, server.URL)
	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		t.Fatal(err)
	}
	// Keep the size limit and signal handling isolated to the subprocess.
	signal.Ignore(syscall.SIGXFSZ)
	limit.Cur = 96
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
		t.Fatal(err)
	}
	stdout, err := captureAccountOutput(t, func() error { return runAccountUse(accountUseCmd, []string{"research"}) })
	var cliErr *clierror.Error
	if !errors.As(err, &cliErr) || cliErr.Category != errorcodes.CategorySystem || cliErr.Code != errorcodes.ErrConfigWriteFailed || stdout != "" {
		t.Fatalf("write failure: error=%v stdout=%q, want config error and no success output", err, stdout)
	}
	got, err := session.LoadState()
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("failed write changed saved credentials or local state")
	}
}
