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
                "X-Linked-ETag": "etag", "X-Linked-Size": "65000000000",
            })
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file",
                                   headers={"Authorization": "Bearer test-token"}, follow_redirects=False)
        self.assertEqual(len(requests), 2)
        self.assertEqual(requests[0].headers["Authorization"], "Bearer test-token")
        self.assertNotIn("Authorization", requests[1].headers)
        self.assertEqual(response.headers["X-Xet-Hash"], "hash")
        self.assertEqual(response.headers["X-Linked-ETag"], "etag")
        self.assertEqual(response.headers["X-Linked-Size"], "65000000000")
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
                            ("HEAD", "https://other.example.test/org/repo/resolve/main/file"),
                            ("HEAD", "https://hf-mirror.com/api/resolve-cache/models/org/repo/commit/file")):
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

    def test_relative_redirect_uses_current_response_origin(self):
        for location, expected in (
            ("/api/resolve-cache/models/org/repo/commit/file", "https://huggingface.co/api/resolve-cache/models/org/repo/commit/file"),
            ("../commit/file", "https://huggingface.co/org/repo/resolve/commit/file"),
            ("./cached/file", "https://huggingface.co/org/repo/resolve/main/cached/file"),
            ("?download=true", "https://huggingface.co/org/repo/resolve/main/file?download=true"),
        ):
            for with_commit in (False, True):
                with self.subTest(location=location, with_commit=with_commit):
                    requests, responses = [], []
                    def handler(request):
                        requests.append(request)
                        if len(requests) == 1:
                            response = httpx.Response(308, headers={"Location": "https://huggingface.co/org/repo/resolve/main/file"})
                        elif len(requests) == 2:
                            headers = {"Location": location}
                            if with_commit:
                                headers["X-Repo-Commit"] = "commit"
                            response = httpx.Response(307, headers=headers)
                        else:
                            self.assertEqual(len(requests), 3)
                            self.assertEqual(str(request.url), expected)
                            response = httpx.Response(200, headers={"X-Repo-Commit": "commit", "ETag": "etag", "Content-Length": "12"})
                        responses.append(response)
                        return response
                    with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
                        response = client.head("https://hf-mirror.com/org/repo/resolve/main/file",
                                               headers={"Authorization": "Bearer test-token"}, follow_redirects=False)
                    self.assertEqual(len(requests), 3)
                    self.assertEqual(response.status_code, 200)
                    self.assertEqual(response.headers["X-Repo-Commit"], "commit")
                    self.assertTrue(all(r.is_closed for r in responses[:-1]))
                    self.assertTrue(all("Authorization" not in r.headers for r in requests[1:]))

    def test_scheme_relative_redirect_is_normalized_without_following_cdn(self):
        requests = []
        def handler(request):
            requests.append(request)
            return httpx.Response(302, headers={
                "Location": "//cdn.example.test/file?signature=value",
                "X-Repo-Commit": "commit", "X-Xet-Hash": "hash",
                "X-Linked-ETag": "etag", "X-Linked-Size": "12",
            })
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
        self.assertEqual(len(requests), 1)
        self.assertEqual(response.headers["Location"], "https://cdn.example.test/file?signature=value")
        self.assertEqual(response.headers["X-Xet-Hash"], "hash")

    def test_scheme_relative_redirect_without_metadata_is_followed(self):
        requests = []
        def handler(request):
            requests.append(request)
            if len(requests) == 1:
                return httpx.Response(308, headers={"Location": "//huggingface.co/org/repo/resolve/main/file"})
            self.assertEqual(request.url.host, "huggingface.co")
            self.assertNotIn("Authorization", request.headers)
            return httpx.Response(200, headers={"X-Repo-Commit": "commit"})
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file",
                                   headers={"Authorization": "Bearer test-token"}, follow_redirects=False)
        self.assertEqual(len(requests), 2)
        self.assertEqual(response.status_code, 200)

    def test_terminal_responses_are_not_retried_or_given_metadata(self):
        for status in (200, 302, 304, 401, 403, 404):
            with self.subTest(status=status):
                requests = []
                def handler(request):
                    requests.append(request)
                    return httpx.Response(status)
                with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
                    response = client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
                self.assertEqual(len(requests), 1)
                self.assertEqual(response.status_code, status)
                self.assertNotIn("X-Repo-Commit", response.headers)

    def test_later_https_downgrade_is_not_followed(self):
        requests = []
        def handler(request):
            requests.append(request)
            if len(requests) == 1:
                return httpx.Response(308, headers={"Location": "https://huggingface.co/org/repo/resolve/main/file"})
            return httpx.Response(307, headers={"Location": "http://huggingface.co/cache/file", "X-Repo-Commit": "commit"})
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler)) as client:
            response = client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
        self.assertEqual(len(requests), 2)
        self.assertEqual(response.status_code, 307)

    def test_relative_redirect_loop_with_commit_is_bounded(self):
        requests, responses = [], []
        def handler(request):
            requests.append(request)
            response = httpx.Response(307, headers={"Location": "./file", "X-Repo-Commit": "commit"})
            responses.append(response)
            return response
        with HuggingFaceMetadataClient(transport=httpx.MockTransport(handler), max_redirects=3) as client:
            with self.assertRaises(httpx.TooManyRedirects):
                client.head("https://hf-mirror.com/org/repo/resolve/main/file", follow_redirects=False)
        self.assertEqual(len(requests), 4)
        self.assertTrue(all(r.is_closed for r in responses))

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

func TestHuggingFaceMetadataRedirectSDK(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("Python is required for the pinned SDK metadata redirect test")
	}
	checkVersion := fmt.Sprintf("import huggingface_hub, httpx; assert huggingface_hub.__version__ == %q", huggingFaceHubVersion)
	if output, err := exec.CommandContext(t.Context(), python, "-c", checkVersion).CombinedOutput(); err != nil {
		t.Skipf("huggingface_hub==%s and httpx are required: %s", huggingFaceHubVersion, output)
	}
	script := `
import os
import tempfile
import unittest
import httpx

# Keep SDK user-agent discovery out of the mocked metadata requests.
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"

from huggingface_hub import get_hf_file_metadata, hf_hub_download, set_client_factory
from huggingface_hub.errors import HfHubHTTPError, LocalEntryNotFoundError
from huggingface_hub.utils import close_session

os.environ["HF_ENDPOINT"] = "https://hf-mirror.com"
` + huggingFaceHTTPClientScript + `

class SDKMetadataRedirectTests(unittest.TestCase):
    def tearDown(self):
        close_session()

    def use_handler(self, handler):
        close_session()
        set_client_factory(lambda: HuggingFaceMetadataClient(transport=httpx.MockTransport(handler), follow_redirects=True))

    def test_mirror_to_official_relative_cache_metadata(self):
        requests = []
        cache_url = "https://huggingface.co/api/resolve-cache/models/org/repo/commit/file"
        def handler(request):
            requests.append(request)
            self.assertEqual(request.method, "HEAD")
            if len(requests) == 1:
                return httpx.Response(308, headers={"Location": "https://huggingface.co/org/repo/resolve/main/file"})
            if len(requests) == 2:
                return httpx.Response(307, headers={"Location": "/api/resolve-cache/models/org/repo/commit/file", "X-Repo-Commit": "commit"})
            self.assertEqual(len(requests), 3)
            self.assertEqual(str(request.url), cache_url)
            return httpx.Response(200, headers={"X-Repo-Commit": "commit", "ETag": '"etag"', "Content-Length": "12"})
        self.use_handler(handler)
        metadata = get_hf_file_metadata("https://hf-mirror.com/org/repo/resolve/main/file", token="test-token")
        self.assertEqual(len(requests), 3)
        self.assertEqual(metadata.commit_hash, "commit")
        self.assertEqual(metadata.etag, "etag")
        self.assertEqual(metadata.size, 12)
        self.assertEqual(metadata.location, cache_url)
        self.assertTrue(all("Authorization" not in r.headers for r in requests[1:]))

    def test_cdn_metadata_preserves_xet_without_requesting_cdn(self):
        for location in ("https://cdn.example.test/file", "//cdn.example.test/file"):
            with self.subTest(location=location):
                requests = []
                def handler(request):
                    requests.append(request)
                    self.assertEqual(request.url.host, "hf-mirror.com")
                    return httpx.Response(302, headers={
                        "Location": location, "X-Repo-Commit": "commit", "X-Xet-Hash": "hash",
                        "X-Xet-Refresh-Route": "https://huggingface.co/api/models/org/repo/xet-read-token/commit",
                        "X-Linked-ETag": '"etag"', "X-Linked-Size": "65000000000",
                    })
                self.use_handler(handler)
                metadata = get_hf_file_metadata("https://hf-mirror.com/org/repo/resolve/main/file", token=False)
                self.assertEqual(len(requests), 1)
                self.assertEqual(metadata.commit_hash, "commit")
                self.assertEqual(metadata.etag, "etag")
                self.assertEqual(metadata.size, 65000000000)
                self.assertEqual(metadata.location, "https://cdn.example.test/file")
                self.assertEqual(metadata.xet_file_data.file_hash, "hash")

    def test_sdk_rejects_success_response_without_commit(self):
        def handler(request):
            self.assertEqual(request.method, "HEAD")
            return httpx.Response(200, headers={"ETag": '"etag"', "Content-Length": "12"})
        self.use_handler(handler)
        with tempfile.TemporaryDirectory() as cache:
            with self.assertRaises(LocalEntryNotFoundError):
                hf_hub_download("org/repo", "file", endpoint="https://hf-mirror.com", cache_dir=cache, token=False)

    def test_sdk_reports_http_errors(self):
        for status in (401, 403, 404):
            with self.subTest(status=status):
                def handler(request):
                    return httpx.Response(status)
                self.use_handler(handler)
                with self.assertRaises(HfHubHTTPError):
                    get_hf_file_metadata("https://hf-mirror.com/org/repo/resolve/main/file", token=False)

unittest.main()
`
	if output, err := exec.CommandContext(t.Context(), python, "-c", script).CombinedOutput(); err != nil {
		t.Fatalf("Pinned SDK metadata redirect tests failed: %v\n%s", err, output)
	}
}
