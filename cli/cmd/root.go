package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/raids-lab/crater/cli/internal/config"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	noInteractive bool
	outputJSON    bool
)

type ErrorResponse struct {
	Category string                 `json:"category"`
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

type CLIError struct {
	Category string
	Code     string
	Message  string
	Context  map[string]interface{}
}

func (e *CLIError) Error() string {
	return e.Message
}

func exitCodeFor(err error) int {
	var e *CLIError
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
	// Scheme A: initialize language and update help texts
	// BEFORE Cobra decides to render help/usage.
	initLanguageAndHelp()

	if err := rootCmd.Execute(); err != nil {
		handleError(err)
		os.Exit(exitCodeFor(err))
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noInteractive, "no-interactive", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output in raw JSON format")
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Help for crater")

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return &CLIError{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrInvalidFlagValue,
			Message:  err.Error(),
		}
	})

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}

func handleError(err error) {
	if outputJSON {
		category := errorcodes.CategorySystem
		code := errorcodes.ErrCommandExecution
		var context map[string]interface{}

		if cliErr, ok := err.(*CLIError); ok {
			category = cliErr.Category
			code = cliErr.Code
			context = cliErr.Context
		}

		resp := ErrorResponse{
			Category: category,
			Code:     code,
			Message:  err.Error(),
			Context:  context,
		}
		jsonBytes, _ := json.Marshal(resp)
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Error:\n  %v\n", err)
	}
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

func initLanguageAndHelp() {
	cm, _ := config.NewConfigManager()

	// Priority (Scheme A): Config > Env > Detect
	lang := ""
	if cm != nil && cm.State.Language != "" {
		lang = cm.State.Language
	}
	if lang == "" {
		lang = i18n.DetectLanguage()
	}
	i18n.SetLanguage(lang)

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

func MarshalJSON(v interface{}) error {
	if !outputJSON {
		return nil
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
