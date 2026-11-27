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

package vcjob

import (
	"testing"

	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
)

func TestGetLabelAndAnnotationsIncludesClientInfo(t *testing.T) {
	t.Parallel()

	token := util.JWTMessage{Username: "alice", AccountName: "default", UserID: 7}
	common := &CreateJobCommon{Name: "demo", Template: `{"version":1}`}

	tests := []struct {
		name        string
		clientInfo  util.ClientInfo
		wantType    string
		wantVersion string
	}{
		{
			name:        "cli with version",
			clientInfo:  util.ClientInfo{Type: model.ClientTypeCLI, Version: "1.4.1"},
			wantType:    string(model.ClientTypeCLI),
			wantVersion: "1.4.1",
		},
		{
			name:       "web drops version",
			clientInfo: util.ClientInfo{Type: model.ClientTypeWeb, Version: "1.4.1"},
			wantType:   string(model.ClientTypeWeb),
		},
		{
			name:       "unknown",
			clientInfo: util.ClientInfo{Type: model.ClientTypeUnknown},
			wantType:   string(model.ClientTypeUnknown),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, jobAnnotations, _ := getLabelAndAnnotations(
				CraterJobTypeJupyter, token, "base", common, nil, test.clientInfo,
			)
			if got := jobAnnotations[vcjobservice.AnnotationKeyClientType]; got != test.wantType {
				t.Fatalf("client type annotation = %q, want %q", got, test.wantType)
			}
			if got := jobAnnotations[vcjobservice.AnnotationKeyClientVersion]; got != test.wantVersion {
				t.Fatalf("client version annotation = %q, want %q", got, test.wantVersion)
			}
		})
	}
}

func TestJobClientVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *model.Job
		want string
	}{
		{name: "nil job", want: ""},
		{
			name: "cli with version",
			job:  &model.Job{ClientType: model.ClientTypeCLI, ClientVersion: ptr.To("1.4.1")},
			want: "1.4.1",
		},
		{
			name: "cli without version",
			job:  &model.Job{ClientType: model.ClientTypeCLI},
			want: "",
		},
		{
			name: "web drops version",
			job:  &model.Job{ClientType: model.ClientTypeWeb, ClientVersion: ptr.To("1.4.1")},
			want: "",
		},
		{
			name: "invalid version is dropped",
			job:  &model.Job{ClientType: model.ClientTypeCLI, ClientVersion: ptr.To("../etc/passwd")},
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := jobClientVersion(test.job); got != test.want {
				t.Fatalf("jobClientVersion() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestConvertJobRespIncludesClientInfo(t *testing.T) {
	t.Parallel()

	got := convertJobResp([]*model.Job{
		{
			Name:          "demo",
			JobName:       "jpt-demo",
			User:          model.User{Name: "alice", Nickname: "Alice"},
			JobType:       model.JobTypeJupyter,
			ClientType:    model.ClientTypeCLI,
			ClientVersion: ptr.To("1.4.1"),
		},
		{
			Name:    "legacy",
			JobName: "sg-legacy",
			User:    model.User{Name: "bob", Nickname: "Bob"},
			JobType: model.JobTypeCustom,
		},
	})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ClientType != string(model.ClientTypeCLI) || got[0].ClientVersion != "1.4.1" {
		t.Fatalf("cli response = %+v", got[0])
	}
	if got[1].ClientType != string(model.ClientTypeUnknown) || got[1].ClientVersion != "" {
		t.Fatalf("legacy response = %+v", got[1])
	}
}
