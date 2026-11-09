// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vcjob

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/internal/util"
)

func setupProfilingUserQuery(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	// An empty user table exercises the existing fallback without external services.
	if err := db.Table("users").AutoMigrate(&struct {
		ID        uint `gorm:"primaryKey"`
		Name      string
		DeletedAt gorm.DeletedAt
	}{}); err != nil {
		t.Fatal(err)
	}
	original := query.User
	testQuery := query.Use(db)
	query.User = &testQuery.User
	t.Cleanup(func() { query.User = original })
}

func TestUserPodBuildersEnableProfiling(t *testing.T) {
	// Sequential because GenerateEnvs reads the shared generated user query.
	setupProfilingUserQuery(t)
	const profilingTestPort = 8080
	ctx := context.Background()
	token := util.JWTMessage{UserID: 42, Username: "profiling-user"}
	image := ImageBaseInfo{ImageLink: "example.invalid/profiling:test"}
	tests := []struct {
		name  string
		build func() (corev1.PodSpec, error)
	}{
		{"custom", func() (corev1.PodSpec, error) {
			return GenerateCustomPodSpec(ctx, token, "custom-test", &CreateCustomReq{Image: image})
		}},
		{"jupyter", func() (corev1.PodSpec, error) {
			return generateInteractivePodSpec(ctx, token, "jupyter-test", &CreateJobCommon{}, nil, image, nil, JupyterPort, "jupyter", false)
		}},
		{"webide", func() (corev1.PodSpec, error) {
			return generateInteractivePodSpec(ctx, token, "webide-test", &CreateJobCommon{}, nil, image, nil, profilingTestPort, "webide", false)
		}},
		{"distributed-worker", func() (corev1.PodSpec, error) {
			return generatePodSpecForParallelJob(&TaskReq{Name: "worker", Image: image}, nil, nil, nil, nil, nil, nil, false), nil
		}},
		{"distributed-ps", func() (corev1.PodSpec, error) {
			return generatePodSpecForParallelJob(&TaskReq{Name: "ps", Image: image}, nil, nil, nil, nil, nil, nil, false), nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := tt.build()
			if err != nil {
				t.Fatal(err)
			}
			assertProfilingPod(t, &spec)
			if tt.name == "webide" && spec.Containers[0].Ports[0].ContainerPort != profilingTestPort {
				t.Fatal("profiling changed the requested interactive port")
			}
		})
	}
}

func assertProfilingPod(t *testing.T, spec *corev1.PodSpec) {
	t.Helper()
	if spec.HostPID || spec.HostIPC || spec.HostNetwork {
		t.Fatal("profiling enabled a host namespace")
	}
	if len(spec.Containers) == 0 {
		t.Fatal("builder produced no user container")
	}
	for i := range spec.Containers {
		assertProfilingContainer(t, &spec.Containers[i])
	}
}

func assertProfilingContainer(t *testing.T, container *corev1.Container) {
	t.Helper()
	sc := container.SecurityContext
	if sc == nil || sc.Capabilities == nil {
		t.Fatal("missing profiling security context")
	}
	for _, cap := range []corev1.Capability{"IPC_LOCK", "SYS_PTRACE"} {
		if !slices.Contains(sc.Capabilities.Add, cap) {
			t.Errorf("missing capability %s", cap)
		}
	}
	if slices.Contains(sc.Capabilities.Add, "PERFMON") {
		t.Fatal("unexpected default PERFMON")
	}
	if slices.Contains(sc.Capabilities.Add, "SYS_ADMIN") || (sc.Privileged != nil && *sc.Privileged) {
		t.Fatal("profiling enabled broad administrative privileges")
	}
	if sc.SeccompProfile != nil && sc.SeccompProfile.Type == corev1.SeccompProfileTypeUnconfined {
		t.Fatal("profiling disabled seccomp")
	}
}

//nolint:gocyclo // Exercise all pod builders through the real submission path before and after revocation.
func TestVolcanoSubmissionAppliesCurrentUserGrant(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Name: "profiling", Space: "profiling", Role: model.RoleUser, Status: model.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	q := query.Use(db)
	old := query.User
	query.User = &q.User
	t.Cleanup(func() { query.User = old })
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node", Labels: map[string]string{"kubernetes.io/arch": "amd64"}}, Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}}}).Build()
	mgr := &VolcanojobMgr{client: fakeClient, configService: service.NewConfigService(q)}
	token := util.JWTMessage{UserID: user.ID, Username: user.Name}
	image := ImageBaseInfo{ImageLink: "example.invalid/profiling:test"}
	custom, err := GenerateCustomPodSpec(t.Context(), token, "custom", &CreateCustomReq{Image: image})
	if err != nil {
		t.Fatal(err)
	}
	jupyter, err := generateInteractivePodSpec(t.Context(), token, "jupyter", &CreateJobCommon{}, nil, image, nil, JupyterPort, "jupyter", false)
	if err != nil {
		t.Fatal(err)
	}
	webide, err := generateInteractivePodSpec(t.Context(), token, "webide", &CreateJobCommon{}, nil, image, nil, 8080, "webide", false)
	if err != nil {
		t.Fatal(err)
	}
	parallel := generatePodSpecForParallelJob(&TaskReq{Name: "worker", Image: image}, nil, nil, nil, nil, nil, nil, false)
	original := &batch.Job{ObjectMeta: metav1.ObjectMeta{Namespace: "jobs", Annotations: map[string]string{"crater.raids.io/user-id": fmt.Sprint(user.ID)}}, Spec: batch.JobSpec{Tasks: []batch.TaskSpec{
		{Name: "custom", Replicas: 1, Template: corev1.PodTemplateSpec{Spec: custom}},
		{Name: "jupyter", Replicas: 1, Template: corev1.PodTemplateSpec{Spec: jupyter}},
		{Name: "webide", Replicas: 1, Template: corev1.PodTemplateSpec{Spec: webide}},
		{Name: "worker", Replicas: 2, Template: corev1.PodTemplateSpec{Spec: parallel}},
	}}}
	for i, tc := range []struct{ granted, requested bool }{
		{false, false}, {true, false}, {true, true}, {false, true}, {false, false},
	} {
		enabled := tc.granted
		if err := db.Model(&model.User{}).Where("id = ?", user.ID).Update("container_sys_admin_enabled", enabled).Error; err != nil {
			t.Fatal(err)
		}
		job := original.DeepCopy()
		job.Name = fmt.Sprintf("profiling-%d", i)
		err := mgr.submitJob(t.Context(), token, job, tc.requested)
		if tc.requested && !tc.granted {
			if !errors.Is(err, bizerr.Forbidden.Base) {
				t.Fatalf("expected permission rejection, got %v", err)
			}
			var rejected batch.Job
			if err := fakeClient.Get(t.Context(), client.ObjectKeyFromObject(job), &rejected); !apierrors.IsNotFound(err) {
				t.Fatalf("denied job was submitted: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var stored batch.Job
		if err := fakeClient.Get(t.Context(), client.ObjectKeyFromObject(job), &stored); err != nil {
			t.Fatal(err)
		}
		for _, task := range stored.Spec.Tasks {
			for _, container := range task.Template.Spec.Containers {
				if slices.Contains(container.SecurityContext.Capabilities.Add, "SYS_ADMIN") != tc.requested || slices.Contains(container.SecurityContext.Capabilities.Add, "PERFMON") {
					t.Fatalf("%s grant %v: %+v", task.Name, enabled, container.SecurityContext)
				}
			}
		}
		original = stored.DeepCopy()
		original.ResourceVersion = ""
		original.UID = ""
	}
}
