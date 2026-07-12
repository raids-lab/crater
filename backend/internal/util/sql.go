package util

import "strings"

func ContainsPattern(search string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(search))
	return "%" + escaped + "%"
}
