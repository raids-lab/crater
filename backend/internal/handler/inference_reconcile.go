package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
)

const kthenaRequestHashAnnotation = "crater.raids.io/request-hash"

func kthenaRequestHash(req *CreateKthenaReq) (string, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (mgr *KthenaMgr) reconcileKthenaNetworking(ctx context.Context, serving *unstructured.Unstructured) error {
	if !serving.GetDeletionTimestamp().IsZero() {
		return bizerr.Conflict.ResourceStatusError.New("deployment is being deleted")
	}
	deployment, err := kthena.ReadDeployment(serving)
	if err != nil {
		return err
	}
	req := &CreateKthenaReq{
		BackendType: deployment.Engine,
		Layout:      deployment.Layout,
		Port:        deployment.Port,
		ServedModel: deployment.ServedModel,
	}
	for _, desired := range buildKthenaNetworking(serving, req) {
		err := mgr.client.Create(ctx, desired)
		if err == nil {
			continue
		}
		{
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			current := &unstructured.Unstructured{}
			current.SetGroupVersionKind(desired.GroupVersionKind())
			if err := mgr.client.Get(ctx, client.ObjectKeyFromObject(desired), current); err != nil {
				return err
			}
			if !isRelatedKthenaObject(current, serving) {
				return bizerr.Conflict.ResourceAlreadyExists.New("routing resource is owned by another deployment")
			}
			// Only repair the spec of resources owned by this exact serving UID.
			if reflect.DeepEqual(current.Object["spec"], desired.Object["spec"]) {
				continue
			}
			current.Object["spec"] = desired.Object["spec"]
			if err := mgr.client.Update(ctx, current); err != nil {
				return err
			}
		}
	}
	return nil
}

// UserReconcileKthenaService godoc
// @Summary Repair an incomplete deployment's routing resources
// @Tags kthena
// @Security Bearer
// @Produce json
// @Param name path string true "Inference service name"
// @Success 200 {object} resputil.Response[string]
// @Failure 409 {object} resputil.Response[any]
// @Router /v1/kthena/inference-services/{name}/reconcile [post]
func (mgr *KthenaMgr) UserReconcileKthenaService(c *gin.Context) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}
	if err := mgr.reconcileKthenaNetworking(c.Request.Context(), obj); err != nil {
		resputil.HandleError(c,
			bizerr.Conflict.ResourceStatusError.Wrap(err,
				"deployment repair failed; inspect resource ownership and cluster access"))
		return
	}
	resputil.Success(c, "deployment routing repaired")
}
