package tensorboard

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	interutil "github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/crclient"
)

func TestTensorboardGarbageCollectorRequiresLeaderElection(t *testing.T) {
	gc := NewTensorboardGarbageCollector(nil, "jobs")
	if !gc.NeedLeaderElection() {
		t.Fatal("TensorBoard garbage collector must require leader election")
	}
}

func TestCleanExpiredTensorboards(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add apps/v1 to scheme: %v", err)
	}

	now := time.Now()
	objects := []client.Object{
		tensorboardDeploymentWithExpiration("expired", "jobs", now.Add(-time.Minute).Format(time.RFC3339)),
		tensorboardDeploymentWithExpiration("active", "jobs", now.Add(time.Hour).Format(time.RFC3339)),
		tensorboardDeploymentWithExpiration("invalid", "jobs", "not-a-time"),
		tensorboardDeploymentWithExpiration("other-namespace", "other", now.Add(-time.Minute).Format(time.RFC3339)),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-task",
				Namespace: "jobs",
				Labels:    map[string]string{crclient.LabelKeyTaskType: "training"},
				Annotations: map[string]string{
					interutil.AnnotationKeyExpirationTime: now.Add(-time.Minute).Format(time.RFC3339),
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	gc := NewTensorboardGarbageCollector(k8sClient, "jobs")
	gc.cleanExpiredTensorboards(context.Background())

	assertDeploymentExists(t, k8sClient, "jobs", "expired", false)
	assertDeploymentExists(t, k8sClient, "jobs", "active", true)
	assertDeploymentExists(t, k8sClient, "jobs", "invalid", true)
	assertDeploymentExists(t, k8sClient, "other", "other-namespace", true)
	assertDeploymentExists(t, k8sClient, "jobs", "other-task", true)
}

func tensorboardDeploymentWithExpiration(name, namespace, expiration string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				crclient.LabelKeyTaskType: interutil.LabelKeyTypeTensorboard,
			},
			Annotations: map[string]string{
				interutil.AnnotationKeyExpirationTime: expiration,
			},
		},
	}
}

func assertDeploymentExists(
	t *testing.T,
	k8sClient client.Client,
	namespace string,
	name string,
	want bool,
) {
	t.Helper()
	var deployment appsv1.Deployment
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: namespace, Name: name}, &deployment)
	if want && err != nil {
		t.Fatalf("expected deployment %s/%s to exist: %v", namespace, name, err)
	}
	if !want && !k8serrors.IsNotFound(err) {
		t.Fatalf("expected deployment %s/%s to be deleted, got error: %v", namespace, name, err)
	}
}
