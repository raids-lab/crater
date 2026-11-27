package vcjob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/extender"
)

func TestQueueSnapshotScopesAndStages(t *testing.T) {
	db := phaseTestDB(t)
	for _, statement := range []string{
		"ALTER TABLE jobs ADD COLUMN attributes jsonb",
		"INSERT INTO accounts (id, name, nickname) VALUES (10,'q-test','Lab'),(20,'other','Other')",
		"INSERT INTO users (id,name,nickname) VALUES (1,'self','Self'),(2,'peer','Peer')",
		`INSERT INTO jobs (id, account_id, user_id, name, job_name, job_type, status, pod_group_phase, creation_timestamp) VALUES
		(1,10,2,'Old','old','pytorch','Pending','Pending','2024-02-01'),
		(2,10,1,'Queued','queued','jupyter','Pending','Inqueue','2024-02-02'),
		(3,10,1,'Starting','starting','custom','Pending','Running','2024-02-03'),
		(4,20,1,'Other','other','custom','','','2024-02-01'),
		(5,10,1,'Aux','aux','tensorboard','Pending','Pending','2024-02-01'),
		(6,10,2,'Legacy','legacy','custom','Prequeue','','2024-02-04'),
		(7,10,1,'New','new','custom','','','2024-02-04'),
		(8,10,1,'Running','running','custom','Running','Running','2024-02-01'),
		(9,10,1,'Deleted','deleted','custom','Pending','Inqueue','2024-02-01')`,
		"UPDATE jobs SET deleted_at='2024-02-02' WHERE id=9",
		`UPDATE jobs SET attributes='{"metadata":{"uid":"old-uid"}}', resources='{"cpu":"4"}' WHERE id=1`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	mgr := &VolcanojobMgr{kubeClient: fake.NewSimpleClientset()}
	for _, tc := range []struct {
		query string
		admin bool
		want  []string
	}{
		{"?accountId=20", false, []string{"old", "queued", "legacy", "new"}},
		{"?mine=true", false, []string{"queued", "new"}},
		{"", true, []string{"old", "other", "queued", "legacy", "new"}},
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/queue"+tc.query, http.NoBody)
		util.SetJWTContext(c, util.JWTMessage{AccountID: 10, UserID: 1})
		if tc.admin {
			mgr.GetAdminQueue(c)
		} else {
			mgr.GetAccountQueue(c)
		}
		var result resputil.Response[QueueSnapshot]
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK || !result.Data.ReasonsAvailable {
			t.Fatalf("response %d: %s", w.Code, w.Body.String())
		}
		var names []string
		for _, job := range result.Data.Jobs {
			names = append(names, job.JobName)
			resources := job.Resources.Data()
			if job.JobName == "old" && (job.Account != "Lab" || job.UserInfo.Username != "peer" || resources.Cpu().String() != "4") {
				t.Fatalf("wrong projection: %+v", job)
			}
		}
		if !reflect.DeepEqual(names, tc.want) {
			t.Fatalf("scope %q admin=%v got %v, want %v", tc.query, tc.admin, names, tc.want)
		}
	}
}

func TestQueueReasonsUseLatestFreshEventForWaitingUID(t *testing.T) {
	ns := config.GetConfig().Namespaces.Job
	now := time.Now()
	event := func(uid, reason string, when time.Time) *v1.Event {
		return &v1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: uid + reason, Namespace: ns},
			InvolvedObject: v1.ObjectReference{Name: "held", UID: types.UID(uid), Kind: "Job", APIVersion: "batch.volcano.sh/v1alpha1"},
			Source:         v1.EventSource{Component: "crater-volcano-extender"}, Type: v1.EventTypeWarning, Reason: reason, LastTimestamp: metav1.NewTime(when),
		}
	}
	stale := event("stale", extender.ReasonQueueQuotaExceeded, now.Add(-time.Hour))
	latest := event("current", extender.ReasonQueueQuotaExceeded, now.Add(-9*time.Minute))
	latest.Series = &v1.EventSeries{LastObservedTime: metav1.NewMicroTime(now)}
	mgr := &VolcanojobMgr{kubeClient: fake.NewSimpleClientset(stale, latest, event("current", extender.ReasonBlockedByTimedOutJob, now.Add(-time.Minute)), event("previous", extender.ReasonExceedsQueueCapability, now))}
	rows := []QueueJob{
		{JobUID: "current", JobName: "held", Phase: "Pending"},
		{JobUID: "stale", JobName: "held", Phase: "Init"},
		{JobUID: "previous", JobName: "held", Phase: "Inqueue"},
		{JobUID: "unrelated", JobName: "held", Phase: "Pending"},
	}
	if err := mgr.addQueueReasons(t.Context(), rows); err != nil {
		t.Fatal(err)
	}
	for i, j := range rows {
		if (i == 0 && j.Reason != extender.ReasonQueueQuotaExceeded) || (i != 0 && j.Reason != "") {
			t.Fatalf("unexpected reason on row %d: %q", i, j.Reason)
		}
	}
}
