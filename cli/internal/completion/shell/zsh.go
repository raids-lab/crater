package shell

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/cli/internal/completion"
)

// ZshInlineBlock returns a zsh rc snippet to be embedded into ~/.zshrc.
// It sets up compinit (if needed), defines _crater, and registers it via compdef.
// It delegates candidate computation to the crater `__complete` fast path.
func ZshInlineBlock(exePath string) string {
	return fmt.Sprintf(`autoload -Uz compinit && compinit

_crater() {
  local -a lines lines_eq lines_rest
  lines=("${(@f)$( command %[1]q __complete zsh "$CURRENT" "${words[@]}" )}")

  # Avoid inserting a trailing space after completing non-bool flags like "--mode=".
  # We keep the default behavior for other candidates, but for matches whose value
  # ends with '=', we pass -S '' to compadd (via _describe) to disable the space.
  local l
  for l in "${lines[@]}"; do
    if [[ "$l" == *"=:"* ]]; then
      lines_eq+=("$l")
    else
      lines_rest+=("$l")
    fi
  done
  (( ${#lines_eq[@]} )) && _describe -t crater 'crater' lines_eq -S ''
  (( ${#lines_rest[@]} )) && _describe -t crater 'crater' lines_rest
}

compdef _crater crater
`, exePath)
}

// ParseZshArgs parses: <CURRENT> <word>...
// CURRENT is 1-based (zsh). Words usually include the executable name at index 0.
func ParseZshArgs(args []string) (completion.Context, error) {
	if len(args) < 2 {
		return completion.Context{}, fmt.Errorf("usage: crater __complete zsh <CURRENT> <word>...")
	}
	cur, err := strconv.Atoi(args[0])
	if err != nil || cur < 1 {
		return completion.Context{}, fmt.Errorf("invalid CURRENT for zsh completion")
	}
	words := append([]string(nil), args[1:]...)
	// zsh may provide fewer words than CURRENT when completing a new word.
	for len(words) < cur {
		words = append(words, "")
	}
	return completion.Context{
		Shell:   "zsh",
		Words:   words,
		Current: cur,
	}, nil
}

// RenderZsh prints candidates as "value:description" lines for zsh `_describe`.
// Note: value/description fields escape ':' and '\' inside fields, but keep the
// separator ':' unescaped.
func RenderZsh(w io.Writer, cands []completion.Candidate) error {
	for _, c := range cands {
		_, err := fmt.Fprintf(w, "%s:%s\n", zshEscapeField(c.Value), zshEscapeField(c.Description))
		if err != nil {
			return err
		}
	}
	return nil
}

func zshEscapeField(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ":", "\\:")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

