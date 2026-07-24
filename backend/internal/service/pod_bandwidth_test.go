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
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

const (
	testJobIngressBandwidth = "200M"
	testJobEgressBandwidth  = "300M"
)

func TestPodBandwidthConfigAndAnnotations(t *testing.T) {
	configService := newPodBandwidthTestConfigService(t, "pod_bandwidth_config")

	cfg, err := configService.GetPodBandwidthConfig(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	assertDefaultPodBandwidthConfig(t, cfg)
	annotations, err := JobPodBandwidthAnnotations(
		t.Context(), configService, supportedFlannelClient(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if annotations != nil {
		t.Fatalf("disabled config should not return annotations: %#v", annotations)
	}

	if err := configService.UpdatePodBandwidthConfig(t.Context(), PodBandwidthConfig{
		Enabled:                true,
		ModelDownloadBandwidth: "100M",
		JobIngressBandwidth:    testJobIngressBandwidth,
		JobEgressBandwidth:     testJobEgressBandwidth,
	}); err != nil {
		t.Fatal(err)
	}
	annotations, err = JobPodBandwidthAnnotations(
		t.Context(), configService, supportedFlannelClient(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if annotations[podIngressBandwidthAnnotation] != testJobIngressBandwidth ||
		annotations[podEgressBandwidthAnnotation] != testJobEgressBandwidth {
		t.Fatalf("unexpected job bandwidth annotations: %#v", annotations)
	}

	annotations, err = ModelDownloadPodBandwidthAnnotations(
		t.Context(), configService, supportedFlannelClient(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if annotations[podIngressBandwidthAnnotation] != "100M" {
		t.Fatalf("unexpected model download bandwidth annotations: %#v", annotations)
	}
	if _, exists := annotations[podEgressBandwidthAnnotation]; exists {
		t.Fatalf("model download should not receive an egress limit: %#v", annotations)
	}
}

func TestApplyJobPodBandwidthUsesCurrentConfig(t *testing.T) {
	configService := newPodBandwidthTestConfigService(t, "apply_pod_bandwidth_config")
	if err := configService.UpdatePodBandwidthConfig(t.Context(), PodBandwidthConfig{
		Enabled:                true,
		ModelDownloadBandwidth: "100M",
		JobIngressBandwidth:    testJobIngressBandwidth,
		JobEgressBandwidth:     testJobEgressBandwidth,
	}); err != nil {
		t.Fatal(err)
	}

	job := &batch.Job{Spec: batch.JobSpec{Tasks: []batch.TaskSpec{
		{Template: corev1.PodTemplateSpec{}},
		{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			"existing":                    "value",
			podIngressBandwidthAnnotation: "stale",
			podEgressBandwidthAnnotation:  "stale",
		}}}},
	}}}
	if err := ApplyJobPodBandwidth(t.Context(), configService, supportedFlannelClient(), job); err != nil {
		t.Fatal(err)
	}
	for taskIndex, task := range job.Spec.Tasks {
		if task.Template.Annotations[podIngressBandwidthAnnotation] != testJobIngressBandwidth ||
			task.Template.Annotations[podEgressBandwidthAnnotation] != testJobEgressBandwidth {
			t.Fatalf("task %d annotations = %#v", taskIndex, task.Template.Annotations)
		}
	}
	if job.Spec.Tasks[1].Template.Annotations["existing"] != "value" {
		t.Fatal("existing pod annotations must be preserved")
	}

	if err := configService.UpdatePodBandwidthConfig(t.Context(), PodBandwidthConfig{
		Enabled:                false,
		ModelDownloadBandwidth: "100M",
		JobIngressBandwidth:    testJobIngressBandwidth,
		JobEgressBandwidth:     testJobEgressBandwidth,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyJobPodBandwidth(t.Context(), configService, supportedFlannelClient(), job); err != nil {
		t.Fatal(err)
	}
	for taskIndex, task := range job.Spec.Tasks {
		if _, exists := task.Template.Annotations[podIngressBandwidthAnnotation]; exists {
			t.Fatalf("task %d retained ingress annotation: %#v", taskIndex, task.Template.Annotations)
		}
		if _, exists := task.Template.Annotations[podEgressBandwidthAnnotation]; exists {
			t.Fatalf("task %d retained egress annotation: %#v", taskIndex, task.Template.Annotations)
		}
	}
}

func TestApplyJobPodBandwidthCoversNormalAndBackfillJobs(t *testing.T) {
	configService := newPodBandwidthTestConfigService(t, "pod_bandwidth_schedule_types")
	if err := configService.UpdatePodBandwidthConfig(t.Context(), PodBandwidthConfig{
		Enabled:                true,
		ModelDownloadBandwidth: "100M",
		JobIngressBandwidth:    testJobIngressBandwidth,
		JobEgressBandwidth:     testJobEgressBandwidth,
	}); err != nil {
		t.Fatal(err)
	}

	for _, scheduleType := range []model.ScheduleType{model.ScheduleTypeNormal, model.ScheduleTypeBackfill} {
		job := &batch.Job{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"crater.raids.io/schedule-type": scheduleType.String(),
			}},
			Spec: batch.JobSpec{Tasks: []batch.TaskSpec{{Template: corev1.PodTemplateSpec{}}}},
		}
		if err := ApplyJobPodBandwidth(t.Context(), configService, supportedFlannelClient(), job); err != nil {
			t.Fatal(err)
		}
		annotations := job.Spec.Tasks[0].Template.Annotations
		if annotations[podIngressBandwidthAnnotation] != testJobIngressBandwidth ||
			annotations[podEgressBandwidthAnnotation] != testJobEgressBandwidth {
			t.Fatalf("schedule type %s annotations = %#v", scheduleType.String(), annotations)
		}
	}
}

func TestPodBandwidthConfigValidation(t *testing.T) {
	configService := newPodBandwidthTestConfigService(t, "pod_bandwidth_validation")
	invalidConfigs := []PodBandwidthConfig{
		{
			Enabled:                true,
			ModelDownloadBandwidth: "",
			JobIngressBandwidth:    testJobIngressBandwidth,
			JobEgressBandwidth:     testJobEgressBandwidth,
		},
		{
			Enabled:                true,
			ModelDownloadBandwidth: "100M",
			JobIngressBandwidth:    "0M",
			JobEgressBandwidth:     testJobEgressBandwidth,
		},
		{
			Enabled:                true,
			ModelDownloadBandwidth: "100M",
			JobIngressBandwidth:    testJobIngressBandwidth,
			JobEgressBandwidth:     "1Gi",
		},
	}
	for _, cfg := range invalidConfigs {
		if err := configService.UpdatePodBandwidthConfig(t.Context(), cfg); err == nil {
			t.Fatalf("invalid bandwidth config should be rejected: %+v", cfg)
		}
	}
}

func newPodBandwidthTestConfigService(t *testing.T, databaseName string) *ConfigService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+databaseName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}); err != nil {
		t.Fatal(err)
	}
	return NewConfigService(query.Use(db))
}

func assertDefaultPodBandwidthConfig(t *testing.T, cfg *PodBandwidthConfig) {
	t.Helper()
	if cfg.Enabled ||
		cfg.ModelDownloadBandwidth != defaultPodBandwidth ||
		cfg.JobIngressBandwidth != defaultPodBandwidth ||
		cfg.JobEgressBandwidth != defaultPodBandwidth {
		t.Fatalf("unexpected default pod bandwidth config: %+v", cfg)
	}
}

func TestCheckFlannelBandwidthCNISupport(t *testing.T) {
	if err := CheckFlannelBandwidthCNISupport(t.Context(), supportedFlannelClient()); err != nil {
		t.Fatalf("supported Flannel configuration was rejected: %v", err)
	}

	tests := []struct {
		name       string
		configJSON string
		installer  bool
		ready      int32
		observed   int64
		mutate     func(*appsv1.DaemonSet)
		want       string
	}{
		{
			name:       "bandwidth plugin missing",
			configJSON: `{"plugins":[{"type":"flannel"}]}`,
			installer:  true,
			ready:      2,
			observed:   2,
			want:       "does not enable the bandwidth plugin",
		},
		{
			name:       "installer missing",
			configJSON: supportedCNIConfig,
			installer:  false,
			ready:      2,
			observed:   2,
			want:       "has no install-bandwidth-plugin initContainer",
		},
		{
			name:       "unrelated bandwidth argument is not an installer",
			configJSON: supportedCNIConfig,
			installer:  false,
			ready:      2,
			observed:   2,
			mutate: func(daemonSet *appsv1.DaemonSet) {
				daemonSet.Spec.Template.Spec.InitContainers = append(
					daemonSet.Spec.Template.Spec.InitContainers,
					corev1.Container{Name: "other", Args: []string{"/tmp/bandwidth"}},
				)
			},
			want: "has no install-bandwidth-plugin initContainer",
		},
		{
			name:       "installer destination mismatch",
			configJSON: supportedCNIConfig,
			installer:  true,
			ready:      2,
			observed:   2,
			mutate: func(daemonSet *appsv1.DaemonSet) {
				for index := range daemonSet.Spec.Template.Spec.InitContainers {
					container := &daemonSet.Spec.Template.Spec.InitContainers[index]
					if container.Name == flannelBandwidthInstallerContainerName {
						container.Args[len(container.Args)-1] = "/tmp/bandwidth"
					}
				}
			},
			want: "must copy",
		},
		{
			name:       "installer host mount missing",
			configJSON: supportedCNIConfig,
			installer:  true,
			ready:      2,
			observed:   2,
			mutate: func(daemonSet *appsv1.DaemonSet) {
				for index := range daemonSet.Spec.Template.Spec.InitContainers {
					container := &daemonSet.Spec.Template.Spec.InitContainers[index]
					if container.Name == flannelBandwidthInstallerContainerName {
						container.VolumeMounts = nil
					}
				}
			},
			want: "must mount host CNI directory",
		},
		{
			name:       "generation not observed",
			configJSON: supportedCNIConfig,
			installer:  true,
			ready:      2,
			observed:   1,
			want:       "rollout is incomplete",
		},
		{
			name:       "rollout incomplete",
			configJSON: supportedCNIConfig,
			installer:  true,
			ready:      1,
			observed:   2,
			want:       "rollout is incomplete",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := flannelClient(tc.configJSON, tc.installer, tc.ready, tc.observed)
			if tc.mutate != nil {
				daemonSet, err := client.AppsV1().DaemonSets(flannelNamespace).Get(
					t.Context(), flannelDaemonSetName, metav1.GetOptions{},
				)
				if err != nil {
					t.Fatal(err)
				}
				tc.mutate(daemonSet)
				if _, err := client.AppsV1().DaemonSets(flannelNamespace).Update(
					t.Context(), daemonSet, metav1.UpdateOptions{},
				); err != nil {
					t.Fatal(err)
				}
			}
			err := CheckFlannelBandwidthCNISupport(t.Context(), client)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CheckFlannelBandwidthCNISupport() error = %v, want %q", err, tc.want)
			}
		})
	}
}

const supportedCNIConfig = `{
  "plugins": [
    {"type": "flannel"},
    {"type": "bandwidth", "capabilities": {"bandwidth": true}}
  ]
}`

func supportedFlannelClient() *fake.Clientset {
	return flannelClient(supportedCNIConfig, true, 2, 2)
}

func flannelClient(configJSON string, installer bool, ready int32, observed int64) *fake.Clientset {
	initContainers := []corev1.Container{{
		Name: "install-cni-plugin", Args: []string{"-f", "/flannel", "/opt/cni/bin/flannel"},
	}}
	if installer {
		initContainers = append(initContainers, corev1.Container{
			Name:    flannelBandwidthInstallerContainerName,
			Command: []string{"cp"},
			Args: []string{
				"-f", flannelBandwidthBinarySource, flannelBandwidthBinaryDestination,
			},
			VolumeMounts: []corev1.VolumeMount{{
				Name: "cni-plugin", MountPath: flannelCNIBinVolumeMountPath,
			}},
		})
	}
	return fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: flannelConfigMapName, Namespace: flannelNamespace},
			Data:       map[string]string{flannelCNIConfigKey: configJSON},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: flannelDaemonSetName, Namespace: flannelNamespace, Generation: 2,
			},
			Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Volumes: []corev1.Volume{{
						Name: "cni-plugin",
						VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
							Path: flannelCNIBinVolumeHostPath,
						}},
					}},
				},
			}},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				UpdatedNumberScheduled: 2,
				NumberReady:            ready,
				ObservedGeneration:     observed,
			},
		},
	)
}
