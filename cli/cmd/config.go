package cmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/clierror"
	"github.com/raids-lab/crater/cli/internal/config"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: i18n.T("config_short"),
	Long:  i18n.T("config_long"),
}

var languageCmd = &cobra.Command{
	Use:   "language [LANG]",
	Short: i18n.T("lang_short"),
	RunE: func(cmd *cobra.Command, args []string) error {
		cm, err := config.NewConfigManager()
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
		cm.State.Language = targetLang
		if err := cm.Save(); err != nil {
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
}
