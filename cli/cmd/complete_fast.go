package cmd

import (
	"fmt"
	"os"

	"github.com/raids-lab/crater/cli/internal/completion"
	compshell "github.com/raids-lab/crater/cli/internal/completion/shell"
)

// RunCompleteFast 处理 `crater __complete ...` 快路径（不经 Execute / rootCmd.Execute）。
// 成功时 stdout 仅输出候选（每行一个 value）；错误写到 stderr 并返回非零退出码。
func RunCompleteFast(argv []string) int {
	if len(argv) < 1 {
		fmt.Fprintln(os.Stderr, "usage: crater __complete <shell> ...")
		return 2
	}
	switch argv[0] {
	case "zsh":
		return runCompleteWithAdapter("zsh", argv[1:])
	case "bash":
		return runCompleteWithAdapter("bash", argv[1:])
	default:
		fmt.Fprintf(os.Stderr, "unsupported shell: %q\n", argv[0])
		return 2
	}
}

func runCompleteWithAdapter(shellName string, args []string) int {
	initLanguageOnly()
	var (
		ctx completion.Context
		err error
	)
	switch shellName {
	case "zsh":
		ctx, err = compshell.ParseZshArgs(args)
	case "bash":
		ctx, err = compshell.ParseBashArgs(args)
	default:
		err = fmt.Errorf("unsupported shell: %q", shellName)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	cands, err := completion.Complete(rootCmd, ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	switch shellName {
	case "zsh":
		if err := compshell.RenderZsh(os.Stdout, cands); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 2
		}
	case "bash":
		if err := compshell.RenderBash(os.Stdout, cands); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 2
		}
	}
	return 0
}
