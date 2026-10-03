package ceph

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/raids-lab/crater/pkg/storagequota"
)

func TestStorageQuotaEnabledRequiresExplicitOptIn(t *testing.T) {
	t.Parallel()

	enabled := true
	disabled := false
	tests := []struct {
		name    string
		enabled *bool
		want    bool
	}{
		{name: "omitted", enabled: nil, want: false},
		{name: "disabled", enabled: &disabled, want: false},
		{name: "enabled", enabled: &enabled, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := storageQuotaEnabled(tt.enabled); got != tt.want {
				t.Fatalf("storageQuotaEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetDirectoryQuotaRejectsAmbiguousValues(t *testing.T) {
	t.Parallel()

	for _, quota := range []int64{-2, 0} {
		if err := setDirectoryQuota(nil, nil, "", "users/alice", quota); err == nil {
			t.Errorf("setDirectoryQuota() accepted quota %d", quota)
		}
	}
}

func TestNormalizeCephQuota(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		got  int64
		want int64
	}{
		{name: "Ceph unlimited sentinel", got: normalizeCephQuota(0), want: -1},
		{name: "positive quota", got: normalizeCephQuota(1024), want: 1024},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.got != test.want {
				t.Fatalf("normalizeCephQuota() = %d, want %d", test.got, test.want)
			}
		})
	}
}

func TestReconcileQuotaWriteError(t *testing.T) {
	t.Parallel()

	writeErr := io.ErrUnexpectedEOF
	tests := []struct {
		name        string
		requested   int64
		readback    int64
		readErr     error
		wantErr     bool
		wantUnknown bool
	}{
		{name: "write applied before response failed", requested: 1024, readback: 1024},
		{name: "unlimited write applied before response failed", requested: -1, readback: 0},
		{name: "write not applied", requested: 1024, readback: 512, wantErr: true},
		{name: "write outcome cannot be read", requested: 1024, readErr: io.EOF, wantErr: true, wantUnknown: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := reconcileQuotaWriteError(writeErr, tt.requested, func() (int64, error) {
				return tt.readback, tt.readErr
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("reconcileQuotaWriteError() error = %v, wantErr %v", err, tt.wantErr)
			}
			if errors.Is(err, ErrQuotaWriteOutcomeUnknown) != tt.wantUnknown {
				t.Fatalf("unknown outcome = %v, want %v", errors.Is(err, ErrQuotaWriteOutcomeUnknown), tt.wantUnknown)
			}
		})
	}
}

func TestSetStorageServerQuotaConfirmsWriteAfterResponseDisconnect(t *testing.T) {
	t.Parallel()

	var appliedQuota atomic.Int64
	appliedQuota.Store(-1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/internal/storage/quota":
			var request storagequota.Quota
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode quota request: %v", err)
				return
			}
			appliedQuota.Store(request.MaxBytes)
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack response: %v", err)
				return
			}
			_ = connection.Close()
		case r.Method == http.MethodGet && r.URL.Path == "/internal/storage/quota":
			_ = json.NewEncoder(w).Encode(storagequota.Quota{
				Path: r.URL.Query().Get("path"), MaxBytes: appliedQuota.Load(),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := storagequota.NewClient(server.URL, "test-secret")
	if err := setStorageServerQuota(client, "users/alice", 1024); err != nil {
		t.Fatalf("setStorageServerQuota() returned an error after confirmed write: %v", err)
	}
	if appliedQuota.Load() != 1024 {
		t.Fatalf("applied quota = %d, want 1024", appliedQuota.Load())
	}
}

func TestLogicalPathToStorageRelativePath(t *testing.T) {
	t.Parallel()

	prefixes := StoragePrefixConfig{User: "users", Account: "accounts", Public: "public"}
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "user", path: "/user/alice", want: "users/alice"},
		{name: "nested user path", path: "/user/alice/checkpoints", want: "users/alice/checkpoints"},
		{name: "public root", path: "/public", want: "public"},
		{name: "account", path: "/account/lab", want: "accounts/lab"},
		{name: "missing user space", path: "/user", wantErr: true},
		{name: "escape prefix", path: "/user/../../public", wantErr: true},
		{name: "unknown type", path: "/other/alice", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := logicalPathToStorageRelativePath(tt.path, prefixes)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}
