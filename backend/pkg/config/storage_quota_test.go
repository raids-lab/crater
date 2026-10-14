package config

import "testing"

const customRookNamespace = "storage-system"

func TestStorageQuotaProviderValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider string
		want     bool
	}{
		{provider: "", want: true},
		{provider: "auto", want: true},
		{provider: "storageServer", want: true},
		{provider: "storage-server", want: true},
		{provider: "toolbox", want: true},
		{provider: "disabled", want: true},
		{provider: "cephfs", want: false},
	}

	for _, tt := range tests {
		if got := isStorageQuotaProviderValid(tt.provider); got != tt.want {
			t.Errorf("isStorageQuotaProviderValid(%q) = %v, want %v", tt.provider, got, tt.want)
		}
	}
}

func TestStorageQuotaClusterDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	var defaults Config
	if got := defaults.StorageQuotaRookNamespace(); got != "rook-ceph" {
		t.Fatalf("default Rook namespace = %q", got)
	}
	if got := defaults.StorageQuotaCephFSCSIDriver(); got != "rook-ceph.cephfs.csi.ceph.com" {
		t.Fatalf("default CephFS CSI driver = %q", got)
	}
	if got := defaults.StorageQuotaToolboxLabelSelector(); got != "app=rook-ceph-tools" {
		t.Fatalf("default toolbox selector = %q", got)
	}
	if got := defaults.StorageQuotaCephFSName(); got != "cephfs" {
		t.Fatalf("default CephFS name = %q", got)
	}

	var custom Config
	custom.Storage.Quota.RookNamespace = customRookNamespace
	custom.Storage.Quota.CephFSCSIDriver = "custom.cephfs.csi.example.com"
	custom.Storage.Quota.ToolboxLabelSelector = "app=ceph-toolbox"
	custom.Storage.Quota.CephFSName = "shared-fs"
	if got := custom.StorageQuotaRookNamespace(); got != customRookNamespace {
		t.Fatalf("custom Rook namespace = %q", got)
	}
	if got := custom.StorageQuotaCephFSCSIDriver(); got != "custom.cephfs.csi.example.com" {
		t.Fatalf("custom CephFS CSI driver = %q", got)
	}
	if got := custom.StorageQuotaToolboxLabelSelector(); got != "app=ceph-toolbox" {
		t.Fatalf("custom toolbox selector = %q", got)
	}
	if got := custom.StorageQuotaCephFSName(); got != "shared-fs" {
		t.Fatalf("custom CephFS name = %q", got)
	}

	var derived Config
	derived.Storage.Quota.RookNamespace = customRookNamespace
	if got := derived.StorageQuotaCephFSCSIDriver(); got != customRookNamespace+".cephfs.csi.ceph.com" {
		t.Fatalf("derived CephFS CSI driver = %q", got)
	}
}
