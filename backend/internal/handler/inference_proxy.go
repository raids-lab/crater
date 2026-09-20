package handler

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"k8s.io/client-go/rest"

	"github.com/raids-lab/crater/internal/bizerr"
)

// Use the authenticated Kubernetes transport directly so the status, headers,
// and incremental response body survive the service proxy unchanged.
func (mgr *KthenaMgr) openKthenaRouterResponse(
	ctx context.Context, method, targetPath string, body []byte, headers http.Header,
) (*http.Response, error) {
	if mgr.kubeClient == nil {
		return nil, bizerr.Internal.ServiceError.New("kubernetes client is not initialized")
	}
	client, ok := mgr.kubeClient.CoreV1().RESTClient().(*rest.RESTClient)
	if !ok || client == nil {
		return nil, bizerr.Internal.ServiceError.New("kubernetes REST transport is unavailable")
	}
	endpoint := client.Verb(method).Namespace(kthenaNamespace).Resource("services").
		Name(kthenaRouterService + ":http").SubResource("proxy").Suffix(strings.Split(targetPath, "/")...).URL()
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// Never forward caller credentials to the Kubernetes API server.
	for _, name := range []string{"Accept", "Content-Type"} {
		if value := headers.Get(name); value != "" {
			request.Header.Set(name, value)
		}
	}
	request.Header.Set("Content-Type", "application/json")
	return client.Client.Do(request)
}

func forwardKthenaResponse(writer http.ResponseWriter, response *http.Response) {
	for _, name := range []string{"Content-Type", "Cache-Control", "Retry-After", "X-Request-Id"} {
		if value := response.Header.Get(name); value != "" {
			writer.Header().Set(name, value)
		}
	}
	writer.WriteHeader(response.StatusCode)
	const responseBufferSize = 32 * 1024
	buffer := make([]byte, responseBufferSize)
	for {
		n, err := response.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := writer.Write(buffer[:n]); writeErr != nil {
				return
			}
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}
