package shell

import (
	"fmt"
	"io"
	"strconv"

	"github.com/raids-lab/crater/cli/internal/completion"
)

// BashInlineBlock returns a bash rc snippet to be embedded into ~/.bashrc.
// It defines _crater and registers it via `complete -F`.
// It delegates candidate computation to the crater `__complete` fast path.
func BashInlineBlock(exePath string) string {
	// Use a bash function + complete -F to register completion for `crater`.
	// The script calls: crater __complete bash "$COMP_CWORD" "${COMP_WORDS[@]}"
	return fmt.Sprintf(`# bash completion for crater
_crater() {
  local IFS=$'\n'
  local -a lines
  lines=($(command %q __complete bash "$COMP_CWORD" "${COMP_WORDS[@]}"))
  COMPREPLY=()
  local line
  local nospace=0
  for line in "${lines[@]}"; do
    COMPREPLY+=("$line")
    [[ "$line" == *"=" ]] && nospace=1
  done
  # Avoid inserting a trailing space after completing non-bool flags like "--mode=".
  # This only affects the current completion invocation.
  if (( nospace )); then
    compopt -o nospace 2>/dev/null
  fi
  return 0
}

complete -F _crater crater
`, exePath)
}

// ParseBashArgs parses: <COMP_CWORD> <word>...
// COMP_CWORD is 0-based (bash): index of current word in COMP_WORDS.
// We convert it to completion.Context.Current (1-based) for engine reuse.
func ParseBashArgs(args []string) (completion.Context, error) {
	if len(args) < 2 {
		return completion.Context{}, fmt.Errorf("usage: crater __complete bash <COMP_CWORD> <word>...")
	}
	cword, err := strconv.Atoi(args[0])
	if err != nil || cword < 0 {
		return completion.Context{}, fmt.Errorf("invalid COMP_CWORD for bash completion")
	}
	words := append([]string(nil), args[1:]...)
	// Current is 1-based index of the word being completed.
	cur := cword + 1
	if cur < 1 {
		cur = 1
	}
	for len(words) < cur {
		words = append(words, "")
	}
	return completion.Context{
		Shell:   "bash",
		Words:   words,
		Current: cur,
	}, nil
}

// RenderBash prints candidate values one per line (bash typically ignores descriptions).
func RenderBash(w io.Writer, cands []completion.Candidate) error {
	for _, c := range cands {
		if _, err := fmt.Fprintln(w, c.Value); err != nil {
			return err
		}
	}
	return nil
}
