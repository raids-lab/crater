package handler

import (
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
)

func TestServingUpdateKeepsIdentityAndRequiresPauseForResources(t *testing.T) {
	req := nativeTestRequest(t)
	obj := nativeTestServing(t, req)
	c := nativeTestClient(obj)
	mgr := &KthenaMgr{client: c, namespace: "jobs"}
	token := util.JWTMessage{UserID: 1, AccountID: 2}
	if err := c.Get(t.Context(), client.ObjectKeyFromObject(obj), obj); err != nil {
		t.Fatal(err)
	}
	identity := obj.GetLabels()[kthena.PodDeploymentLabel]
	plugins, _, _ := unstructured.NestedSlice(obj.Object, "spec", "plugins")
	req.Roles[0].Entry.Image = "test/engine:updated"
	req.Roles[0].Entry.CPU = "2000m"
	if err := mgr.updateServing(t.Context(), obj, req, token); err != nil {
		t.Fatal(err)
	}
	roles, _, _ := unstructured.NestedSlice(obj.Object, "spec", "template", "roles")
	actual, _, _ := unstructured.NestedString(
		roles[0].(map[string]any),
		"entryTemplate",
		"metadata",
		"labels",
		kthena.PodDeploymentLabel,
	)
	if actual != identity || obj.GetUID() != "native-uid" {
		t.Fatal("rolling update changed routing identity")
	}
	updatedPlugins, _, _ := unstructured.NestedSlice(obj.Object, "spec", "plugins")
	if !reflect.DeepEqual(updatedPlugins, plugins) {
		t.Fatalf("rolling update changed plugins from %v to %v", plugins, updatedPlugins)
	}
	req.Roles[0].Entry.CPU = "3"
	if err := mgr.updateServing(t.Context(), obj, req, token); err == nil {
		t.Fatal("active resource growth bypassed admission")
	}
	if err := unstructured.SetNestedField(obj.Object, int64(0), "spec", "replicas"); err != nil {
		t.Fatal(err)
	}
	annotations := obj.GetAnnotations()
	annotations[servingPausedAnnotation] = "2"
	obj.SetAnnotations(annotations)
	if err := c.Update(t.Context(), obj); err != nil {
		t.Fatal(err)
	}
	if err := mgr.updateServing(t.Context(), obj, req, token); err != nil {
		t.Fatal(err)
	}
	d, err := kthena.ReadDeployment(obj)
	if err != nil {
		t.Fatal(err)
	}
	if d.Replicas != 0 || d.Roles[0].Entry.CPU != "3" {
		t.Fatal("paused update resumed or lost resource change")
	}
}

func TestServingUpdateKeepsLegacyPluginChain(t *testing.T) {
	req := nativeTestRequest(t)
	obj := nativeTestServing(t, req)
	unstructured.RemoveNestedField(obj.Object, "spec", "plugins")
	mgr := &KthenaMgr{namespace: "jobs"}
	token := util.JWTMessage{UserID: 1, AccountID: 2}
	req.Roles[0].Entry.Image = "test/engine:updated"
	if err := mgr.prepareServingUpdate(t.Context(), obj, req, token); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(obj.Object, "spec", "plugins"); found {
		t.Fatal("single-node edit added an immutable plugin to an existing deployment")
	}
}

func TestLegacyServingRequiresCloneBeforeStartingRay(t *testing.T) {
	req := nativeTestRequest(t)
	mgr := &KthenaMgr{namespace: "jobs"}
	token := util.JWTMessage{UserID: 1, AccountID: 2}
	single := nativeTestServing(t, req)
	unstructured.RemoveNestedField(single.Object, "spec", "plugins")
	req.Roles[0].Execution = kthena.ExecutionRay
	req.Roles[0].NodesPerInstance = 2
	req.Roles[0].PipelineParallel = 2
	if err := mgr.prepareServingUpdate(t.Context(), single, req, token); err == nil || !strings.Contains(err.Error(), "headless-service") {
		t.Fatalf("legacy edit error = %v, want headless-service conflict", err)
	}

	ray := nativeTestServing(t, req)
	unstructured.RemoveNestedField(ray.Object, "spec", "plugins")
	if err := unstructured.SetNestedField(ray.Object, int64(0), "spec", "replicas"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.prepareServingScale(t.Context(), ray, token, req.Replicas, ray.GetAnnotations()); err == nil ||
		!strings.Contains(err.Error(), "headless-service") {
		t.Fatalf("legacy resume error = %v, want headless-service conflict", err)
	}
}

func TestServingSecretOwnershipAndMissingKey(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "model-access",
			Namespace: "jobs",
			Labels:    map[string]string{inferenceServiceLabelUserID: "1", inferenceServiceLabelAccountID: "2"},
		},
		Data: map[string][]byte{"token": []byte("fixture")},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
	mgr := &KthenaMgr{client: c, namespace: "jobs"}
	req := nativeTestRequest(t)
	req.Roles[0].Entry.SecretEnv = map[string]kthena.SecretKeyRef{"HF_AUTH_TOKEN": {Name: secret.Name, Key: "token"}}
	if err := mgr.validateServingSecrets(t.Context(), req, util.JWTMessage{UserID: 1, AccountID: 2}); err != nil {
		t.Fatal(err)
	}
	if err := mgr.validateServingSecrets(t.Context(), req, util.JWTMessage{UserID: 1, AccountID: 3}); err == nil {
		t.Fatal("cross-account credential accepted")
	}
	req.Roles[0].Entry.SecretEnv["HF_AUTH_TOKEN"] = kthena.SecretKeyRef{Name: secret.Name, Key: "missing"}
	if err := mgr.validateServingSecrets(t.Context(), req, util.JWTMessage{UserID: 1, AccountID: 2}); err == nil {
		t.Fatal("missing credential key accepted")
	}
}
