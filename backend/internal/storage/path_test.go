package storage

import "testing"

func TestCleanURLPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "leading slash", input: "/user/a.pdf", want: "user/a.pdf"},
		{name: "already clean", input: "user/a.pdf", want: "user/a.pdf"},
		{name: "windows separators", input: `\\user\\a.pdf`, want: "user/a.pdf"},
		{name: "dot path", input: "/", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "normalize", input: "/user//nested/../a.pdf", want: "user/a.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := cleanURLPath(tt.input)
			if got != tt.want {
				t.Fatalf("cleanURLPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetFirstToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "user path", input: "/user/a.pdf", want: "user"},
		{name: "windows separators", input: `\\user\\a.pdf`, want: "user"},
		{name: "root", input: "/", want: ""},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getFirstToken(tt.input)
			if got != tt.want {
				t.Fatalf("getFirstToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveDatasetStoragePath(t *testing.T) {
	t.Parallel()

	const base = "sugon-gpu-incoming/Models/Qwen/Qwen3-32B"
	tests := []struct {
		name      string
		relative  string
		want      string
		wantError bool
	}{
		{name: "resource root", relative: "", want: base},
		{name: "one level", relative: "/figures", want: base + "/figures"},
		{name: "nested", relative: "/figures/examples", want: base + "/figures/examples"},
		{name: "normalize", relative: "/figures/../config", want: base + "/config"},
		{name: "reject traversal", relative: "/../../other-model", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveDatasetStoragePath(base, tt.relative)
			if tt.wantError {
				if err == nil {
					t.Fatalf("resolveDatasetStoragePath(%q, %q) accepted traversal", base, tt.relative)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveDatasetStoragePath(%q, %q) error = %v", base, tt.relative, err)
			}
			if got != tt.want {
				t.Fatalf("resolveDatasetStoragePath(%q, %q) = %q, want %q", base, tt.relative, got, tt.want)
			}
		})
	}
}
