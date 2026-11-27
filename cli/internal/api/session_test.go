package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imroc/req/v3"
)

const expiredSessionBody = `{"code":40101,"msg":"access expired"}`

var originalTokens = Tokens{AccessToken: "old-access", RefreshToken: "old-refresh"}
var renewedTokens = Tokens{AccessToken: "new-access", RefreshToken: "new-refresh"}

func sessionJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func respondToRefresh(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
	}
	if r.Method != http.MethodPost || r.Header.Get("Authorization") != "" ||
		len(body) != 1 || body["refreshToken"] != originalTokens.RefreshToken {
		t.Error("refresh request must send only the refresh credential")
	}
	sessionJSON(w, http.StatusOK, `{"code":0,"data":{"accessToken":"new-access","refreshToken":"new-refresh"}}`)
}

func testRenewingClient(t *testing.T, server *httptest.Server, tokens Tokens) *Client {
	t.Helper()
	return NewClient(server.URL).SetSession(tokens, func(Tokens) error { return nil }, func(previous, next Tokens, requireActive bool) error {
		if previous != tokens || next != renewedTokens || !requireActive {
			t.Error("renewal must replace exactly the original pair")
		}
		return nil
	})
}

func TestResolveSessionPlatform(t *testing.T) {
	for _, tc := range []struct {
		name, base, location, want string
		probe, rejected            bool
	}{
		{name: "HTTPS", base: "https://platform.invalid"},
		{name: "localhost", base: "http://localhost:8089"},
		{name: "IPv4 loopback", base: "http://127.0.0.1:8089"},
		{name: "IPv6 loopback", base: "http://[::1]:8089"},
		{name: "direct HTTP", base: "http://platform.invalid", probe: true},
		{name: "HTTPS upgrade", base: "http://platform.invalid:80/crater", probe: true,
			location: "https://platform.invalid:443/crater" + AuthCheckPath, want: "https://platform.invalid:443/crater"},
		{name: "nonstandard HTTP port", base: "http://platform.invalid:8080", probe: true, rejected: true,
			location: "https://platform.invalid" + AuthCheckPath},
		{name: "nonstandard HTTPS port", base: "http://platform.invalid", probe: true, rejected: true,
			location: "https://platform.invalid:8443" + AuthCheckPath},
		{name: "different host", base: "http://platform.invalid", probe: true, rejected: true,
			location: "https://other.invalid" + AuthCheckPath},
		{name: "different path", base: "http://platform.invalid", probe: true, rejected: true,
			location: "https://platform.invalid/other" + AuthCheckPath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(tc.base)
			var probes int
			client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
				return func(r *http.Request) (*http.Response, error) {
					probes++
					if r.Method != http.MethodGet || r.URL.String() != tc.base+AuthCheckPath || r.Header.Get("Authorization") != "" {
						t.Errorf("expected an unauthenticated GET to the configured platform, got %s %s", r.Method, r.URL)
					}
					if tc.location == "" {
						return simulatedJSONResponse(r, http.StatusUnauthorized, expiredSessionBody), nil
					}
					response := simulatedJSONResponse(r, http.StatusPermanentRedirect, "")
					response.Header.Set("Location", tc.location)
					return response, nil
				}
			})
			err := client.resolveSessionPlatform(t.Context())
			var redirect *RequestError
			if tc.rejected {
				if !errors.As(err, &redirect) || redirect.HTTPStatus != http.StatusPermanentRedirect {
					t.Fatalf("expected an HTTP redirect error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			want := tc.want
			if want == "" {
				want = tc.base
			}
			if client.BaseURL != want || client.httpClient.BaseURL != want || (probes == 1) != tc.probe || probes > 1 {
				t.Fatalf("expected platform %s and probe=%t, got %s / %s and %d requests", want, tc.probe, client.BaseURL, client.httpClient.BaseURL, probes)
			}
		})
	}
}

func TestUpgradedSessionUsesHTTPSForRequestsAndRenewal(t *testing.T) {
	const httpBase = "http://example.com/crater"
	const httpsBase = "https://example.com/crater"
	var probes, accountRequests, refreshes atomic.Int32
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probes.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/crater"+AuthCheckPath || r.Header.Get("Authorization") != "" {
			t.Error("the HTTP server must see only the unauthenticated GET probe")
		}
		http.Redirect(w, r, httpsBase+AuthCheckPath, http.StatusPermanentRedirect)
	}))
	defer httpServer.Close()
	httpsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crater" + AuthRefreshPath:
			refreshes.Add(1)
			respondToRefresh(t, w, r)
		case "/crater" + AuthCheckPath:
			access := r.Header.Get("Authorization")
			attempt := accountRequests.Add(1)
			switch attempt {
			case 1, 2:
				if access != "Bearer old-access" {
					t.Error("the original access credential must reach the HTTPS server")
				}
				if attempt == 2 {
					sessionJSON(w, http.StatusUnauthorized, expiredSessionBody)
					return
				}
			case 3:
				if access != "Bearer new-access" {
					t.Error("the retried request must use the new access credential")
				}
			}
			sessionJSON(w, http.StatusOK, `{"data":{"context":{"queue":"research"}}}`)
		default:
			t.Errorf("unexpected HTTPS request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer httpsServer.Close()

	// Route both logical default ports to local test listeners. TLS still checks
	// example.com against the trusted certificate supplied by httptest.
	roots := x509.NewCertPool()
	roots.AddCert(httpsServer.Certificate())
	client := NewClient(httpBase)
	dialer := net.Dialer{}
	client.httpClient.SetProxy(nil).SetTLSClientConfig(&tls.Config{RootCAs: roots}).
		SetDial(func(ctx context.Context, network, address string) (net.Conn, error) {
			switch address {
			case "example.com:80":
				address = httpServer.Listener.Addr().String()
			case "example.com:443":
				address = httpsServer.Listener.Addr().String()
			default:
				return nil, fmt.Errorf("unexpected network address %s", address)
			}
			return dialer.DialContext(ctx, network, address)
		})
	if err := client.resolveSessionPlatform(t.Context()); err != nil {
		t.Fatal(err)
	}
	var saves int
	client.SetSession(originalTokens, func(Tokens) error { return nil }, func(previous, next Tokens, requireActive bool) error {
		saves++
		if previous != originalTokens || next != renewedTokens || !requireActive {
			t.Error("renewal must replace the original pair for the active identity")
		}
		return nil
	})
	if account, err := client.CurrentAccount(); err != nil || account.Queue != "research" {
		t.Fatalf("expected the HTTPS account request to succeed, got %v, %v", account, err)
	}
	if account, err := client.CurrentAccount(); err != nil || account.Queue != "research" {
		t.Fatalf("expected the account request to succeed after HTTPS renewal, got %v, %v", account, err)
	}
	if probes.Load() != 1 || accountRequests.Load() != 3 || refreshes.Load() != 1 || saves != 1 || client.currentTokens() != renewedTokens {
		t.Fatal("expected one probe, three HTTPS account requests, and one successful HTTPS renewal and save")
	}
}

func TestSessionRenewsAndReplaysAPostOnce(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == AuthRefreshPath {
			respondToRefresh(t, w, r)
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["queue"] != "research" {
			t.Error("original request body was not preserved")
		}
		attempt := attempts.Add(1)
		if attempt == 1 && r.Header.Get("Authorization") == "Bearer old-access" {
			sessionJSON(w, http.StatusUnauthorized, expiredSessionBody)
			return
		}
		if attempt != 2 || r.Header.Get("Authorization") != "Bearer new-access" {
			t.Error("retry must occur once with the renewed access credential")
		}
		sessionJSON(w, http.StatusOK, `{"code":0,"data":{"accessToken":"scoped","context":{"queue":"research"}}}`)
	}))
	defer server.Close()
	client := testRenewingClient(t, server, originalTokens)
	if _, err := client.SwitchAccount("research"); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 || client.currentTokens() != renewedTokens {
		t.Fatal("failed to renew and replay the original request exactly once")
	}
}

func TestSessionPreservesAPIErrorAndBoundsRenewal(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		initial, refresh                    string
		refreshStatus, wantStatus, wantCode int
		hasRefresh, wantSignIn              bool
		wantAttempts, wantRefreshes         int
	}{
		{"legacy access-only", expiredSessionBody, "", 0, 401, 40101, false, true, 1, 0},
		{"generic unauthorized", `{"code":999,"msg":"other unauthorized error"}`, "", 0, 401, 999, true, false, 1, 0},
		{"invalid access does not renew", `{"code":40102,"msg":"invalid access"}`, "", 0, 401, 40102, true, true, 1, 0},
		{"expired refresh", expiredSessionBody, `{"code":40101,"msg":"refresh expired"}`, 401, 401, 40101, true, true, 1, 1},
		{"invalid refresh", expiredSessionBody, `{"code":40102,"msg":"invalid refresh"}`, 401, 401, 40102, true, true, 1, 1},
		{"refresh endpoint missing", expiredSessionBody, `{"code":40401,"msg":"route not found"}`, 404, 404, 40401, true, false, 1, 1},
		{"service failure", expiredSessionBody, `{"code":50001,"msg":"service unavailable"}`, 503, 503, 50001, true, false, 1, 1},
		{"incomplete pair", expiredSessionBody, `{"data":{"accessToken":"new-access"}}`, 200, 200, 0, true, false, 1, 1},
		{"second expired", expiredSessionBody, `{"data":{"accessToken":"new-access","refreshToken":"new-refresh"}}`, 200, 401, 40101, true, true, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var attempts, refreshes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == AuthRefreshPath {
					refreshes.Add(1)
					sessionJSON(w, tc.refreshStatus, tc.refresh)
					return
				}
				attempts.Add(1)
				sessionJSON(w, 401, tc.initial)
			}))
			defer server.Close()
			tokens := originalTokens
			if !tc.hasRefresh {
				tokens.RefreshToken = ""
			}
			client := NewClient(server.URL).SetSession(tokens, func(Tokens) error { return nil }, func(Tokens, Tokens, bool) error {
				if tc.wantAttempts != 2 {
					t.Error("failed renewal must not save credentials")
				}
				return nil
			})
			_, err := client.CurrentAccount()
			var failure *RequestError
			if !errors.As(err, &failure) || failure.HTTPStatus != tc.wantStatus || failure.CraterCode != tc.wantCode ||
				failure.ShouldSignInAgain() != tc.wantSignIn || failure.Msg == "" {
				t.Fatalf("unexpected error: %+v", failure)
			}
			if tc.refreshStatus == http.StatusNotFound &&
				(!failure.DuringRefresh || failure.Msg != "route not found" || client.currentTokens() != tokens) {
				t.Fatal("missing refresh endpoint must preserve the error and original credentials")
			}
			if attempts.Load() != int32(tc.wantAttempts) || refreshes.Load() != int32(tc.wantRefreshes) {
				t.Fatalf("unexpected request counts: protected=%d refresh=%d", attempts.Load(), refreshes.Load())
			}
		})
	}
}

func accessTokenForStorageTest(expiry time.Time) string {
	payload := fmt.Sprintf(`{"exp":%d}`, expiry.Unix())
	return "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
}

func TestStorageListGoesDirectlyToStorage(t *testing.T) {
	for _, tc := range []struct {
		name, access, refresh string
	}{
		{"unexpired", accessTokenForStorageTest(time.Now().Add(time.Hour)), originalTokens.RefreshToken},
		{"unreadable expiry", "access-without-readable-expiry", originalTokens.RefreshToken},
		{"legacy access only", accessTokenForStorageTest(time.Now().Add(-time.Hour)), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path != FileListPath || r.Header.Get("Authorization") != "Bearer "+tc.access {
					t.Errorf("expected a direct storage request, got %s", r.URL.Path)
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				sessionJSON(w, http.StatusOK, `{"data":[{"name":"example.txt"}]}`)
			}))
			defer server.Close()
			client := testRenewingClient(t, server, Tokens{AccessToken: tc.access, RefreshToken: tc.refresh})
			files, err := client.ListFiles("")
			if err != nil || len(files) != 1 || files[0].Name != "example.txt" || requests.Load() != 1 {
				t.Fatalf("expected one direct file-list request: files=%v, requests=%d, err=%v", files, requests.Load(), err)
			}
		})
	}
}

func TestExpiredStorageAccessRenewsBeforeOneUpload(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var uploads, renewals atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == AuthRefreshPath:
					renewals.Add(1)
					respondToRefresh(t, w, r)
				case strings.HasPrefix(r.URL.Path, FileUploadPath+"/"):
					uploads.Add(1)
					body, _ := io.ReadAll(r.Body)
					if string(body) != "contents" || r.Header.Get("Authorization") != "Bearer new-access" || renewals.Load() != 1 {
						t.Error("upload must send the original stream once with the renewed access token")
					}
					if status == http.StatusUnauthorized {
						sessionJSON(w, status, expiredSessionBody)
						return
					}
					sessionJSON(w, status, `{"data":{"remote_path":"user/file","bytes":8,"overwritten":false}}`)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			tokens := originalTokens
			tokens.AccessToken = accessTokenForStorageTest(time.Now().Add(-time.Hour))
			client := testRenewingClient(t, server, tokens)
			result, err := client.UploadFile(t.Context(), "user/file", strings.NewReader("contents"), false)
			if status == http.StatusUnauthorized {
				var failure *RequestError
				if !errors.As(err, &failure) || failure.CraterCode != tokenExpiredCode {
					t.Fatalf("expected original storage error: %v", err)
				}
			} else if err != nil || result.Bytes != 8 {
				t.Fatalf("expected successful upload: result=%+v, err=%v", result, err)
			}
			if uploads.Load() != 1 || renewals.Load() != 1 {
				t.Fatalf("expected one renewal and one upload: renewals=%d uploads=%d", renewals.Load(), uploads.Load())
			}
		})
	}
}

func TestExecRenewsHandshakeAndStartsCommandOnce(t *testing.T) {
	var handshakes, starts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == AuthRefreshPath {
			respondToRefresh(t, w, r)
			return
		}
		handshakes.Add(1)
		switch r.Header.Get("Authorization") {
		case "Bearer old-access":
			sessionJSON(w, 401, expiredSessionBody)
			return
		case "Bearer new-access":
		default:
			t.Error("exec received the wrong credential")
			w.WriteHeader(401)
			return
		}
		upgrader := websocket.Upgrader{Subprotocols: []string{execSubprotocol}}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer ws.Close()
		var frame execFrame
		if err := ws.ReadJSON(&frame); err != nil || frame.Type != "start" || fmt.Sprint(frame.Command) != "[id]" {
			t.Errorf("unexpected start frame: %+v (%v)", frame, err)
			return
		}
		starts.Add(1)
		code := 0
		for _, reply := range []execFrame{execReady(), {Type: "stdout", Data: []byte("alice\n")}, {Type: "exit", ExitCode: &code}} {
			if err := ws.WriteJSON(reply); err != nil {
				t.Error(err)
				return
			}
		}
	}))
	defer server.Close()
	var stdout bytes.Buffer
	result, err := testRenewingClient(t, server, originalTokens).StreamJobExec(t.Context(), "job", JobExecOptions{Command: []string{"id"}}, &stdout, io.Discard)
	if err != nil || result.ExitCode != 0 || stdout.String() != "alice\n" || handshakes.Load() != 2 || starts.Load() != 1 {
		t.Fatalf("exec result=%+v, stdout=%q, err=%v, handshakes=%d, starts=%d", result, &stdout, err, handshakes.Load(), starts.Load())
	}
}
