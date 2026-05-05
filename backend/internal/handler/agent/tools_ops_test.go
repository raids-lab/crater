package agent

import "testing"

func TestIsAllowedDomain(t *testing.T) {
	allowed := []string{"kubernetes.io", "docs.nvidia.com"}
	if !isAllowedDomain("kubernetes.io", allowed) {
		t.Fatalf("exact allowed domain should pass")
	}
	if !isAllowedDomain("www.kubernetes.io", allowed) {
		t.Fatalf("subdomain should pass")
	}
	if !isAllowedDomain("developer.docs.nvidia.com", allowed) {
		t.Fatalf("nested subdomain should pass")
	}
	if isAllowedDomain("evil-kubernetes.io", allowed) {
		t.Fatalf("suffix-like domain should be rejected")
	}
	if isAllowedDomain("example.com", allowed) {
		t.Fatalf("non-allowlisted domain should be rejected")
	}
}

func TestSanitizeSandboxRelativePath(t *testing.T) {
	path, root, err := sanitizeSandboxRelativePath("runbooks/storage/pvc.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "runbooks/storage/pvc.md" || root != "runbooks" {
		t.Fatalf("unexpected normalized path/root: %q %q", path, root)
	}

	rejected := []string{
		"../runbooks/secret",
		"/runbooks/secret",
		"backend/internal",
		"runbooks/../../etc/passwd",
	}
	for _, candidate := range rejected {
		if _, _, err := sanitizeSandboxRelativePath(candidate); err == nil {
			t.Fatalf("expected %q to be rejected", candidate)
		}
	}
}
