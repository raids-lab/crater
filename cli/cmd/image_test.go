package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/raids-lab/crater/cli/internal/api"
	"github.com/raids-lab/crater/cli/internal/i18n"
	"github.com/spf13/cobra"
)

func TestPrintImageTableIncludesVisibility(t *testing.T) {
	previousLanguage := i18n.GetCurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })

	got := captureImageTestStdout(t, func() {
		printImageTable([]api.ImageInfo{{
			ID:               1,
			ImageLink:        "registry.example/demo:v1",
			TaskType:         "custom",
			ImageShareStatus: "Private",
			Archs:            []string{"linux/amd64"},
			UserInfo:         api.UserInfo{Nickname: "alice"},
		}})
	})
	if !strings.Contains(got, "VISIBILITY") || !strings.Contains(got, "Private") {
		t.Fatalf("table must display image visibility, got %q", got)
	}
	if strings.Contains(got, "CREATED") {
		t.Fatalf("table must not display the removed CREATED column, got %q", got)
	}
}

func TestImageCommandGroupsRejectUnknownSubcommands(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "build", cmd: imageBuildCmd},
		{name: "share", cmd: imageShareCmd},
		{name: "user cuda", cmd: imageCudaCmd},
		{name: "harbor", cmd: imageHarborCmd},
		{name: "quota", cmd: imageQuotaCmd},
		{name: "admin image", cmd: adminImageCmd},
		{name: "admin cuda", cmd: adminImageCudaCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cobra treats arguments on a command group as positional input when
			// no matching child exists. Exercise the group's RunE directly so the
			// contract remains independent of the process-wide root command state.
			err := tt.cmd.RunE(tt.cmd, []string{"unknown"})
			if err == nil || !strings.Contains(err.Error(), `unknown command "unknown"`) {
				t.Fatalf("unknown subcommand error = %v", err)
			}
		})
	}
}

func captureImageTestStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe: %v", err)
	}
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	return string(output)
}
