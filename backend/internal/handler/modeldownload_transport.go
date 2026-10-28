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

package handler

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// The pinned SDK follows only relative HEAD redirects. Some mirrors return an
// absolute redirect without metadata; follow it without changing HF_ENDPOINT.
const huggingFaceHTTPClientScript = `
class HuggingFaceMetadataClient(httpx.Client):
    def send(self, request, **kwargs):
        response = super().send(request, **kwargs)
        endpoint = httpx.URL(os.environ.get("HF_ENDPOINT", "https://huggingface.co"))
        origin = (request.url.scheme, request.url.host, request.url.port)
        configured_origin = (endpoint.scheme, endpoint.host, endpoint.port)
        if request.method != "HEAD" or origin != configured_origin or "/resolve/" not in request.url.path:
            return response
        redirects = 0
        while response.is_redirect and not response.headers.get("X-Repo-Commit"):
            next_request = response.next_request
            if next_request is None or next_request.url.scheme != "https":
                return response
            if redirects >= self.max_redirects:
                response.close()
                raise httpx.TooManyRedirects("Too many Hub metadata redirects", request=request)
            # HTTPX builds next_request with cross-origin credentials removed.
            response.close()
            print("[METADATA] following HTTPS metadata redirect", flush=True)
            response = super().send(next_request, **kwargs)
            redirects += 1
        return response
`

// Preserve the configured Hub endpoint and let the SDK select Xet from file
// metadata. Both its Python and Xet clients read proxy settings from the environment.
func huggingFaceDownloadEnv(endpoint, httpsProxy, httpProxy, noProxy string) []corev1.EnvVar {
	env := []corev1.EnvVar{{Name: "HF_ENDPOINT", Value: endpoint}}
	return append(env, downloadProxyEnv(httpsProxy, httpProxy, noProxy)...)
}

func downloadProxyEnv(httpsProxy, httpProxy, noProxy string) []corev1.EnvVar {
	var env []corev1.EnvVar
	if httpsProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTPS_PROXY", Value: normalizeProxyURL(httpsProxy)})
	}
	if httpProxy != "" {
		env = append(env, corev1.EnvVar{Name: "HTTP_PROXY", Value: normalizeProxyURL(httpProxy)})
	}
	if noProxy != "" {
		env = append(env, corev1.EnvVar{Name: "NO_PROXY", Value: noProxy})
	}
	return env
}

func normalizeProxyURL(proxy string) string {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" || strings.Contains(proxy, "://") {
		return proxy
	}
	return "http://" + proxy
}
