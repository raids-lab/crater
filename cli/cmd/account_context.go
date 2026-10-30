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

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
)

var accountCurrentCmd = &cobra.Command{
	Use: "current", Short: "Show the current account/queue", Args: noArgs, RunE: runAccountCurrent,
}

var accountUseCmd = &cobra.Command{
	Use: "use <name>", Short: "Switch the current account/queue", Args: exactArgs(1, "queue"), RunE: runAccountUse,
}

func runAccountCurrent(_ *cobra.Command, _ []string) error {
	client, active, err := activeAPIClientWithContext()
	if err != nil {
		return err
	}
	context, err := client.CurrentAccount()
	if err != nil {
		return cliErrFromAPI(err)
	}
	return writeAccountContext(active, context)
}

func runAccountUse(_ *cobra.Command, args []string) error {
	name, err := requiredArg(args, "account_label_queue", "name")
	if err != nil {
		return err
	}
	client, active, err := activeAPIClientWithContext()
	if err != nil {
		return err
	}
	result, err := client.SwitchAccount(name)
	if err != nil {
		return cliErrFromAPI(err)
	}
	if err := session.ReplaceToken(active, result.AccessToken); err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  i18n.T("err_config_write", err.Error()),
		}
	}
	return writeAccountContext(active, &result.Context)
}

func writeAccountContext(active session.ActiveContext, context *api.AccountContext) error {
	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]any{
			"platform_url": active.PlatformURL,
			"username":     active.Username,
			"context":      context,
		}))
	}
	fmt.Printf("%s: %s\n", i18n.T("table_platform"), active.PlatformURL)
	fmt.Printf("%s: %s\n", i18n.T("table_username"), active.Username)
	fmt.Printf("%s: %s\n", i18n.T("account_label_queue"), context.Queue)
	fmt.Printf("%s: %d\n", i18n.T("table_role"), context.RoleQueue)
	return nil
}

func init() {
	accountCmd.AddCommand(accountCurrentCmd, accountUseCmd)
}
