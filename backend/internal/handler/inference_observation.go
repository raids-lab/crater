package handler

import (
	"context"
	"sort"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/resputil"
)

func (mgr *KthenaMgr) readKthenaResources(ctx context.Context, serving *unstructured.Unstructured) ([]KthenaResource, error) {
	resources := []KthenaResource{kthenaResourceFromObject(serving.GetKind(), serving)}
	for _, gvk := range []schema.GroupVersionKind{kthena.ServerGVK, kthena.RouteGVK} {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		if err := mgr.client.Get(ctx, client.ObjectKeyFromObject(serving), obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		if isRelatedKthenaObject(obj, serving) {
			resources = append(resources, kthenaResourceFromObject(obj.GetKind(), obj))
		}
	}
	return resources, nil
}

func (mgr *KthenaMgr) readKthenaPods(ctx context.Context, serving *unstructured.Unstructured) ([]corev1.Pod, error) {
	if mgr.kubeClient == nil {
		return nil, nil
	}
	selector := labels.Set{"modelserving.volcano.sh/name": serving.GetName(),
		kthena.PodDeploymentLabel: serving.GetLabels()[kthena.PodDeploymentLabel]}
	pods, err := mgr.kubeClient.CoreV1().Pods(serving.GetNamespace()).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

// UserKthenaDiagnostics godoc
// @Summary Inspect model deployment diagnostics on demand
// @Tags kthena
// @Security Bearer
// @Produce json
// @Param name path string true "Inference service name"
// @Success 200 {object} resputil.Response[[]KthenaDiagnostic]
// @Failure 500 {object} resputil.Response[any]
// @Router /v1/kthena/inference-services/{name}/diagnostics [get]
func (mgr *KthenaMgr) UserKthenaDiagnostics(c *gin.Context) {
	obj, ok := mgr.loadKthenaService(c, false)
	if !ok {
		return
	}
	pods, err := mgr.readKthenaPods(c.Request.Context(), obj)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.K8sServiceError.Wrap(err, "read deployment pods failed"))
		return
	}
	diagnostics := make([]KthenaDiagnostic, 0)
	for i := range pods {
		diagnostics = append(diagnostics, mgr.diagnosticsFromPod(c.Request.Context(), &pods[i])...)
		diagnostics = append(diagnostics, mgr.diagnosticsFromPodEvents(c.Request.Context(), &pods[i])...)
	}
	sort.SliceStable(diagnostics, func(i, j int) bool { return diagnostics[i].Timestamp.After(diagnostics[j].Timestamp) })
	if len(diagnostics) > kthenaMaxDiagnostics {
		diagnostics = diagnostics[:kthenaMaxDiagnostics]
	}
	resputil.Success(c, diagnostics)
}
