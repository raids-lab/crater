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

func commaSeparatedValueCompleter(values []string, descKey func(string) string) func(completion.Context) ([]completion.Candidate, error) {
	return func(ctx completion.Context) ([]completion.Candidate, error) {
		current := completion.CurrentWordPrefix(ctx)
		head := ""
		prefix := current
		if index := strings.LastIndex(current, ","); index >= 0 {
			head = current[:index+1]
			prefix = current[index+1:]
		}
		selected := map[string]struct{}{}
		for _, value := range strings.Split(strings.TrimSuffix(head, ","), ",") {
			value = strings.ToLower(strings.TrimSpace(value))
			if value != "" {
				selected[value] = struct{}{}
			}
		}
		prefix = strings.ToLower(strings.TrimSpace(prefix))
		out := make([]completion.Candidate, 0, len(values))
		for _, value := range values {
			lower := strings.ToLower(value)
			if _, ok := selected[lower]; ok {
				continue
			}
			if prefix != "" && !strings.HasPrefix(lower, prefix) {
				continue
			}
			candidate := completion.Candidate{Value: head + value}
			if descKey != nil {
				candidate.Description = i18n.T(descKey(value))
			}
			out = append(out, candidate)
		}
		return out, nil
	}
}
