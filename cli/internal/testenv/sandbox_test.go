package testenv

import "testing"

func Test_truthyEnv(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1", true},
		{" true ", true},
		{"YES", true},
		{"y", true},
		{"on", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
		{"timeout", false},
	}
	for _, tc := range cases {
		if got := truthyEnv(tc.in); got != tc.want {
			t.Fatalf("truthyEnv(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestSandboxEnabled(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "")
	if SandboxEnabled() {
		t.Fatal("SandboxEnabled() should be false when env empty")
	}

	t.Setenv("CRATER_TEST_SANDBOX", "1")
	if !SandboxEnabled() {
		t.Fatal("SandboxEnabled() should be true when CRATER_TEST_SANDBOX=1")
	}
}

func TestSandboxSessionEnabled(t *testing.T) {
	// neither enabled
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")
	if SandboxSessionEnabled() {
		t.Fatal("SandboxSessionEnabled() should be false when both env empty")
	}

	// feature-specific session sandbox only
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "true")
	if !SandboxSessionEnabled() {
		t.Fatal("SandboxSessionEnabled() should be true when CRATER_TEST_SANDBOX_SESSION=true")
	}

	// global sandbox implies session sandbox
	t.Setenv("CRATER_TEST_SANDBOX", "yes")
	t.Setenv("CRATER_TEST_SANDBOX_SESSION", "")
	if !SandboxSessionEnabled() {
		t.Fatal("SandboxSessionEnabled() should be true when CRATER_TEST_SANDBOX=yes")
	}
}

func TestSandboxHTTPMode_truthyMeansTimeout(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "1")
	if got := SandboxHTTPMode(); got != "timeout" {
		t.Fatalf("SandboxHTTPMode()=%q want %q", got, "timeout")
	}
}

func TestSandboxHTTPMode_globalSandboxEmptyFallsBackToTimeout(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "1")
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "")
	if got := SandboxHTTPMode(); got != "timeout" {
		t.Fatalf("SandboxHTTPMode()=%q want %q", got, "timeout")
	}
}

func TestSandboxHTTPMode_explicitAllowedModes(t *testing.T) {
	// When sandbox is not enabled, explicit allowed modes still opt into HTTP simulation.
	t.Setenv("CRATER_TEST_SANDBOX", "")
	for _, v := range []string{"timeout", "hang", "error404", "404"} {
		t.Setenv("CRATER_TEST_SANDBOX_HTTP", v)
		if got := SandboxHTTPMode(); got != v {
			t.Fatalf("SandboxHTTPMode(HTTP=%q)=%q want %q", v, got, v)
		}
	}
}

func TestSandboxHTTPMode_explicitAllowedModesAreCaseInsensitive(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "")
	cases := map[string]string{
		" Timeout ": "timeout",
		"HANG":      "hang",
		"Error404":  "error404",
	}
	for in, want := range cases {
		t.Setenv("CRATER_TEST_SANDBOX_HTTP", in)
		if got := SandboxHTTPMode(); got != want {
			t.Fatalf("SandboxHTTPMode(HTTP=%q)=%q want %q", in, got, want)
		}
	}
}

func TestSandboxHTTPMode_invalidMode_withoutSandboxKeepsValue(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "")
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "weird-mode")
	if got := SandboxHTTPMode(); got != "weird-mode" {
		t.Fatalf("SandboxHTTPMode()=%q want %q", got, "weird-mode")
	}
}

func TestSandboxHTTPMode_invalidMode_withSandboxFallsBackToTimeout(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "1")
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "weird-mode")
	if got := SandboxHTTPMode(); got != "timeout" {
		t.Fatalf("SandboxHTTPMode()=%q want %q", got, "timeout")
	}
}
