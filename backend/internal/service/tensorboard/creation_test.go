/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tensorboard

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
)

func setupTensorboardCreationTest(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test database connection: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	// Only the personal-storage lookup needs a database in this creation flow.
	user := struct {
		gorm.Model
		Name  string
		Space string
	}{Model: gorm.Model{ID: 42}, Name: "alice", Space: "alice-space"}
	if err := db.Table("users").AutoMigrate(&user); err != nil {
		t.Fatalf("create test user table: %v", err)
	}
	if err := db.Table("users").Create(&user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}

	// This test is intentionally sequential because the generated query is global.
	originalUser := query.User
	testQuery := query.Use(db)
	query.User = &testQuery.User
	t.Cleanup(func() { query.User = originalUser })

	cfg := config.GetConfig()
	originalConfig := *cfg
	t.Cleanup(func() { *cfg = originalConfig })
	cfg.Tensorboard.Image = "tensorboard:test"
	cfg.Tensorboard.ImagePullPolicy = "IfNotPresent"
	cfg.Storage.PVC.ReadWriteMany = "creation-test-pvc"
	cfg.Storage.PVC.ReadOnlyMany = nil
}

const tensorboardCreationJobKind = "Job"

type tensorboardCreationCase struct {
	name       string
	failKind   string
	cancel     bool
	deleteFail bool
}

type tensorboardCreationClient struct {
	client.Client
	t           *testing.T
	test        tensorboardCreationCase
	cancel      context.CancelFunc
	creationErr error
	deleteCalls int
	createdJob  *batch.Job
}

func (cli *tensorboardCreationClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	kind := ""
	switch obj.(type) {
	case *batch.Job:
		kind = tensorboardCreationJobKind
	case *corev1.Service:
		kind = "Service"
	case *networkingv1.Ingress:
		kind = "Ingress"
	}
	if kind == cli.test.failKind {
		if cli.test.cancel {
			cli.cancel()
			cli.creationErr = ctx.Err()
		}
		return cli.creationErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if job, ok := obj.(*batch.Job); ok {
		job.UID = "test-job-uid"
		cli.createdJob = job.DeepCopy()
	}
	return cli.Client.Create(ctx, obj, opts...)
}

func (cli *tensorboardCreationClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	cli.deleteCalls++
	if err := ctx.Err(); err != nil {
		cli.t.Errorf("rollback used a canceled context: %v", err)
		return err
	}
	job, ok := obj.(*batch.Job)
	if !ok || cli.createdJob == nil || job.Name != cli.createdJob.Name || job.UID != cli.createdJob.UID {
		cli.t.Fatalf("rollback targeted an unexpected resource: %#v", obj)
	}
	if cli.test.deleteFail {
		return bizerr.Internal.K8sServiceError.New("injected deletion failure")
	}
	return cli.Client.Delete(ctx, obj, opts...)
}

func assertTensorboardCreationResult(
	t *testing.T,
	cli *tensorboardCreationClient,
	result *payload.CreateTensorboardResp,
	err error,
) {
	t.Helper()
	if cli.test.failKind == "" {
		if err != nil || result == nil || result.Name != "Training comparison" || result.AccessPath == "" {
			t.Fatalf("unexpected creation result: %#v, error: %v", result, err)
		}
		return
	}
	assertTensorboardErrorCode(t, err, bizerr.Internal.K8sServiceError)
	if result != nil || !errors.Is(err, cli.creationErr) {
		t.Fatalf("original creation error was not preserved: result: %#v, error: %v", result, err)
	}
}

func assertTensorboardRollbackJobs(t *testing.T, cli *tensorboardCreationClient) {
	t.Helper()
	wantDeletes := 0
	if cli.test.failKind != "" && cli.test.failKind != tensorboardCreationJobKind {
		wantDeletes = 1
	}
	if cli.deleteCalls != wantDeletes {
		t.Fatalf("rollback calls = %d, want %d", cli.deleteCalls, wantDeletes)
	}
	var jobs batch.JobList
	if err := cli.List(t.Context(), &jobs); err != nil {
		t.Fatalf("list test jobs: %v", err)
	}
	wantJobs := 0
	if cli.test.failKind == "" || cli.test.deleteFail {
		wantJobs = 1
	}
	if len(jobs.Items) != wantJobs {
		t.Fatalf("remaining Jobs = %d, want %d", len(jobs.Items), wantJobs)
	}
}

func assertTensorboardServiceOwners(t *testing.T, cli *tensorboardCreationClient) {
	t.Helper()
	// The fake client does not run GC; verify the ownership needed for real cascading deletion.
	var services corev1.ServiceList
	if err := cli.List(t.Context(), &services); err != nil {
		t.Fatalf("list test services: %v", err)
	}
	for i := range services.Items {
		service := &services.Items[i]
		if len(service.OwnerReferences) != 1 || cli.createdJob == nil ||
			service.OwnerReferences[0].UID != cli.createdJob.UID || service.OwnerReferences[0].Kind != tensorboardCreationJobKind {
			t.Fatalf("Service does not belong to the created Job: %#v", service.OwnerReferences)
		}
	}
}

func TestTensorboardCreateRollback(t *testing.T) {
	setupTensorboardCreationTest(t)
	tests := []tensorboardCreationCase{
		{name: "success"},
		{name: "Job creation failure", failKind: tensorboardCreationJobKind},
		{name: "Service creation failure", failKind: "Service"},
		{name: "Ingress creation failure", failKind: "Ingress"},
		{name: "request canceled during Service creation", failKind: "Service", cancel: true},
		{name: "request canceled during Ingress creation", failKind: "Ingress", cancel: true},
		{name: "rollback failure preserves creation error", failKind: "Ingress", deleteFail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			crClient := &tensorboardCreationClient{
				Client: newTensorboardTestClient(t), t: t, test: tt, cancel: cancel,
				creationErr: bizerr.Internal.K8sServiceError.New("injected creation failure"),
			}
			if err := networkingv1.AddToScheme(crClient.Scheme()); err != nil {
				t.Fatalf("register Ingress resources: %v", err)
			}
			svc := NewTensorboardService(crClient, crclient.NewServiceManager(crClient, nil))
			result, err := svc.Create(ctx, interutil.JWTMessage{
				Username: "alice", UserID: 42, AccountID: model.DefaultAccountID,
			}, &payload.CreateTensorboardReq{Name: "Training comparison", LogDir: "/home/alice/logs"})
			assertTensorboardCreationResult(t, crClient, result, err)
			assertTensorboardRollbackJobs(t, crClient)
			assertTensorboardServiceOwners(t, crClient)
		})
	}
}
