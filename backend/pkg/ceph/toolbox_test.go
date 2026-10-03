package ceph

import (
	"context"
	"errors"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestToolboxStoragePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		root     string
		relative string
		want     string
		wantErr  bool
	}{
		{name: "root", root: "/mnt/ceph/volume", relative: ".", want: "/mnt/ceph/volume"},
		{name: "user", root: "/mnt/ceph/volume", relative: "users/alice", want: "/mnt/ceph/volume/users/alice"},
		{name: "escape", root: "/mnt/ceph/volume", relative: "../other", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := toolboxStoragePath(test.root, test.relative)
			if (err != nil) != test.wantErr {
				t.Fatalf("toolboxStoragePath() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("toolboxStoragePath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestValidateToolboxResolvedPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		root    string
		target  string
		want    string
		wantErr bool
	}{
		{name: "root", root: "/mnt/ceph/volume", target: "/mnt/ceph/volume", want: "/mnt/ceph/volume"},
		{name: "child", root: "/mnt/ceph/volume", target: "/mnt/ceph/volume/user/alice", want: "/mnt/ceph/volume/user/alice"},
		{name: "similar prefix", root: "/mnt/ceph/volume", target: "/mnt/ceph/volume-other", wantErr: true},
		{name: "symlink escape", root: "/mnt/ceph/volume", target: "/etc", wantErr: true},
		{name: "relative", root: "/mnt/ceph/volume", target: "user/alice", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateToolboxResolvedPath(test.root, test.target)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateToolboxResolvedPath() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("validateToolboxResolvedPath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIsToolboxXattrNotFound(t *testing.T) {
	t.Parallel()

	for _, message := range []string{
		"ceph.quota.max_bytes: No such attribute",
		"getxattr failed: No data available",
		"errno=ENODATA",
	} {
		if !isToolboxXattrNotFound(fmt.Errorf("exec failed: %s", message)) {
			t.Fatalf("isToolboxXattrNotFound(%q) = false", message)
		}
	}
	if isToolboxXattrNotFound(errors.New("permission denied")) {
		t.Fatal("permission error must not be treated as a missing xattr")
	}
}

func TestCephQuotaXattrValue(t *testing.T) {
	t.Parallel()

	if got := cephQuotaXattrValue(-1); got != 0 {
		t.Fatalf("unlimited quota xattr = %d, want 0", got)
	}
	if got := cephQuotaXattrValue(4 * 1024 * 1024); got != 4*1024*1024 {
		t.Fatalf("limited quota xattr = %d", got)
	}
}

func TestFindToolboxPod(t *testing.T) {
	t.Parallel()
	const selector = "component=ceph-tools"

	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "not-running", Namespace: "storage-system", Labels: map[string]string{"component": "ceph-tools"}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "custom-tools-abc", Namespace: "storage-system", Labels: map[string]string{"component": "ceph-tools"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "toolbox-with-wrong-label", Namespace: "storage-system", Labels: map[string]string{"app": "other"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	pod, err := findToolboxPod(context.Background(), clientset, "storage-system", selector)
	if err != nil {
		t.Fatalf("findToolboxPod() error = %v", err)
	}
	if pod.Name != "custom-tools-abc" {
		t.Fatalf("findToolboxPod() = %q", pod.Name)
	}
}

func TestVolumeHandleUUIDPattern(t *testing.T) {
	t.Parallel()

	handle := "0001-0009-rook-ceph-0000000000000001-7ca9d703-bbbc-4ae5-a03d-ae9f9f7be59e"
	if got := volumeHandleUUIDPattern.FindString(handle); got != "7ca9d703-bbbc-4ae5-a03d-ae9f9f7be59e" {
		t.Fatalf("UUID = %q", got)
	}
}
