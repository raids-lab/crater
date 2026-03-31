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
		tt := tt
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getFirstToken(tt.input)
			if got != tt.want {
				t.Fatalf("getFirstToken(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
