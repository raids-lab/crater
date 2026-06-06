package cmd

import "testing"

func TestCollectLoginUsageIssuesNonInteractiveAggregates(t *testing.T) {
	t.Parallel()
	issues := collectLoginUsageIssues(loginInput{
		platformURL: "http://example",
		mode:        authModeLDAP,
	}, true)
	if len(issues) != 2 {
		t.Fatalf("want 2 issues (username, password), got %d", len(issues))
	}
}

func TestCollectLoginUsageIssuesInvalidModeAndMissing(t *testing.T) {
	t.Parallel()
	issues := collectLoginUsageIssues(loginInput{
		mode: "bad",
	}, true)
	if len(issues) != 4 {
		t.Fatalf("want 4 issues, got %d", len(issues))
	}
}
