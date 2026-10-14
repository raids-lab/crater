package aijob

import (
	"testing"
	"time"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/dao/model"
	aijobapi "github.com/raids-lab/crater/pkg/apis/aijob/v1alpha1"
)

func TestResolveAIJobNamePrefersPersistedName(t *testing.T) {
	task := &model.AITask{Model: gorm.Model{ID: 42}, Namespace: "workspace", JobName: "persisted-job"}
	client := fake.NewClientBuilder().Build()

	got, err := resolveAIJobName(t.Context(), client, task)
	if err != nil {
		t.Fatalf("resolve persisted job name: %v", err)
	}
	if got != "persisted-job" {
		t.Fatalf("resolved name = %q, want persisted-job", got)
	}
}

func TestResolveAIJobNameRecoversOldestTaskResource(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := aijobapi.AddToScheme(scheme); err != nil {
		t.Fatalf("add AIJob scheme: %v", err)
	}
	created := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	older := &aijobapi.AIJob{ObjectMeta: metav1.ObjectMeta{
		Name: "older-job", Namespace: "workspace", CreationTimestamp: metav1.Time{Time: created},
		Labels: map[string]string{aijobapi.LabelKeyTaskID: "42"},
	}}
	newer := &aijobapi.AIJob{ObjectMeta: metav1.ObjectMeta{
		Name: "newer-job", Namespace: "workspace", CreationTimestamp: metav1.Time{Time: created.Add(time.Hour)},
		Labels: map[string]string{aijobapi.LabelKeyTaskID: "42"},
	}}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newer, older).Build()

	got, err := resolveAIJobName(t.Context(), client, &model.AITask{
		Model: gorm.Model{ID: 42}, Namespace: "workspace",
	})
	if err != nil {
		t.Fatalf("resolve historical job name: %v", err)
	}
	if got != "older-job" {
		t.Fatalf("resolved name = %q, want oldest matching resource", got)
	}
}

func TestResolveAIJobNameReportsMissingResource(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := aijobapi.AddToScheme(scheme); err != nil {
		t.Fatalf("add AIJob scheme: %v", err)
	}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	_, err := resolveAIJobName(t.Context(), client, &model.AITask{
		Model: gorm.Model{ID: 42}, Namespace: "workspace",
	})
	if err == nil {
		t.Fatal("resolve missing resource returned nil error")
	}
}
