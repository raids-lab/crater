package cmd

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/config"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
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
		cm, _ := config.NewConfigManager()

		// Language is initialized before Execute(), so no need to set it here.
		_ = cm

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
		handleError(err)
		os.Exit(exitCodeFor(err))
	}
}

func bootstrapJSONFlagFromArgs() {
	// 最后一次出现的 `--json` 生效；在 Cobra 解析失败前也要能识别，以便错误走 JSON。
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--json=") {
			v := strings.TrimPrefix(a, "--json=")
			if b, err := strconv.ParseBool(v); err == nil {
				outputJSON = b
			}
			continue
		}
		if a == "--json" {
			if i+1 < len(args) {
				if b, err := strconv.ParseBool(args[i+1]); err == nil {
					outputJSON = b
					i++
					continue
				}
			}
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
	cm, _ := config.NewConfigManager()
	lang := ""
	if cm != nil && cm.State.Language != "" {
		lang = cm.State.Language
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
		flagKey := "flag_" + f.Name
		usage := i18n.T(flagKey)
		if usage != flagKey {
			f.Usage = usage
		}
	})

	// 4. Recurse into children
	for _, sub := range cmd.Commands() {
		updateAllCommands(sub)
	}
}
