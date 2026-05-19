package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/internal/session"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T("config_short"),
	Long:  i18n.T("config_long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If user mistypes a subcommand (e.g. `crater config langauge`), avoid
		// Cobra's default "print usage and exit 0" behavior. Instead, return a
		// structured usage error so --json and exit codes remain consistent.
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var languageCmd = &cobra.Command{
	Use:   "language [LANG]",
	Short: i18n.T("lang_short"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errTooManyArgs(cmd, len(args), 1)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := session.LoadState()
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		var targetLang string
		if len(args) > 0 {
			targetLang = args[0]
			supported := false
			for _, lang := range i18n.GetSupportedLanguages() {
				if targetLang == lang {
					supported = true
					break
				}
			}
			if !supported {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrInvalidFlagValue, Message: i18n.T("err_invalid_lang", targetLang)}
			}
		} else {
			if viper.GetBool("no-interactive") {
				return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_language_arg")}
			}

			langs := i18n.GetSupportedLanguages()
			displayMap := i18n.GetLanguageDisplay()
			options := make([]string, len(langs))
			current := i18n.GetCurrentLanguage()
			defaultOption := ""

			for i, l := range langs {
				options[i] = fmt.Sprintf("%-10s (%s)", l, displayMap[l])
				if l == current {
					defaultOption = options[i]
				}
			}

			var selection string
			prompt := &survey.Select{
				Message: i18n.T("select_language"),
				Options: options,
				Default: defaultOption,
			}
			if err := survey.AskOne(prompt, &selection); err != nil {
				return errSurveyOrSame(err)
			}

			// Extract language code from selection (e.g., "en         (English)" -> "en")
			for _, l := range langs {
				if selection[:len(l)] == l {
					targetLang = l
					break
				}
			}
		}

		// Update config
		st.Language = targetLang
		if err := session.SaveState(st); err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: i18n.T("err_config_write", err.Error())}
		}

		// Apply language immediately for the success message
		i18n.SetLanguage(targetLang)

		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"language": targetLang,
			}))
		}
		fmt.Println(i18n.T("lang_switch_success", targetLang))
		return nil
	},
}

func init() {
	configCmd.AddCommand(languageCmd)
	rootCmd.AddCommand(configCmd)

	// Advanced completion (phase 1): crater config language [LANG]
	// token 不翻译；description 使用语言展示名，当前语言带 UI 语言下的简短标记。
	completion.RegisterPositional([]string{"config", "language"}, 0, func(ctx completion.Context) ([]completion.Candidate, error) {
		prefix := completion.CurrentWordPrefix(ctx)
		langs := i18n.GetSupportedLanguages()
		display := i18n.GetLanguageDisplay()
		current := i18n.GetCurrentLanguage()
		var out []completion.Candidate
		for _, l := range langs {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(l), strings.ToLower(prefix)) {
				continue
			}
			desc := display[l]
			if l == current {
				desc = i18n.T("lang_completion_label_current", display[l])
			}
			out = append(out, completion.Candidate{
				Value:       l,
				Description: desc,
			})
		}
		return out, nil
	})
}
