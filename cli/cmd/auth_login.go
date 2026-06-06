package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loginInput 为 auth login 在本地校验与交互补全阶段的输入视图。
type loginInput struct {
	platformURL string
	username    string
	password    string
	mode        string
}

func readLoginInput(cmd *cobra.Command) loginInput {
	platformURL, _ := cmd.Flags().GetString("platform")
	username, _ := cmd.Flags().GetString("username")
	mode, _ := cmd.Flags().GetString("mode")
	password, _ := cmd.Flags().GetString("password")
	return loginInput{
		platformURL: strings.TrimSpace(platformURL),
		username:    strings.TrimSpace(username),
		password:    password,
		mode:        strings.TrimSpace(mode),
	}
}

// collectLoginUsageIssues 汇总可在发起 API 请求前本地判定的用法错误（缺参、非法 mode 等）。
func collectLoginUsageIssues(in loginInput, requireAllFlags bool) []usageIssue {
	var issues []usageIssue
	if !authModeValid(in.mode) {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrInvalidFlagValue,
			Message: i18n.T("err_invalid_auth_mode", in.mode),
			Field:   "mode",
		})
	}
	if !requireAllFlags {
		return issues
	}
	if in.platformURL == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("login_label_platform"), "platform"),
			Field:   "platform",
		})
	}
	if in.username == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("login_label_username"), "username"),
			Field:   "username",
		})
	}
	if in.password == "" {
		issues = append(issues, usageIssue{
			Code:    errorcodes.ErrMissingRequiredFlag,
			Message: i18n.T("err_missing_flag_non_interactive", i18n.T("login_label_password"), "password"),
			Field:   "password",
		})
	}
	return issues
}

func resolveLoginInteractively(in *loginInput) error {
	if in.platformURL == "" {
		if err := survey.AskOne(&survey.Input{Message: i18nPromptLabel("prompt_platform")}, &in.platformURL); err != nil {
			return errSurveyOrSame(err)
		}
		in.platformURL = strings.TrimSpace(in.platformURL)
		if in.platformURL == "" {
			return errUsageFromIssues([]usageIssue{{
				Code:    errorcodes.ErrMissingRequiredFlag,
				Message: i18n.T("err_prompt_empty", i18n.T("login_label_platform")),
				Field:   "platform",
			}})
		}
	}
	if in.username == "" {
		if err := survey.AskOne(&survey.Input{Message: i18nPromptLabel("prompt_username")}, &in.username); err != nil {
			return errSurveyOrSame(err)
		}
		in.username = strings.TrimSpace(in.username)
		if in.username == "" {
			return errUsageFromIssues([]usageIssue{{
				Code:    errorcodes.ErrMissingRequiredFlag,
				Message: i18n.T("err_prompt_empty", i18n.T("login_label_username")),
				Field:   "username",
			}})
		}
	}
	if in.password == "" {
		if err := survey.AskOne(&survey.Password{Message: i18nPromptLabel("prompt_password")}, &in.password); err != nil {
			return errSurveyOrSame(err)
		}
	}
	return nil
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	in := readLoginInput(cmd)
	noInter := viper.GetBool("no-interactive")

	if noInter {
		if issues := collectLoginUsageIssues(in, true); len(issues) > 0 {
			return errUsageFromIssues(issues)
		}
	} else {
		if issues := collectLoginUsageIssues(in, false); len(issues) > 0 {
			return errUsageFromIssues(issues)
		}
		if err := resolveLoginInteractively(&in); err != nil {
			return err
		}
	}

	authClient := api.NewAuthClient(in.platformURL)
	loginResp, err := authClient.Login(in.username, in.password, in.mode)
	if err != nil {
		return cliErrFromAPI(err)
	}

	roleToStr := func(r int) string {
		if r == 3 {
			return "admin"
		}
		return "user"
	}

	newAuth := session.AuthInfo{
		PlatformURL: in.platformURL,
		Username:    in.username,
		Method:      in.mode,
		UserID:      loginResp.User.ID,
		Nickname:    loginResp.User.Nickname,
		Role:        roleToStr(loginResp.Context.RolePlatform),
	}

	if err := session.SaveLogin(newAuth, loginResp.AccessToken); err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrSecureStorageError,
			Message:  err.Error(),
		}
	}

	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"user": newAuth,
		}))
	}
	fmt.Printf("%s\n", i18n.T("login_success", in.platformURL, in.username, in.mode, roleForAuthInfo(newAuth.Role)))
	return nil
}
