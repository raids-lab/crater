package service

import (
	"testing"

	"github.com/raids-lab/crater/dao/model"
	vcjobservice "github.com/raids-lab/crater/internal/service/vcjob"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

func TestLiveJobReservationsCoverObserverLagWithoutDoubleCounting(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	jobs := []client.Object{}
	for _, name := range []string{"observed", "unobserved", "foreign", "completed", "backfill"} {
		job := &batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "jobs",
				Annotations: map[string]string{
					"crater.raids.io/user-id":              "1",
					"crater.raids.io/account-id":           "2",
					vcjobservice.AnnotationKeyScheduleType: "normal",
				},
			},
		}
		switch name {
		case "foreign":
			job.Annotations["crater.raids.io/account-id"] = "3"
		case "completed":
			job.Status.State.Phase = batch.Completed
		case "backfill":
			job.Annotations[vcjobservice.AnnotationKeyScheduleType] = "backfill"
		}
		jobs = append(jobs, job)
	}
	s := &PrequeueService{
		servingClient:    fake.NewClientBuilder().WithScheme(scheme).WithObjects(jobs...).Build(),
		servingNamespace: "jobs",
	}
	records := []*model.Job{{JobName: "observed"}}
	reserved, err := s.liveJobReservations(t.Context(), 1, 2, records, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reserved) != 2 {
		t.Fatalf("quota reservations=%d, want DB + unobserved normal job", len(reserved))
	}
	all, err := s.liveJobReservations(t.Context(), 1, 2, records, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("workload count=%d, want normal + backfill without foreign/completed", len(all))
	}
}
