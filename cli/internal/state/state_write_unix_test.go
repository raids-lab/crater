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

package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestManagerSaveWriteFailurePreservesState(t *testing.T) {
	const helperPathEnv = "CRATER_STATE_WRITE_FAILURE_HELPER_PATH"
	if path := os.Getenv(helperPathEnv); path != "" {
		m := &Manager{Path: path}
		if err := m.Load(); err != nil {
			t.Fatal(err)
		}
		m.State.AuthInfos[0].Token = strings.Repeat("replacement-token", 128)
		var limit syscall.Rlimit
		if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
			t.Fatal(err)
		}
		// Restrict only this child, forcing a real partial write without filling the disk.
		signal.Ignore(syscall.SIGXFSZ)
		limit.Cur = 96
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
			t.Fatal(err)
		}
		if err := m.Save(); !errors.Is(err, syscall.EFBIG) {
			t.Fatalf("Save error = %v, want file size limit failure", err)
		}
		return
	}

	m := &Manager{
		Path: filepath.Join(t.TempDir(), "state.json"),
		State: State{
			AuthInfos: []AuthInfo{
				{Username: "alice", Token: "original-token"},
				{Username: "bob", Token: "other-token"},
			},
			ActiveContext: ActiveContext{Username: "alice", Method: "ldap"},
			Language:      "zh-CN",
		},
	}
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(m.Path)
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestManagerSaveWriteFailurePreservesState$")
	child.Env = append(os.Environ(), helperPathEnv+"="+m.Path)
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("write-failure subprocess: %v\n%s", err, output)
	}
	got, err := os.ReadFile(m.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) || !json.Valid(got) {
		t.Fatal("failed write changed or corrupted the original credentials")
	}
	entries, err := os.ReadDir(filepath.Dir(m.Path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Fatal("failed write left temporary credentials on disk")
	}
}
