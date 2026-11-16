// Copyright 2026 The Crater Project Team, RAIDS-Lab

package vcjob

import (
	v1 "k8s.io/api/core/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

// SessionKind identifies the access policy of an existing SSH or exec session.
type SessionKind uint8

const (
	// ExecSession requires ownership in the current account, including for admins.
	ExecSession SessionKind = iota
	// SSHSession preserves the existing platform administrator access policy.
	SSHSession
)

// CheckSessionAccess authorizes the resolved job before accessing live resources.
func CheckSessionAccess(job *model.Job, userID, accountID uint, role model.Role, kind SessionKind) error {
	if kind != ExecSession && kind != SSHSession {
		return bizerr.Forbidden.PermissionDenied.New("unsupported job session access policy")
	}
	if kind == SSHSession && role == model.RoleAdmin {
		return nil
	}
	if job.UserID != userID || job.AccountID != accountID {
		return bizerr.Forbidden.PermissionDenied.New("job access requires ownership in the current user and project")
	}
	return nil
}

// SelectSSHTarget retains the first task's main container despite injected sidecars.
func SelectSSHTarget(job *batch.Job, pod *v1.Pod) (*v1.Pod, string, error) {
	if len(job.Spec.Tasks) == 0 || len(job.Spec.Tasks[0].Template.Spec.Containers) == 0 {
		return nil, "", bizerr.Conflict.ResourceStatusError.New("job has no main container for SSH")
	}
	main := job.Spec.Tasks[0].Template.Spec.Containers[0].Name
	return SelectExecTarget(job, []v1.Pod{*pod}, pod.Name, main)
}
