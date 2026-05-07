package completion

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// 本文件测试会修改 DefaultRegistry；请勿在子测试中使用 t.Parallel()。

func cloneRegistry(r *Registry) *Registry {
	c := NewRegistry()
	for k, m := range r.positional {
		inner := make(map[int]func(Context) ([]Candidate, error), len(m))
		for ik, fn := range m {
			inner[ik] = fn
		}
		c.positional[k] = inner
	}
	for k, m := range r.flagValue {
		inner := make(map[string]func(Context) ([]Candidate, error), len(m))
		for ik, fn := range m {
			inner[ik] = fn
		}
		c.flagValue[k] = inner
	}
	return c
}

func restoreRegistry(backup *Registry) {
	DefaultRegistry.positional = backup.positional
	DefaultRegistry.flagValue = backup.flagValue
}

func withRegistryReset(t *testing.T) {
	t.Helper()
	backup := cloneRegistry(DefaultRegistry)
	t.Cleanup(func() { restoreRegistry(backup) })
}

func candidateValues(cands []Candidate) []string {
	s := make([]string, len(cands))
	for i := range cands {
		s[i] = cands[i].Value
	}
	sort.Strings(s)
	return s
}

func testTreeFlagMode(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "crater", Short: "root"}
	grp := &cobra.Command{Use: "grp", Short: "grp"}
	leaf := &cobra.Command{Use: "leaf", Short: "leaf"}
	leaf.Flags().String("mode", "", "auth mode")
	grp.AddCommand(leaf)
	root.AddCommand(grp)
	return root
}

func testTreeUnknownSub(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "crater", Short: "root"}
	box := &cobra.Command{Use: "box", Short: "box"}
	box.AddCommand(&cobra.Command{Use: "alpha", Short: "alpha"})
	box.AddCommand(&cobra.Command{Use: "beta", Short: "beta"})
	root.AddCommand(box)
	return root
}

func modeCompleterLDAPNormal(ctx Context) ([]Candidate, error) {
	p := strings.ToLower(CurrentWordPrefix(ctx))
	out := make([]Candidate, 0, 2)
	for _, v := range []string{"ldap", "normal"} {
		if p == "" || strings.HasPrefix(v, p) {
			out = append(out, Candidate{Value: v})
		}
	}
	return out, nil
}

func TestComplete_invalidContext(t *testing.T) {
	root := &cobra.Command{Use: "crater"}
	_, err := Complete(root, Context{Words: []string{"crater"}, Current: 0})
	if err == nil {
		t.Fatal("expected error for Current < 1")
	}
	_, err = Complete(nil, Context{Words: []string{"crater", "x"}, Current: 2})
	if err == nil {
		t.Fatal("expected error for nil root")
	}
}

func TestComplete_unknownSubcommand_thenSiblings(t *testing.T) {
	root := testTreeUnknownSub(t)
	// crater box typo <TAB> — 停在 box，应列出 alpha、beta
	ctx := Context{
		Words:   []string{"crater", "box", "typo", ""},
		Current: 4,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := candidateValues(cands)
	want := []string{"alpha", "beta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("values: got %v want %v", got, want)
	}
}

func TestNormalizeCommandKey_mixedCaseRegister(t *testing.T) {
	withRegistryReset(t)
	root := testTreeFlagMode(t)
	RegisterFlagValue([]string{"GRP", "Leaf"}, "mode", modeCompleterLDAPNormal)

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", "--mode=ld"},
		Current: 4,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := candidateValues(cands)
	if len(got) != 1 || got[0] != "--mode=ldap" {
		t.Fatalf("got %v want [--mode=ldap]", got)
	}
}

func TestComplete_flagValue_inlineEquals(t *testing.T) {
	withRegistryReset(t)
	root := testTreeFlagMode(t)
	RegisterFlagValue([]string{"grp", "leaf"}, "mode", modeCompleterLDAPNormal)

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", "--mode=ld"},
		Current: 4,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := candidateValues(cands); len(got) != 1 || got[0] != "--mode=ldap" {
		t.Fatalf("got %v want [--mode=ldap]", got)
	}
}

func TestComplete_flagValue_separateToken(t *testing.T) {
	withRegistryReset(t)
	root := testTreeFlagMode(t)
	RegisterFlagValue([]string{"grp", "leaf"}, "mode", modeCompleterLDAPNormal)

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", "--mode", "ld"},
		Current: 5,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := candidateValues(cands); len(got) != 1 || got[0] != "ldap" {
		t.Fatalf("got %v want [ldap]", got)
	}
}

func TestComplete_flagValue_bashSplitEquals(t *testing.T) {
	withRegistryReset(t)
	root := testTreeFlagMode(t)
	RegisterFlagValue([]string{"grp", "leaf"}, "mode", modeCompleterLDAPNormal)

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", "--mode", "=", "ld"},
		Current: 6,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := candidateValues(cands); len(got) != 1 || got[0] != "ldap" {
		t.Fatalf("got %v want [ldap]", got)
	}
}

func TestComplete_positionalAfterPersistentBoolFlag(t *testing.T) {
	withRegistryReset(t)
	root := &cobra.Command{Use: "crater", Short: "root"}
	root.PersistentFlags().Bool("no-interactive", false, "")
	grp := &cobra.Command{Use: "grp", Short: "grp"}
	leaf := &cobra.Command{Use: "leaf", Short: "leaf"}
	grp.AddCommand(leaf)
	root.AddCommand(grp)

	RegisterPositional([]string{"grp", "leaf"}, 0, func(ctx Context) ([]Candidate, error) {
		p := strings.ToLower(CurrentWordPrefix(ctx))
		var out []Candidate
		for _, v := range []string{"en", "zh-CN"} {
			if p == "" || strings.HasPrefix(v, p) {
				out = append(out, Candidate{Value: v})
			}
		}
		return out, nil
	})

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", "--no-interactive", ""},
		Current: 5,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := candidateValues(cands)
	want := []string{"en", "zh-CN"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRegistry_duplicateRegisterOverwrites(t *testing.T) {
	withRegistryReset(t)
	root := testTreeFlagMode(t)

	RegisterFlagValue([]string{"grp", "leaf"}, "mode", func(Context) ([]Candidate, error) {
		return []Candidate{{Value: "first"}}, nil
	})
	RegisterFlagValue([]string{"grp", "leaf"}, "mode", modeCompleterLDAPNormal)

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", "--mode=n"},
		Current: 4,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := candidateValues(cands)
	if len(got) != 1 || got[0] != "--mode=normal" {
		t.Fatalf("second register should win: got %v want [--mode=normal]", got)
	}
}

func TestRegistry_duplicatePositionalOverwrites(t *testing.T) {
	withRegistryReset(t)
	root := testTreeFlagMode(t)

	RegisterPositional([]string{"grp", "leaf"}, 0, func(Context) ([]Candidate, error) {
		return []Candidate{{Value: "only-second"}}, nil
	})
	RegisterPositional([]string{"grp", "leaf"}, 0, func(Context) ([]Candidate, error) {
		return []Candidate{{Value: "winner"}}, nil
	})

	ctx := Context{
		Words:   []string{"crater", "grp", "leaf", ""},
		Current: 4,
	}
	cands, err := Complete(root, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].Value != "winner" {
		t.Fatalf("got %+v want one candidate winner", cands)
	}
}
