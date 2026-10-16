// Copyright 2026 The Crater Project Team, RAIDS-Lab

package vcjob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/pkg/jobsession"
)

// UserOpenSSH godoc
//
// @Summary Enable SSH access, optionally preparing the task user's environment
// @Description Returns ip and port. session=user additionally prepares and returns user and session_command.
// @Description Default clients retain native sshd sessions without shared launcher installation.
// @Description Platform administrators retain access to other users' jobs; ordinary users require ownership in the current project.
// @Tags VolcanoJob
// @Produce json
// @Security Bearer
// @Param name path string true "Platform job name"
// @Param session query string false "Set user to prepare the shared HOME/shell environment"
// @Success 200 {object} resputil.Response[SSHInfo] "SSH endpoint and shared session"
// @Failure 400 {object} resputil.Response[any] "Unsupported session mode"
// @Failure 403 {object} resputil.Response[any] "Job is outside the current user/project"
// @Failure 404 {object} resputil.Response[any] "Job, pod or SSH service not found"
// @Failure 409 {object} resputil.Response[any] "Job or container is not running"
// @Failure 500 {object} resputil.Response[any] "SSH service or session setup failed"
// @Router /v1/vcjobs/{name}/ssh [post]
func (mgr *VolcanojobMgr) UserOpenSSH(c *gin.Context) {
	mode := c.Query("session")
	if mode != "" && mode != "user" {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("session must be user or omitted"))
		return
	}
	job, live, err := mgr.liveSessionJob(c, vcjobservice.SSHSession)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if job.Status != batch.Running {
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("job is not running"))
		return
	}
	podName, podLabels := getPodNameAndLabelFromJob(live)
	var candidate v1.Pod
	if err = mgr.client.Get(c, client.ObjectKey{Namespace: live.Namespace, Name: podName}, &candidate); err != nil {
		resputil.HandleError(c, execLookupError(err))
		return
	}
	// Preserve SSH's first-task selection while verifying the live controller UID.
	pod, container, err := vcjobservice.SelectSSHTarget(live, &candidate)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	const setupTimeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(c.Request.Context(), setupTimeout)
	defer cancel()
	run := mgr.sessionRunner(pod, container)
	info := SSHInfo{}
	if mode == "user" {
		var username bytes.Buffer
		if err = run(ctx, jobsession.InstallSSHCommand(), &username, io.Discard); err != nil {
			resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err,
				"could not prepare SSH environment; check the task account and writable /run/crater directory"))
			return
		}
		info.User, info.SessionCommand = username.String(), jobsession.SSHCommand
	}
	if cached := pod.Annotations[AnnotationKeySSHEnabled]; cached != "" {
		info.IP, info.Port, err = net.SplitHostPort(cached)
		if err != nil {
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("cached SSH address is invalid"))
			return
		}
		resputil.Success(c, info)
		return
	}
	if err = run(ctx, []string{"/bin/sh", "-c", startSSHD}, io.Discard, io.Discard); err != nil {
		resputil.HandleError(c, bizerr.NotFound.ServiceSshdNotFound.Wrap(err,
			"could not start SSH; use job exec or configure sshd in the image"))
		return
	}
	ip, port, err := mgr.serviceManager.CreateNodePort(c, []metav1.OwnerReference{{
		APIVersion: "batch.volcano.sh/v1alpha1", Kind: "Job", Name: live.Name,
		UID: live.UID, BlockOwnerDeletion: ptr.To(true),
	}}, podLabels, &v1.ServicePort{
		Name: "ssh", Port: SSHPort, Protocol: v1.ProtocolTCP, TargetPort: intstr.FromInt(SSHPort),
	}, job.User.Name)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "could not expose the container SSH service"))
		return
	}
	info.IP, info.Port = ip, fmt.Sprintf("%d", port)
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[AnnotationKeySSHEnabled] = net.JoinHostPort(ip, info.Port)
	if err = mgr.client.Update(c, pod); err != nil {
		klog.ErrorS(err, "Could not cache SSH address")
	}
	resputil.Success(c, info)
}

var startSSHD = strings.TrimSpace(`
set -eu
if ! id sshd >/dev/null 2>&1; then
    getent group sshd >/dev/null 2>&1 || groupadd --system sshd
    useradd --system -g sshd -d /var/lib/sshd -s /sbin/nologin sshd
fi
mkdir -p /run/sshd
if command -v service >/dev/null 2>&1; then
    service ssh restart || service sshd restart
else
    /usr/sbin/sshd
fi
`)
