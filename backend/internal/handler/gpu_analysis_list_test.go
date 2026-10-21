package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindGpuAnalysisListPageQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
		wantSort string
		wantStat []uint8
		wantRisk []string
		wantErr  bool
	}{
		{name: "defaults", query: "", wantPage: 1, wantSize: 10, wantSort: "-Phase2Score,-CreatedAt"},
		{
			name:     "normalizes filters",
			query:    "page=2&page_size=20&search=job&type=ignored&ReviewStatus=1&ReviewStatus=3&Phase2Score=high&Phase2Score=low&sort=-CreatedAt,JobName",
			wantPage: 2,
			wantSize: 20,
			wantSort: "-CreatedAt,JobName",
			wantStat: []uint8{1, 3},
			wantRisk: []string{"high", "low"},
		},
		{name: "rejects status", query: "ReviewStatus=4", wantErr: true},
		{name: "rejects risk", query: "Phase2Score=critical", wantErr: true},
		{name: "rejects sort", query: "sort=Nodes", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/gpu-analysis?"+tt.query, http.NoBody)

			got, statuses, risks, err := bindGpuAnalysisListPageQuery(context)
			if (err != nil) != tt.wantErr {
				t.Fatalf("bindGpuAnalysisListPageQuery() error = %v, wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Page != tt.wantPage || got.PageSize != tt.wantSize || got.Sort != tt.wantSort {
				t.Fatalf("unexpected query: %#v", got)
			}
			if strings.Join(intsToStrings(statuses), ",") != strings.Join(intsToStrings(tt.wantStat), ",") {
				t.Fatalf("statuses = %v, want %v", statuses, tt.wantStat)
			}
			if strings.Join(risks, ",") != strings.Join(tt.wantRisk, ",") {
				t.Fatalf("risks = %v, want %v", risks, tt.wantRisk)
			}
		})
	}
}

func TestGpuAnalysisSortClauses(t *testing.T) {
	got := gpuAnalysisSortClauses("-Phase2Score,JobName")
	want := []string{"gpu_analyses.phase2_score DESC", "gpu_analyses.job_name ASC", "gpu_analyses.id DESC"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("gpuAnalysisSortClauses() = %v, want %v", got, want)
	}
}

func intsToStrings(values []uint8) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strconv.Itoa(int(value))
	}
	return result
}
