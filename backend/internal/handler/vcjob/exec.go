package vcjob

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/constants"
	"github.com/raids-lab/crater/pkg/jobexec"
	"github.com/raids-lab/crater/pkg/jobsession"
)

const (
	execAdminScope  = "admin"
	execScopeHeader = "X-Crater-Exec-Scope"
)

type execSessionTarget struct {
	job       *model.Job
	pod       *v1.Pod
	container string
}

// UserExec godoc
//
// @Summary Open a user shell or execute argv in a running job
// @Description Subprotocol crater.exec.v1: start, stdin/eof/resize, ready/stdout/stderr/exit/error.
// @Description The caller must own the job in the current project by default.
// @Description scope=admin requires a platform administrator and a Bearer header.
// @Description Clients using scope=admin must wait for X-Crater-Exec-Scope: admin on upgrade before sending start.
// @Description Empty command opens a user login shell with stdin/tty.
// @Description env_mode=user (default) loads user shell configuration; container skips probing.
// @Description Non-TTY commands use a bounded environment probe; TTY commands use the user login shell.
// @Description Exec resolves HOME and shell from the task account. Byte payloads use Base64 JSON strings.
// @Tags VolcanoJob
// @Security Bearer
// @Param name path string true "Platform job name"
// @Param pod query string false "Pod name; required for multiple pods"
// @Param container query string false "Container name; required for multiple regular containers"
// @Param scope query string false "Request administrator job access" Enums(admin)
// @Success 101 {string} string "WebSocket exec session"
// @Header 101 {string} X-Crater-Exec-Scope "admin for an authorized administrator exec request"
// @Failure 400 {object} resputil.Response[any] "Invalid target, scope or subprotocol"
// @Failure 401 {object} resputil.Response[any] "Invalid credentials or missing administrator Bearer header"
// @Failure 403 {object} resputil.Response[any] "Job or requested scope is outside the caller's permissions"
// @Failure 404 {object} resputil.Response[any] "Job or pod not found"
// @Failure 409 {object} resputil.Response[any] "Container is not running"
// @Failure 500 {object} resputil.Response[any] "Platform failure"
// @Router /v1/vcjobs/{name}/exec [get]
func (mgr *VolcanojobMgr) UserExec(c *gin.Context) {
	kind := vcjobservice.ExecSession
	var headers http.Header
	if scopes, present := c.GetQueryArray("scope"); present {
		if len(scopes) != 1 || scopes[0] != execAdminScope {
			resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("exec scope must be admin when specified"))
			return
		}
		// AuthProtected validates a supplied Bearer header. Admin exec excludes its browser token fallback.
		if c.GetHeader("Authorization") == "" {
			resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("Administrator exec requires an Authorization Bearer header."))
			return
		}
		if err := vcjobservice.CheckAdminExecAccess(util.GetToken(c).RolePlatform); err != nil {
			resputil.HandleError(c, err)
			return
		}
		kind = vcjobservice.AdminExecSession
		headers = http.Header{execScopeHeader: {execAdminScope}}
	}
	target, err := mgr.execTarget(c, kind)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	if !slices.Contains(websocket.Subprotocols(c.Request), jobexec.Subprotocol) {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("exec requires WebSocket subprotocol crater.exec.v1"))
		return
	}
	upgrader := websocket.Upgrader{Subprotocols: []string{jobexec.Subprotocol}}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, headers)
	if err != nil {
		return
	}
	defer ws.Close()
	mgr.runUserExec(c, ws, target, kind)
}

func (mgr *VolcanojobMgr) execTarget(c *gin.Context, kind vcjobservice.SessionKind) (execSessionTarget, error) {
	job, live, err := mgr.liveSessionJob(c, kind)
	if err != nil {
		return execSessionTarget{}, err
	}
	pods := &v1.PodList{}
	if err = mgr.client.List(c.Request.Context(), pods, client.InNamespace(live.Namespace),
		client.MatchingLabels{batch.JobNameKey: live.Name}); err != nil {
		return execSessionTarget{}, execLookupError(err)
	}
	pod, container, err := vcjobservice.SelectExecTarget(live, pods.Items, c.Query("pod"), c.Query("container"))
	if err != nil {
		return execSessionTarget{}, err
	}
	return execSessionTarget{job: job, pod: pod, container: container}, nil
}

func (mgr *VolcanojobMgr) liveSessionJob(c *gin.Context, kind vcjobservice.SessionKind) (job *model.Job, live *batch.Job, err error) {
	token := util.GetToken(c)
	job, err = getJob(c, c.Param("name"), &token)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if kind == vcjobservice.AdminExecSession {
			return nil, nil, bizerr.NotFound.DataBaseNotFound.New("job not found")
		}
		return nil, nil, bizerr.NotFound.DataBaseNotFound.New("job not found in the current user and project")
	}
	if err != nil {
		return nil, nil, bizerr.Internal.DatabaseError.Wrap(err, "could not look up job")
	}
	if err := vcjobservice.CheckSessionAccess(job, token.UserID, token.AccountID, token.RolePlatform, kind); err != nil {
		return nil, nil, err
	}
	stored := job.Attributes.Data()
	if stored == nil || stored.Namespace == "" {
		return nil, nil, bizerr.Conflict.ResourceStatusError.New("job has no active Kubernetes namespace")
	}
	live = &batch.Job{}
	if err = mgr.client.Get(c.Request.Context(), client.ObjectKey{Namespace: stored.Namespace, Name: job.JobName}, live); err != nil {
		return nil, nil, execLookupError(err)
	}
	return job, live, nil
}

func (mgr *VolcanojobMgr) runUserExec(
	c *gin.Context, ws *websocket.Conn, target execSessionTarget, kind vcjobservice.SessionKind,
) {
	start, err := jobexec.ReadStart(ws)
	message := "could not initialize exec; check the command and platform connection"
	var prepared jobsession.Prepared
	pod, container := target.pod, target.container
	if err == nil {
		if kind == vcjobservice.AdminExecSession {
			// Record the accepted session before environment probing or command execution.
			target.recordAdminExec(c)
		}
		prepared, err = jobsession.Command(c.Request.Context(), mgr.sessionRunner(pod, container), start.Command, start.EnvMode, start.TTY)
		if err != nil {
			message = err.Error()
		}
	}
	if err == nil {
		var executor remotecommand.Executor
		executor, err = mgr.sessionExecutor(pod, container, prepared.Argv, start.Stdin || len(prepared.Input) > 0, start.TTY)
		if err == nil {
			jobexec.Run(c.Request.Context(), ws, executor, &start, &jobexec.Message{
				Type: "ready", Namespace: pod.Namespace, Pod: pod.Name, Container: container,
			}, prepared.Input)
			return
		}
	}
	klog.ErrorS(err, "Could not initialize container exec")
	const initializationWriteTimeout = 10 * time.Second
	if deadlineErr := ws.SetWriteDeadline(time.Now().Add(initializationWriteTimeout)); deadlineErr == nil {
		frame := jobexec.Message{Type: "error", Message: message}
		if writeErr := ws.WriteJSON(frame); writeErr != nil {
			klog.ErrorS(writeErr, "Exec initialization error could not be delivered")
		}
	}
}

func (target execSessionTarget) recordAdminExec(c *gin.Context) {
	operator := util.GetToken(c)
	handler.RecordOperationLog(c, constants.OpTypeAdminJobExec, target.job.JobName, constants.OpStatusSuccess, "", map[string]any{
		"operatorUsername": operator.Username,
		"operatorUserID":   operator.UserID,
		"owner":            target.job.User.Name,
		"ownerUserID":      target.job.UserID,
		"account":          target.job.Account.Name,
		"accountID":        target.job.AccountID,
		"namespace":        target.pod.Namespace,
		"pod":              target.pod.Name,
		"container":        target.container,
	})
}

func (mgr *VolcanojobMgr) sessionRunner(pod *v1.Pod, container string) jobsession.Runner {
	return func(ctx context.Context, argv []string, stdout, stderr io.Writer) error {
		executor, err := mgr.sessionExecutor(pod, container, argv, false, false)
		if err != nil {
			return err
		}
		return executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: stdout, Stderr: stderr})
	}
}

func (mgr *VolcanojobMgr) sessionExecutor(pod *v1.Pod, container string, argv []string, stdin, tty bool) (remotecommand.Executor, error) {
	request := mgr.kubeClient.CoreV1().RESTClient().Post().Resource("pods").Namespace(pod.Namespace).
		Name(pod.Name).SubResource("exec").VersionedParams(&v1.PodExecOptions{
		Command: argv, Container: container, Stdin: stdin, Stdout: true, Stderr: !tty, TTY: tty,
	}, scheme.ParameterCodec)
	return remotecommand.NewSPDYExecutor(mgr.config, http.MethodPost, request.URL())
}

func execLookupError(err error) error {
	if apierrors.IsNotFound(err) {
		return bizerr.NotFound.K8sResourceNotFound.New("job or pod no longer exists")
	}
	return bizerr.Internal.K8sServiceError.Wrap(err, "could not locate the job containers")
}
