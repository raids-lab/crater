package completion

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/raids-lab/crater/cli/internal/i18n"
)

// Complete 根据 Cobra 命令树与上下文返回候选项（不含 shell 编码）。
func Complete(root *cobra.Command, ctx Context) ([]Candidate, error) {
	if root == nil {
		return nil, fmt.Errorf("root command is nil")
	}
	if ctx.Current < 1 || len(ctx.Words) == 0 {
		return nil, fmt.Errorf("invalid completion context")
	}

	// words[0] 应为可执行名（crater）；若缺失则假定从根开始。
	words := ctx.Words
	idx := ctx.Current - 1 // 0-based index of word being completed
	if idx < 0 || idx >= len(words) {
		return nil, fmt.Errorf("current index out of range")
	}
	if idx == 0 {
		// 正在补全可执行名本身，不提供候选。
		return nil, nil
	}
	prefix := CurrentWordPrefix(ctx)

	cmd, err := walkToCommand(root, words, idx)
	if err != nil {
		return nil, err
	}

	// Flag value completion (registered): handle "--flag=value" (including empty value) and
	// "--flag value"/"-p value". This runs before flag-name completion so that values can still
	// complete when the current token begins with '-' (e.g. "--mode=n" or "--mode=").
	if cands, ok := completeFlagValues(cmd, ctx); ok {
		return cands, nil
	}

	if strings.HasPrefix(prefix, "-") {
		return completeFlags(cmd, prefix), nil
	}
	// 1) 子命令补全
	if subs := completeSubcommands(cmd, prefix); len(subs) > 0 {
		return subs, nil
	}

	// 2) 位置参数补全（通过注册表）
	if cands, ok := completePositionals(cmd, ctx); ok {
		return cands, nil
	}
	return nil, nil
}

func completeFlagValues(cmd *cobra.Command, ctx Context) ([]Candidate, bool) {
	words := ctx.Words
	completingIdx := ctx.Current - 1
	if cmd == nil || completingIdx <= 0 || completingIdx >= len(words) {
		return nil, false
	}

	flags := visibleFlags(cmd)

	cur := words[completingIdx]
	prev := ""
	if completingIdx-1 >= 0 {
		prev = words[completingIdx-1]
	}

	var (
		flagName     string // long name without "--"
		inlinePrefix string // "--name=" if completing inline assignment
		valuePrefix  string // current value prefix (without "--name=")
	)

	// Case 1: inline form: --name=valuePrefix
	if strings.HasPrefix(cur, "--") && strings.Contains(cur, "=") {
		before, after, ok := strings.Cut(cur[2:], "=")
		if ok && before != "" {
			if f := flags.Lookup(before); f != nil && f.Value.Type() != "bool" {
				flagName = f.Name
				inlinePrefix = "--" + f.Name + "="
				valuePrefix = after
			}
		}
	}

	// Case 2: separate value: --name <valuePrefix> or -p <valuePrefix>
	if flagName == "" && prev != "" && strings.HasPrefix(prev, "-") {
		// Only complete values when the previous token is a non-bool flag expecting a value.
		if strings.HasPrefix(prev, "--") {
			name := strings.TrimPrefix(prev, "--")
			if name != "" {
				if f := flags.Lookup(name); f != nil && f.Value.Type() != "bool" {
					flagName = f.Name
					valuePrefix = cur
				}
			}
		} else if strings.HasPrefix(prev, "-") && len(prev) == 2 {
			sh := strings.TrimPrefix(prev, "-")
			if f := flags.ShorthandLookup(sh); f != nil && f.Value.Type() != "bool" {
				flagName = f.Name
				valuePrefix = cur
			}
		}
	}

	// Case 3 (bash wordbreaks): --name = <valuePrefix>
	// Bash may split '=' into its own token (COMP_WORDBREAKS contains '='),
	// so "--mode=n" can appear as ["--mode", "=", "n"].
	if flagName == "" && prev == "=" && completingIdx-2 >= 0 {
		prev2 := words[completingIdx-2]
		if strings.HasPrefix(prev2, "--") {
			name := strings.TrimPrefix(prev2, "--")
			if name != "" {
				if f := flags.Lookup(name); f != nil && f.Value.Type() != "bool" {
					flagName = f.Name
					valuePrefix = cur
				}
			}
		} else if strings.HasPrefix(prev2, "-") && len(prev2) == 2 {
			sh := strings.TrimPrefix(prev2, "-")
			if f := flags.ShorthandLookup(sh); f != nil && f.Value.Type() != "bool" {
				flagName = f.Name
				valuePrefix = cur
			}
		}
	}

	if flagName == "" {
		return nil, false
	}

	commandKey := commandKeyForCmd(cmd)
	if len(commandKey) == 0 {
		return nil, false
	}
	fn, ok := DefaultRegistry.flagValueCompleter(commandKey, flagName)
	if !ok {
		return nil, false
	}

	// Provide a cleaner view to the completer: current word is the raw value prefix.
	ctx2 := ctx
	words2 := append([]string(nil), words...)
	words2[completingIdx] = valuePrefix
	ctx2.Words = words2

	cands, err := fn(ctx2)
	if err != nil {
		return nil, false
	}
	if inlinePrefix != "" {
		for i := range cands {
			cands[i].Value = inlinePrefix + cands[i].Value
		}
	}
	return cands, true
}

// walkToCommand 根据已确定的词元（不含正在补全的那一个）定位到目标 *cobra.Command。
func walkToCommand(root *cobra.Command, words []string, completingIdx int) (*cobra.Command, error) {
	cmd := root
	// 从第一个子词开始（跳过程序名 words[0]），直到 completingIdx 之前。
	for i := 1; i < completingIdx && i < len(words); i++ {
		w := words[i]
		if w == "" {
			break
		}
		if w == "--" {
			break
		}
		if strings.HasPrefix(w, "-") {
			flags := visibleFlags(cmd)
			if flagConsumesNext(flags, w) {
				i++
			}
			continue
		}
		next := findChildByName(cmd, w)
		if next == nil {
			// 未知子命令：仍返回当前 cmd，由前缀匹配子命令名。
			break
		}
		cmd = next
	}
	return cmd, nil
}

func visibleFlags(cmd *cobra.Command) *pflag.FlagSet {
	flags := pflag.NewFlagSet("completion", pflag.ContinueOnError)
	for _, a := range ancestors(cmd) {
		flags.AddFlagSet(a.PersistentFlags())
	}
	flags.AddFlagSet(cmd.LocalFlags())
	return flags
}

func flagConsumesNext(flags *pflag.FlagSet, token string) bool {
	if flags == nil || token == "" || token == "--" {
		return false
	}
	if strings.HasPrefix(token, "--") {
		name := strings.TrimPrefix(token, "--")
		if name == "" || strings.Contains(name, "=") {
			return false
		}
		if f := flags.Lookup(name); f != nil {
			return f.Value.Type() != "bool"
		}
		return false
	}
	if strings.HasPrefix(token, "-") && len(token) == 2 {
		sh := strings.TrimPrefix(token, "-")
		if f := flags.ShorthandLookup(sh); f != nil {
			return f.Value.Type() != "bool"
		}
	}
	return false
}

func findChildByName(parent *cobra.Command, name string) *cobra.Command {
	name = strings.ToLower(name)
	for _, c := range parent.Commands() {
		if !c.Hidden && c.Name() == name {
			return c
		}
		for _, a := range c.Aliases {
			if strings.ToLower(a) == name {
				return c
			}
		}
	}
	return nil
}

func completeSubcommands(cmd *cobra.Command, prefix string) []Candidate {
	prefix = strings.ToLower(prefix)
	var out []Candidate
	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		n := sub.Name()
		if !strings.HasPrefix(strings.ToLower(n), prefix) {
			continue
		}
		out = append(out, Candidate{Value: n, Description: commandShortI18n(sub)})
	}
	return out
}

func completePositionals(cmd *cobra.Command, ctx Context) ([]Candidate, bool) {
	commandKey := commandKeyForCmd(cmd)
	if len(commandKey) == 0 {
		return nil, false
	}
	argIndex := positionalArgIndex(cmd, ctx.Words, ctx.Current-1)
	fn, ok := DefaultRegistry.positionalCompleter(commandKey, argIndex)
	if !ok {
		return nil, false
	}
	cands, err := fn(ctx)
	if err != nil {
		return nil, false
	}
	return cands, true
}

func commandKeyForCmd(cmd *cobra.Command) []string {
	// CommandPath() 包含根命令名 crater：例如 "crater config language"
	parts := strings.Split(cmd.CommandPath(), " ")
	if len(parts) <= 1 {
		return nil
	}
	return parts[1:]
}

func positionalArgIndex(cmd *cobra.Command, words []string, completingIdx int) int {
	// 计算正在补全的“位置参数序号”（0-based）。
	// 规则：从命令路径后的第一个 token 开始，跳过 flags 与它们的值，统计纯位置参数数量。
	// 对 --flag=value 视为一个 token。
	if completingIdx < 0 || len(words) == 0 {
		return 0
	}
	pathLen := len(strings.Split(cmd.CommandPath(), " "))
	start := pathLen
	if start < 0 {
		start = 0
	}
	if start > completingIdx {
		return 0
	}

	flags := visibleFlags(cmd)

	count := 0
	for i := start; i < completingIdx && i < len(words); i++ {
		w := words[i]
		if w == "" {
			continue
		}
		if strings.HasPrefix(w, "--") {
			if strings.Contains(w, "=") {
				continue
			}
			name := strings.TrimPrefix(w, "--")
			if f := flags.Lookup(name); f != nil && f.Value.Type() != "bool" {
				i++ // skip value
				continue
			}
			continue
		}
		if strings.HasPrefix(w, "-") && len(w) == 2 {
			// 简单处理单短旗标 -p value
			sh := strings.TrimPrefix(w, "-")
			if f := flags.ShorthandLookup(sh); f != nil && f.Value.Type() != "bool" {
				i++ // skip value
			}
			continue
		}
		// 非 flag token 视为位置参数
		count++
	}
	return count
}

func commandShortI18n(cmd *cobra.Command) string {
	// Avoid relying on Execute() help-text overwriting in __complete fast path.
	// Use the same key derivation rule as cmd/root.go (updateAllCommands):
	// crater auth login -> auth_login_short
	if cmd == nil {
		return ""
	}
	parts := strings.Split(cmd.CommandPath(), " ")
	keyPath := "root"
	if len(parts) > 1 {
		keyPath = strings.Join(parts[1:], "_")
	}
	k := keyPath + "_short"
	s := strings.TrimSpace(i18n.T(k))
	if s == "" || s == k {
		return strings.TrimSpace(cmd.Short)
	}
	return s
}

func completeFlags(cmd *cobra.Command, prefix string) []Candidate {
	prefix = strings.ToLower(prefix)
	var out []Candidate
	seen := make(map[string]struct{})
	add := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			long := "--" + f.Name
			if _, ok := seen[long]; ok {
				return
			}
			if strings.HasPrefix(long, prefix) {
				seen[long] = struct{}{}
				val := long
				if f.Value.Type() != "bool" {
					val = long + "="
				}
				out = append(out, Candidate{Value: val, Description: flagUsageI18n(f)})
			}
			if f.Shorthand != "" && f.Shorthand != "-" {
				sh := "-" + string(f.Shorthand)
				if strings.HasPrefix(strings.ToLower(sh), prefix) {
					if _, ok := seen[sh]; !ok {
						seen[sh] = struct{}{}
						out = append(out, Candidate{Value: sh, Description: flagUsageI18n(f)})
					}
				}
			}
		})
	}

	// PersistentFlags：自根向叶遍历，后者覆盖前者（与 Cobra 解析语义一致）。
	for _, a := range ancestors(cmd) {
		add(a.PersistentFlags())
	}
	add(cmd.LocalFlags())
	return out
}

func flagUsageI18n(f *pflag.Flag) string {
	if f == nil {
		return ""
	}
	k := "flag_" + f.Name
	s := strings.TrimSpace(i18n.T(k))
	if s != "" && s != k {
		return s
	}
	return strings.TrimSpace(f.Usage)
}

func ancestors(leaf *cobra.Command) []*cobra.Command {
	var chain []*cobra.Command
	for c := leaf; c != nil; c = c.Parent() {
		chain = append(chain, c)
	}
	// chain 为 leaf..root，反转为 root..leaf
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}
