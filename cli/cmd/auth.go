package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// roleForAuthInfo 返回用于展示的权限级别字符串（与 auth ls 最后一列一致）。
func roleForAuthInfo(role string) string {
	if role == "" {
		return "-"
	}
	return role
}

// roleForActiveContext 根据 state 中匹配的 AuthInfo 解析当前激活账号的权限级别。
func roleForActiveContext(st session.State, ac session.ActiveContext) string {
	for _, info := range st.AuthInfos {
		if info.PlatformURL == ac.PlatformURL && info.Username == ac.Username && info.Method == ac.Method {
			return roleForAuthInfo(info.Role)
		}
	}
	return "-"
}

const (
	authModeLDAP    = "ldap"
	authModeNormal  = "normal"
	authModeDefault = authModeLDAP
)

// authLoginModes 为 login / 补全 / 校验共用的认证方式 token 有序列表（唯一来源）。
var authLoginModes = []string{authModeLDAP, authModeNormal}

func authModeOrdered() []string {
	return slices.Clone(authLoginModes)
}

func authModeValid(m string) bool {
	return slices.Contains(authLoginModes, m)
}

func authModeDescI18nKey(mode string) string {
	return "auth_mode_" + mode + "_desc"
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication and credentials management",
	Long:  "Manage login sessions and switch between different saved credentials (platform, identity, and method).",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to a Crater platform instance",
	Long:  "Authenticate with a platform using Platform URL, Username, and Method.",
	RunE: func(cmd *cobra.Command, args []string) error {
		platformURL, _ := cmd.Flags().GetString("platform")
		username, _ := cmd.Flags().GetString("username")
		mode, _ := cmd.Flags().GetString("mode")
		password, _ := cmd.Flags().GetString("password")

		noInter := viper.GetBool("no-interactive")

		if platformURL == "" {
			if noInter {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "platform URL", "platform")}
			}
			fmt.Print(i18n.T("prompt_platform"))
			fmt.Scanln(&platformURL)
			if platformURL == "" {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "platform URL", "platform")}
			}
		}

		if username == "" {
			if noInter {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "username", "username")}
			}
			fmt.Print(i18n.T("prompt_username"))
			fmt.Scanln(&username)
			if username == "" {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "username", "username")}
			}
		}

		if password == "" {
			if noInter {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_password")}
			}
			fmt.Print(i18n.T("prompt_password"))
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = string(bytePassword)
			fmt.Println()
		}

		mode = strings.TrimSpace(mode)
		if !authModeValid(mode) {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_invalid_auth_mode", mode)}
		}

		authClient := api.NewAuthClient(platformURL)
		loginResp, err := authClient.Login(username, password, mode)
		if err != nil {
			return cliErrFromAPI(err)
		}

		// NOTE: we intentionally don't need to load state here; SaveLogin will load and persist.

		roleToStr := func(r int) string {
			if r == 3 {
				return "admin"
			}
			return "user"
		}

		newAuth := session.AuthInfo{
			PlatformURL: platformURL,
			Username:    username,
			Method:      mode,
			UserID:      loginResp.User.ID,
			Nickname:    loginResp.User.Nickname,
			Role:        roleToStr(loginResp.Context.RolePlatform),
		}

		if err := session.SaveLogin(newAuth, loginResp.AccessToken); err != nil {
			// session.SaveLogin touches both secure storage and state; map errors to system_error.
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrSecureStorageError, Message: err.Error()}
		}

		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"user": newAuth,
			}))
		}
		fmt.Printf("%s\n", i18n.T("login_success", platformURL, username, mode, roleForAuthInfo(newAuth.Role)))
		return nil
	},
}

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch active credentials",
	Long:  "Switch between different saved platform credentials with fuzzy matching and interactive selection.",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, _ := cmd.Flags().GetString("platform")
		u, _ := cmd.Flags().GetString("username")
		m, _ := cmd.Flags().GetString("mode")
		m = strings.TrimSpace(m)
		if m != "" && !authModeValid(m) {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_invalid_auth_mode", m)}
		}

		st, err := session.LoadState()
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		active := st.ActiveContext
		if !viper.GetBool("json") && !viper.GetBool("no-interactive") {
			if active.PlatformURL != "" {
				fmt.Printf("%s\n", i18n.T("current_active_context", active.PlatformURL, active.Username, active.Method, roleForActiveContext(st, active)))
			} else {
				fmt.Println(i18n.T("no_active_context"))
			}
		}

		var candidates []session.AuthInfo
		for _, info := range st.AuthInfos {
			if info.PlatformURL == active.PlatformURL && info.Username == active.Username && info.Method == active.Method {
				continue
			}
			if (p == "" || info.PlatformURL == p) && (u == "" || info.Username == u) && (m == "" || info.Method == m) {
				candidates = append(candidates, info)
			}
		}

		if len(candidates) == 0 {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrNotFound, Message: i18n.T("err_not_found")}
		}

		var target session.AuthInfo
		if len(candidates) == 1 {
			target = candidates[0]
		} else {
			if viper.GetBool("no-interactive") {
				msg := i18n.T("err_multiple_matches") + "\n"
				for _, c := range candidates {
					msg += fmt.Sprintf("  - %s (%s, %s, %s)\n", c.PlatformURL, c.Username, c.Method, roleForAuthInfo(c.Role))
				}
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: msg}
			}

			options := make([]string, len(candidates))
			for i, c := range candidates {
				options[i] = fmt.Sprintf("%s (%s, %s, %s)", c.PlatformURL, c.Username, c.Method, roleForAuthInfo(c.Role))
			}
			var selection string
			prompt := &survey.Select{
				Message: i18n.T("select_context"),
				Options: options,
			}
			if err := survey.AskOne(prompt, &selection); err != nil {
				return errSurveyOrSame(err)
			}
			for i, opt := range options {
				if opt == selection {
					target = candidates[i]
					break
				}
			}
		}

		st.ActiveContext = session.ActiveContext{
			PlatformURL: target.PlatformURL,
			Username:    target.Username,
			Method:      target.Method,
		}
		if err := session.SaveState(st); err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"active": st.ActiveContext,
			}))
		}
		fmt.Printf("%s\n", i18n.T("switch_success", target.PlatformURL, target.Username, target.Method, roleForAuthInfo(target.Role)))
		return nil
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, _ := cmd.Flags().GetString("platform")
		u, _ := cmd.Flags().GetString("username")
		m, _ := cmd.Flags().GetString("mode")
		m = strings.TrimSpace(m)
		if m != "" && !authModeValid(m) {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_invalid_auth_mode", m)}
		}

		st, err := session.LoadState()
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		var filtered []session.AuthInfo
		active := st.ActiveContext
		for _, info := range st.AuthInfos {
			if (p == "" || info.PlatformURL == p) && (u == "" || info.Username == u) && (m == "" || info.Method == m) {
				filtered = append(filtered, info)
			}
		}

		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"active_context": active,
				"auth_infos":     filtered,
			}))
		}

		header := fmt.Sprintf("%s %s %s %s %s",
			i18n.PadRight(i18n.T("table_active"), 8),
			i18n.PadRight(i18n.T("table_platform"), 30),
			i18n.PadRight(i18n.T("table_username"), 15),
			i18n.PadRight(i18n.T("table_method"), 10),
			i18n.PadRight(i18n.T("table_privilege"), 10))
		fmt.Println(header)

		for _, info := range filtered {
			isActive := " "
			if info.PlatformURL == active.PlatformURL && info.Username == active.Username && info.Method == active.Method {
				isActive = "*"
			}
			row := fmt.Sprintf("%s %s %s %s %s",
				i18n.PadRight(isActive, 8),
				i18n.PadRight(info.PlatformURL, 30),
				i18n.PadRight(info.Username, 15),
				i18n.PadRight(info.Method, 10),
				i18n.PadRight(info.Role, 10))
			fmt.Println(row)
		}
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, _ := cmd.Flags().GetString("platform")
		u, _ := cmd.Flags().GetString("username")
		m, _ := cmd.Flags().GetString("mode")
		m = strings.TrimSpace(m)
		if m != "" && !authModeValid(m) {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_invalid_auth_mode", m)}
		}
		force, _ := cmd.Flags().GetBool("yes")
		noInter := viper.GetBool("no-interactive")

		st, err := session.LoadState()
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		var toRemove []session.AuthInfo
		var remaining []session.AuthInfo
		for _, info := range st.AuthInfos {
			if (p == "" || info.PlatformURL == p) && (u == "" || info.Username == u) && (m == "" || info.Method == m) {
				toRemove = append(toRemove, info)
			} else {
				remaining = append(remaining, info)
			}
		}

		if len(toRemove) == 0 {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrNotFound, Message: i18n.T("err_not_found")}
		}

		if noInter && !force {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "yes", "yes")}
		}

		if !force {
			fmt.Println(i18n.T("remove_confirm_title"))
			for _, r := range toRemove {
				fmt.Printf("  - %s (%s, %s, %s)\n", r.PlatformURL, r.Username, r.Method, roleForAuthInfo(r.Role))
			}
			var confirm bool
			prompt := &survey.Confirm{Message: i18n.T("remove_confirm_ask"), Default: false}
			if err := survey.AskOne(prompt, &confirm); err != nil {
				return errSurveyOrSame(err)
			}
			if !confirm {
				return errOperationCancelled()
			}
		}

		for _, r := range toRemove {
			_ = session.DeleteToken(session.ActiveContext{PlatformURL: r.PlatformURL, Username: r.Username, Method: r.Method})
			if r.PlatformURL == st.ActiveContext.PlatformURL && r.Username == st.ActiveContext.Username && r.Method == st.ActiveContext.Method {
				st.ActiveContext = session.ActiveContext{}
			}
		}

		st.AuthInfos = remaining
		if err := session.SaveState(st); err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"removed_count": len(toRemove),
			}))
		}
		fmt.Printf("%s\n", i18n.T("remove_success", len(toRemove)))
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of the active saved credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("yes")
		noInter := viper.GetBool("no-interactive")

		st, err := session.LoadState()
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		active := st.ActiveContext
		if active.PlatformURL == "" {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrNotFound, Message: i18n.T("err_no_active")}
		}

		if noInter && !force {
			return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "yes", "yes")}
		}

		if !force {
			var confirm bool
			prompt := &survey.Confirm{Message: i18n.T("logout_confirm", active.PlatformURL, active.Username, active.Method, roleForActiveContext(st, active)), Default: true}
			if err := survey.AskOne(prompt, &confirm); err != nil {
				return errSurveyOrSame(err)
			}
			if !confirm {
				return errOperationCancelled()
			}
		}

		_ = session.DeleteToken(active)

		var remaining []session.AuthInfo
		for _, info := range st.AuthInfos {
			if info.PlatformURL == active.PlatformURL && info.Username == active.Username && info.Method == active.Method {
				continue
			}
			remaining = append(remaining, info)
		}
		st.AuthInfos = remaining

		if len(remaining) > 0 {
			st.ActiveContext = session.ActiveContext{
				PlatformURL: remaining[0].PlatformURL,
				Username:    remaining[0].Username,
				Method:      remaining[0].Method,
			}
			if !outputJSON {
				fmt.Printf("%s\n", i18n.T("logout_success_switched", st.ActiveContext.PlatformURL, st.ActiveContext.Username, st.ActiveContext.Method, roleForAuthInfo(remaining[0].Role)))
			}
		} else {
			st.ActiveContext = session.ActiveContext{}
			if !outputJSON {
				fmt.Println(i18n.T("logout_success_no_other"))
			}
		}

		if err := session.SaveState(st); err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
		}

		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"next_active": st.ActiveContext,
			}))
		}
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("platform", "p", "", "Platform base URL")
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("mode", "m", authModeDefault, "Authentication mode")
	loginCmd.Flags().String("password", "", "Password")

	switchCmd.Flags().StringP("platform", "p", "", "Platform URL")
	switchCmd.Flags().StringP("username", "u", "", "Username")
	switchCmd.Flags().StringP("mode", "m", "", "Method")

	lsCmd.Flags().StringP("platform", "p", "", "Filter by platform")
	lsCmd.Flags().StringP("username", "u", "", "Filter by username")
	lsCmd.Flags().StringP("mode", "m", "", "Filter by method")

	rmCmd.Flags().StringP("platform", "p", "", "Filter platform")
	rmCmd.Flags().StringP("username", "u", "", "Filter username")
	rmCmd.Flags().StringP("mode", "m", "", "Filter method")
	rmCmd.Flags().BoolP("yes", "y", false, "Force removal")

	logoutCmd.Flags().BoolP("yes", "y", false, "Force logout")

	// Advanced completion: crater auth * --mode/-m（候选与 authLoginModes 一致）
	modeCompleter := func(ctx completion.Context) ([]completion.Candidate, error) {
		prefix := completion.CurrentWordPrefix(ctx)
		modes := authModeOrdered()
		out := make([]completion.Candidate, 0, len(modes))
		for _, m := range modes {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(m), strings.ToLower(prefix)) {
				continue
			}
			out = append(out, completion.Candidate{
				Value:       m,
				Description: i18n.T(authModeDescI18nKey(m)),
			})
		}
		return out, nil
	}
	completion.RegisterFlagValue([]string{"auth", "login"}, "mode", modeCompleter)
	completion.RegisterFlagValue([]string{"auth", "switch"}, "mode", modeCompleter)
	completion.RegisterFlagValue([]string{"auth", "ls"}, "mode", modeCompleter)
	completion.RegisterFlagValue([]string{"auth", "rm"}, "mode", modeCompleter)

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(switchCmd)
	authCmd.AddCommand(lsCmd)
	authCmd.AddCommand(rmCmd)
	authCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(authCmd)
}
