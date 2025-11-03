package tool

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/config"
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	handler.Registers = append(handler.Registers, NewWebsocketMgr)
}

type WebsocketMgr struct {
	name       string
	config     *rest.Config
	client     client.Client
	kubeClient kubernetes.Interface
}

func NewWebsocketMgr(conf *handler.RegisterConfig) handler.Manager {
	return &WebsocketMgr{
		name:       "websocket",
		config:     conf.KubeConfig,
		client:     conf.Client,
		kubeClient: conf.KubeClient,
	}
}

func (mgr *WebsocketMgr) GetName() string { return mgr.name }

func (mgr *WebsocketMgr) RegisterPublic(_ *gin.RouterGroup) {}
func (mgr *WebsocketMgr) RegisterAdmin(_ *gin.RouterGroup)  {}

func (mgr *WebsocketMgr) RegisterProtected(g *gin.RouterGroup) {
	g.GET("namespaces/:namespace/pods/:name/containers/:container/terminal", mgr.GetPodContainerTerminal)
}

type (
	PodContainerTerminalReq struct {
		// from uri
		Namespace     string `uri:"namespace" binding:"required"`
		PodName       string `uri:"name" binding:"required"`
		ContainerName string `uri:"container" binding:"required"`
	}
)

const (
	// WriteTimeout specifies the maximum duration for completing a write operation.
	WriteTimeout = 10 * time.Second
	// EndOfTransmission represents the signal for ending the transmission (Ctrl+D).
	EndOfTransmission = "\u0004"
)

// 首先定义终端大小消息的结构
type TerminalMessage struct {
	Op   string `json:"op"`   // 操作类型: "stdin", "stdout", "resize"等
	Data string `json:"data"` // 对于stdin/stdout是内容，对于resize是宽高
	Cols uint16 `json:"cols"` // 列数
	Rows uint16 `json:"rows"` // 行数
}

type streamHandler struct {
	ws       *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
}

// 实现TerminalSizeQueue接口的Next方法
func (h *streamHandler) Next() *remotecommand.TerminalSize {
	select {
	case size := <-h.sizeChan:
		return &size
	case <-h.doneChan:
		return nil
	}
}

func (h *streamHandler) Write(p []byte) (int, error) {
	if err := h.ws.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
		// If setting the write deadline fails, return the error immediately.
		return 0, err
	}
	err := h.ws.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

// References:
// - https://github.com/kubernetes/client-go/issues/554
// - https://github.com/juicedata/juicefs-csi-driver/pull/1053
func (h *streamHandler) Read(p []byte) (int, error) {
	_, message, err := h.ws.ReadMessage()
	if err != nil {
		// Returns "0x04" on error
		return copy(p, EndOfTransmission), err
	}

	// 尝试解析为终端消息
	var msg TerminalMessage
	if err := json.Unmarshal(message, &msg); err == nil {
		// 如果是resize操作
		if msg.Op == "resize" {
			h.sizeChan <- remotecommand.TerminalSize{
				Width:  msg.Cols,
				Height: msg.Rows,
			}
			return 0, nil
		}

		// 如果是stdin操作，使用Data字段
		if msg.Op == "stdin" {
			return copy(p, msg.Data), nil
		}
	}

	return copy(p, message), nil
}

// TODO: 通过 Pod Labels，检查当前访问的 Pod 是否属于当前用户
func (mgr *WebsocketMgr) GetPodContainerTerminal(c *gin.Context) {
	var req PodContainerTerminalReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	var upgrade = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	// Allow all origins in debug mode
	if config.IsDebugMode() {
		upgrade.CheckOrigin = func(_ *http.Request) bool {
			return true
		}
	}
	ws, err := upgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	stream := &streamHandler{
		ws:       ws,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
	}
	defer close(stream.doneChan)

	// Reference: https://github.com/juicedata/juicefs-csi-driver/pull/1053
	request := mgr.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec")
	request.VersionedParams(&v1.PodExecOptions{
		Command:   []string{"sh", "-c", "bash || sh"},
		Container: req.ContainerName,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(mgr.config, "POST", request.URL())
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stream,
		Stdout:            stream,
		Stderr:            stream,
		Tty:               true,
		TerminalSizeQueue: stream,
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
}
