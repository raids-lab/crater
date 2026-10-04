package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestServingPreviewUsesMutationRulesWithoutPersisting(t *testing.T) {
	for _, test := range []struct {
		name, mode      string
		paused, invalid bool
	}{
		{"active resource growth", "update", false, true},
		{"paused resource update", "update", true, false},
		{"create name collision", "create", false, true},
		{"invalid intent", "resume", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := nativeTestRequest(t)
			obj := nativeTestServing(t, req)
			if test.paused {
				if err := unstructured.SetNestedField(obj.Object, int64(0), "spec", "replicas"); err != nil {
					t.Fatal(err)
				}
			}
			c := nativeTestClient(obj)
			mgr := &KthenaMgr{client: c, namespace: "jobs"}
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/preview?mode="+test.mode, http.NoBody)
			util.SetJWTContext(ctx, util.JWTMessage{UserID: 1, AccountID: 2})
			req.Roles[0].Entry.CPU = "3"
			preview, _, err := mgr.prepareServingPreview(ctx, req, obj.DeepCopy())
			if (err != nil) != test.invalid {
				t.Fatalf("error=%v, want invalid=%v", err, test.invalid)
			}
			stored := obj.DeepCopy()
			if err := c.Get(t.Context(), client.ObjectKeyFromObject(obj), stored); err != nil {
				t.Fatal(err)
			}
			deployment, err := kthena.ReadDeployment(stored)
			if err != nil {
				t.Fatal(err)
			}
			if deployment.Roles[0].Entry.CPU == "3" {
				t.Fatal("preview persisted mutation")
			}
			if !test.invalid {
				d, err := kthena.ReadDeployment(preview)
				if err != nil {
					t.Fatal(err)
				}
				if d.Replicas != 0 || d.Roles[0].Entry.CPU != "3" {
					t.Fatal("preview lost suspended update")
				}
			}
		})
	}
}
