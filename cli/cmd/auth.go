package cmd

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/auth"
	"github.com/raids-lab/crater/cli/internal/config"
	"github.com/raids-lab/crater/cli/internal/i18n"
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
func roleForActiveContext(st config.State, ac config.ActiveContext) string {
	for _, info := range st.AuthInfos {
		if info.PlatformURL == ac.PlatformURL && info.Username == ac.Username && info.Method == ac.Method {
			return roleForAuthInfo(info.Role)
		}
	}
	return "-"
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication and credentials management",
	Long:  "Manage login sessions and switch between different saved credentials (platform, identity, and method).",
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
				return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "platform URL", "platform")}
			}
			fmt.Print(i18n.T("prompt_platform"))
			fmt.Scanln(&platformURL)
			if platformURL == "" {
				return errors.New(i18n.T("err_missing_required", "platform URL", "platform"))
			}
		}

		if username == "" {
			if noInter {
				return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "username", "username")}
			}
			fmt.Print(i18n.T("prompt_username"))
			fmt.Scanln(&username)
			if username == "" {
				return errors.New(i18n.T("err_missing_required", "username", "username"))
			}
		}

		if password == "" {
			if noInter {
				return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_password")}
			}
			fmt.Print(i18n.T("prompt_password"))
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			password = string(bytePassword)
			fmt.Println()
		}

		apiClient := api.NewClient(platformURL)
		loginResp, err := apiClient.Login(username, password, mode)
		if err != nil {
			return &CLIError{Category: errorcodes.CategoryAPI, Code: errorcodes.ErrUnauthorized, Message: i18n.T("err_unauthorized", err.Error())}
		}

		authManager, err := auth.NewAuthManager()
		if err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrSecureStorageError, Message: err.Error()}
		}
		key := fmt.Sprintf("%s|%s|%s", platformURL, username, mode)
		if err := authManager.StoreToken("crater", key, loginResp.AccessToken); err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrSecureStorageError, Message: err.Error()}
		}

		cm, err := config.NewConfigManager()
		if err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		roleToStr := func(r int) string {
			if r == 3 {
				return "admin"
			}
			return "user"
		}

		newAuth := config.AuthInfo{
			PlatformURL: platformURL,
			Username:    username,
			Method:      mode,
			UserID:      loginResp.User.ID,
			Nickname:    loginResp.User.Nickname,
			Role:        roleToStr(loginResp.Context.RolePlatform),
		}

		found := false
		for i, info := range cm.State.AuthInfos {
			if info.PlatformURL == platformURL && info.Username == username && info.Method == mode {
				cm.State.AuthInfos[i] = newAuth
				found = true
				break
			}
		}
		if !found {
			cm.State.AuthInfos = append(cm.State.AuthInfos, newAuth)
		}

		cm.State.ActiveContext = config.ActiveContext{
			PlatformURL: platformURL,
			Username:    username,
			Method:      mode,
		}

		if err := cm.Save(); err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		if outputJSON {
			return MarshalJSON(map[string]interface{}{"status": "success", "user": newAuth})
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

		cm, err := config.NewConfigManager()
		if err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		active := cm.State.ActiveContext
		if !viper.GetBool("json") && !viper.GetBool("no-interactive") {
			if active.PlatformURL != "" {
				fmt.Printf("%s\n", i18n.T("current_active_context", active.PlatformURL, active.Username, active.Method, roleForActiveContext(cm.State, active)))
			} else {
				fmt.Println(i18n.T("no_active_context"))
			}
		}

		var candidates []config.AuthInfo
		for _, info := range cm.State.AuthInfos {
			if info.PlatformURL == active.PlatformURL && info.Username == active.Username && info.Method == active.Method {
				continue
			}
			if (p == "" || info.PlatformURL == p) && (u == "" || info.Username == u) && (m == "" || info.Method == m) {
				candidates = append(candidates, info)
			}
		}

		if len(candidates) == 0 {
			return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrNotFound, Message: i18n.T("err_not_found")}
		}

		var target config.AuthInfo
		if len(candidates) == 1 {
			target = candidates[0]
		} else {
			if viper.GetBool("no-interactive") {
				msg := i18n.T("err_multiple_matches") + "\n"
				for _, c := range candidates {
					msg += fmt.Sprintf("  - %s (%s, %s, %s)\n", c.PlatformURL, c.Username, c.Method, roleForAuthInfo(c.Role))
				}
				return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: msg}
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
				if err.Error() == "interrupt" {
					return nil
				}
				return err
			}
			for i, opt := range options {
				if opt == selection {
					target = candidates[i]
					break
				}
			}
		}

		cm.State.ActiveContext = config.ActiveContext{
			PlatformURL: target.PlatformURL,
			Username:    target.Username,
			Method:      target.Method,
		}
		if err := cm.Save(); err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		if outputJSON {
			return MarshalJSON(map[string]interface{}{"status": "switched", "active": cm.State.ActiveContext})
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

		cm, err := config.NewConfigManager()
		if err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		var filtered []config.AuthInfo
		active := cm.State.ActiveContext
		for _, info := range cm.State.AuthInfos {
			if (p == "" || info.PlatformURL == p) && (u == "" || info.Username == u) && (m == "" || info.Method == m) {
				filtered = append(filtered, info)
			}
		}

		if outputJSON {
			type Result struct {
				ActiveContext config.ActiveContext `json:"active_context"`
				AuthInfos     []config.AuthInfo    `json:"auth_infos"`
			}
			return MarshalJSON(Result{ActiveContext: active, AuthInfos: filtered})
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
		force, _ := cmd.Flags().GetBool("yes")
		noInter := viper.GetBool("no-interactive")

		cm, err := config.NewConfigManager()
		if err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		var toRemove []config.AuthInfo
		var remaining []config.AuthInfo
		for _, info := range cm.State.AuthInfos {
			if (p == "" || info.PlatformURL == p) && (u == "" || info.Username == u) && (m == "" || info.Method == m) {
				toRemove = append(toRemove, info)
			} else {
				remaining = append(remaining, info)
			}
		}

		if len(toRemove) == 0 {
			return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrNotFound, Message: i18n.T("err_not_found")}
		}

		if noInter && !force {
			return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "yes", "yes")}
		}

		if !force {
			fmt.Println(i18n.T("remove_confirm_title"))
			for _, r := range toRemove {
				fmt.Printf("  - %s (%s, %s, %s)\n", r.PlatformURL, r.Username, r.Method, roleForAuthInfo(r.Role))
			}
			var confirm bool
			prompt := &survey.Confirm{Message: i18n.T("remove_confirm_ask"), Default: false}
			if err := survey.AskOne(prompt, &confirm); err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				return &CLIError{Category: errorcodes.CategoryCancelled, Code: errorcodes.ErrOperationCancelled, Message: i18n.T("err_operation_cancelled")}
			}
			if !confirm {
				return &CLIError{Category: errorcodes.CategoryCancelled, Code: errorcodes.ErrOperationCancelled, Message: i18n.T("err_operation_cancelled")}
			}
		}

		authManager, _ := auth.NewAuthManager()
		for _, r := range toRemove {
			if authManager != nil {
				key := fmt.Sprintf("%s|%s|%s", r.PlatformURL, r.Username, r.Method)
				_ = authManager.RemoveToken("crater", key)
			}
			if r.PlatformURL == cm.State.ActiveContext.PlatformURL && r.Username == cm.State.ActiveContext.Username && r.Method == cm.State.ActiveContext.Method {
				cm.State.ActiveContext = config.ActiveContext{}
			}
		}

		cm.State.AuthInfos = remaining
		if err := cm.Save(); err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		if outputJSON {
			return MarshalJSON(map[string]interface{}{"status": "removed", "count": len(toRemove)})
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

		cm, err := config.NewConfigManager()
		if err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		active := cm.State.ActiveContext
		if active.PlatformURL == "" {
			return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrNotFound, Message: i18n.T("err_no_active")}
		}

		if noInter && !force {
			return &CLIError{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "yes", "yes")}
		}

		if !force {
			var confirm bool
			prompt := &survey.Confirm{Message: i18n.T("logout_confirm", active.PlatformURL, active.Username, active.Method, roleForActiveContext(cm.State, active)), Default: true}
			if err := survey.AskOne(prompt, &confirm); err != nil {
				if err.Error() == "interrupt" {
					return nil
				}
				return &CLIError{Category: errorcodes.CategoryCancelled, Code: errorcodes.ErrOperationCancelled, Message: i18n.T("err_operation_cancelled")}
			}
			if !confirm {
				return &CLIError{Category: errorcodes.CategoryCancelled, Code: errorcodes.ErrOperationCancelled, Message: i18n.T("err_operation_cancelled")}
			}
		}

		authManager, _ := auth.NewAuthManager()
		if authManager != nil {
			key := fmt.Sprintf("%s|%s|%s", active.PlatformURL, active.Username, active.Method)
			_ = authManager.RemoveToken("crater", key)
		}

		var remaining []config.AuthInfo
		for _, info := range cm.State.AuthInfos {
			if info.PlatformURL == active.PlatformURL && info.Username == active.Username && info.Method == active.Method {
				continue
			}
			remaining = append(remaining, info)
		}
		cm.State.AuthInfos = remaining

		if len(remaining) > 0 {
			cm.State.ActiveContext = config.ActiveContext{
				PlatformURL: remaining[0].PlatformURL,
				Username:    remaining[0].Username,
				Method:      remaining[0].Method,
			}
			if !outputJSON {
				fmt.Printf("%s\n", i18n.T("logout_success_switched", cm.State.ActiveContext.PlatformURL, cm.State.ActiveContext.Username, cm.State.ActiveContext.Method, roleForAuthInfo(remaining[0].Role)))
			}
		} else {
			cm.State.ActiveContext = config.ActiveContext{}
			if !outputJSON {
				fmt.Println(i18n.T("logout_success_no_other"))
			}
		}

		if err := cm.Save(); err != nil {
			return &CLIError{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
		}

		if outputJSON {
			return MarshalJSON(map[string]interface{}{"status": "logged_out", "next_active": cm.State.ActiveContext})
		}
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("platform", "p", "", "Platform base URL")
	loginCmd.Flags().StringP("username", "u", "", "Username")
	loginCmd.Flags().StringP("mode", "m", "ldap", "Authentication mode")
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

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(switchCmd)
	authCmd.AddCommand(lsCmd)
	authCmd.AddCommand(rmCmd)
	authCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(authCmd)
}
