package cmd

import (
	"os"
	"testing"

	"github.com/raids-lab/crater/cli/internal/i18n"
)

func TestBootstrapJSONFlagFromArgsMatchesPflagBoolSemantics(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "bare json enables output",
			args: []string{"crater", "--json"},
			want: true,
		},
		{
			name: "equals false disables output",
			args: []string{"crater", "--json=false"},
			want: false,
		},
		{
			name: "space separated false is not consumed as bool value",
			args: []string{"crater", "--json", "false"},
			want: true,
		},
		{
			name: "double dash stops flag scanning",
			args: []string{"crater", "--", "--json"},
			want: false,
		},
		{
			name: "last json occurrence wins",
			args: []string{"crater", "--json", "--json=false"},
			want: false,
		},
	}

	oldArgs := os.Args
	oldOutputJSON := outputJSON
	t.Cleanup(func() {
		os.Args = oldArgs
		outputJSON = oldOutputJSON
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			outputJSON = false

			bootstrapJSONFlagFromArgs()

			if outputJSON != tt.want {
				t.Fatalf("outputJSON = %v, want %v", outputJSON, tt.want)
			}
		})
	}
}

func TestTranslatedFlagUsagePrecedence(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	tests := []struct {
		name     string
		keyPath  string
		flagName string
		want     string
	}{
		{
			name:     "command-specific text wins",
			keyPath:  "order_submit",
			flagName: "name",
			want:     "Approval target name",
		},
		{
			name:     "image-domain text wins over global text",
			keyPath:  "image_upload",
			flagName: "type",
			want:     "Image task type",
		},
		{
			name:     "admin image uses image-domain text",
			keyPath:  "admin_image_type",
			flagName: "type",
			want:     "Image task type",
		},
		{
			name:     "non-image command falls back to global text",
			keyPath:  "resource_ls",
			flagName: "type",
			want:     "Filter by type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := translatedFlagUsage(tt.keyPath, tt.flagName)
			if !ok {
				t.Fatalf("translatedFlagUsage(%q, %q) did not find a translation", tt.keyPath, tt.flagName)
			}
			if got != tt.want {
				t.Fatalf("translatedFlagUsage(%q, %q) = %q, want %q", tt.keyPath, tt.flagName, got, tt.want)
			}
		})
	}
}
