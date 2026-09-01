package tensorboard

import (
	"testing"

	"gorm.io/datatypes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/payload"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

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
			deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Labels: tt.labels}}
			if got := isTensorboardOwner(deploy, tt.username); got != tt.want {
				t.Fatalf("isTensorboardOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func readyTensorboardDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tb-panel-1",
			Namespace: "jobs",
			Labels: map[string]string{
				interutil.LabelKeyTensorboardID: "panel-1",
				crclient.LabelKeyTaskUser:       "alice",
				crclient.LabelKeyTaskType:       interutil.LabelKeyTypeTensorboard,
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			ReadyReplicas:     1,
		},
	}
}

func TestGetStatusUsesDeploymentState(t *testing.T) {
	tests := []struct {
		name        string
		deployment  *appsv1.Deployment
		wantStatus  payload.TensorboardStatus
		wantReason  payload.TensorboardStatusReason
		wantMessage string
	}{
		{
			name:        "ready replicas available",
			deployment:  readyTensorboardDeployment(),
			wantStatus:  payload.TensorboardStatusReady,
			wantReason:  payload.TensorboardStatusReasonReady,
			wantMessage: "The panel is ready.",
		},
		{
			name:        "replicas not ready",
			deployment:  &appsv1.Deployment{},
			wantStatus:  payload.TensorboardStatusStarting,
			wantReason:  payload.TensorboardStatusReasonDeploymentStarting,
			wantMessage: "The panel is starting. The first startup may take several minutes.",
		},
		{
			name: "deployment failed",
			deployment: &appsv1.Deployment{Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionFalse,
				}},
			}},
			wantStatus:  payload.TensorboardStatusFailed,
			wantReason:  payload.TensorboardStatusReasonDeploymentFailed,
			wantMessage: "The panel failed to start. Check the Pod events or contact an administrator.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, reason, message := getStatus(tt.deployment)
			if status != tt.wantStatus || reason != tt.wantReason || message != tt.wantMessage {
				t.Fatalf(
					"getStatus() = (%q, %q, %q), want (%q, %q, %q)",
					status,
					reason,
					message,
					tt.wantStatus,
					tt.wantReason,
					tt.wantMessage,
				)
			}
		})
	}
}
