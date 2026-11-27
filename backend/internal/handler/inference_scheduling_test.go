package handler

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

func schedulingTestClient(t *testing.T, queues ...client.Object) client.WithWatch {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := scheduling.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(queues...).Build()
}

func TestKthenaUsesSameVolcanoQueueAsOtherJobs(t *testing.T) {
	for _, token := range []util.JWTMessage{
		{UserID: 7, AccountID: model.DefaultAccountID, AccountName: vcqueue.PublicQueueName},
		{UserID: 7, AccountID: 9, AccountName: "research"},
	} {
		t.Run(token.AccountName, func(t *testing.T) {
			leaf := &scheduling.Queue{ObjectMeta: metav1.ObjectMeta{Name: vcqueue.ResolveJobQueueName(token)}}
			queues := []client.Object{leaf}
			if token.AccountID != model.DefaultAccountID {
				leaf.Spec.Parent = vcqueue.GetAccountLogicQueueName(token.AccountID)
				queues = append(queues, &scheduling.Queue{ObjectMeta: metav1.ObjectMeta{Name: leaf.Spec.Parent}})
			}
			mgr := &KthenaMgr{client: schedulingTestClient(t, queues...), namespace: kthenaConversationTestNamespace}
			if err := mgr.prepareKthenaScheduling(t.Context(), token); err != nil {
				t.Fatal(err)
			}
			req := nativeTestRequest(t)
			serving, err := buildModelServing(req, token, mgr.namespace, "")
			if err != nil {
				t.Fatal(err)
			}
			if serving.GetAnnotations()[scheduling.QueueNameAnnotationKey] != leaf.Name {
				t.Fatal("inference queue differs from job queue")
			}
			scheduler, _, _ := unstructured.NestedString(serving.Object, "spec", "schedulerName")
			if scheduler != kthenaSchedulerName {
				t.Fatalf("scheduler = %s", scheduler)
			}
			minimum, _, _ := unstructured.NestedInt64(serving.Object, "spec", "template", "gangPolicy", "minRoleReplicas", "server")
			if minimum != req.Roles[0].Instances {
				t.Fatalf("gang minimum = %d", minimum)
			}
			roles, _, _ := unstructured.NestedSlice(serving.Object, "spec", "template", "roles")
			podScheduler, _, _ := unstructured.NestedString(roles[0].(map[string]any), "entryTemplate", "spec", "schedulerName")
			if podScheduler != scheduler {
				t.Fatal("pod uses a different scheduler")
			}
		})
	}
}

func TestKthenaRejectsMissingVolcanoPodGroupAPI(t *testing.T) {
	c := interceptor.NewClient(schedulingTestClient(t),
		interceptor.Funcs{List: func(context.Context,
			client.WithWatch,
			client.ObjectList,
			...client.ListOption) error {
			return apierrors.NewNotFound(schema.GroupResource{Group: scheduling.SchemeGroupVersion.Group, Resource: "podgroups"}, "")
		}})
	mgr := &KthenaMgr{client: c, namespace: kthenaConversationTestNamespace}
	if err := mgr.prepareKthenaScheduling(t.Context(),
		util.JWTMessage{UserID: 1,
			AccountID:   model.DefaultAccountID,
			AccountName: vcqueue.PublicQueueName}); err == nil {
		t.Fatal("missing PodGroup API accepted")
	}
}

func TestKthenaRejectsClosedOrForeignUserQueue(t *testing.T) {
	token := util.JWTMessage{UserID: 7, AccountID: 9}
	parent := vcqueue.GetAccountLogicQueueName(token.AccountID)
	for _, test := range []struct {
		name, parent string
		state        scheduling.QueueState
	}{
		{"closed", parent, "Closed"}, {"foreign", "q-a10", "Open"},
	} {
		t.Run(test.name, func(t *testing.T) {
			queue := &scheduling.Queue{ObjectMeta: metav1.ObjectMeta{Name: vcqueue.ResolveJobQueueName(token)},
				Spec:   scheduling.QueueSpec{Parent: test.parent},
				Status: scheduling.QueueStatus{State: test.state}}
			c := schedulingTestClient(t, queue, &scheduling.Queue{ObjectMeta: metav1.ObjectMeta{Name: parent}})
			mgr := &KthenaMgr{client: c, namespace: kthenaConversationTestNamespace}
			if err := mgr.prepareKthenaScheduling(t.Context(), token); err == nil {
				t.Fatal("invalid inference queue accepted")
			}
		})
	}
}

func TestServingPreflightQueueChecksDoNotProvision(t *testing.T) {
	token := util.JWTMessage{UserID: 7, AccountID: 9}
	c := schedulingTestClient(t)
	mgr := &KthenaMgr{client: c, namespace: "jobs"}
	if err := mgr.checkKthenaScheduling(t.Context(), token, true); err != nil {
		t.Fatal(err)
	}
	queues := &scheduling.QueueList{}
	if err := c.List(t.Context(), queues); err != nil {
		t.Fatal(err)
	}
	if len(queues.Items) != 0 {
		t.Fatal("preview created queues")
	}
	parent := &scheduling.Queue{
		ObjectMeta: metav1.ObjectMeta{Name: vcqueue.GetAccountLogicQueueName(token.AccountID)},
		Status:     scheduling.QueueStatus{State: "Closed"},
	}
	mgr.client = schedulingTestClient(t, parent)
	if mgr.checkKthenaScheduling(t.Context(), token, true) == nil {
		t.Fatal("preview accepted closed account queue")
	}
	if mgr.prepareKthenaScheduling(t.Context(), token) == nil {
		t.Fatal("submission accepted closed account queue")
	}
}
