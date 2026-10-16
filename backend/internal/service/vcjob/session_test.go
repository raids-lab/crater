// Copyright 2026 The Crater Project Team, RAIDS-Lab

package vcjob

import (
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

func TestSessionAccessPreservesWebSSHAdminAccess(t *testing.T) {
	job := &model.Job{UserID: 7, AccountID: 11}
	for _, tt := range []struct {
		name      string
		user      uint
		account   uint
		role      model.Role
		kind      SessionKind
		permitted bool
	}{
		{"owner SSH", 7, 11, model.RoleUser, SSHSession, true},
		{"owner exec", 7, 11, model.RoleUser, ExecSession, true},
		{"admin SSH to other user and account", 99, 22, model.RoleAdmin, SSHSession, true},
		{"admin exec still requires ownership", 99, 22, model.RoleAdmin, ExecSession, false},
		{"same account other user SSH denied", 99, 11, model.RoleUser, SSHSession, false},
		{"same user other account SSH denied", 7, 22, model.RoleUser, SSHSession, false},
		{"same user other account exec denied", 7, 22, model.RoleUser, ExecSession, false},
		{"unknown session denied for admin", 99, 22, model.RoleAdmin, SessionKind(99), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSessionAccess(job, tt.user, tt.account, tt.role, tt.kind)
			if tt.permitted {
				if err != nil {
					t.Fatalf("access failed: %v", err)
				}
			} else if !errors.Is(err, bizerr.Forbidden.Base) {
				t.Fatalf("error=%v, want forbidden", err)
			}
		})
	}
}

func TestSSHTargetPreservesMainContainerWithSidecars(t *testing.T) {
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "jpt-user-job", Namespace: "tasks", UID: types.UID("current-job")},
		Spec: batch.JobSpec{Tasks: []batch.TaskSpec{{Template: v1.PodTemplateSpec{Spec: v1.PodSpec{
			Containers: []v1.Container{{Name: jobTypeJupyter}},
		}}}}},
	}
	for _, tt := range []struct {
		name   string
		change func(*batch.Job, *v1.Pod)
		valid  bool
	}{
		{"one main container", nil, true},
		{"injected sidecar before main", func(_ *batch.Job, p *v1.Pod) {
			p.Spec.Containers = append([]v1.Container{{Name: "proxy"}}, p.Spec.Containers...)
			p.Status.ContainerStatuses = append(p.Status.ContainerStatuses, v1.ContainerStatus{
				Name: "proxy", State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
			})
		}, true},
		{"stale pod rejected", func(_ *batch.Job, p *v1.Pod) { p.OwnerReferences[0].UID = "old-job" }, false},
		{"main stopped with running sidecar", func(_ *batch.Job, p *v1.Pod) {
			p.Status.ContainerStatuses[0].State = v1.ContainerState{}
			p.Status.ContainerStatuses = append(p.Status.ContainerStatuses, v1.ContainerStatus{
				Name: "proxy", State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
			})
		}, false},
		{"missing task", func(j *batch.Job, _ *v1.Pod) { j.Spec.Tasks = nil }, false},
		{"missing template container", func(j *batch.Job, _ *v1.Pod) {
			j.Spec.Tasks[0].Template.Spec.Containers = nil
		}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			live := job.DeepCopy()
			pod := execTestPod(live)
			pod.Spec.Containers[0].Name = jobTypeJupyter
			pod.Status.ContainerStatuses[0].Name = jobTypeJupyter
			if tt.change != nil {
				tt.change(live, &pod)
			}
			got, container, err := SelectSSHTarget(live, &pod)
			if !tt.valid {
				if err == nil {
					t.Fatal("accepted an invalid SSH target")
				}
				return
			}
			if err != nil || container != jobTypeJupyter || got.Name != pod.Name {
				t.Fatalf("target=%v/%q error=%v, want the job's jupyter container", got, container, err)
			}
		})
	}
}
