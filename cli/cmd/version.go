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
	"io"
	"text/tabwriter"

	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	internalversion "github.com/raids-lab/crater/cli/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: i18n.T("version_short"),
	Long:  i18n.T("version_long"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errTooManyArgs(cmd, len(args), 0)
		}
		return nil
	},
	RunE: runVersion,
}

func validateRootArgs(cmd *cobra.Command, args []string) error {
	if rootVersion {
		if outputJSON {
			return errUsageFromIssues([]usageIssue{
				invalidIssue("version", i18n.T("err_version_json_conflict")),
			})
		}
		if len(args) > 0 {
			return errTooManyArgs(cmd, len(args), 0)
		}
		return nil
	}
	if len(args) > 0 {
		return errUnknownSubcommand(cmd, args[0])
	}
	return nil
}

func runRoot(cmd *cobra.Command, _ []string) error {
	if !rootVersion {
		return cmd.Help()
	}
	return writeShortVersionResult(cmd.OutOrStdout(), internalversion.CurrentBuildInfo())
}

func runVersion(cmd *cobra.Command, _ []string) error {
	return writeVersionResult(cmd.OutOrStdout(), outputJSON, internalversion.CurrentBuildInfo())
}

func writeVersionResult(w io.Writer, jsonOutput bool, info internalversion.BuildInfo) error {
	if jsonOutput {
		return output.WriteSuccessJSON(w, output.SuccessEnvelope(map[string]interface{}{
			"version": info,
		}))
	}

	if _, err := fmt.Fprintf(w, "%s:\n", i18n.T("version_heading")); err != nil {
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	lines := []struct {
		label string
		value interface{}
	}{
		{label: i18n.T("version_label_product_version"), value: info.ProductVersion},
		{label: i18n.T("version_label_api_version"), value: info.APIVersion},
		{label: i18n.T("version_label_min_backend"), value: info.MinSupportedBackendAPIVersion},
		{label: i18n.T("version_label_go_version"), value: info.GoVersion},
		{label: i18n.T("version_label_commit_sha"), value: info.CommitSHA},
		{label: i18n.T("version_label_build_time"), value: info.BuildTime},
		{label: i18n.T("version_label_platform"), value: info.OS + "/" + info.Arch},
		{label: i18n.T("version_label_build_type"), value: info.BuildType},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(tw, " %s:\t%v\n", line.label, line.value); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeShortVersionResult(w io.Writer, info internalversion.BuildInfo) error {
	_, err := fmt.Fprintln(
		w,
		i18n.T(
			"root_version_output",
			info.ProductVersion,
			internalversion.ShortCommitSHA(info.CommitSHA),
		),
	)
	return err
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
