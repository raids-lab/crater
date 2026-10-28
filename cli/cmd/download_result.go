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
	"fmt"
	"os"
	"strings"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
)

// Retain only non-sensitive request identity through creation and polling.
type downloadCreateResult struct {
	source, revision, message string
}

func (result downloadCreateResult) reuseWarning(download *api.ModelDownloadResp) string {
	if download == nil {
		return ""
	}
	sourceChanged := normalizeDownloadSource(result.source) != download.Source
	revisionChanged := result.revision != download.Revision
	if !sourceChanged && !revisionChanged {
		return ""
	}
	var reasons []string
	if sourceChanged {
		reasons = append(reasons, i18n.T("download_requested_source_not_used"))
	}
	if revisionChanged {
		reasons = append(reasons, i18n.T("download_requested_revision_not_downloaded"))
	}
	return i18n.T("download_reuse_warning",
		normalizeDownloadSource(result.source), downloadRevisionLabel(result.revision),
		download.Source, downloadRevisionLabel(download.Revision), download.Status, download.Path,
		strings.Join(reasons, " "),
	)
}

func downloadRevisionLabel(revision string) string {
	if revision == "" {
		return i18n.T("download_revision_unspecified")
	}
	return revision
}

func (result downloadCreateResult) write(download *api.ModelDownloadResp) error {
	message := result.message
	if warning := result.reuseWarning(download); warning != "" {
		if message != "" {
			message += "\n"
		}
		message += warning
	}
	if outputJSON {
		envelope := output.SuccessEnvelope(map[string]interface{}{"download": download})
		if message != "" {
			envelope["message"] = message
		}
		return output.WriteSuccessJSON(os.Stdout, envelope)
	}
	if message != "" {
		fmt.Println(message)
	}
	printDownload(download)
	return nil
}
