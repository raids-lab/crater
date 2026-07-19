package cmd

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	noInteractive bool
	outputJSON    bool
)

func exitCodeFor(err error) int {
	var e *clierror.Error
	if errors.As(err, &e) {
		return errorcodes.ExitCodeForCategory(e.Category)
	}
	return errorcodes.ExitFailure
}

var rootCmd = &cobra.Command{
	Use:   "crater",
	Short: "Crater CLI",
	Long:  "Crater CLI is an AI-friendly command line tool.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Language is initialized before Execute(), so no need to set it here.

		if outputJSON {
			noInteractive = true
		}

		if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
			noInteractive = true
		}

		viper.Set("no-interactive", noInteractive)
		viper.Set("json", outputJSON)

		return nil
	},
}

func Execute() {
	// Ensure `--json` works regardless of argument order.
	// Cobra/pflag may fail fast on unknown flags before PersistentPreRunE runs,
	// so we pre-scan os.Args to decide whether errors should render as JSON.
	bootstrapJSONFlagFromArgs()
	if outputJSON {
		noInteractive = true
	}
	// Keep viper in sync even if flag parsing fails early.
	viper.Set("no-interactive", noInteractive)
	viper.Set("json", outputJSON)

	// Scheme A: initialize language and update help texts
	// BEFORE Cobra decides to render help/usage.
	initLanguageAndHelp()

	if err := rootCmd.Execute(); err != nil {
		err = normalizeCobraExecutionError(err)
		handleError(err)
		os.Exit(exitCodeFor(err))
	}
}

func normalizeCobraExecutionError(err error) error {
	if err == nil {
		return nil
	}
	var ce *clierror.Error
	if errors.As(err, &ce) {
		return err
	}
	if typed, ok := cobraUnknownCommand(err); ok {
		return errUnknownSubcommand(rootCmd, typed)
	}
	return err
}

func cobraUnknownCommand(err error) (string, bool) {
	const prefix = "unknown command "
	msg := err.Error()
	if !strings.HasPrefix(msg, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(msg, prefix)
	if !strings.HasPrefix(rest, "\"") {
		return "", true
	}
	end := strings.Index(rest[1:], "\"")
	if end < 0 {
		return "", true
	}
	return rest[1 : end+1], true
}

func bootstrapJSONFlagFromArgs() {
	// 最后一次出现的 `--json` 生效；在 Cobra 解析失败前也要能识别，以便错误走 JSON。
	// 预扫描必须与 pflag 的 bool flag 语义一致：支持 `--json` 与 `--json=<bool>`，
	// 不支持空格分隔的 `--json <bool>`，并在 `--` 后停止解析 flag。
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return
		}
		if strings.HasPrefix(a, "--json=") {
			v := strings.TrimPrefix(a, "--json=")
			if b, err := strconv.ParseBool(v); err == nil {
				outputJSON = b
			}
			continue
		}
		if a == "--json" {
			outputJSON = true
		}
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noInteractive, "no-interactive", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output in raw JSON format")
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Help for crater")

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrInvalidFlagValue,
			Message:  err.Error(),
		}
	})

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}

func handleError(err error) {
	output.WriteError(os.Stderr, outputJSON, err)
}

func updateHelpTexts(root *cobra.Command) {
	// Update Root
	root.Short = i18n.T("root_short")
	root.Long = i18n.T("root_long")

	// Update Global Flags
	root.PersistentFlags().Lookup("no-interactive").Usage = i18n.T("flag_no-interactive")
	root.PersistentFlags().Lookup("json").Usage = i18n.T("flag_json")
	root.PersistentFlags().Lookup("help").Usage = i18n.T("flag_help")

	// Update all commands recursively
	updateAllCommands(root)
}

// initLanguageOnly 仅设置 i18n 语言（供 crater __complete 快路径使用，不做 help 全树覆盖）。
func initLanguageOnly() {
	lang := ""
	st, err := session.LoadState()
	if err == nil && st.Language != "" {
		lang = st.Language
	}
	if lang == "" {
		lang = i18n.DetectLanguage()
	}
	i18n.SetLanguage(lang)
}

func initLanguageAndHelp() {
	initLanguageOnly()
	updateHelpTexts(rootCmd)
}

func updateAllCommands(cmd *cobra.Command) {
	// 1. Generate Key from Command Path
	// crater auth login -> auth_login
	fullPath := cmd.CommandPath()
	parts := strings.Split(fullPath, " ")
	var keyPath string
	if len(parts) > 1 {
		keyPath = strings.Join(parts[1:], "_")
	} else {
		keyPath = "root"
	}

	shortKey := keyPath + "_short"
	longKey := keyPath + "_long"

	// 2. Update Descriptions
	shortTranslated := i18n.T(shortKey)
	if shortTranslated != shortKey {
		cmd.Short = shortTranslated
	}

	longTranslated := i18n.T(longKey)
	if longTranslated != longKey {
		cmd.Long = longTranslated
	}

	// 3. Update Flags for this command
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if usage, ok := translatedFlagUsage(keyPath, f.Name); ok {
			f.Usage = usage
		}
	})

	// 4. Recurse into children
	for _, sub := range cmd.Commands() {
		updateAllCommands(sub)
	}
}

func translatedFlagUsage(keyPath, flagName string) (string, bool) {
	keys := []string{keyPath + "_flag_" + flagName}
	if isImageCommandPath(keyPath) {
		keys = append(keys, "image_flag_"+flagName)
	}
	keys = append(keys, "flag_"+flagName)
	for _, key := range keys {
		if usage := i18n.T(key); usage != key {
			return usage, true
		}
	}
	return "", false
}

func isImageCommandPath(keyPath string) bool {
	return keyPath == "image" || strings.HasPrefix(keyPath, "image_") ||
		keyPath == "admin_image" || strings.HasPrefix(keyPath, "admin_image_")
}
