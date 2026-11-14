package service

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
)

func TestLiveJobReservationsCoverObserverLagWithoutDoubleCounting(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := scheduling.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	jobs := []client.Object{}
	for _, name := range []string{"observed", "unobserved", "foreign", "completed", "unadmitted"} {
		job := &batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "jobs",
				UID:       types.UID(name),
				Annotations: map[string]string{
					"crater.raids.io/user-id":    "1",
					"crater.raids.io/account-id": "2",
				},
			},
		}
		switch name {
		case "foreign":
			job.Annotations["crater.raids.io/account-id"] = "3"
		case "completed":
			job.Status.State.Phase = batch.Completed
		case "unobserved":
			jobs = append(jobs, &scheduling.PodGroup{
				ObjectMeta: metav1.ObjectMeta{Name: name + "-pg", Namespace: "jobs",
					OwnerReferences: []metav1.OwnerReference{{Kind: "Job", APIVersion: batch.SchemeGroupVersion.String(),
						Name: name, UID: types.UID(name), Controller: ptr.To(true)}}},
				Status: scheduling.PodGroupStatus{Phase: scheduling.PodGroupInqueue},
			})
		}
		jobs = append(jobs, job)
	}
	s := &QueueQuotaService{
		servingClient:    fake.NewClientBuilder().WithScheme(scheme).WithObjects(jobs...).Build(),
		servingNamespace: "jobs",
	}
	records := []*model.Job{{JobName: "observed"}}
	reserved, err := s.liveJobReservations(t.Context(), 1, 2, records, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(reserved) != 2 {
		t.Fatalf("quota reservations=%d, want DB + unobserved admitted job", len(reserved))
	}
	all, err := s.liveJobReservations(t.Context(), 1, 2, records, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("workload count=%d, want admitted + unadmitted without foreign/completed", len(all))
	}
}
