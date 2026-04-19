package prequeuewatcher

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/crclient"
)

func TestClaimAndActivatePrequeueJobKeepsPendingOnSuccess(t *testing.T) {
	w, db := newTestWatcher(t)
	record := createPrequeueJob(t, db, "job-success")

	activated, err := w.claimAndActivatePrequeueJob(context.Background(), record)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !activated {
		t.Fatalf("expected activation")
	}
	assertJobStatus(t, w.q, record.ID, batch.Pending)
	assertVolcanoJobExists(t, w, "job-success")
}

func TestClaimAndActivatePrequeueJobRollsBackOnFailure(t *testing.T) {
	w, db := newTestWatcher(t)
	record := createPrequeueJobWithUserID(t, db, "job-failed", "invalid")

	activated, err := w.claimAndActivatePrequeueJob(context.Background(), record)
	if err == nil || !strings.Contains(err.Error(), "invalid user id annotation") {
		t.Fatalf("expected invalid user id error, got %v", err)
	}
	if activated {
		t.Fatalf("expected no activation")
	}
	assertJobStatus(t, w.q, record.ID, model.Prequeue)
	assertVolcanoJobMissing(t, w, "job-failed")
}

func TestClaimAndActivatePrequeueJobSkipsUnclaimedRecord(t *testing.T) {
	w, db := newTestWatcher(t)
	record := createJobWithStatus(t, db, "job-running", batch.Running)

	activated, err := w.claimAndActivatePrequeueJob(context.Background(), record)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if activated {
		t.Fatalf("expected no activation")
	}
	assertJobStatus(t, w.q, record.ID, batch.Running)
	assertVolcanoJobMissing(t, w, "job-running")
}

func TestClaimAndActivatePrequeueJobKeepsPendingOnAlreadyExists(t *testing.T) {
	w, db := newTestWatcher(t)
	record := createPrequeueJob(t, db, "job-existing")
	if err := w.k8sClient.Create(context.Background(), newTestVolcanoJob("job-existing", "1")); err != nil {
		t.Fatalf("create existing volcano job: %v", err)
	}

	activated, err := w.claimAndActivatePrequeueJob(context.Background(), record)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !activated {
		t.Fatalf("expected activation")
	}
	assertJobStatus(t, w.q, record.ID, batch.Pending)
	assertVolcanoJobExists(t, w, "job-existing")
}

func newTestWatcher(t *testing.T) (*PrequeueWatcher, *gorm.DB) {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Migrator().CreateTable(&testJobRow{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	scheme := runtime.NewScheme()
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatalf("add volcano scheme: %v", err)
	}
	return &PrequeueWatcher{
		q:         query.Use(db),
		k8sClient: fake.NewClientBuilder().WithScheme(scheme).Build(),
	}, db
}

type testJobRow struct {
	ID                uint `gorm:"primaryKey"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt
	Name              string
	JobName           string
	UserID            uint
	AccountID         uint
	JobType           model.JobType
	ScheduleType      *model.ScheduleType
	Status            batch.JobPhase
	Queue             string
	CreationTimestamp time.Time
	Resources         datatypes.JSONType[v1.ResourceList]
	Attributes        datatypes.JSONType[*batch.Job]
}

func (testJobRow) TableName() string {
	return "jobs"
}

func createPrequeueJob(t *testing.T, db *gorm.DB, name string) *model.Job {
	t.Helper()
	return createJobWithStatus(t, db, name, model.Prequeue)
}

func createPrequeueJobWithUserID(t *testing.T, db *gorm.DB, name, userID string) *model.Job {
	t.Helper()
	return createJobWithStatusAndUserID(t, db, name, model.Prequeue, userID)
}

func createJobWithStatus(t *testing.T, db *gorm.DB, name string, status batch.JobPhase) *model.Job {
	t.Helper()
	return createJobWithStatusAndUserID(t, db, name, status, "1")
}

func createJobWithStatusAndUserID(
	t *testing.T,
	db *gorm.DB,
	name string,
	status batch.JobPhase,
	userID string,
) *model.Job {
	t.Helper()

	scheduleType := model.ScheduleTypeNormal
	row := &testJobRow{
		Name:              name,
		JobName:           name,
		UserID:            1,
		AccountID:         1,
		JobType:           model.JobTypeCustom,
		ScheduleType:      ptr.To(scheduleType),
		Status:            status,
		Queue:             "default",
		CreationTimestamp: time.Now(),
		Resources: datatypes.NewJSONType(v1.ResourceList{
			v1.ResourceCPU: resource.MustParse("1"),
		}),
		Attributes: datatypes.NewJSONType(newTestVolcanoJob(name, userID)),
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	return &model.Job{
		Model:      gorm.Model{ID: row.ID},
		Attributes: row.Attributes,
	}
}

func newTestVolcanoJob(name, userID string) *batch.Job {
	return &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				crclient.LabelKeyTaskType: string(crclient.CraterJobTypeCustom),
				crclient.LabelKeyTaskUser: "test-user",
				crclient.LabelKeyBaseURL:  "test-user",
			},
			Annotations: map[string]string{
				"crater.raids.io/user-id": userID,
			},
		},
	}
}

func assertJobStatus(t *testing.T, q *query.Query, id uint, want batch.JobPhase) {
	t.Helper()

	record, err := q.Job.WithContext(context.Background()).Where(q.Job.ID.Eq(id)).First()
	if err != nil {
		t.Fatalf("load job: %v", err)
	}
	if record.Status != want {
		t.Fatalf("expected status %s, got %s", want, record.Status)
	}
}

func assertVolcanoJobExists(t *testing.T, w *PrequeueWatcher, name string) {
	t.Helper()

	job := &batch.Job{}
	err := w.k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, job)
	if err != nil {
		t.Fatalf("expected volcano job %s to exist: %v", name, err)
	}
}

func assertVolcanoJobMissing(t *testing.T, w *PrequeueWatcher, name string) {
	t.Helper()

	job := &batch.Job{}
	err := w.k8sClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, job)
	if err == nil {
		t.Fatalf("expected volcano job %s to be missing", name)
	}
}
