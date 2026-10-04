package handler

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/kthena"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/vcqueue"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

const (
	servingProbePeriod     int64 = 10
	servingProbeTimeout    int64 = 5
	servingStartupFailures int64 = 180
	servingRayProbePeriod  int64 = 15
)

func buildServingPod(
	req *CreateKthenaReq,
	role *kthena.RoleSpec,
	profile *kthena.PodProfile,
	entry bool,
	token util.JWTMessage,
	labels map[string]string,
	downloader string,
) (map[string]any, error) {
	env := servingEnvironment(profile)
	volume, mount, modelPath, init, err := buildKthenaModelStorage(req, profile, env, downloader)
	if err != nil {
		return nil, err
	}
	env = append(
		env,
		map[string]any{"name": "POD_IP", "valueFrom": map[string]any{"fieldRef": map[string]any{"fieldPath": "status.podIP"}}},
	)
	resources := map[string]any{}
	for k, v := range profile.Requests() {
		resources[string(k)] = v.String()
	}
	cmd := kthena.RuntimeCommand(req.BackendType, req.Layout, modelPath, req.ServedModel, kthenaConfiguredPort(req), role, entry)
	command := []any{}
	for _, arg := range cmd {
		command = append(command, arg)
	}
	container := map[string]any{"name": kthena.EngineContainer, "image": profile.Image, "command": command, "env": env,
		"resources": map[string]any{
			"requests": resources,
			"limits":   resources,
		}, "volumeMounts": []any{mount, map[string]any{"name": "dshm", "mountPath": "/dev/shm"}}}

	if len(profile.ExtendedResources) > 0 {
		container["securityContext"] = map[string]any{"capabilities": map[string]any{"add": []any{"IPC_LOCK"}}}
	}
	if role.Execution == kthena.ExecutionRay && !entry {
		probe := map[string]any{"exec": map[string]any{"command": []any{"python3", "-c", `import ray,os
ray.init(address="auto")
assert any(n["Alive"] and n["NodeManagerAddress"]==os.environ["POD_IP"] for n in ray.nodes())`},
		}, "periodSeconds": servingRayProbePeriod, "timeoutSeconds": servingProbePeriod}
		container["readinessProbe"] = probe
	}
	if entry {
		port := kthenaConfiguredPort(req)
		container["ports"] = []any{map[string]any{"name": "http", "containerPort": port}}
		container["readinessProbe"] = map[string]any{
			"httpGet":        map[string]any{"path": "/health", "port": port},
			"periodSeconds":  servingProbePeriod,
			"timeoutSeconds": servingProbeTimeout,
		}
		container["startupProbe"] = map[string]any{
			"httpGet":          map[string]any{"path": "/health", "port": port},
			"periodSeconds":    servingProbePeriod,
			"timeoutSeconds":   servingProbeTimeout,
			"failureThreshold": servingStartupFailures,
		}
	}
	spec := map[string]any{"schedulerName": kthenaSchedulerName, "containers": []any{container}, "initContainers": init,
		"automountServiceAccountToken": false,
		"volumes": []any{volume, map[string]any{
			"name":     "dshm",
			"emptyDir": map[string]any{"medium": "Memory", "sizeLimit": profile.Memory},
		}},
	}
	applyServingScheduling(spec, req, role, profile, token, labels)
	runtimeConfig, _ := json.Marshal(
		map[string]any{
			"execution":        role.Execution,
			"tensorParallel":   role.TensorParallel,
			"pipelineParallel": role.PipelineParallel,
			"engineOptions":    role.EngineOptions,
			"gpuModel":         profile.GPUModel,
		},
	)
	annotations := map[string]any{
		kthena.RoleConfigAnnotation:       string(runtimeConfig),
		scheduling.QueueNameAnnotationKey: vcqueue.ResolveJobQueueName(token),
	}
	if entry {
		annotations["prometheus.io/scrape"] = strconv.FormatBool(entry)
		annotations["prometheus.io/path"] = "/metrics"
		annotations["prometheus.io/port"] = strconv.FormatInt(kthenaConfiguredPort(req), 10)
	}
	return map[string]any{
		"metadata": map[string]any{
			"labels": map[string]any{
				kthena.PodDeploymentLabel:      labels[kthena.PodDeploymentLabel],
				kthena.ManagedByLabel:          kthena.ManagedByValue,
				inferenceServiceLabelUserID:    labels[inferenceServiceLabelUserID],
				inferenceServiceLabelAccountID: labels[inferenceServiceLabelAccountID],
			},
			"annotations": annotations,
		},
		"spec": spec,
	}, nil
}

func servingEnvironment(profile *kthena.PodProfile) []any {
	env := []any{}
	keys := []string{}
	for k := range profile.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		env = append(env, map[string]any{"name": k, "value": profile.Env[k]})
	}

	secretNames := []string{}
	for name := range profile.SecretEnv {
		secretNames = append(secretNames, name)
	}
	sort.Strings(secretNames)
	for _, name := range secretNames {
		ref := profile.SecretEnv[name]
		env = append(
			env,
			map[string]any{
				"name":      name,
				"valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": ref.Name, "key": ref.Key}},
			},
		)
	}

	return env
}

func applyServingScheduling(
	spec map[string]any,
	req *CreateKthenaReq,
	role *kthena.RoleSpec,
	profile *kthena.PodProfile,
	token util.JWTMessage,
	labels map[string]string,
) {
	selectors := append([]corev1.NodeSelectorRequirement{}, profile.Selectors...)
	if len(profile.ImageArchs) > 0 {
		selectors = append(
			selectors,
			corev1.NodeSelectorRequirement{Key: "kubernetes.io/arch", Operator: corev1.NodeSelectorOpIn, Values: profile.ImageArchs},
		)
	}
	affinity := buildWorkerAffinity(selectors)
	if role.Execution == kthena.ExecutionRay {
		if affinity == nil {
			affinity = map[string]any{}
		}
		// Spread each distributed role over physical nodes. The deployment UUID
		// prevents interference with another user's deployment with the same name.
		affinity["podAntiAffinity"] = map[string]any{"requiredDuringSchedulingIgnoredDuringExecution": []any{map[string]any{
			"topologyKey":    "kubernetes.io/hostname",
			"matchLabelKeys": []any{kthena.GroupLabel, "modelserving.volcano.sh/role-id"},
			"labelSelector": map[string]any{
				"matchLabels": map[string]any{
					kthena.PodDeploymentLabel: labels[kthena.PodDeploymentLabel],
					kthena.RoleLabel:          role.Name,
				},
			},
		}}}
	}
	if affinity != nil {
		spec["affinity"] = affinity
	}
	tolerations := append([]corev1.Toleration{}, profile.Tolerations...)
	if token.AccountID != model.DefaultAccountID && token.AccountName != "" {
		tolerations = append(
			tolerations,
			corev1.Toleration{
				Key:      "crater.raids.io/account",
				Operator: corev1.TolerationOpEqual,
				Value:    token.AccountName,
				Effect:   corev1.TaintEffectNoSchedule,
			},
		)
	}
	tmp, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&corev1.PodSpec{Tolerations: tolerations})
	if value, ok := tmp["tolerations"]; ok {
		spec["tolerations"] = value
	}
	if secret := config.GetConfig().Secrets.ImagePullSecretName; secret != "" {
		spec["imagePullSecrets"] = []any{map[string]any{"name": secret}}
	}
	if req.BillingGate {
		spec["schedulingGates"] = []any{map[string]any{"name": servingBillingGate}}
	}
}
