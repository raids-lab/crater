package handler

import (
	"testing"

	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
