package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
)

// UserProxyKthenaService godoc
//
//	@Summary		Proxy OpenAI-compatible inference request
//	@Description	Proxy an OpenAI-compatible request to kthena-router for an inference service owned by the current user and account.
//	@Tags			kthena
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string				true	"Inference service name"
//	@Param			path	path		string				true	"OpenAI-compatible API path"
//	@Param			request	body		KthenaProxyReq	false	"OpenAI-compatible request body"
//	@Success		200		{object}	any
//	@Failure		400		{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		404		{object}	any				"Model route or runtime pod not found"
//	@Failure		500		{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/kthena/inference-services/{name}/openai/{path} [post]
func (mgr *KthenaMgr) UserProxyKthenaService(c *gin.Context) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}

	targetPath := strings.TrimPrefix(c.Param("path"), "/")
	if targetPath == "" {
		targetPath = kthenaChatCompletionsPath
	}
	if !strings.HasPrefix(targetPath, "v1/") {
		targetPath = "v1/" + targetPath
	}

	// Router discovery and arbitrary paths cannot be authorized by one deployment.
	if !isKthenaInferenceEndpoint(c.Request.Method, targetPath) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, kthenaMaxProxyBodyBytes)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "failed to read request body"))
		return
	}
	body, err = withDefaultModel(body, obj.GetName())
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.Wrap(err, err.Error()))
		return
	}

	response, err := mgr.openKthenaRouterResponse(c.Request.Context(), c.Request.Method, targetPath, body, c.Request.Header)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "proxy inference request failed"))
		return
	}
	defer func() { _ = response.Body.Close() }()
	forwardKthenaResponse(c.Writer, response)
}

func (mgr *KthenaMgr) proxyKthenaRouter(
	ctx context.Context,
	method string,
	targetPath string,
	body []byte,
	headers http.Header,
) ([]byte, error) {
	if mgr.proxyKthenaRouterFn != nil {
		return mgr.proxyKthenaRouterFn(ctx, method, targetPath, body, headers)
	}
	if mgr.kubeClient == nil {
		return nil, bizerr.Internal.K8sServiceError.New("kubernetes client is not initialized")
	}
	response, err := mgr.openKthenaRouterResponse(ctx, method, targetPath, body, headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, kthenaMaxProxyBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > kthenaMaxProxyBodyBytes {
		return nil, bizerr.Internal.ServiceError.New("completion exceeds size limit")
	}
	if response.StatusCode < 100 ||
		response.StatusCode > 599 {
		return nil,
			bizerr.Internal.ServiceError.New("invalid inference HTTP status")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return raw, &errors.StatusError{
			ErrStatus: metav1.Status{Code: int32(response.StatusCode), Message: "inference request failed"},
		}
	}
	return raw, nil
}

func kthenaProxyHTTPStatus(err error) int {
	type apiStatus interface {
		Status() metav1.Status
	}
	if statusErr, ok := err.(apiStatus); ok {
		statusCode := int(statusErr.Status().Code)
		if statusCode >= http.StatusBadRequest && statusCode <= 599 {
			return statusCode
		}
	}
	return http.StatusBadGateway
}

func withDefaultModel(body []byte, modelName string) ([]byte, error) {
	if len(bytes.TrimSpace(body)) == 0 || strings.TrimSpace(modelName) == "" {
		return nil, bizerr.BadRequest.InvalidRequest.New("request body and deployment model are required")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, bizerr.BadRequest.InvalidRequest.New("request body must be valid JSON")
	}
	if payload == nil {
		return nil, bizerr.BadRequest.InvalidRequest.New("request body must be a JSON object")
	}
	requested := strings.TrimSpace(stringValue(payload["model"]))
	if requested != "" && requested != modelName {
		return nil, bizerr.BadRequest.ParameterError.New("model must match the authorized deployment route")
	}
	payload["model"] = modelName
	return json.Marshal(payload)
}
