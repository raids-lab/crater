package cmd

import (
	"reflect"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/completion"
)

func TestNormalizeRemotePath(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		allowEmpty bool
		want       string
		wantErr    bool
	}{
		{name: "visible root", allowEmpty: true, want: ""},
		{name: "logical root", input: "user", want: "user"},
		{name: "leading and trailing slash", input: "/public/实验 data/", want: "public/实验 data"},
		{name: "account nested", input: "account/projects/run #1", want: "account/projects/run #1"},
		{name: "empty rejected", input: "", wantErr: true},
		{name: "unknown root", input: "admin/secret", wantErr: true},
		{name: "parent traversal", input: "user/../public", wantErr: true},
		{name: "current segment normalized", input: "user/./file", want: "user/file"},
		{name: "duplicate slash normalized", input: "user//file", want: "user/file"},
		{name: "backslash", input: `user\file`, wantErr: true},
		{name: "control byte", input: "user/file\nname", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeRemotePath(test.input, test.allowEmpty)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeRemotePath(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("normalizeRemotePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestFileRootCompleterUsesStaticLogicalRoots(t *testing.T) {
	candidates, err := fileRootCompleter(completion.Context{
		Words:   []string{"crater", "file", "ls", "u"},
		Current: 4,
	})
	if err != nil {
		t.Fatalf("fileRootCompleter: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Value != "user" {
		t.Fatalf("candidates = %#v, want user", candidates)
	}
}

func TestSortFileInfosDirectoriesFirstThenName(t *testing.T) {
	files := []api.FileInfo{
		{Name: "z.bin", Size: 1},
		{Name: "beta", IsDir: true},
		{Name: "Alpha.txt", Size: 2},
		{Name: "alpha", IsDir: true},
	}
	sortFileInfos(files)

	got := []string{files[0].Name, files[1].Name, files[2].Name, files[3].Name}
	want := []string{"alpha", "beta", "Alpha.txt", "z.bin"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted names = %#v, want %#v", got, want)
	}
}

func TestSortFileInfosKeepsEmptySliceStable(t *testing.T) {
	files := []api.FileInfo{}
	sortFileInfos(files)
	if files == nil || len(files) != 0 {
		t.Fatalf("files = %#v, want non-nil empty slice", files)
	}
}

func TestDisplayFileNameEscapesTerminalControlCharacters(t *testing.T) {
	if got := displayFileName("safe 文件.txt"); got != "safe 文件.txt" {
		t.Fatalf("safe name = %q", got)
	}
	if got := displayFileName("line\nname"); got != `"line\nname"` {
		t.Fatalf("control-character name = %q", got)
	}
	if got := displayFileName("color\x1b[31m"); got != `"color\x1b[31m"` {
		t.Fatalf("escape-sequence name = %q", got)
	}
}
