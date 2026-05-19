package completion

// Context 是补全内核的统一输入。
// 目前由各 shell 适配层（internal/completion/shell/*）构造并填充。
type Context struct {
	Shell string   // 例如 "zsh" / "bash"（用于调试与将来 directive）
	Words []string // 完整词元，通常含根命令名（如 crater）
	// Current 为 1-based，与 zsh 的 CURRENT 对齐：正在补全的词下标为 Current-1。
	Current int
}

// CurrentWordPrefix 返回正在补全词中的已输入片段（可能为空）；下标越界时返回空串。
// RegisterPositional / RegisterFlagValue 的回调可用其代替手写 ctx.Words[ctx.Current-1]。
// flag 取值路径下，传入的 ctx 已由 completeFlagValues 将当前槽位归一为纯值前缀，本函数与该语义一致。
func CurrentWordPrefix(ctx Context) string {
	i := ctx.Current - 1
	if i < 0 || i >= len(ctx.Words) {
		return ""
	}
	return ctx.Words[i]
}

// Candidate 表示一个补全候选项。
type Candidate struct {
	Value       string // 插入命令行的 token（不翻译）
	Description string // 可选，人类可读描述（i18n）
}
