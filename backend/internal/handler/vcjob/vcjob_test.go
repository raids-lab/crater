package vcjob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

func TestResolveDeleteSettlementTime(t *testing.T) {
	t.Parallel()

	recordCompletedAt := time.Date(2026, 4, 15, 9, 30, 0, 0, time.UTC)
	jobTransitionAt := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		record *model.Job
		job    *batch.Job
		check  func(time.Time) bool
	}{
		{
			name:   "prefers record completed timestamp",
			record: &model.Job{CompletedTimestamp: recordCompletedAt},
			job: &batch.Job{Status: batch.JobStatus{
				State: batch.JobState{LastTransitionTime: metav1.NewTime(jobTransitionAt)},
			}},
			check: func(got time.Time) bool {
				return got.Equal(recordCompletedAt)
			},
		},
		{
			name:   "falls back to job transition timestamp",
			record: &model.Job{},
			job: &batch.Job{Status: batch.JobStatus{
				State: batch.JobState{LastTransitionTime: metav1.NewTime(jobTransitionAt)},
			}},
			check: func(got time.Time) bool {
				return got.Equal(jobTransitionAt)
			},
		},
		{
			name: "falls back to current time when timestamps missing",
			check: func(got time.Time) bool {
				return !got.IsZero()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveDeleteSettlementTime(tc.record, tc.job)
			if !tc.check(got) {
				t.Fatalf("resolveDeleteSettlementTime() = %v", got)
			}
		})
	}
}

func TestJobDetailLookupErrors(t *testing.T) {
	const adminPath = "/api/v1/admin/vcjobs/missing/detail"
	const userPath = "/api/v1/vcjobs/missing/detail"
	const legacyUnspecifiedCode = 99999

	db := setupRegularJobListDB(t)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		util.SetJWTContext(c, util.JWTMessage{RolePlatform: model.RoleAdmin})
		c.Next()
	})
	mgr := &VolcanojobMgr{}
	mgr.RegisterAdmin(router.Group("/api/v1/admin/vcjobs"))
	mgr.RegisterProtected(router.Group("/api/v1/vcjobs"))

	tests := []struct {
		name        string
		path        string
		breakLookup bool
		wantStatus  int
		wantCode    bizerr.BizCode
		wantMessage string
	}{
		{"admin missing job", adminPath, false, http.StatusNotFound, bizerr.NotFound.DataBaseNotFound, "job not found"},
		{"protected missing job", userPath, false, http.StatusInternalServerError, legacyUnspecifiedCode, "record not found"},
		{"admin database failure", adminPath, true, http.StatusInternalServerError, bizerr.Internal.DatabaseError, "failed to load job details"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.breakLookup {
				if err := db.Migrator().DropTable(&model.Job{}); err != nil {
					t.Fatal(err)
				}
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, http.NoBody))
			if recorder.Code != tc.wantStatus {
				t.Fatalf("HTTP status = %d, want %d: %s", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
			var body resputil.Response[any]
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tc.wantCode || body.Message != tc.wantMessage {
				t.Errorf("response = (%d, %q), want (%d, %q)", body.Code, body.Message, tc.wantCode, tc.wantMessage)
			}
		})
	}
}
