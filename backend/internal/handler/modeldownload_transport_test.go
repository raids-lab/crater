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
	"fmt"
	"maps"
	"os/exec"
	"testing"
)

func TestHuggingFaceDownloadEnvPreservesEndpointAndXet(t *testing.T) {
	for _, endpoint := range []string{
		"https://huggingface.co",
		"https://hf-mirror.com",
		"https://gateway.example.test/hf",
	} {
		for _, withProxy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/proxy=%t", endpoint, withProxy), func(t *testing.T) {
				t.Parallel()
				var httpsProxy, httpProxy, noProxy string
				want := map[string]string{"HF_ENDPOINT": endpoint}
				if withProxy {
					httpsProxy, httpProxy, noProxy = "proxy:1080", "http://proxy:1081", "localhost,127.0.0.1"
					want["HTTPS_PROXY"] = "http://proxy:1080"
					want["HTTP_PROXY"] = httpProxy
					want["NO_PROXY"] = noProxy
				}
				env := huggingFaceDownloadEnv(endpoint, httpsProxy, httpProxy, noProxy)
				got := make(map[string]string, len(env))
				for _, variable := range env {
					if _, exists := got[variable.Name]; exists {
						t.Fatalf("duplicate environment variable %q", variable.Name)
					}
					got[variable.Name] = variable.Value
				}
				if !maps.Equal(got, want) {
					t.Fatalf("download environment = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestHuggingFaceMetadataRedirectClient(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("Python is required for the HTTPX metadata redirect test")
	}
	if err := exec.CommandContext(t.Context(), python, "-c", "import httpx").Run(); err != nil {
		t.Skip("httpx is required for the metadata redirect test")
	}
	script := `
import os
import httpx
import unittest

os.environ["HF_ENDPOINT"] = "https://hf-mirror.com"
` + huggingFaceHTTPClientScript + `

class MetadataRedirectTests(unittest.TestCase):
    def test_metadata_redirect_preserves_headers_and_strips_credentials(self):
        requests = []
        def handler(request):
            requests.append(request)
            if request.url.host == "hf-mirror.com":
                return httpx.Response(308, headers={"Location": "https://huggingface.co/org/repo/resolve/main/file"})
            return httpx.Response(302, headers={
                "Location": "https://cdn.example.test/file",
                "X-Repo-Commit": "commit", "X-Xet-Hash": "hash",
            })
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file",
                                   headers={"Authorization": "Bearer test-token"}, follow_redirects=False)
        self.assertEqual(len(requests), 2)
        self.assertEqual(requests[0].headers["Authorization"], "Bearer test-token")
        self.assertNotIn("Authorization", requests[1].headers)
        self.assertEqual(response.headers["X-Xet-Hash"], "hash")
        self.assertEqual(os.environ["HF_ENDPOINT"], "https://hf-mirror.com")

    def test_redirect_with_metadata_is_not_followed(self):
        requests = []
        def handler(request):
            requests.append(request)
            return httpx.Response(302, headers={"Location": "https://cdn.example.test/file", "X-Repo-Commit": "commit"})
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
        self.assertEqual(len(requests), 1)
        self.assertEqual(response.status_code, 302)

    def test_get_and_unrelated_head_requests_are_unchanged(self):
        for method, url in (("GET", "https://hf-mirror.com/org/repo/resolve/main/file"),
                            ("HEAD", "https://other.example.test/org/repo/resolve/main/file")):
            with self.subTest(method=method, url=url):
                requests = []
                def handler(request):
                    requests.append(request)
                    return httpx.Response(308, headers={"Location": "https://huggingface.co/org/repo/resolve/main/file"})
                with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
                    response = client.request(method, url, follow_redirects=False)
                self.assertEqual(len(requests), 1)
                self.assertEqual(response.status_code, 308)

    def test_https_downgrade_is_not_followed(self):
        requests = []
        def handler(request):
            requests.append(request)
            return httpx.Response(308, headers={"Location": "http://huggingface.co/org/repo/resolve/main/file"})
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
        self.assertEqual(len(requests), 1)
        self.assertEqual(response.status_code, 308)

    def test_redirect_loop_is_bounded(self):
        requests = []
        def handler(request):
            requests.append(request)
            return httpx.Response(308, headers={"Location": "https://hf-mirror.com/org/repo/resolve/main/next"})
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler), max_redirects=3) as client:
            with self.assertRaises(httpx.TooManyRedirects):
                client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
        self.assertEqual(len(requests), 4)

unittest.main()
`
	if output, err := exec.CommandContext(t.Context(), python, "-c", script).CombinedOutput(); err != nil {
		t.Fatalf("HTTPX metadata redirect tests failed: %v\n%s", err, output)
	}
}
