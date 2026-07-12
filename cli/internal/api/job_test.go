package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/imroc/req/v3"
)

func jobTestClient(t *testing.T, handler http.HandlerFunc) *Client {
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

func writeJobTestResponse(t *testing.T, w http.ResponseWriter, data interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": data,
		"msg":  "",
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func decodeJobTestBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	return body
}

func TestJobClientAdminListRoute(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/admin/vcjobs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("days"); got != "-1" {
			t.Errorf("days = %q, want -1", got)
		}
		writeJobTestResponse(t, w, Page[JobInfo]{})
	})

	page, err := client.ListJobs(JobListOptions{Admin: true, Days: -1})
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("jobs = %#v, want empty", page.Items)
	}
}

func TestJobClientDetailDecodesTerminatedStates(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/vcjobs/job-1/detail" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		writeJobTestResponse(t, w, map[string]interface{}{
			"name":             "demo",
			"jobName":          "job-1",
			"terminatedStates": []map[string]interface{}{{"exitCode": 1}},
		})
	})

	job, err := client.GetJob("job-1")
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	if len(job.TerminatedStates) != 1 {
		t.Fatalf("terminatedStates = %#v", job.TerminatedStates)
	}
}

func TestJobClientCreateUsesBackendVolumeMountDTO(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/vcjobs/jupyter" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		body := decodeJobTestBody(t, r)
		if _, ok := body["datasetMounts"]; ok {
			t.Error("request contains unused datasetMounts field")
		}
		mounts, ok := body["volumeMounts"].([]interface{})
		if !ok || len(mounts) != 1 {
			t.Fatalf("volumeMounts = %#v", body["volumeMounts"])
		}
		mount, ok := mounts[0].(map[string]interface{})
		if !ok || mount["type"] != float64(2) || mount["datasetID"] != float64(7) {
			t.Fatalf("dataset volume = %#v", mounts[0])
		}
		writeJobTestResponse(t, w, map[string]interface{}{
			"metadata": map[string]interface{}{"name": "jpt-user-test"},
		})
	})

	data, err := client.CreateJupyterJob(CreateInteractiveJobRequest{
		JobCommonRequest: JobCommonRequest{
			Name: "demo",
			VolumeMounts: []VolumeMount{{
				Type:      2,
				DatasetID: 7,
				MountPath: "/data",
			}},
		},
		Resource: ResourceList{"cpu": "1", "memory": "1Gi"},
		Image:    ImageBaseInfo{ImageLink: "example/image:latest"},
	})
	if err != nil {
		t.Fatalf("CreateJupyterJob: %v", err)
	}
	metadata, ok := data["metadata"].(map[string]interface{})
	if !ok || metadata["name"] != "jpt-user-test" {
		t.Fatalf("data = %#v", data)
	}
}

func TestJobClientCleanupContracts(t *testing.T) {
	t.Run("waiting request keeps backend typo and normalizes arrays", func(t *testing.T) {
		client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/operations/clean/clean-waiting-jupyter-job" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			body := decodeJobTestBody(t, r)
			if body["waitMinitues"] != float64(30) {
				t.Errorf("waitMinitues = %#v", body["waitMinitues"])
			}
			writeJobTestResponse(t, w, map[string]interface{}{"deleted": []string{"job-1"}})
		})

		result, err := client.AdminCleanWaitingJupyter(30)
		if err != nil {
			t.Fatalf("AdminCleanWaitingJupyter: %v", err)
		}
		if result.Reminded == nil || len(result.Reminded) != 0 {
			t.Fatalf("reminded = %#v, want non-nil empty slice", result.Reminded)
		}
		if len(result.Deleted) != 1 || result.Deleted[0] != "job-1" {
			t.Fatalf("deleted = %#v", result.Deleted)
		}
	})

	t.Run("low gpu request includes required wait time", func(t *testing.T) {
		client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/operations/clean/clean-low-gpu-usage-job" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			body := decodeJobTestBody(t, r)
			if body["timeRange"] != float64(90) || body["waitTime"] != float64(30) || body["util"] != float64(10) {
				t.Errorf("body = %#v", body)
			}
			writeJobTestResponse(t, w, map[string]interface{}{"reminded": []string{}, "deleted": []string{}})
		})

		_, err := client.AdminCleanLowGPUUsage(CleanLowGPUUsageRequest{
			TimeRange: 90,
			WaitTime:  30,
			Util:      10,
		})
		if err != nil {
			t.Fatalf("AdminCleanLowGPUUsage: %v", err)
		}
	})
}

func TestJobClientDeleteAcceptsNullSuccessData(t *testing.T) {
	client := jobTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/vcjobs/job-1" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		writeJobTestResponse(t, w, nil)
	})

	message, err := client.DeleteJob("job-1")
	if err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
	if message != "" {
		t.Fatalf("message = %q, want empty backend message", message)
	}
}

func TestListJobsSendsPagingAndServerFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != VCJobListPath+"/all" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("page") != "2" || query.Get("page_size") != "25" || query.Get("sort") != "-createdAt" {
			t.Fatalf("unexpected paging query: %v", query)
		}
		if query.Get("days") != "14" || query.Get("status") != "Running" || query.Get("node") != "gpu-01" {
			t.Fatalf("unexpected filters: %v", query)
		}
		if !reflect.DeepEqual(query["job_type"], []string{"jupyter", "webide"}) {
			t.Fatalf("unexpected job types: %v", query["job_type"])
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[Page[JobInfo]]{
			Data: Page[JobInfo]{Items: []JobInfo{{Name: "job"}}, Total: 1, Page: 2, PageSize: 25},
		})
	}))
	defer server.Close()

	page, err := NewClient(server.URL).ListJobs(JobListOptions{
		ListOptions: ListOptions{Page: 2, PageSize: 25, Sort: "-createdAt"},
		All:         true,
		Days:        14,
		Status:      "Running",
		Node:        "gpu-01",
		Interactive: true,
	})
	if err != nil {
		t.Fatalf("ListJobs returned error: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestFetchAllJobPagesSequentially(t *testing.T) {
	requested := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		page := request.URL.Query().Get("page")
		requested = append(requested, page)
		items := map[string][]JobInfo{
			"1": {{Name: "job-1"}, {Name: "job-2"}},
			"2": {{Name: "job-3"}},
		}[page]
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(Response[Page[JobInfo]]{
			Data: Page[JobInfo]{Items: items, Total: 3, PageSize: 2},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	items, err := FetchAllPages(ListOptions{PageSize: 2}, func(options ListOptions) (Page[JobInfo], error) {
		return client.ListJobs(JobListOptions{ListOptions: options})
	})
	if err != nil {
		t.Fatalf("FetchAllPages returned error: %v", err)
	}
	if !reflect.DeepEqual(requested, []string{"1", "2"}) {
		t.Fatalf("unexpected request order: %v", requested)
	}
	if len(items) != 3 {
		t.Fatalf("unexpected items: %#v", items)
	}
}
