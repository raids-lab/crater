package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"strings"

	"github.com/google/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/vcqueue"
)

const (
	kthenaDefaultDownloaderImage       = "ghcr.io/volcano-sh/downloader:v1.0.0"
	kthenaEnginePort             int64 = 8000
)

func buildModelServing(
	req *CreateKthenaReq,
	token util.JWTMessage,
	namespace, downloaderImage string,
) (*unstructured.Unstructured, error) {
	obj := inferenceMetadata(req, token, namespace)
	obj.SetGroupVersionKind(kthena.ServingGVK)
	labels := obj.GetLabels()
	labels[kthena.LayoutLabel] = kthena.NativeLayout
	labels[kthena.PodDeploymentLabel] = uuid.NewString()
	obj.SetLabels(labels)
	annotations := obj.GetAnnotations()
	annotations[scheduling.QueueNameAnnotationKey] = vcqueue.ResolveJobQueueName(token)
	annotations[kthena.ModelURIAnnotation] = req.ModelURI
	annotations["crater.raids.io/model-revision"] = req.ModelRevision
	annotations[kthena.CacheURIAnnotation] = req.CacheURI
	annotations[kthena.EngineAnnotation] = req.BackendType
	annotations[kthena.LayoutAnnotation] = req.Layout
	annotations[kthena.ModelAnnotation] = req.ServedModel
	obj.SetAnnotations(annotations)
	roles := []any{}
	minimum := map[string]any{}
	for i := range req.Roles {
		role := &req.Roles[i]
		entry, err := buildServingPod(req, role, &role.Entry, true, token, labels, downloaderImage)
		if err != nil {
			return nil, err
		}
		item := map[string]any{
			"name":           role.Name,
			"replicas":       role.Instances,
			"workerReplicas": role.NodesPerInstance - 1,
			"entryTemplate":  entry,
		}
		if role.NodesPerInstance > 1 {
			profile := &role.Entry
			if role.Worker != nil {
				profile = role.Worker
			}
			worker, err := buildServingPod(req, role, profile, false, token, labels, downloaderImage)
			if err != nil {
				return nil, err
			}
			item["workerTemplate"] = worker
		}
		roles = append(roles, item)
		minimum[role.Name] = role.Instances
	}
	obj.Object["spec"] = map[string]any{
		"schedulerName":  kthenaSchedulerName,
		"replicas":       req.Replicas,
		"recoveryPolicy": "ServingGroupRecreate",
		"template":       map[string]any{"gangPolicy": map[string]any{"minRoleReplicas": minimum}, "roles": roles},
	}
	return obj, nil
}

func kthenaConfiguredPort(req *CreateKthenaReq) int64 {
	if req.Port > 0 {
		return req.Port
	}
	return kthenaEnginePort
}

// Networking resources are owned by the created ModelServing, never by a name
// alone. Garbage collection removes both after the root is deleted.
func buildKthenaNetworking(serving *unstructured.Unstructured, req *CreateKthenaReq) []*unstructured.Unstructured {
	server := kthenaOwnedObject(serving, kthena.ServerGVK)
	server.Object["spec"] = map[string]any{
		"model": req.ServedModel, "inferenceEngine": req.BackendType,
		"workloadSelector": map[string]any{"matchLabels": map[string]any{
			kthena.PodDeploymentLabel:      serving.GetLabels()[kthena.PodDeploymentLabel],
			kthena.EntryLabel:              "true",
			"modelserving.volcano.sh/name": serving.GetName(),
		}},
		"workloadPort": map[string]any{"port": kthenaConfiguredPort(req)},
	}
	if req.Layout == kthena.LayoutPD {
		spec := server.Object["spec"].(map[string]any)
		spec["workloadSelector"].(map[string]any)["pdGroup"] = map[string]any{
			"groupKey":      kthena.GroupLabel,
			"prefillLabels": map[string]any{kthena.RoleLabel: "prefill"},
			"decodeLabels":  map[string]any{kthena.RoleLabel: "decode"},
		}
		if req.BackendType == kthena.EngineVLLM {
			spec["kvConnector"] = map[string]any{"type": "nixl"}
		}
	}
	route := kthenaOwnedObject(serving, kthena.RouteGVK)
	route.Object["spec"] = map[string]any{
		"modelName": serving.GetName(),
		"rules": []any{
			map[string]any{"name": "default", "targetModels": []any{map[string]any{"modelServerName": server.GetName()}}},
		},
	}
	return []*unstructured.Unstructured{server, route}
}

func kthenaOwnedObject(serving *unstructured.Unstructured, gvk schema.GroupVersionKind) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	obj.SetName(serving.GetName())
	obj.SetNamespace(serving.GetNamespace())
	obj.SetLabels(serving.GetLabels())
	controller := true
	obj.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: serving.GetAPIVersion(), Kind: serving.GetKind(), Name: serving.GetName(),
		UID: serving.GetUID(), Controller: &controller,
	}})
	return obj
}

// The downloader is bounded by the same CPU/memory budget as the engine, so
// Kubernetes max(init, app) pod requests agree with Kthena's PodGroup sum.
// Downloading does not request an accelerator separately.
func kthenaDownloaderResources(profile *kthena.PodProfile) map[string]any {
	return map[string]any{
		"requests": map[string]any{kthenaResourceCPUKey: profile.CPU, kthenaResourceMemoryKey: profile.Memory},
		"limits":   map[string]any{kthenaResourceCPUKey: profile.CPU, kthenaResourceMemoryKey: profile.Memory},
	}
}

func buildKthenaModelStorage(req *CreateKthenaReq, profile *kthena.PodProfile, env []any, downloaderImage string) (
	volume map[string]any, modelMount map[string]any, modelPath string, initContainers []any, err error,
) {
	volume = map[string]any{"name": "model-cache", "emptyDir": map[string]any{}}
	mountPath := "/model-cache"
	hash := sha256.Sum256([]byte(req.ModelURI + "@" + req.ModelRevision))
	modelPath = path.Join(mountPath, hex.EncodeToString(hash[:]))
	modelMount = map[string]any{"name": "model-cache", "mountPath": mountPath}
	initContainers = []any{}
	if req.ModelSource == inferenceModelSourcePlatform {
		if req.ModelPVC == "" ||
			req.ModelSubPath == "" ||
			path.Clean(req.ModelSubPath) != req.ModelSubPath ||
			path.IsAbs(req.ModelSubPath) ||
			strings.HasPrefix(req.ModelSubPath,
				"..") {
			return nil, nil, "", nil, bizerr.BadRequest.ParameterError.New("authorized model storage is required")
		}
		volume = map[string]any{
			"name":                  "model-cache",
			"persistentVolumeClaim": map[string]any{"claimName": req.ModelPVC, "readOnly": true},
		}
		modelMount["subPath"] = req.ModelSubPath
		modelMount["readOnly"] = true
		modelPath = mountPath
	} else {
		if req.ModelRevision != "" {
			name := "HF_REVISION"
			if strings.HasPrefix(req.ModelURI, "ms://") {
				name = "MS_REVISION"
			}
			env = append(append([]any{}, env...), map[string]any{"name": name, "value": req.ModelRevision})
		}
		if strings.HasPrefix(req.ModelURI, "pvc://") || req.CacheURI != "" {
			return nil, nil, "", nil, bizerr.BadRequest.ParameterError.New("external models cannot mount PVC or hostPath storage")
		}
		initContainers = append(initContainers, map[string]any{
			"name": "model-downloader", "image": nonEmpty(downloaderImage, kthenaDefaultDownloaderImage),
			"args": []any{"--source", req.ModelURI, "--output-dir", modelPath}, "env": env, "volumeMounts": []any{modelMount},
			"resources": kthenaDownloaderResources(profile),
		})
	}
	return volume, modelMount, modelPath, initContainers, nil
}
