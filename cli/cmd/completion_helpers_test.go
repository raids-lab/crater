package cmd

import (
	"reflect"
	"testing"

	"github.com/raids-lab/crater/cli/internal/completion"
)

func TestCommaSeparatedValueCompleter(t *testing.T) {
	complete := commaSeparatedValueCompleter([]string{"Running", "Pending", "Failed"}, nil)
	candidates, err := complete(completion.Context{
		Words:   []string{"Running,F"},
		Current: 1,
	})
	if err != nil {
		t.Fatalf("completion returned error: %v", err)
	}
	want := []completion.Candidate{{Value: "Running,Failed"}}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestCommaSeparatedValueCompleterOmitsSelectedValues(t *testing.T) {
	complete := commaSeparatedValueCompleter([]string{"Running", "Pending"}, nil)
	candidates, err := complete(completion.Context{
		Words:   []string{"Running,"},
		Current: 1,
	})
	if err != nil {
		t.Fatalf("completion returned error: %v", err)
	}
	want := []completion.Candidate{{Value: "Running,Pending"}}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}
