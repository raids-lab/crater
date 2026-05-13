package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/raids-lab/crater/cli/internal/clierror"
	compshell "github.com/raids-lab/crater/cli/internal/completion/shell"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/raids-lab/crater/cli/internal/output"
	"github.com/raids-lab/crater/cli/pkg/errorcodes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var completionCmd = &cobra.Command{
	Use:     "completion",
	Aliases: []string{"comp"},
	Short:   i18n.T("completion_short"),
	Long:    i18n.T("completion_long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errUnknownSubcommand(cmd, args[0])
		}
		return cmd.Help()
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: i18n.T("completion_zsh_short"),
	Long:  i18n.T("completion_zsh_long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			exe = "crater"
		}
		script := compshell.ZshInlineBlock(exe)
		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"shell":  "zsh",
				"script": script,
			}))
		}
		fmt.Print(script)
		return nil
	},
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: i18n.T("completion_bash_short"),
	Long:  i18n.T("completion_bash_long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		exe, err := os.Executable()
		if err != nil {
			exe = "crater"
		}
		script := compshell.BashInlineBlock(exe)
		if outputJSON {
			return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
				"shell":  "bash",
				"script": script,
			}))
		}
		fmt.Print(script)
		return nil
	},
}

var completionInstallCmd = &cobra.Command{
	Use:   "install",
	Short: i18n.T("completion_install_short"),
	Long:  i18n.T("completion_install_long"),
}

var completionInstallZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: i18n.T("completion_install_zsh_short"),
	Long:  i18n.T("completion_install_zsh_long"),
	RunE:  runCompletionInstallZsh,
}

var completionInstallBashCmd = &cobra.Command{
	Use:   "bash",
	Short: i18n.T("completion_install_bash_short"),
	Long:  i18n.T("completion_install_bash_long"),
	RunE:  runCompletionInstallBash,
}

var completionUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: i18n.T("completion_uninstall_short"),
	Long:  i18n.T("completion_uninstall_long"),
}

var completionUninstallZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: i18n.T("completion_uninstall_zsh_short"),
	Long:  i18n.T("completion_uninstall_zsh_long"),
	RunE:  runCompletionUninstallZsh,
}

var completionUninstallBashCmd = &cobra.Command{
	Use:   "bash",
	Short: i18n.T("completion_uninstall_bash_short"),
	Long:  i18n.T("completion_uninstall_bash_long"),
	RunE:  runCompletionUninstallBash,
}

func init() {
	completionInstallZshCmd.Flags().BoolP("yes", "y", false, i18n.T("flag_yes"))
	completionUninstallZshCmd.Flags().BoolP("yes", "y", false, i18n.T("flag_yes"))
	completionInstallBashCmd.Flags().BoolP("yes", "y", false, i18n.T("flag_yes"))
	completionUninstallBashCmd.Flags().BoolP("yes", "y", false, i18n.T("flag_yes"))

	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionBashCmd)
	completionInstallCmd.AddCommand(completionInstallZshCmd)
	completionInstallCmd.AddCommand(completionInstallBashCmd)
	completionUninstallCmd.AddCommand(completionUninstallZshCmd)
	completionUninstallCmd.AddCommand(completionUninstallBashCmd)
	completionCmd.AddCommand(completionInstallCmd)
	completionCmd.AddCommand(completionUninstallCmd)
	rootCmd.AddCommand(completionCmd)
}

func zshrcPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zshrc"), nil
}

const (
	zshrcBeginMarker = "# >>> crater completion >>>"
	zshrcEndMarker   = "# <<< crater completion <<<"
)

func zshrcBlock(inlineScript string) string {
	return strings.Join([]string{
		zshrcBeginMarker,
		inlineScript,
		zshrcEndMarker,
		"",
	}, "\n")
}

func upsertZshrcBlock(content []byte, inlineScript string) ([]byte, bool, error) {
	begin := []byte(zshrcBeginMarker)
	end := []byte(zshrcEndMarker)
	bi := bytes.Index(content, begin)
	ei := bytes.Index(content, end)
	block := []byte(zshrcBlock(inlineScript))

	// No existing block -> append.
	if bi == -1 || ei == -1 || ei < bi {
		out := content
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, block...)
		return out, true, nil
	}

	// Replace existing block.
	eiEnd := ei + len(end)
	// include trailing newline if exists
	if eiEnd < len(content) && content[eiEnd] == '\n' {
		eiEnd++
	}
	out := append([]byte(nil), content[:bi]...)
	out = append(out, block...)
	out = append(out, content[eiEnd:]...)
	return out, false, nil
}

func removeZshrcBlock(content []byte) ([]byte, bool, error) {
	begin := []byte(zshrcBeginMarker)
	end := []byte(zshrcEndMarker)
	bi := bytes.Index(content, begin)
	ei := bytes.Index(content, end)
	if bi == -1 || ei == -1 || ei < bi {
		return content, false, nil
	}
	eiEnd := ei + len(end)
	if eiEnd < len(content) && content[eiEnd] == '\n' {
		eiEnd++
	}
	out := append([]byte(nil), content[:bi]...)
	out = append(out, content[eiEnd:]...)
	return out, true, nil
}

func confirmCompletionAction(message string) error {
	var confirmed bool
	prompt := &survey.Confirm{Message: message, Default: false}
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		if mapped := errSurveyOrSame(err); mapped != err {
			return mapped
		}
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrCommandExecution,
			Message:  err.Error(),
		}
	}
	if !confirmed {
		return errOperationCancelled()
	}
	return nil
}

func runCompletionInstallZsh(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	noInter := viper.GetBool("no-interactive")
	if noInter && !yes {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrMissingRequiredFlag,
			Message:  i18n.T("err_missing_required", "yes", "yes"),
		}
	}

	zshrc, err := zshrcPath()
	if err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  err.Error(),
		}
	}
	exe, err := os.Executable()
	if err != nil {
		exe = "crater"
	}
	inline := compshell.ZshInlineBlock(exe)
	block := zshrcBlock(inline)

	if !noInter && !yes {
		fmt.Print(i18n.T("completion_install_plan", zshrc, block))
		if err := confirmCompletionAction(i18n.T("completion_install_confirm")); err != nil {
			return err
		}
	}

	// Update ~/.zshrc (user config) with marker block (idempotent).
	zshrcContent, err := os.ReadFile(zshrc)
	if err != nil && !os.IsNotExist(err) {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  err.Error(),
		}
	}
	hadZshBlock := err == nil && bytes.Contains(zshrcContent, []byte(zshrcBeginMarker))
	newZshrcContent, inserted, err := upsertZshrcBlock(zshrcContent, inline)
	if err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  err.Error(),
		}
	}
	if err := os.WriteFile(zshrc, newZshrcContent, 0644); err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  err.Error(),
		}
	}

	if outputJSON {
		updated := hadZshBlock || !inserted
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"shell":           "zsh",
			"installed_paths": []string{zshrc},
			"updated":         updated,
			"inserted_zshrc":  inserted,
		}))
	}
	fmt.Print(i18n.T("completion_install_done", zshrc))
	fmt.Print(i18n.T("completion_install_next_steps"))
	fmt.Print(i18n.T("completion_install_uninstall_hint"))
	return nil
}

func runCompletionUninstallZsh(cmd *cobra.Command, args []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	noInter := viper.GetBool("no-interactive")
	if noInter && !yes {
		return &clierror.Error{
			Category: errorcodes.CategoryUsage,
			Code:     errorcodes.ErrMissingRequiredFlag,
			Message:  i18n.T("err_missing_required", "yes", "yes"),
		}
	}

	zshrc, err := zshrcPath()
	if err != nil {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  err.Error(),
		}
	}

	if !noInter && !yes {
		fmt.Print(i18n.T("completion_uninstall_plan", zshrc))
		if err := confirmCompletionAction(i18n.T("completion_uninstall_confirm")); err != nil {
			return err
		}
	}

	// Remove marker block from ~/.zshrc (best-effort, idempotent).
	zshrcContent, err := os.ReadFile(zshrc)
	if err != nil && !os.IsNotExist(err) {
		return &clierror.Error{
			Category: errorcodes.CategorySystem,
			Code:     errorcodes.ErrConfigWriteFailed,
			Message:  err.Error(),
		}
	}
	removed := []string{}
	if err == nil {
		newZshrcContent, changed, err := removeZshrcBlock(zshrcContent)
		if err != nil {
			return &clierror.Error{
				Category: errorcodes.CategorySystem,
				Code:     errorcodes.ErrConfigWriteFailed,
				Message:  err.Error(),
			}
		}
		if changed {
			if err := os.WriteFile(zshrc, newZshrcContent, 0644); err != nil {
				return &clierror.Error{
					Category: errorcodes.CategorySystem,
					Code:     errorcodes.ErrConfigWriteFailed,
					Message:  err.Error(),
				}
			}
			removed = append(removed, zshrc)
		}
	}

	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"shell":         "zsh",
			"removed_paths": removed,
		}))
	}
	if len(removed) > 0 {
		fmt.Print(i18n.T("completion_uninstall_done", strings.Join(removed, "\n")))
		fmt.Print(i18n.T("completion_uninstall_next_steps"))
	}
	return nil
}

func bashrcPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".bashrc"), nil
}

const (
	bashrcBeginMarker = "# >>> crater completion >>>"
	bashrcEndMarker   = "# <<< crater completion <<<"
)

func bashrcBlock(inlineScript string) string {
	return strings.Join([]string{
		bashrcBeginMarker,
		inlineScript,
		bashrcEndMarker,
		"",
	}, "\n")
}

func upsertBashrcBlock(content []byte, inlineScript string) ([]byte, bool, error) {
	begin := []byte(bashrcBeginMarker)
	end := []byte(bashrcEndMarker)
	bi := bytes.Index(content, begin)
	ei := bytes.Index(content, end)
	block := []byte(bashrcBlock(inlineScript))

	if bi == -1 || ei == -1 || ei < bi {
		out := content
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, block...)
		return out, true, nil
	}

	eiEnd := ei + len(end)
	if eiEnd < len(content) && content[eiEnd] == '\n' {
		eiEnd++
	}
	out := append([]byte(nil), content[:bi]...)
	out = append(out, block...)
	out = append(out, content[eiEnd:]...)
	return out, false, nil
}

func removeBashrcBlock(content []byte) ([]byte, bool, error) {
	begin := []byte(bashrcBeginMarker)
	end := []byte(bashrcEndMarker)
	bi := bytes.Index(content, begin)
	ei := bytes.Index(content, end)
	if bi == -1 || ei == -1 || ei < bi {
		return content, false, nil
	}
	eiEnd := ei + len(end)
	if eiEnd < len(content) && content[eiEnd] == '\n' {
		eiEnd++
	}
	out := append([]byte(nil), content[:bi]...)
	out = append(out, content[eiEnd:]...)
	return out, true, nil
}

func runCompletionInstallBash(cmd *cobra.Command, args []string) error {
	bashrc, err := bashrcPath()
	if err != nil {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
	}

	yes, _ := cmd.Flags().GetBool("yes")
	noInter := viper.GetBool("no-interactive")
	if noInter && !yes {
		return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "yes", "yes")}
	}

	exe, err := os.Executable()
	if err != nil {
		exe = "crater"
	}
	inline := compshell.BashInlineBlock(exe)
	block := bashrcBlock(inline)
	if !noInter && !yes {
		fmt.Print(i18n.T("completion_bash_install_plan", bashrc, block))
		if err := confirmCompletionAction(i18n.T("completion_install_confirm")); err != nil {
			return err
		}
	}

	bashrcContent, err := os.ReadFile(bashrc)
	if err != nil && !os.IsNotExist(err) {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
	}
	hadBashBlock := err == nil && bytes.Contains(bashrcContent, []byte(bashrcBeginMarker))
	newBashrcContent, inserted, err := upsertBashrcBlock(bashrcContent, inline)
	if err != nil {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
	}
	if err := os.WriteFile(bashrc, newBashrcContent, 0644); err != nil {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
	}

	if outputJSON {
		updated := hadBashBlock || !inserted
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"shell":           "bash",
			"installed_paths": []string{bashrc},
			"updated":         updated,
			"inserted_bashrc": inserted,
		}))
	}
	fmt.Print(i18n.T("completion_bash_install_done", bashrc))
	fmt.Print(i18n.T("completion_bash_install_next_steps"))
	fmt.Print(i18n.T("completion_install_uninstall_hint_bash"))
	return nil
}

func runCompletionUninstallBash(cmd *cobra.Command, args []string) error {
	bashrc, err := bashrcPath()
	if err != nil {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
	}

	yes, _ := cmd.Flags().GetBool("yes")
	noInter := viper.GetBool("no-interactive")
	if noInter && !yes {
		return &clierror.Error{Category: errorcodes.CategoryUsage, Code: errorcodes.ErrMissingRequiredFlag, Message: i18n.T("err_missing_required", "yes", "yes")}
	}
	if !noInter && !yes {
		fmt.Print(i18n.T("completion_bash_uninstall_plan", bashrc))
		if err := confirmCompletionAction(i18n.T("completion_uninstall_confirm")); err != nil {
			return err
		}
	}

	removed := []string{}

	bashrcContent, err := os.ReadFile(bashrc)
	if err != nil && !os.IsNotExist(err) {
		return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
	}
	if err == nil {
		newContent, changed, err := removeBashrcBlock(bashrcContent)
		if err != nil {
			return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
		}
		if changed {
			if err := os.WriteFile(bashrc, newContent, 0644); err != nil {
				return &clierror.Error{Category: errorcodes.CategorySystem, Code: errorcodes.ErrConfigWriteFailed, Message: err.Error()}
			}
			removed = append(removed, bashrc)
		}
	}

	if outputJSON {
		return output.WriteSuccessJSON(os.Stdout, output.SuccessEnvelope(map[string]interface{}{
			"shell":         "bash",
			"removed_paths": removed,
		}))
	}
	if len(removed) > 0 {
		fmt.Print(i18n.T("completion_uninstall_done", strings.Join(removed, "\n")))
		fmt.Print(i18n.T("completion_bash_uninstall_next_steps"))
	}
	return nil
}
