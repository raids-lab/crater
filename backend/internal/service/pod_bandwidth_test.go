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

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestPodBandwidthConfigAndAnnotations(t *testing.T) {
	configService := newPodBandwidthTestConfigService(t, "pod_bandwidth_config")

	cfg, err := configService.GetPodBandwidthConfig(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	assertDefaultPodBandwidthConfig(t, cfg)
	annotations, err := NormalJobPodBandwidthAnnotations(
		t.Context(), configService, supportedFlannelClient(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if annotations != nil {
		t.Fatalf("disabled config should not return annotations: %#v", annotations)
	}

	if err := configService.UpdatePodBandwidthConfig(t.Context(), PodBandwidthConfig{
		Enabled:                   true,
		ModelDownloadBandwidth:    "100M",
		NormalJobIngressBandwidth: "200M",
		NormalJobEgressBandwidth:  "300M",
	}); err != nil {
		t.Fatal(err)
	}
	annotations, err = NormalJobPodBandwidthAnnotations(
		t.Context(), configService, supportedFlannelClient(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if annotations[podIngressBandwidthAnnotation] != "200M" ||
		annotations[podEgressBandwidthAnnotation] != "300M" {
		t.Fatalf("unexpected normal job bandwidth annotations: %#v", annotations)
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

func TestPodBandwidthConfigValidation(t *testing.T) {
	configService := newPodBandwidthTestConfigService(t, "pod_bandwidth_validation")
	invalidConfigs := []PodBandwidthConfig{
		{
			Enabled:                   true,
			ModelDownloadBandwidth:    "",
			NormalJobIngressBandwidth: "200M",
			NormalJobEgressBandwidth:  "300M",
		},
		{
			Enabled:                   true,
			ModelDownloadBandwidth:    "100M",
			NormalJobIngressBandwidth: "0M",
			NormalJobEgressBandwidth:  "300M",
		},
		{
			Enabled:                   true,
			ModelDownloadBandwidth:    "100M",
			NormalJobIngressBandwidth: "200M",
			NormalJobEgressBandwidth:  "1Gi",
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
		cfg.NormalJobIngressBandwidth != defaultPodBandwidth ||
		cfg.NormalJobEgressBandwidth != defaultPodBandwidth {
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
			want:       "does not install the bandwidth binary",
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
			Name: "install-bandwidth-plugin",
			Args: []string{"-f", "/opt/cni/bin/bandwidth", "/host/opt/cni/bin/bandwidth"},
		})
	}
	return fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: flannelConfigMapName, Namespace: flannelNamespace},
			Data:       map[string]string{"cni-conf.json": configJSON},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: flannelDaemonSetName, Namespace: flannelNamespace, Generation: 2,
			},
			Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{InitContainers: initContainers},
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
