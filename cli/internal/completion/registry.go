package completion

// Registry 提供高级补全注册表。
type Registry struct {
	positional map[string]map[int]func(ctx Context) ([]Candidate, error)    // cmdKey -> argIndex -> fn
	flagValue  map[string]map[string]func(ctx Context) ([]Candidate, error) // cmdKey -> flagName -> fn
}

func NewRegistry() *Registry {
	return &Registry{
		positional: make(map[string]map[int]func(ctx Context) ([]Candidate, error)),
		flagValue:  make(map[string]map[string]func(ctx Context) ([]Candidate, error)),
	}
}

func (r *Registry) RegisterPositional(commandKey []string, argIndex int, fn func(ctx Context) ([]Candidate, error)) {
	if r == nil || fn == nil || argIndex < 0 {
		return
	}
	k := normalizeCommandKey(commandKey)
	m := r.positional[k]
	if m == nil {
		m = make(map[int]func(ctx Context) ([]Candidate, error))
		r.positional[k] = m
	}
	m[argIndex] = fn
}

func (r *Registry) RegisterFlagValue(commandKey []string, flagName string, fn func(ctx Context) ([]Candidate, error)) {
	if r == nil || fn == nil || flagName == "" {
		return
	}
	k := normalizeCommandKey(commandKey)
	m := r.flagValue[k]
	if m == nil {
		m = make(map[string]func(ctx Context) ([]Candidate, error))
		r.flagValue[k] = m
	}
	m[flagName] = fn
}

func (r *Registry) positionalCompleter(commandKey []string, argIndex int) (func(ctx Context) ([]Candidate, error), bool) {
	if r == nil {
		return nil, false
	}
	k := normalizeCommandKey(commandKey)
	m := r.positional[k]
	if m == nil {
		return nil, false
	}
	fn, ok := m[argIndex]
	return fn, ok
}

func (r *Registry) flagValueCompleter(commandKey []string, flagName string) (func(ctx Context) ([]Candidate, error), bool) {
	if r == nil || flagName == "" {
		return nil, false
	}
	k := normalizeCommandKey(commandKey)
	m := r.flagValue[k]
	if m == nil {
		return nil, false
	}
	fn, ok := m[flagName]
	return fn, ok
}

func normalizeCommandKey(parts []string) string {
	// 与临时设计文档一致：小写、用 / 连接（不含根命令 crater）。
	var b []byte
	for i, p := range parts {
		if i > 0 {
			b = append(b, '/')
		}
		for _, ch := range p {
			if ch >= 'A' && ch <= 'Z' {
				ch += 'a' - 'A'
			}
			b = append(b, byte(ch))
		}
	}
	return string(b)
}

// DefaultRegistry 是默认注册表，供 cmd 在 init() 中注册补全。
var DefaultRegistry = NewRegistry()

// RegisterPositional 通过默认注册表注册位置参数补全。
// 回调内可用 CurrentWordPrefix(ctx) 取得当前正在补全词中的已输入片段（可能为空）。
func RegisterPositional(commandKey []string, argIndex int, fn func(ctx Context) ([]Candidate, error)) {
	DefaultRegistry.RegisterPositional(commandKey, argIndex, fn)
}

// RegisterFlagValue 通过默认注册表注册 flag 值补全。
// 回调内可用 CurrentWordPrefix(ctx) 取得归一后的值前缀（可能为空）。
func RegisterFlagValue(commandKey []string, flagName string, fn func(ctx Context) ([]Candidate, error)) {
	DefaultRegistry.RegisterFlagValue(commandKey, flagName, fn)
}
