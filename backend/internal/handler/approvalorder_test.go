package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/utils"
)

func TestApplyAgentEmergencyOrderLocksRunningJobAndKeepsOrderPending(t *testing.T) {
	db := newApprovalOrderTestDB(t)
	mgr := &ApprovalOrderMgr{}
	now := utils.GetLocalTime().Truncate(time.Second)
	job := approvalOrderTestJob{
		JobName:         "job-emergency",
		Status:          batch.Running,
		LockedTimestamp: now,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	order := approvalOrderTestOrder{
		Name:   job.JobName,
		Type:   string(model.ApprovalOrderTypeJob),
		Status: string(model.ApprovalOrderStatusPending),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	err := mgr.applyAgentApprovedOrderWithDB(
		context.Background(),
		db,
		order.ID,
		job.JobName,
		6,
		map[string]any{
			"review_notes": "agent emergency lock applied: 6h; remaining 42h require manual review",
			"agent_report": `{"verdict":"approve_emergency"}`,
		},
	)
	if err != nil {
		t.Fatalf("apply emergency lock: %v", err)
	}

	var gotJob approvalOrderTestJob
	if err := db.First(&gotJob, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if !gotJob.LockedTimestamp.After(now) {
		t.Fatalf("expected emergency lock to extend job, got %s base %s", gotJob.LockedTimestamp, now)
	}
	var gotOrder approvalOrderTestOrder
	if err := db.First(&gotOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if gotOrder.Status != string(model.ApprovalOrderStatusPending) {
		t.Fatalf("expected order to stay pending, got %s", gotOrder.Status)
	}
	if gotOrder.ReviewNotes == "" || gotOrder.AgentReport == "" {
		t.Fatalf("expected emergency suggestion and report to be recorded, got notes=%q report=%q", gotOrder.ReviewNotes, gotOrder.AgentReport)
	}
}

func TestApplyAgentApprovedOrderSkipsNonPendingOrder(t *testing.T) {
	db := newApprovalOrderTestDB(t)
	mgr := &ApprovalOrderMgr{}
	job := approvalOrderTestJob{
		JobName:         "job-manual-approved",
		Status:          batch.Running,
		LockedTimestamp: utils.GetLocalTime(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	order := approvalOrderTestOrder{
		Name:   job.JobName,
		Type:   string(model.ApprovalOrderTypeJob),
		Status: string(model.ApprovalOrderStatusRejected),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	err := mgr.applyAgentApprovedOrderWithDB(context.Background(), db, order.ID, job.JobName, 2, map[string]any{
		"status": string(model.ApprovalOrderStatusApproved),
	})

	if !errors.Is(err, errApprovalOrderNoLongerPending) {
		t.Fatalf("expected no-longer-pending error, got %v", err)
	}
	var gotJob approvalOrderTestJob
	if err := db.First(&gotJob, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if !gotJob.LockedTimestamp.Equal(job.LockedTimestamp) {
		t.Fatalf("expected job lock to remain unchanged, got %s want %s", gotJob.LockedTimestamp, job.LockedTimestamp)
	}
	var gotOrder approvalOrderTestOrder
	if err := db.First(&gotOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if gotOrder.Status != string(model.ApprovalOrderStatusRejected) {
		t.Fatalf("expected manual status to remain rejected, got %s", gotOrder.Status)
	}
}

func TestRecordAgentReportOnlyDoesNotOverwriteManualDecision(t *testing.T) {
	db := newApprovalOrderTestDB(t)
	mgr := &ApprovalOrderMgr{}
	job := approvalOrderTestJob{
		JobName:         "job-manual-report",
		Status:          batch.Running,
		LockedTimestamp: utils.GetLocalTime(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	order := approvalOrderTestOrder{
		Name:        job.JobName,
		Type:        string(model.ApprovalOrderTypeJob),
		Status:      string(model.ApprovalOrderStatusRejected),
		ReviewNotes: "manual reject",
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	rows, err := mgr.recordAgentReportWithDB(context.Background(), db, order.ID, `{"verdict":"approve"}`)
	if err != nil {
		t.Fatalf("record agent report: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected one order report update, got %d", rows)
	}

	var gotJob approvalOrderTestJob
	if err := db.First(&gotJob, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if !gotJob.LockedTimestamp.Equal(job.LockedTimestamp) {
		t.Fatalf("expected job lock to remain unchanged, got %s want %s", gotJob.LockedTimestamp, job.LockedTimestamp)
	}
	var gotOrder approvalOrderTestOrder
	if err := db.First(&gotOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if gotOrder.Status != string(model.ApprovalOrderStatusRejected) {
		t.Fatalf("expected manual status to remain rejected, got %s", gotOrder.Status)
	}
	if gotOrder.ReviewNotes != "manual reject" {
		t.Fatalf("expected manual notes to remain unchanged, got %q", gotOrder.ReviewNotes)
	}
	if gotOrder.AgentReport != `{"verdict":"approve"}` {
		t.Fatalf("expected agent report to be recorded, got %q", gotOrder.AgentReport)
	}
}

func TestApplyAgentApprovedOrderRejectsNonRunningJobInsideTransaction(t *testing.T) {
	db := newApprovalOrderTestDB(t)
	mgr := &ApprovalOrderMgr{}
	job := approvalOrderTestJob{
		JobName:         "job-finished",
		Status:          batch.Failed,
		LockedTimestamp: utils.GetLocalTime(),
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	order := approvalOrderTestOrder{
		Name:   job.JobName,
		Type:   string(model.ApprovalOrderTypeJob),
		Status: string(model.ApprovalOrderStatusPending),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	err := mgr.applyAgentApprovedOrderWithDB(context.Background(), db, order.ID, job.JobName, 2, map[string]any{
		"status": string(model.ApprovalOrderStatusApproved),
	})

	if err == nil || errors.Is(err, errApprovalOrderNoLongerPending) {
		t.Fatalf("expected non-running job error, got %v", err)
	}
	var gotOrder approvalOrderTestOrder
	if err := db.First(&gotOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if gotOrder.Status != string(model.ApprovalOrderStatusPending) {
		t.Fatalf("expected order to stay pending, got %s", gotOrder.Status)
	}
}

func TestApplyAgentApprovedOrderLocksRunningJobAndUpdatesPendingOrder(t *testing.T) {
	db := newApprovalOrderTestDB(t)
	mgr := &ApprovalOrderMgr{}
	now := utils.GetLocalTime().Truncate(time.Second)
	job := approvalOrderTestJob{
		JobName:         "job-running",
		Status:          batch.Running,
		LockedTimestamp: now,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}
	order := approvalOrderTestOrder{
		Name:   job.JobName,
		Type:   string(model.ApprovalOrderTypeJob),
		Status: string(model.ApprovalOrderStatusPending),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	err := mgr.applyAgentApprovedOrderWithDB(context.Background(), db, order.ID, job.JobName, 3, map[string]any{
		"status":        string(model.ApprovalOrderStatusApproved),
		"review_source": string(model.ReviewSourceAgentAuto),
	})

	if err != nil {
		t.Fatalf("apply agent approval: %v", err)
	}
	var gotJob approvalOrderTestJob
	if err := db.First(&gotJob, job.ID).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if !gotJob.LockedTimestamp.After(now) {
		t.Fatalf("expected job lock to be extended, got %s base %s", gotJob.LockedTimestamp, now)
	}
	var gotOrder approvalOrderTestOrder
	if err := db.First(&gotOrder, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if gotOrder.Status != string(model.ApprovalOrderStatusApproved) {
		t.Fatalf("expected order approved, got %s", gotOrder.Status)
	}
	if gotOrder.ReviewSource != string(model.ReviewSourceAgentAuto) {
		t.Fatalf("expected agent review source, got %s", gotOrder.ReviewSource)
	}
}

func newApprovalOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&approvalOrderTestOrder{}, &approvalOrderTestJob{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

type approvalOrderTestOrder struct {
	gorm.Model
	Name         string `gorm:"type:varchar(256)"`
	Type         string `gorm:"type:varchar(32)"`
	Status       string `gorm:"type:varchar(32)"`
	ReviewNotes  string `gorm:"type:varchar(512)"`
	ReviewSource string `gorm:"type:varchar(32)"`
	AgentReport  string `gorm:"type:text"`
}

func (approvalOrderTestOrder) TableName() string {
	return "approval_orders"
}

type approvalOrderTestJob struct {
	gorm.Model
	JobName         string
	Status          batch.JobPhase
	LockedTimestamp time.Time
}

func (approvalOrderTestJob) TableName() string {
	return "jobs"
}
