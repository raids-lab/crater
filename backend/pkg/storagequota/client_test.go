package storagequota

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	t.Parallel()

	token := DeriveInternalToken("secret")
	if !Authenticate("secret", token) {
		t.Fatal("expected derived token to authenticate")
	}
	if Authenticate("secret", token+"x") {
		t.Fatal("expected invalid token to be rejected")
	}
	if !AuthenticateToken(token, " "+token+" ") {
		t.Fatal("expected direct internal token to authenticate")
	}
	if AuthenticateToken(token, "") {
		t.Fatal("expected empty direct token to be rejected")
	}
}

func TestClientGetUsageAndSetQuota(t *testing.T) {
	t.Parallel()

	const secret = "test-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Authenticate(secret, r.Header.Get(InternalTokenHeader)) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/internal/storage/usage":
			if got := r.URL.Query().Get("path"); got != "users/alice" {
				t.Errorf("unexpected usage path %q", got)
			}
			_ = json.NewEncoder(w).Encode(Usage{Path: "users/alice", Bytes: 42})
		case r.Method == http.MethodPut && r.URL.Path == "/internal/storage/quota":
			var request Quota
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode quota request: %v", err)
			}
			_ = json.NewEncoder(w).Encode(request)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, secret)
	usage, err := client.GetUsage(context.Background(), "users/alice")
	if err != nil || usage.Bytes != 42 {
		t.Fatalf("GetUsage() = %+v, %v", usage, err)
	}
	quota, err := client.SetQuota(context.Background(), "users/alice", 1024)
	if err != nil || quota.MaxBytes != 1024 {
		t.Fatalf("SetQuota() = %+v, %v", quota, err)
	}
}

func TestNormalizeProvider(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":               ProviderAuto,
		"AUTO":           ProviderAuto,
		"storage-server": ProviderStorageServer,
		"storageServer":  ProviderStorageServer,
		"toolbox":        ProviderToolbox,
		"disabled":       ProviderDisabled,
		"invalid":        ProviderDisabled,
	}
	for input, want := range tests {
		if got := NormalizeProvider(input); got != want {
			t.Errorf("NormalizeProvider(%q) = %q, want %q", input, got, want)
		}
	}
}
