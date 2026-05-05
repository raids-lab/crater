package patrol

import (
	"context"
	"testing"

	"gorm.io/datatypes"
)

type stubAdminOpsService struct {
	adminReqs   []TriggerAdminOpsReportRequest
	storageReqs []TriggerStorageAuditRequest
}

func (s *stubAdminOpsService) TriggerAdminOpsReport(_ context.Context, req TriggerAdminOpsReportRequest) (map[string]any, error) {
	s.adminReqs = append(s.adminReqs, req)
	return map[string]any{"status": "ok"}, nil
}

func (s *stubAdminOpsService) TriggerStorageAudit(_ context.Context, req TriggerStorageAuditRequest) (map[string]any, error) {
	s.storageReqs = append(s.storageReqs, req)
	return map[string]any{"status": "ok"}, nil
}

func TestGetPatrolFuncStorageAuditUsesDedicatedService(t *testing.T) {
	service := &stubAdminOpsService{}
	clients := &Clients{AdminOpsService: service}

	f, err := GetPatrolFunc(TRIGGER_STORAGE_DAILY_AUDIT_JOB, clients, datatypes.JSON([]byte(`{}`)))
	if err != nil {
		t.Fatalf("GetPatrolFunc() error = %v", err)
	}
	if _, err := f(context.Background()); err != nil {
		t.Fatalf("patrol func returned error = %v", err)
	}

	if len(service.adminReqs) != 0 {
		t.Fatalf("expected no admin ops report call, got %d", len(service.adminReqs))
	}
	if len(service.storageReqs) != 1 {
		t.Fatalf("expected one storage audit call, got %d", len(service.storageReqs))
	}
	if service.storageReqs[0].Days != 1 || service.storageReqs[0].PVCLimit != 200 {
		t.Fatalf("unexpected storage audit defaults: %+v", service.storageReqs[0])
	}
}

func TestGetPatrolFuncStorageAuditReadsPVCConfig(t *testing.T) {
	service := &stubAdminOpsService{}
	clients := &Clients{AdminOpsService: service}

	f, err := GetPatrolFunc(
		TRIGGER_STORAGE_DAILY_AUDIT_JOB,
		clients,
		datatypes.JSON([]byte(`{"days":2,"pvc_limit":64,"dry_run":true}`)),
	)
	if err != nil {
		t.Fatalf("GetPatrolFunc() error = %v", err)
	}
	if _, err := f(context.Background()); err != nil {
		t.Fatalf("patrol func returned error = %v", err)
	}

	if len(service.storageReqs) != 1 {
		t.Fatalf("expected one storage audit call, got %d", len(service.storageReqs))
	}
	req := service.storageReqs[0]
	if req.Days != 2 || req.PVCLimit != 64 || !req.DryRun {
		t.Fatalf("unexpected storage audit request: %+v", req)
	}
}

func TestGetPatrolFuncAdminOpsReportReadsNotificationPolicy(t *testing.T) {
	service := &stubAdminOpsService{}
	clients := &Clients{AdminOpsService: service}

	f, err := GetPatrolFunc(
		TRIGGER_ADMIN_OPS_REPORT_JOB,
		clients,
		datatypes.JSON([]byte(`{
			"days":1,
			"notification":{
				"enabled":true,
				"notify_admins":true,
				"notify_job_owners":true
			}
		}`)),
	)
	if err != nil {
		t.Fatalf("GetPatrolFunc() error = %v", err)
	}
	if _, err := f(context.Background()); err != nil {
		t.Fatalf("patrol func returned error = %v", err)
	}

	if len(service.adminReqs) != 1 {
		t.Fatalf("expected one admin ops report call, got %d", len(service.adminReqs))
	}
	policy := service.adminReqs[0].Notification
	if policy == nil || !policy.Enabled || !policy.NotifyAdmins || !policy.NotifyJobOwners {
		t.Fatalf("expected notification policy to be preserved, got %+v", policy)
	}
	if policy.FailureJobThreshold != 10 || policy.CooldownHours != 12 || policy.MaxJobOwnerEmails != 10 {
		t.Fatalf("expected notification defaults to be normalized, got %+v", policy)
	}
}
