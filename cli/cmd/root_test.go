package cmd

import (
	"os"
	"testing"
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
