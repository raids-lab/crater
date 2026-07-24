// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
)

const (
	defaultPodBandwidth = "1G"

	podIngressBandwidthAnnotation = "kubernetes.io/ingress-bandwidth"
	podEgressBandwidthAnnotation  = "kubernetes.io/egress-bandwidth"

	flannelNamespace                       = "kube-flannel"
	flannelConfigMapName                   = "kube-flannel-cfg"
	flannelDaemonSetName                   = "kube-flannel-ds"
	flannelCNIConfigKey                    = "cni-conf.json"
	flannelBandwidthInstallerContainerName = "install-bandwidth-plugin"
	flannelCNIBinVolumeHostPath            = "/opt/cni/bin"
	flannelCNIBinVolumeMountPath           = "/host/opt/cni/bin"
	flannelBandwidthBinarySource           = "/opt/cni/bin/bandwidth"
	flannelBandwidthBinaryDestination      = "/host/opt/cni/bin/bandwidth"
)

// PodBandwidthConfig controls the limits applied to newly created Crater
// workload Pods.
type PodBandwidthConfig struct {
	Enabled                bool
	ModelDownloadBandwidth string
	JobIngressBandwidth    string
	JobEgressBandwidth     string
}

func (s *ConfigService) GetPodBandwidthConfig(ctx context.Context) (*PodBandwidthConfig, error) {
	configMap, err := s.getConfigs(
		ctx,
		model.ConfigKeyPodBandwidthEnabled,
		model.ConfigKeyModelDownloadBandwidth,
		model.ConfigKeyJobIngressBandwidth,
		model.ConfigKeyJobEgressBandwidth,
	)
	if err != nil {
		return nil, err
	}

	enabled := false
	if value := configMap[model.ConfigKeyPodBandwidthEnabled]; value != "" {
		enabled, err = strconv.ParseBool(value)
		if err != nil {
			return nil, bizerr.Internal.DatabaseError.New(
				"invalid pod bandwidth config " + model.ConfigKeyPodBandwidthEnabled +
					"=" + strconv.Quote(value),
			)
		}
	}

	modelDownloadBandwidth, err := storedPodBandwidth(configMap, model.ConfigKeyModelDownloadBandwidth)
	if err != nil {
		return nil, err
	}
	jobIngressBandwidth, err := storedPodBandwidth(configMap, model.ConfigKeyJobIngressBandwidth)
	if err != nil {
		return nil, err
	}
	jobEgressBandwidth, err := storedPodBandwidth(configMap, model.ConfigKeyJobEgressBandwidth)
	if err != nil {
		return nil, err
	}

	return &PodBandwidthConfig{
		Enabled:                enabled,
		ModelDownloadBandwidth: modelDownloadBandwidth,
		JobIngressBandwidth:    jobIngressBandwidth,
		JobEgressBandwidth:     jobEgressBandwidth,
	}, nil
}

func (s *ConfigService) UpdatePodBandwidthConfig(ctx context.Context, cfg PodBandwidthConfig) error {
	values := []struct {
		name  string
		value *string
	}{
		{name: "model download bandwidth", value: &cfg.ModelDownloadBandwidth},
		{name: "job ingress bandwidth", value: &cfg.JobIngressBandwidth},
		{name: "job egress bandwidth", value: &cfg.JobEgressBandwidth},
	}
	for _, item := range values {
		*item.value = strings.TrimSpace(*item.value)
		if err := validatePodBandwidth(*item.value); err != nil {
			return bizerr.BadRequest.ParameterError.Wrap(err, "invalid "+item.name)
		}
	}

	updates := map[string]string{
		model.ConfigKeyPodBandwidthEnabled:    strconv.FormatBool(cfg.Enabled),
		model.ConfigKeyModelDownloadBandwidth: cfg.ModelDownloadBandwidth,
		model.ConfigKeyJobIngressBandwidth:    cfg.JobIngressBandwidth,
		model.ConfigKeyJobEgressBandwidth:     cfg.JobEgressBandwidth,
	}
	return s.updateConfigs(ctx, updates)
}

func storedPodBandwidth(configMap map[string]string, key string) (string, error) {
	bandwidth := strings.TrimSpace(configMap[key])
	if bandwidth == "" {
		bandwidth = defaultPodBandwidth
	}
	if err := validatePodBandwidth(bandwidth); err != nil {
		return "", bizerr.Internal.DatabaseError.Wrap(
			err, "invalid stored pod bandwidth config "+key,
		)
	}
	return bandwidth, nil
}

func validatePodBandwidth(value string) error {
	if value == "" {
		return bizerr.BadRequest.ParameterError.New("bandwidth is required")
	}
	if len(value) < 2 || !strings.ContainsRune("KMG", rune(value[len(value)-1])) {
		return bizerr.BadRequest.ParameterError.New("bandwidth unit must be K, M, or G")
	}
	amount, err := strconv.ParseFloat(value[:len(value)-1], 64)
	if err != nil || amount <= 0 {
		return bizerr.BadRequest.ParameterError.New("bandwidth value must be greater than zero")
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return bizerr.BadRequest.ParameterError.Wrap(err, "bandwidth must be a quantity such as 100M or 1G")
	}
	if quantity.Sign() <= 0 {
		return bizerr.BadRequest.ParameterError.New("bandwidth must be greater than zero")
	}
	return nil
}

// ModelDownloadPodBandwidthAnnotations returns the ingress limit for a new
// model download Pod.
func ModelDownloadPodBandwidthAnnotations(
	ctx context.Context,
	configService *ConfigService,
	kubeClient kubernetes.Interface,
) (map[string]string, error) {
	cfg, err := enabledPodBandwidthConfig(ctx, configService, kubeClient)
	if err != nil || cfg == nil {
		return nil, err
	}
	return map[string]string{
		podIngressBandwidthAnnotation: cfg.ModelDownloadBandwidth,
	}, nil
}

// JobPodBandwidthAnnotations returns independent ingress and egress limits for
// every Pod in a new Volcano job, including normal and backfill jobs.
func JobPodBandwidthAnnotations(
	ctx context.Context,
	configService *ConfigService,
	kubeClient kubernetes.Interface,
) (map[string]string, error) {
	cfg, err := enabledPodBandwidthConfig(ctx, configService, kubeClient)
	if err != nil || cfg == nil {
		return nil, err
	}
	return map[string]string{
		podIngressBandwidthAnnotation: cfg.JobIngressBandwidth,
		podEgressBandwidthAnnotation:  cfg.JobEgressBandwidth,
	}, nil
}

// ApplyJobPodBandwidth applies the current system configuration immediately
// before a Volcano Job is created. Existing bandwidth annotations are removed
// first so prequeued jobs do not retain a stale configuration snapshot.
func ApplyJobPodBandwidth(
	ctx context.Context,
	configService *ConfigService,
	kubeClient kubernetes.Interface,
	job *batch.Job,
) error {
	if job == nil {
		return bizerr.Internal.ServiceError.New("job is not initialized")
	}
	annotations, err := JobPodBandwidthAnnotations(ctx, configService, kubeClient)
	if err != nil {
		return err
	}
	for taskIndex := range job.Spec.Tasks {
		taskAnnotations := job.Spec.Tasks[taskIndex].Template.Annotations
		delete(taskAnnotations, podIngressBandwidthAnnotation)
		delete(taskAnnotations, podEgressBandwidthAnnotation)
		if len(annotations) == 0 {
			continue
		}
		if taskAnnotations == nil {
			taskAnnotations = make(map[string]string)
			job.Spec.Tasks[taskIndex].Template.Annotations = taskAnnotations
		}
		for key, value := range annotations {
			taskAnnotations[key] = value
		}
	}
	return nil
}

func enabledPodBandwidthConfig(
	ctx context.Context,
	configService *ConfigService,
	kubeClient kubernetes.Interface,
) (*PodBandwidthConfig, error) {
	if configService == nil {
		return nil, bizerr.Internal.ServiceError.New("config service is not initialized")
	}
	cfg, err := configService.GetPodBandwidthConfig(ctx)
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "get pod bandwidth config failed")
	}
	if !cfg.Enabled {
		return nil, nil
	}
	if err := CheckFlannelBandwidthCNISupport(ctx, kubeClient); err != nil {
		return nil, err
	}
	return cfg, nil
}

// CheckFlannelBandwidthCNISupport verifies Crater's supported Flannel deployment
// contract: standard kube-flannel resources, the bandwidth plugin in the CNI
// chain, the chart installer writing to the host CNI directory, and a complete
// DaemonSet rollout.
func CheckFlannelBandwidthCNISupport(ctx context.Context, kubeClient kubernetes.Interface) error {
	if kubeClient == nil {
		return bizerr.Internal.ServiceError.New("kubernetes client is not initialized")
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(flannelNamespace).Get(
		ctx, flannelConfigMapName, metav1.GetOptions{},
	)
	if err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(
			err, "cannot read "+flannelNamespace+"/"+flannelConfigMapName+" ConfigMap",
		)
	}
	cniConfig := configMap.Data[flannelCNIConfigKey]
	if cniConfig == "" {
		return bizerr.Conflict.ResourceStatusError.New(
			flannelNamespace + "/" + flannelConfigMapName + " ConfigMap has no " + flannelCNIConfigKey,
		)
	}
	if err := checkFlannelBandwidthPlugin(cniConfig); err != nil {
		return err
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(flannelNamespace).Get(
		ctx, flannelDaemonSetName, metav1.GetOptions{},
	)
	if err != nil {
		return bizerr.Internal.K8sServiceError.Wrap(
			err, "cannot read "+flannelNamespace+"/"+flannelDaemonSetName+" DaemonSet",
		)
	}
	return checkFlannelBandwidthDaemonSet(daemonSet)
}

func checkFlannelBandwidthPlugin(cniConfig string) error {
	var conflist struct {
		Plugins []struct {
			Type         string          `json:"type"`
			Capabilities map[string]bool `json:"capabilities"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal([]byte(cniConfig), &conflist); err != nil {
		return bizerr.Conflict.ResourceStatusError.Wrap(err, "cannot parse Flannel cni-conf.json")
	}
	bandwidthPluginEnabled := false
	for _, plugin := range conflist.Plugins {
		if plugin.Type == "bandwidth" && plugin.Capabilities["bandwidth"] {
			bandwidthPluginEnabled = true
			break
		}
	}
	if !bandwidthPluginEnabled {
		return bizerr.Conflict.ResourceStatusError.New(
			"flannel CNI chain does not enable the bandwidth plugin",
		)
	}
	return nil
}

func checkFlannelBandwidthDaemonSet(daemonSet *appsv1.DaemonSet) error {
	if err := checkFlannelBandwidthInstaller(daemonSet); err != nil {
		return err
	}

	if daemonSet.Status.ObservedGeneration < daemonSet.Generation {
		return bizerr.Conflict.ResourceStatusError.New(
			"flannel DaemonSet rollout is incomplete: generation=" +
				strconv.FormatInt(daemonSet.Generation, 10) + " observed=" +
				strconv.FormatInt(daemonSet.Status.ObservedGeneration, 10),
		)
	}

	desired := daemonSet.Status.DesiredNumberScheduled
	if desired == 0 || daemonSet.Status.UpdatedNumberScheduled != desired || daemonSet.Status.NumberReady != desired {
		return bizerr.Conflict.ResourceStatusError.New(
			"flannel DaemonSet rollout is incomplete: desired=" +
				strconv.FormatInt(int64(desired), 10) + " updated=" +
				strconv.FormatInt(int64(daemonSet.Status.UpdatedNumberScheduled), 10) + " ready=" +
				strconv.FormatInt(int64(daemonSet.Status.NumberReady), 10),
		)
	}
	return nil
}

func checkFlannelBandwidthInstaller(daemonSet *appsv1.DaemonSet) error {
	if daemonSet == nil {
		return bizerr.Conflict.ResourceStatusError.New("flannel DaemonSet is not initialized")
	}

	hostVolumeName := findHostPathVolumeName(daemonSet, flannelCNIBinVolumeHostPath)
	if hostVolumeName == "" {
		return bizerr.Conflict.ResourceStatusError.New(
			"flannel DaemonSet does not expose host CNI directory " + flannelCNIBinVolumeHostPath,
		)
	}

	installer := findInitContainer(daemonSet, flannelBandwidthInstallerContainerName)
	if installer == nil {
		return bizerr.Conflict.ResourceStatusError.New(
			"flannel DaemonSet has no " + flannelBandwidthInstallerContainerName + " initContainer",
		)
	}

	if !isBandwidthCopyInstaller(installer) {
		return bizerr.Conflict.ResourceStatusError.New(
			"flannel " + flannelBandwidthInstallerContainerName + " initContainer must copy " +
				flannelBandwidthBinarySource + " to " + flannelBandwidthBinaryDestination,
		)
	}

	if hasWritableVolumeMount(installer, hostVolumeName, flannelCNIBinVolumeMountPath) {
		return nil
	}
	return bizerr.Conflict.ResourceStatusError.New(
		"flannel " + flannelBandwidthInstallerContainerName + " initContainer must mount host CNI directory at " +
			flannelCNIBinVolumeMountPath,
	)
}

func findHostPathVolumeName(daemonSet *appsv1.DaemonSet, hostPath string) string {
	for volumeIndex := range daemonSet.Spec.Template.Spec.Volumes {
		volume := &daemonSet.Spec.Template.Spec.Volumes[volumeIndex]
		if volume.HostPath != nil && volume.HostPath.Path == hostPath {
			return volume.Name
		}
	}
	return ""
}

func findInitContainer(daemonSet *appsv1.DaemonSet, name string) *corev1.Container {
	for containerIndex := range daemonSet.Spec.Template.Spec.InitContainers {
		container := &daemonSet.Spec.Template.Spec.InitContainers[containerIndex]
		if container.Name == name {
			return container
		}
	}
	return nil
}

func isBandwidthCopyInstaller(installer *corev1.Container) bool {
	return len(installer.Command) == 1 &&
		(installer.Command[0] == "cp" || installer.Command[0] == "/bin/cp") &&
		containsArgument(installer.Args, flannelBandwidthBinarySource) &&
		containsArgument(installer.Args, flannelBandwidthBinaryDestination)
}

func hasWritableVolumeMount(container *corev1.Container, volumeName, mountPath string) bool {
	for mountIndex := range container.VolumeMounts {
		mount := &container.VolumeMounts[mountIndex]
		if mount.Name == volumeName && mount.MountPath == mountPath && !mount.ReadOnly {
			return true
		}
	}
	return false
}

func containsArgument(arguments []string, expected string) bool {
	for _, argument := range arguments {
		if argument == expected {
			return true
		}
	}
	return false
}
