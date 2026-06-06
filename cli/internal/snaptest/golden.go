package snaptest

import (
	"path/filepath"
	"testing"
)

// GoldenFile returns the path to a txtar golden:
//
//	testdata/snapshots/{domain}/{stem}.{lang}.txtar
//
// under the CLI module root (see ModuleRoot). Example: domain=config, stem=language, lang=en
// -> testdata/snapshots/config/language.en.txtar
func GoldenFile(domain, stem, lang string) (string, error) {
	root, err := ModuleRoot()
	if err != nil {
		return "", err
	}
	name := stem + "." + lang + ".txtar"
	return filepath.Join(root, "testdata", "snapshots", domain, name), nil
}

// GoldenFileT is like GoldenFile but fails the test on error.
func GoldenFileT(t *testing.T, domain, stem, lang string) string {
	t.Helper()
	p, err := GoldenFile(domain, stem, lang)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
