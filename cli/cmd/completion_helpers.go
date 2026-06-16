package cmd

import (
	"strings"

	"github.com/raids-lab/crater/cli/internal/completion"
	"github.com/raids-lab/crater/cli/internal/i18n"
)

func staticValueCompleter(values []string, descKey func(string) string) func(completion.Context) ([]completion.Candidate, error) {
	return func(ctx completion.Context) ([]completion.Candidate, error) {
		prefix := strings.ToLower(completion.CurrentWordPrefix(ctx))
		out := make([]completion.Candidate, 0, len(values))
		for _, v := range values {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(v), prefix) {
				continue
			}
			c := completion.Candidate{Value: v}
			if descKey != nil {
				c.Description = i18n.T(descKey(v))
			}
			out = append(out, c)
		}
		return out, nil
	}
}
