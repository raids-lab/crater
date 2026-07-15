package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imroc/req/v3"
)

func imageTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	client := NewClient("https://example.invalid")
	client.httpClient.GetTransport().WrapRoundTripFunc(func(_ http.RoundTripper) req.HttpRoundTripFunc {
		return func(r *http.Request) (*http.Response, error) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, r)
			return recorder.Result(), nil
		}
	})
	return client
}

func writeImageTestResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": nil,
		"msg":  "",
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestImageClientRoutesMatchBackend(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		query  string
		call   func(*Client) error
	}{
		{"list user builds", http.MethodGet, "/api/v1/images/kaniko", "", func(c *Client) error { _, err := c.ListKaniko(false); return err }},
		{"list admin builds", http.MethodGet, "/api/v1/admin/images/kaniko", "", func(c *Client) error { _, err := c.ListKaniko(true); return err }},
		{"get build", http.MethodGet, "/api/v1/images/getbyname", "name=build-1", func(c *Client) error { _, err := c.GetKanikoByName("build-1"); return err }},
		{"get build template", http.MethodGet, "/api/v1/images/template", "name=build-1", func(c *Client) error { _, err := c.GetKanikoTemplateByName("build-1"); return err }},
		{"get build pod", http.MethodGet, "/api/v1/images/podname", "id=7", func(c *Client) error { _, err := c.GetKanikoPod(7); return err }},
		{"create pip apt build", http.MethodPost, "/api/v1/images/kaniko", "", func(c *Client) error { _, err := c.CreatePipApt(PipAptBuildRequest{}); return err }},
		{"create dockerfile build", http.MethodPost, "/api/v1/images/dockerfile", "", func(c *Client) error { _, err := c.CreateDockerfile(DockerfileBuildRequest{}); return err }},
		{"create envd build", http.MethodPost, "/api/v1/images/envd", "", func(c *Client) error { _, err := c.CreateEnvd(EnvdBuildRequest{}); return err }},
		{"remove user builds", http.MethodPost, "/api/v1/images/remove", "", func(c *Client) error { _, err := c.RemoveKaniko([]uint{7}, false); return err }},
		{"remove admin builds", http.MethodPost, "/api/v1/admin/images/remove", "", func(c *Client) error { _, err := c.RemoveKaniko([]uint{7}, true); return err }},
		{"list user images", http.MethodGet, "/api/v1/images/image", "", func(c *Client) error { _, err := c.ListImageRecords(false); return err }},
		{"list admin images", http.MethodGet, "/api/v1/admin/images/image", "", func(c *Client) error { _, err := c.ListImageRecords(true); return err }},
		{"list available images", http.MethodGet, "/api/v1/images/available", "", func(c *Client) error { _, err := c.ListAvailableImages(); return err }},
		{"upload image", http.MethodPost, "/api/v1/images/image", "", func(c *Client) error { _, err := c.UploadImage(ImageUploadRequest{}); return err }},
		{"delete image", http.MethodDelete, "/api/v1/images/image/7", "", func(c *Client) error { _, err := c.DeleteImage(7); return err }},
		{"delete user images", http.MethodPost, "/api/v1/images/deleteimage", "", func(c *Client) error { _, err := c.DeleteImages([]uint{7}, false); return err }},
		{"delete admin images", http.MethodPost, "/api/v1/admin/images/deleteimage", "", func(c *Client) error { _, err := c.DeleteImages([]uint{7}, true); return err }},
		{"update user description", http.MethodPost, "/api/v1/images/description", "", func(c *Client) error {
			_, err := c.UpdateImageDescription(ImageDescriptionRequest{}, false)
			return err
		}},
		{"update admin description", http.MethodPost, "/api/v1/admin/images/description", "", func(c *Client) error { _, err := c.UpdateImageDescription(ImageDescriptionRequest{}, true); return err }},
		{"update user type", http.MethodPost, "/api/v1/images/type", "", func(c *Client) error { _, err := c.UpdateImageType(ImageTypeRequest{}, false); return err }},
		{"update admin type", http.MethodPost, "/api/v1/admin/images/type", "", func(c *Client) error { _, err := c.UpdateImageType(ImageTypeRequest{}, true); return err }},
		{"update user tags", http.MethodPost, "/api/v1/images/tags", "", func(c *Client) error { _, err := c.UpdateImageTags(ImageTagsRequest{}, false); return err }},
		{"update admin tags", http.MethodPost, "/api/v1/admin/images/tags", "", func(c *Client) error { _, err := c.UpdateImageTags(ImageTagsRequest{}, true); return err }},
		{"update user arch", http.MethodPost, "/api/v1/images/arch", "", func(c *Client) error { _, err := c.UpdateImageArch(ImageArchRequest{}, false); return err }},
		{"update admin arch", http.MethodPost, "/api/v1/admin/images/arch", "", func(c *Client) error { _, err := c.UpdateImageArch(ImageArchRequest{}, true); return err }},
		{"toggle public", http.MethodPost, "/api/v1/admin/images/change/7", "", func(c *Client) error { _, err := c.TogglePublic(7); return err }},
		{"share image", http.MethodPost, "/api/v1/images/share", "", func(c *Client) error { _, err := c.ShareImage(ImageShareRequest{}); return err }},
		{"cancel share", http.MethodDelete, "/api/v1/images/share", "", func(c *Client) error { _, err := c.CancelShareImage(ImageCancelShareRequest{}); return err }},
		{"list grants", http.MethodGet, "/api/v1/images/share", "imageID=7", func(c *Client) error { _, err := c.GetImageGrants(7); return err }},
		{"list ungranted users", http.MethodGet, "/api/v1/images/user", "imageID=7&name=alice", func(c *Client) error { _, err := c.ListUngrantedUsers(7, "alice"); return err }},
		{"list ungranted accounts", http.MethodGet, "/api/v1/images/account", "imageID=7", func(c *Client) error { _, err := c.ListUngrantedAccounts(7); return err }},
		{"validate links", http.MethodPost, "/api/v1/images/valid", "", func(c *Client) error { _, err := c.CheckImageLinks(nil); return err }},
		{"get harbor", http.MethodGet, "/api/v1/images/harbor", "", func(c *Client) error { _, err := c.GetHarbor(); return err }},
		{"get credential", http.MethodPost, "/api/v1/images/credential", "", func(c *Client) error { _, err := c.GetCredential(); return err }},
		{"get quota", http.MethodGet, "/api/v1/images/quota", "", func(c *Client) error { _, err := c.GetQuota(); return err }},
		{"update quota", http.MethodPost, "/api/v1/images/quota", "", func(c *Client) error { _, err := c.UpdateQuota(1024); return err }},
		{"list cuda images", http.MethodGet, "/api/v1/images/cudabaseimage", "", func(c *Client) error { _, err := c.ListCudaBaseImages(); return err }},
		{"add cuda image", http.MethodPost, "/api/v1/admin/images/cudabaseimage", "", func(c *Client) error { _, err := c.AdminAddCudaBaseImage(CudaBaseImageRequest{}); return err }},
		{"delete cuda image", http.MethodDelete, "/api/v1/admin/images/cudabaseimage/7", "", func(c *Client) error { _, err := c.AdminDeleteCudaBaseImage(7); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := imageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tt.method, tt.path)
				}
				if r.URL.RawQuery != tt.query {
					t.Errorf("query = %q, want %q", r.URL.RawQuery, tt.query)
				}
				writeImageTestResponse(t, w)
			})
			if err := tt.call(client); err != nil {
				t.Fatalf("call failed: %v", err)
			}
		})
	}
}

func TestImageClientShareBodiesMatchBackendDTOs(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
		want map[string]interface{}
	}{
		{
			name: "share",
			call: func(c *Client) error {
				_, err := c.ShareImage(ImageShareRequest{IDList: []uint{2, 3}, ImageID: 7, Type: "account"})
				return err
			},
			want: map[string]interface{}{"idList": []interface{}{float64(2), float64(3)}, "imageID": float64(7), "type": "account"},
		},
		{
			name: "cancel",
			call: func(c *Client) error {
				_, err := c.CancelShareImage(ImageCancelShareRequest{ID: 2, ImageID: 7, Type: "user"})
				return err
			},
			want: map[string]interface{}{"id": float64(2), "imageID": float64(7), "type": "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := imageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				var got map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(tt.want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("body = %s, want %s", gotJSON, wantJSON)
				}
				writeImageTestResponse(t, w)
			})
			if err := tt.call(client); err != nil {
				t.Fatalf("call failed: %v", err)
			}
		})
	}
}
