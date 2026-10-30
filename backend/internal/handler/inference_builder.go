package handler

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	kthenaTolerationEffectKey = "effect"
	kthenaSelectorKey         = "key"
	kthenaSelectorOperatorKey = "operator"
)

var inferenceServiceNameRE = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

//nolint:gocyclo // Each field is normalized and validated in a fixed order so API clients receive precise errors.
func validateCreateKthenaReq(ctx context.Context, req *CreateKthenaReq, token util.JWTMessage) error {
	req.Name = strings.TrimSpace(req.Name)
	req.ModelSource = strings.TrimSpace(req.ModelSource)
	req.ModelURI = strings.TrimSpace(req.ModelURI)
	req.ServedModel = strings.TrimSpace(req.ServedModel)
	req.BackendType = strings.TrimSpace(req.BackendType)
	req.CacheURI = strings.TrimSpace(req.CacheURI)

	if !inferenceServiceNameRE.MatchString(req.Name) || len(req.Name) > 63 {
		return bizerr.BadRequest.ParameterError.New(
			"name must be a valid Kubernetes name with lowercase letters, digits, or hyphens",
		)
	}
	if req.ModelSource == "" {
		req.ModelSource = inferenceModelSourcePlatform
	}
	if req.ModelSource == inferenceModelSourcePlatform {
		if req.PlatformModelID == 0 {
			return bizerr.BadRequest.MissingParameter.New("platform model is required")
		}
		dataset, err := loadAccessibleModelDataset(ctx, req.PlatformModelID, token)
		if err != nil {
			return err
		}
		req.ModelSubPath = strings.TrimLeft(dataset.URL, "/")
		if req.ModelSubPath == "" ||
			path.Clean(req.ModelSubPath) != req.ModelSubPath ||
			strings.HasPrefix(req.ModelSubPath,
				"..") ||
			strings.Contains(req.ModelSubPath,
				"://") {
			return bizerr.BadRequest.ParameterError.New("platform model storage path is invalid")
		}
		req.ModelPVC = strings.TrimSpace(config.GetConfig().Storage.PVC.ReadWriteMany)
		if req.ModelPVC == "" {
			return bizerr.Conflict.ResourceStatusError.New("platform model storage is not configured")
		}
		req.ModelURI = datasetToKthenaModelURI(dataset)
		if req.ServedModel == "" {
			req.ServedModel = dataset.Name
		}
		req.CacheURI = datasetModelCacheURI()
	} else if req.ModelSource != inferenceModelSourceExternal {
		return bizerr.BadRequest.ParameterError.New("modelSource must be platform or external")
	}
	_, location, _ := strings.Cut(req.ModelURI, "://")
	if strings.TrimSpace(location) == "" {
		return bizerr.BadRequest.ParameterError.New("modelURI must include a location")
	}
	if req.ModelURI == "" {
		return bizerr.BadRequest.MissingParameter.New("modelURI is required")
	}
	if !strings.HasPrefix(req.ModelURI, "hf://") &&
		!strings.HasPrefix(req.ModelURI, "s3://") &&
		!strings.HasPrefix(req.ModelURI, "pvc://") &&
		!strings.HasPrefix(req.ModelURI, "ms://") {
		return bizerr.BadRequest.ParameterError.New("modelURI must start with hf://, s3://, pvc://, or ms://")
	}
	if req.BackendType == "" {
		req.BackendType = kthenaBackendVLLM
	}
	if req.ModelSource == inferenceModelSourceExternal {
		if strings.HasPrefix(req.ModelURI, "pvc://") || req.CacheURI != "" {
			return bizerr.BadRequest.ParameterError.New("external models require isolated storage; PVC and hostPath are not accepted")
		}
	}

	if req.ModelSource == inferenceModelSourcePlatform && req.ModelRevision != "" {
		return bizerr.BadRequest.ParameterError.New("platform models use their stored dataset revision")
	}
	distributed := req.Layout == kthena.LayoutPD
	for i := range req.Roles {
		r := &req.Roles[i]
		distributed = distributed || r.Execution == kthena.ExecutionRay
	}
	if req.ModelSource == inferenceModelSourceExternal {
		if strings.HasPrefix(req.ModelURI, "s3://") && (req.ModelRevision != "" || distributed) {
			return bizerr.BadRequest.ParameterError.New("save S3 models in the platform model library before distributed serving")
		}
		if distributed && !regexp.MustCompile(`^[a-fA-F0-9]{40}$`).MatchString(req.ModelRevision) {
			return bizerr.BadRequest.ParameterError.New(
				"distributed external models require an immutable 40-character commit revision",
			)
		}
	}
	if req.ServedModel == "" {
		req.ServedModel = inferServedModelName(req.ModelURI)
	}
	if req.Port == 0 {
		req.Port = 8000
	}
	if req.Port < 1 || req.Port > 65535 {
		return bizerr.BadRequest.ParameterError.New("port must be between 1 and 65535")
	}
	if err := kthena.ValidateRoles(req.BackendType, req.Layout, req.Replicas, req.Roles); err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "invalid serving topology")
	}
	return nil
}

func inferenceMetadata(req *CreateKthenaReq, token util.JWTMessage, namespace string) *unstructured.Unstructured {
	labels := map[string]string{
		inferenceServiceLabelManagedBy: inferenceServiceManagedByValue,
		inferenceServiceLabelUserID:    strconv.FormatUint(uint64(token.UserID), 10),
		inferenceServiceLabelAccountID: strconv.FormatUint(uint64(token.AccountID), 10),
	}
	annotations := map[string]string{
		inferenceServiceAnnotationUsername: token.Username,
		inferenceServiceAnnotationAccount:  token.AccountName,
		inferenceServiceAnnotationSource:   req.ModelSource,
	}
	if req.PlatformModelID > 0 {
		annotations[inferenceServiceAnnotationModelID] = strconv.FormatUint(uint64(req.PlatformModelID), 10)
	}

	obj := &unstructured.Unstructured{}
	obj.SetName(req.Name)
	obj.SetNamespace(namespace)
	obj.SetLabels(labels)
	obj.SetAnnotations(annotations)
	return obj
}

func buildWorkerAffinity(selectors []corev1.NodeSelectorRequirement) map[string]any {
	if len(selectors) == 0 {
		return nil
	}
	expressions := make([]any, 0, len(selectors))
	for _, selector := range selectors {
		values := make([]any, 0, len(selector.Values))
		for _, value := range selector.Values {
			values = append(values, value)
		}
		expressions = append(expressions, map[string]any{
			kthenaSelectorKey:         selector.Key,
			kthenaSelectorOperatorKey: string(selector.Operator),
			"values":                  values,
		})
	}
	return map[string]any{
		"nodeAffinity": map[string]any{
			"requiredDuringSchedulingIgnoredDuringExecution": map[string]any{
				"nodeSelectorTerms": []any{
					map[string]any{
						"matchExpressions": expressions,
					},
				},
			},
		},
	}
}
