package snaptest

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/txtar"
)

// Case is one snapshot row: paths in txtar are {lang}/{ID}/{argv,exit,stdout,stderr}.
type Case struct {
	ID   string
	Args []string // argv after program name, e.g. {"config","language","--no-interactive"}
}

// MatchOrUpdateGolden compares each Case result against goldenPath (txtar), or
// rewrites the file when update is true.
func MatchOrUpdateGolden(goldenPath, lang string, cases []Case, results []*Result, update bool) error {
	if len(cases) != len(results) {
		return fmt.Errorf("snaptest: cases/results length mismatch")
	}
	if update {
		ar := buildArchive(lang, cases, results)
		data := txtar.Format(ar)
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(goldenPath, data, 0644)
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		return err
	}
	ar := txtar.Parse(data)
	want := make(map[string][]byte, len(ar.Files))
	for _, f := range ar.Files {
		want[f.Name] = f.Data
	}
	for i := range cases {
		prefix := lang + "/" + cases[i].ID
		if err := comparePrefix(prefix, want, cases[i].Args, results[i]); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
	}
	return nil
}

func comparePrefix(prefix string, want map[string][]byte, args []string, got *Result) error {
	argvWant, ok := want[prefix+"/argv"]
	if !ok {
		return fmt.Errorf("missing golden file %q", prefix+"/argv")
	}
	argvGot := ArgvLine(args) + "\n"
	if trimTrailingNewlines(string(argvWant)) != trimTrailingNewlines(argvGot) {
		return fmt.Errorf("%s/argv mismatch\n--- want ---\n%s--- got ---\n%s", prefix, string(argvWant), argvGot)
	}
	exitWant, ok := want[prefix+"/exit"]
	if !ok {
		return fmt.Errorf("missing golden file %q", prefix+"/exit")
	}
	exitGot := strconv.Itoa(got.ExitCode) + "\n"
	if trimTrailingNewlines(string(exitWant)) != trimTrailingNewlines(exitGot) {
		return fmt.Errorf("%s/exit mismatch: want %q got %q", prefix, string(exitWant), exitGot)
	}
	if err := compareStream(prefix, "stdout", want, got.Stdout); err != nil {
		return err
	}
	if err := compareStream(prefix, "stderr", want, got.Stderr); err != nil {
		return err
	}
	return nil
}

func compareStream(prefix, stream string, want map[string][]byte, gotBody string) error {
	name := prefix + "/" + stream
	exp, ok := want[name]
	if !ok {
		return fmt.Errorf("missing golden file %q", name)
	}
	if trimTrailingNewlines(string(exp)) != trimTrailingNewlines(gotBody) {
		return fmt.Errorf("%q mismatch\n--- want ---\n%s\n--- got ---\n%s", name, string(exp), gotBody)
	}
	return nil
}

func trimTrailingNewlines(s string) string {
	return strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func buildArchive(lang string, cases []Case, results []*Result) *txtar.Archive {
	ar := &txtar.Archive{
		Comment: []byte("# Crater CLI snapshot bundle (txtar). Regenerate: make snapshot-update (or UPDATE_SNAPSHOTS=1 go test ./test/snapshots/...)\n"),
	}
	for i := range cases {
		prefix := lang + "/" + cases[i].ID
		ar.Files = append(ar.Files,
			txtar.File{Name: prefix + "/argv", Data: []byte(ArgvLine(cases[i].Args) + "\n")},
			txtar.File{Name: prefix + "/exit", Data: []byte(strconv.Itoa(results[i].ExitCode) + "\n")},
			txtar.File{Name: prefix + "/stdout", Data: []byte(results[i].Stdout)},
			txtar.File{Name: prefix + "/stderr", Data: []byte(results[i].Stderr)},
		)
	}
	return ar
}
