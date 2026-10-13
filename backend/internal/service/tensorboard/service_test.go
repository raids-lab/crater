/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tensorboard

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

func TestNormalizeTensorboardName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
		code  bizerr.BizCode
	}{
		{name: "trimmed Chinese", input: " 训练对比 ", want: "训练对比"},
		{name: "spaces", input: "training comparison", want: "training comparison"},
		{name: "Unicode boundary", input: strings.Repeat("🚀", maxTensorboardNameLength),
			want: strings.Repeat("🚀", maxTensorboardNameLength)},
		{name: "missing", code: bizerr.BadRequest.MissingParameter},
		{name: "blank", input: "  ", code: bizerr.BadRequest.MissingParameter},
		{name: "too long", input: strings.Repeat("中", maxTensorboardNameLength+1), code: bizerr.BadRequest.ParameterError},
		{name: "control character", input: "training\ncomparison", code: bizerr.BadRequest.ParameterError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeTensorboardName(test.input)
			if test.want == "" {
				assertTensorboardErrorCode(t, err, test.code)
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("normalized name = %q, error = %v, want %q", got, err, test.want)
			}
		})
	}
}

func TestCreateRejectsInvalidNameBeforeClusterOperations(t *testing.T) {
	svc := NewTensorboardService(nil, nil)
	for _, name := range []string{"", "  ", strings.Repeat("x", maxTensorboardNameLength+1), "bad\nname"} {
		_, err := svc.Create(t.Context(), interutil.JWTMessage{}, &payload.CreateTensorboardReq{Name: name})
		if err == nil {
			t.Fatalf("invalid name %q was accepted", name)
		}
	}
}

func sourceJob(volumes []corev1.Volume, mounts []corev1.VolumeMount, logDir string) *model.Job {
	job := &batch.Job{
		Spec: batch.JobSpec{
			Tasks: []batch.TaskSpec{
				{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Volumes: volumes,
							Containers: []corev1.Container{
								{
									Name:         "trainer",
									VolumeMounts: mounts,
									Env: []corev1.EnvVar{
										{Name: tensorboardLogDirEnv, Value: logDir},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &model.Job{Attributes: datatypes.NewJSONType(job)}
}

func TestIsEligibleSourceJob(t *testing.T) {
	tests := []struct {
		name string
		job  *model.Job
		want bool
	}{
		{
			name: "regular job",
			job:  &model.Job{JobType: model.JobTypeCustom},
			want: true,
		},
		{
			name: "TensorBoard job",
			job:  &model.Job{JobType: model.JobType(labelKeyTypeTensorboard)},
			want: false,
		},
		{
			name: "nil job",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEligibleSourceJob(tt.job); got != tt.want {
				t.Fatalf("isEligibleSourceJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func pvcVolume(name, claim string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
		},
	}
}

func TestGetLogStorageReturnsOnlyMatchingReadOnlyMount(t *testing.T) {
	job := sourceJob(
		[]corev1.Volume{
			pvcVolume("home", "home-pvc"),
			pvcVolume("workspace", "workspace-pvc"),
		},
		[]corev1.VolumeMount{
			{Name: "home", MountPath: "/home/alice"},
			{Name: "workspace", MountPath: "/home/alice/workspace", SubPath: "users/alice"},
		},
		"/home/alice/workspace/logs/run-a",
	)

	volume, mount, err := getLogStorage(job, "/home/alice/workspace/logs/run-a")
	if err != nil {
		t.Fatalf("getLogStorage returned an error: %v", err)
	}
	if volume.Name != "workspace" || volume.PersistentVolumeClaim == nil ||
		volume.PersistentVolumeClaim.ClaimName != "workspace-pvc" {
		t.Fatalf("expected only the matching workspace volume, got %#v", volume)
	}
	if mount.Name != "workspace" || mount.MountPath != "/home/alice/workspace" {
		t.Fatalf("unexpected source mount: %#v", mount)
	}
	if mount.SubPath != "users/alice" {
		t.Fatalf("unexpected source subpath: %q", mount.SubPath)
	}
	if !mount.ReadOnly {
		t.Fatal("single-source TensorBoard mounts must be read-only")
	}
}

func TestBuildRunMountUsesMostSpecificDataMount(t *testing.T) {
	job := sourceJob(
		[]corev1.Volume{
			pvcVolume("home", "home-pvc"),
			pvcVolume("workspace", "workspace-pvc"),
		},
		[]corev1.VolumeMount{
			{Name: "home", MountPath: "/home/alice"},
			{Name: "workspace", MountPath: "/home/alice/workspace", SubPath: "users/alice"},
		},
		"/home/alice/workspace/logs/run-a",
	)

	volume, mount, err := buildRunMount(job, "job-a", 2, "")
	if err != nil {
		t.Fatalf("buildRunMount returned an error: %v", err)
	}
	if volume.Name != "tb-run-2" {
		t.Fatalf("unexpected volume name: %q", volume.Name)
	}
	if volume.PersistentVolumeClaim == nil || volume.PersistentVolumeClaim.ClaimName != "workspace-pvc" {
		t.Fatalf("expected the most specific PVC, got %#v", volume.VolumeSource)
	}
	if mount.Name != volume.Name {
		t.Fatalf("mount references %q instead of %q", mount.Name, volume.Name)
	}
	if mount.MountPath != "/tensorboard-runs/job-a" {
		t.Fatalf("unexpected mount path: %q", mount.MountPath)
	}
	if mount.SubPath != "users/alice/logs/run-a" {
		t.Fatalf("unexpected subpath: %q", mount.SubPath)
	}
	if !mount.ReadOnly {
		t.Fatal("aggregated TensorBoard mounts must be read-only")
	}
}

func TestBuildRunMountRejectsInvalidLogDirs(t *testing.T) {
	tests := []struct {
		name   string
		logDir string
	}{
		{name: "relative path", logDir: "logs/run-a"},
		{name: "outside data mount", logDir: "/tmp/logs"},
		{name: "parent traversal outside data mount", logDir: "/workspace/logs/../../secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := sourceJob(
				[]corev1.Volume{pvcVolume("workspace", "workspace-pvc")},
				[]corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
				tt.logDir,
			)

			if _, _, err := buildRunMount(job, "job-a", 0, ""); err == nil {
				t.Fatalf("expected invalid log directory %q to be rejected", tt.logDir)
			}
		})
	}
}

func TestBuildRunMountRejectsDynamicSubPath(t *testing.T) {
	job := sourceJob(
		[]corev1.Volume{pvcVolume("workspace", "workspace-pvc")},
		[]corev1.VolumeMount{
			{Name: "workspace", MountPath: "/workspace", SubPathExpr: "$(POD_NAME)"},
		},
		"/workspace/logs",
	)

	if _, _, err := buildRunMount(job, "job-a", 0, ""); err == nil {
		t.Fatal("expected a dynamic SubPathExpr mount to be rejected")
	}
}

func TestBuildRunMountUsesRequestedLogDirWhenJobDoesNotDeclareOne(t *testing.T) {
	job := sourceJob(
		[]corev1.Volume{pvcVolume("workspace", "workspace-pvc")},
		[]corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace", SubPath: "users/alice"}},
		"",
	)

	_, mount, err := buildRunMount(job, "job-old", 0, "/workspace/logs/run-old")
	if err != nil {
		t.Fatalf("expected a per-job override to support legacy jobs: %v", err)
	}
	if mount.SubPath != "users/alice/logs/run-old" {
		t.Fatalf("unexpected override subpath: %q", mount.SubPath)
	}
}

func TestBuildRunMountRejectsRequestedLogDirOutsideDataMounts(t *testing.T) {
	job := sourceJob(
		[]corev1.Volume{pvcVolume("workspace", "workspace-pvc")},
		[]corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
		"",
	)

	if _, _, err := buildRunMount(job, "job-old", 0, "/tmp/logs"); err == nil {
		t.Fatal("expected an out-of-mount override to be rejected")
	}
}

func TestBuildRunMountRequiresDeclaredOrRequestedLogDir(t *testing.T) {
	job := sourceJob(
		[]corev1.Volume{pvcVolume("workspace", "workspace-pvc")},
		[]corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace"}},
		"",
	)

	if _, _, err := buildRunMount(job, "job-old", 0, ""); err == nil {
		t.Fatal("expected a missing event directory to be rejected")
	}
}

func TestSourceJobsPreferPerJobRequestsAndDeduplicate(t *testing.T) {
	req := &payload.CreateTensorboardReq{
		SourceJobNames: []string{"legacy-job"},
		SourceJobs: []payload.TensorboardSourceJobReq{
			{JobName: " job-a ", LogDir: " /logs/a "},
			{JobName: "job-a", LogDir: "/logs/ignored"},
			{JobName: "job-b"},
		},
	}

	got := sourceJobs(req)
	if len(got) != 2 {
		t.Fatalf("expected two deduplicated sources, got %#v", got)
	}
	if got[0].name != "job-a" || got[0].logDir != "/logs/a" {
		t.Fatalf("unexpected first source: %#v", got[0])
	}
	if got[1].name != "job-b" || got[1].logDir != "" {
		t.Fatalf("unexpected second source: %#v", got[1])
	}
}

func TestSourceJobsUsesLegacySingleLogDir(t *testing.T) {
	req := &payload.CreateTensorboardReq{
		SourceJobName: " job-old ",
		LogDir:        " /workspace/logs ",
	}

	got := sourceJobs(req)
	if len(got) != 1 {
		t.Fatalf("expected one legacy source, got %#v", got)
	}
	if got[0].name != "job-old" || got[0].logDir != "/workspace/logs" {
		t.Fatalf("unexpected legacy source: %#v", got[0])
	}
}

func TestIsLogDirMountedUsesPathBoundaries(t *testing.T) {
	mounts := []corev1.VolumeMount{{MountPath: "/workspace/data"}}
	tests := []struct {
		name   string
		logDir string
		want   bool
	}{
		{name: "mount root", logDir: "/workspace/data", want: true},
		{name: "child", logDir: "/workspace/data/logs", want: true},
		{name: "similar prefix", logDir: "/workspace/database", want: false},
		{name: "parent", logDir: "/workspace", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLogDirMounted(tt.logDir, mounts); got != tt.want {
				t.Fatalf("isLogDirMounted(%q) = %v, want %v", tt.logDir, got, tt.want)
			}
		})
	}
}

func TestIsLogDirMountedAcceptsRootMount(t *testing.T) {
	mounts := []corev1.VolumeMount{{MountPath: "/"}}
	if !isLogDirMounted("/logs/run-a", mounts) {
		t.Fatal("expected an absolute log directory to be accepted under a root mount")
	}
}

func TestBuildPersonalStorageAllowsLogDirWithoutSourceJob(t *testing.T) {
	user := &model.User{Name: "alice", Space: "alice-space"}
	storage, err := buildPersonalStorage(
		user,
		" /home/alice/tensorboard-runs/run-a/ ",
		"storage-rox",
		"users",
	)
	if err != nil {
		t.Fatalf("buildPersonalStorage returned an error: %v", err)
	}
	if storage.logDir != "/home/alice/tensorboard-runs/run-a" {
		t.Fatalf("unexpected log directory: %q", storage.logDir)
	}
	if len(storage.volumes) != 1 || storage.volumes[0].PersistentVolumeClaim == nil ||
		storage.volumes[0].PersistentVolumeClaim.ClaimName != "storage-rox" {
		t.Fatalf("unexpected personal volume: %#v", storage.volumes)
	}
	if len(storage.volumeMounts) != 1 {
		t.Fatalf("expected one personal mount, got %#v", storage.volumeMounts)
	}
	mount := storage.volumeMounts[0]
	if mount.MountPath != "/home/alice" || mount.SubPath != "users/alice-space" || !mount.ReadOnly {
		t.Fatalf("unexpected personal mount: %#v", mount)
	}
}

func TestBuildPersonalStorageRejectsInvalidLogDirs(t *testing.T) {
	user := &model.User{Name: "alice", Space: "alice-space"}
	tests := []struct {
		name   string
		logDir string
	}{
		{name: "missing", logDir: ""},
		{name: "relative", logDir: "tensorboard-runs/run-a"},
		{name: "another user", logDir: "/home/bob/tensorboard-runs/run-a"},
		{name: "parent traversal", logDir: "/home/alice/../bob/tensorboard-runs/run-a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := buildPersonalStorage(user, tt.logDir, "storage-rox", "users"); err == nil {
				t.Fatalf("expected log directory %q to be rejected", tt.logDir)
			}
		})
	}
}

func TestIsTensorboardOwner(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		username string
		want     bool
	}{
		{name: "owner", labels: map[string]string{crclient.LabelKeyTaskUser: "alice"}, username: "alice", want: true},
		{name: "different user", labels: map[string]string{crclient.LabelKeyTaskUser: "alice"}, username: "bob", want: false},
		{name: "missing owner label", labels: map[string]string{}, username: "alice", want: false},
		{name: "nil labels", labels: nil, username: "alice", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &batch.Job{ObjectMeta: metav1.ObjectMeta{Labels: tt.labels}}
			if got := isTensorboardOwner(job, tt.username); got != tt.want {
				t.Fatalf("isTensorboardOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetJobStatusUsesVolcanoAndPodState(t *testing.T) {
	readyPod := &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{
		Type: corev1.PodReady, Status: corev1.ConditionTrue,
	}}}}
	expiredPod := &corev1.Pod{Status: corev1.PodStatus{Reason: "DeadlineExceeded"}}

	tests := []struct {
		name       string
		phase      batch.JobPhase
		pod        *corev1.Pod
		wantStatus payload.TensorboardStatus
		wantReason payload.TensorboardStatusReason
	}{
		{name: "pending", phase: batch.Pending, wantStatus: payload.TensorboardStatusPending,
			wantReason: payload.TensorboardStatusReasonWaitingForSchedule},
		{name: "pod starting", phase: batch.Running, pod: &corev1.Pod{}, wantStatus: payload.TensorboardStatusStarting,
			wantReason: payload.TensorboardStatusReasonPodStarting},
		{name: "ready", phase: batch.Running, pod: readyPod, wantStatus: payload.TensorboardStatusReady,
			wantReason: payload.TensorboardStatusReasonReady},
		{name: "restarting", phase: batch.Restarting, pod: readyPod, wantStatus: payload.TensorboardStatusStarting,
			wantReason: payload.TensorboardStatusReasonPodStarting},
		{name: "runtime expired", phase: batch.Failed, pod: expiredPod, wantStatus: payload.TensorboardStatusExpired,
			wantReason: payload.TensorboardStatusReasonRuntimeExpired},
		{name: "job failed", phase: batch.Failed, pod: &corev1.Pod{}, wantStatus: payload.TensorboardStatusFailed,
			wantReason: payload.TensorboardStatusReasonJobFailed},
		{name: "aborting", phase: batch.Aborting, pod: &corev1.Pod{}, wantStatus: payload.TensorboardStatusFailed,
			wantReason: payload.TensorboardStatusReasonJobFailed},
		{name: "completing", phase: batch.Completing, pod: &corev1.Pod{}, wantStatus: payload.TensorboardStatusCompleted,
			wantReason: payload.TensorboardStatusReasonCompleted},
		{name: "terminating", phase: batch.Terminating, pod: &corev1.Pod{}, wantStatus: payload.TensorboardStatusFailed,
			wantReason: payload.TensorboardStatusReasonJobFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &batch.Job{Status: batch.JobStatus{State: batch.JobState{Phase: tt.phase}}}
			status, reason, _ := getJobStatus(job, tt.pod)
			if status != tt.wantStatus || reason != tt.wantReason {
				t.Fatalf("getJobStatus() = (%q, %q), want (%q, %q)",
					status, reason, tt.wantStatus, tt.wantReason)
			}
		})
	}
}

func newTensorboardTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	// Current panels must work without registering legacy workload kinds.
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register Pod resources: %v", err)
	}
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatalf("register Volcano resources: %v", err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func assertTensorboardErrorCode(t *testing.T, err error, want bizerr.BizCode) {
	t.Helper()
	actual, ok := bizerr.FromError(err)
	if !ok || actual.Code != want {
		t.Fatalf("expected business error code %d, got %v", want, err)
	}
}

func tensorboardTestJob(id, username string, phase batch.JobPhase) *batch.Job {
	return &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "tb-" + id,
			Namespace:   config.GetConfig().Namespaces.Job,
			Annotations: map[string]string{annotationKeyTensorboardName: "Training comparison"},
			Labels: map[string]string{
				labelKeyTensorboardID:     id,
				crclient.LabelKeyTaskUser: username,
				crclient.LabelKeyTaskType: labelKeyTypeTensorboard,
			},
		},
		Status: batch.JobStatus{State: batch.JobState{Phase: phase}},
	}
}

func TestTensorboardQuotaAndListUseVolcanoJobs(t *testing.T) {
	pending := tensorboardTestJob("pending", "alice", batch.Pending)
	running := tensorboardTestJob("running", "alice", batch.Running)
	completed := tensorboardTestJob("completed", "alice", batch.Completed)
	expired := tensorboardTestJob("expired", "alice", batch.Failed)
	otherUser := tensorboardTestJob("other-user", "bob", batch.Pending)
	training := tensorboardTestJob("training", "alice", batch.Pending)
	training.Labels[crclient.LabelKeyTaskType] = "training"
	otherNamespace := tensorboardTestJob("other-namespace", "alice", batch.Pending)
	otherNamespace.Namespace += "-other"
	startedAt := metav1.NewTime(time.Now().Add(-time.Hour))
	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "running-pod", Namespace: running.Namespace, Labels: running.Labels,
		},
		Status: corev1.PodStatus{
			StartTime: &startedAt,
			Conditions: []corev1.PodCondition{{
				Type: corev1.PodReady, Status: corev1.ConditionTrue,
			}},
		},
	}
	expiredPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "expired-pod", Namespace: expired.Namespace, Labels: expired.Labels,
		},
		Status: corev1.PodStatus{Reason: "DeadlineExceeded"},
	}
	crClient := newTensorboardTestClient(t,
		pending, running, completed, expired, otherUser, training, otherNamespace, readyPod, expiredPod,
	)
	svc := NewTensorboardService(crClient, nil)
	ctx := context.Background()
	count, err := svc.activeTensorboardCount(ctx, "alice")
	if err != nil || count != 2 {
		t.Fatalf("active panel count = %d, error = %v, want 2", count, err)
	}
	panels, err := svc.List(ctx, "alice")
	if err != nil {
		t.Fatalf("list Volcano panels: %v", err)
	}
	wantStatuses := map[string]payload.TensorboardStatus{
		"pending": payload.TensorboardStatusPending, "running": payload.TensorboardStatusReady,
		"completed": payload.TensorboardStatusCompleted, "expired": payload.TensorboardStatusExpired,
	}
	if len(panels) != len(wantStatuses) {
		t.Fatalf("listed %d panels, want %d", len(panels), len(wantStatuses))
	}
	for _, panel := range panels {
		if panel.Name != "Training comparison" {
			t.Fatalf("panel name = %q, want stored annotation", panel.Name)
		}
		wantStatus, ok := wantStatuses[panel.ID]
		if !ok || panel.Status != wantStatus {
			t.Fatalf("unexpected panel %q with status %q", panel.ID, panel.Status)
		}
		if panel.ID == "running" && panel.Expiration != startedAt.Add(4*24*time.Hour).Format(time.RFC3339) {
			t.Fatalf("unexpected runtime expiration: %q", panel.Expiration)
		}
	}
}

func TestTensorboardAccessAndDeleteUseVolcanoJobs(t *testing.T) {
	job := tensorboardTestJob("a1b2c3d4", "alice", batch.Running)
	crClient := newTensorboardTestClient(t, job)
	svc := NewTensorboardService(crClient, nil)
	ctx := context.Background()
	accessPath, err := svc.GetAccessPath(ctx, "alice", "a1b2c3d4")
	wantPath := "https://" + config.GetConfig().Host + "/ingress/alice-a1b2c3d4"
	if err != nil || accessPath != wantPath {
		t.Fatalf("access path = %q, error = %v, want %q", accessPath, err, wantPath)
	}
	_, err = svc.GetAccessPath(ctx, "bob", "a1b2c3d4")
	assertTensorboardErrorCode(t, err, bizerr.Forbidden.PermissionDenied)
	assertTensorboardErrorCode(t, svc.Delete(ctx, "bob", "a1b2c3d4"), bizerr.Forbidden.PermissionDenied)
	if err := crClient.Get(ctx, client.ObjectKeyFromObject(job), &batch.Job{}); err != nil {
		t.Fatalf("unauthorized deletion removed the panel: %v", err)
	}
	if err := svc.Delete(ctx, "alice", "a1b2c3d4"); err != nil {
		t.Fatalf("owner cannot delete the panel: %v", err)
	}
	if err := crClient.Get(ctx, client.ObjectKeyFromObject(job), &batch.Job{}); !k8serrors.IsNotFound(err) {
		t.Fatalf("Volcano Job was not deleted: %v", err)
	}
	_, err = svc.GetAccessPath(ctx, "alice", "a1b2c3d4")
	assertTensorboardErrorCode(t, err, bizerr.NotFound.K8sResourceNotFound)
	assertTensorboardErrorCode(t, svc.Delete(ctx, "alice", "a1b2c3d4"), bizerr.NotFound.K8sResourceNotFound)
}

func TestSameNamePanelsHaveIndependentAccessAndDeletion(t *testing.T) {
	first := tensorboardTestJob("11111111", "alice", batch.Running)
	second := tensorboardTestJob("22222222", "alice", batch.Running)
	crClient := newTensorboardTestClient(t, first, second)
	svc := NewTensorboardService(crClient, nil)
	panels, err := svc.List(t.Context(), "alice")
	if err != nil || len(panels) != 2 {
		t.Fatalf("same-name list = %#v, error = %v", panels, err)
	}
	if panels[0].Name != panels[1].Name || panels[0].ID == panels[1].ID {
		t.Fatal("same-name panels did not retain distinct IDs")
	}
	firstURL, err := svc.GetAccessPath(t.Context(), "alice", "11111111")
	if err != nil {
		t.Fatal(err)
	}
	secondURL, err := svc.GetAccessPath(t.Context(), "alice", "22222222")
	if err != nil || firstURL == secondURL {
		t.Fatalf("same-name routes are not independent: %q, %q, %v", firstURL, secondURL, err)
	}
	if err := svc.Delete(t.Context(), "alice", "11111111"); err != nil {
		t.Fatal(err)
	}
	if err := crClient.Get(t.Context(), client.ObjectKeyFromObject(second), &batch.Job{}); err != nil {
		t.Fatalf("deleting one same-name panel affected the other: %v", err)
	}
}

func TestTensorboardAccessRejectsStoppedAndExpiredJobs(t *testing.T) {
	expiredAt := metav1.NewTime(time.Now().Add(-5 * 24 * time.Hour))
	tests := []struct {
		name   string
		phase  batch.JobPhase
		status corev1.PodStatus
	}{
		{name: "completed", phase: batch.Completed},
		{name: "failed", phase: batch.Failed},
		{name: "aborted", phase: batch.Aborted},
		{name: "terminated", phase: batch.Terminated},
		{name: "deadline exceeded", phase: batch.Running,
			status: corev1.PodStatus{Reason: "DeadlineExceeded"}},
		{name: "runtime elapsed", phase: batch.Running,
			status: corev1.PodStatus{StartTime: &expiredAt}},
		{name: "idle timeout before Job transition", phase: batch.Running,
			status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name: tensorboardTaskName,
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					ExitCode: 0, Message: `{"reason":"idle_expired"}`,
				}},
			}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := tensorboardTestJob("a1b2c3d4", "alice", tt.phase)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tb-pod", Namespace: job.Namespace, Labels: job.Labels,
				},
				Status: tt.status,
			}
			svc := NewTensorboardService(newTensorboardTestClient(t, job, pod), nil)
			_, err := svc.GetAccessPath(context.Background(), "alice", "a1b2c3d4")
			assertTensorboardErrorCode(t, err, bizerr.Conflict.ResourceStatusError)
		})
	}
}

func TestTensorboardCompletionReasons(t *testing.T) {
	tests := []struct {
		name       string
		phase      batch.JobPhase
		code       int32
		message    string
		wantStatus payload.TensorboardStatus
		wantReason payload.TensorboardStatusReason
	}{
		{"idle", batch.Completed, 0, `{"reason":"idle_expired"}`, payload.TensorboardStatusCompleted, payload.TensorboardStatusReasonIdleExpired},
		{"maximum runtime", batch.Completed, 0, `{"reason":"runtime_expired"}`, payload.TensorboardStatusCompleted, payload.TensorboardStatusReasonRuntimeExpired},
		{"Pod finished before Job", batch.Running, 0, `{"reason":"idle_expired"}`, payload.TensorboardStatusCompleted, payload.TensorboardStatusReasonIdleExpired},
		{"normal exit", batch.Completed, 0, `{"reason":"completed"}`, payload.TensorboardStatusCompleted, payload.TensorboardStatusReasonCompleted},
		{"invalid message", batch.Completed, 0, "invalid", payload.TensorboardStatusCompleted, payload.TensorboardStatusReasonCompleted},
		{"failure cannot claim expiry", batch.Failed, 2, `{"reason":"idle_expired"}`, payload.TensorboardStatusFailed, payload.TensorboardStatusReasonJobFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &batch.Job{Status: batch.JobStatus{State: batch.JobState{Phase: tt.phase}}}
			pod := &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name: tensorboardTaskName,
				State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					ExitCode: tt.code, Message: tt.message,
				}},
			}}}}
			status, reason, _ := getJobStatus(job, pod)
			if status != tt.wantStatus || reason != tt.wantReason {
				t.Fatalf("status/reason = %s/%s, want %s/%s", status, reason, tt.wantStatus, tt.wantReason)
			}
		})
	}
}

func TestTensorboardExpirationUsesContainerStart(t *testing.T) {
	podStart := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	containerStart := metav1.NewTime(podStart.Add(time.Hour))
	pod := &corev1.Pod{Status: corev1.PodStatus{
		StartTime: &podStart,
		ContainerStatuses: []corev1.ContainerStatus{{
			Name:  tensorboardTaskName,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: containerStart}},
		}},
	}}
	want := containerStart.Add(4 * 24 * time.Hour).Format(time.RFC3339)
	if got := tensorboardExpiration(pod); got != want {
		t.Fatalf("expiration = %q, want %q", got, want)
	}
}
