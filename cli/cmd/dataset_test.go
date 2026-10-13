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

package cmd

import (
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
)

func TestDatasetShareListPathsMatchBackendRoutes(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
		want   string
	}{
		{name: "shared users", suffix: datasetUsersInSuffix, want: api.DatasetPrefix + "/170/usersIn"},
		{name: "shared queues", suffix: datasetQueuesInSuffix, want: api.DatasetPrefix + "/170/queuesIn"},
		{name: "users not shared", suffix: datasetUsersOutSuffix, want: api.DatasetPrefix + "/170/usersNotIn"},
		{name: "queues not shared", suffix: datasetQueuesOutSuffix, want: api.DatasetPrefix + "/170/queuesNotIn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := datasetShareListPath(170, tt.suffix); got != tt.want {
				t.Fatalf("dataset share path = %q, want %q", got, tt.want)
			}
		})
	}
}
