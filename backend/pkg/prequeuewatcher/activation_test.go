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

package prequeuewatcher

import (
	"testing"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/service"
)

func TestRestoreJobForActivationUsesLatestBandwidthConfig(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:prequeue_bandwidth?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}, &model.PrequeueConfig{}); err != nil {
		t.Fatal(err)
	}
	configService := service.NewConfigService(query.Use(db))
	if err := configService.UpdatePodBandwidthConfig(t.Context(), service.PodBandwidthConfig{
		Enabled:                true,
		ModelDownloadBandwidth: "100M",
		JobIngressBandwidth:    "200M",
		JobEgressBandwidth:     "300M",
	}); err != nil {
		t.Fatal(err)
	}

	storedJob := &batch.Job{Spec: batch.JobSpec{Tasks: []batch.TaskSpec{{
		Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			"kubernetes.io/ingress-bandwidth": "10M",
			"kubernetes.io/egress-bandwidth":  "20M",
		}}},
	}}}}
	record := &model.Job{Attributes: datatypes.NewJSONType(storedJob)}
	watcher := &PrequeueWatcher{
		configService: configService,
		kubeClient:    supportedBandwidthFlannelClient(),
	}

	restored, err := watcher.restoreJobForActivation(t.Context(), record)
	if err != nil {
		t.Fatal(err)
	}
	annotations := restored.Spec.Tasks[0].Template.Annotations
	if annotations["kubernetes.io/ingress-bandwidth"] != "200M" ||
		annotations["kubernetes.io/egress-bandwidth"] != "300M" {
		t.Fatalf("activation annotations = %#v, want latest 200M/300M", annotations)
	}

	watcher.kubeClient = fake.NewSimpleClientset()
	if _, err := watcher.restoreJobForActivation(t.Context(), record); err == nil {
		t.Fatal("activation must recheck the current CNI capability")
	}
	watcher.kubeClient = supportedBandwidthFlannelClient()

	if err := configService.UpdatePodBandwidthConfig(t.Context(), service.PodBandwidthConfig{
		Enabled:                false,
		ModelDownloadBandwidth: "100M",
		JobIngressBandwidth:    "200M",
		JobEgressBandwidth:     "300M",
	}); err != nil {
		t.Fatal(err)
	}
	restored, err = watcher.restoreJobForActivation(t.Context(), record)
	if err != nil {
		t.Fatal(err)
	}
	annotations = restored.Spec.Tasks[0].Template.Annotations
	if _, exists := annotations["kubernetes.io/ingress-bandwidth"]; exists {
		t.Fatalf("disabled configuration retained ingress annotation: %#v", annotations)
	}
	if _, exists := annotations["kubernetes.io/egress-bandwidth"]; exists {
		t.Fatalf("disabled configuration retained egress annotation: %#v", annotations)
	}
}

func supportedBandwidthFlannelClient() *fake.Clientset {
	return fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-flannel-cfg", Namespace: "kube-flannel"},
			Data: map[string]string{"cni-conf.json": `{
              "plugins": [
                {"type": "flannel"},
                {"type": "bandwidth", "capabilities": {"bandwidth": true}}
              ]
            }`},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-flannel-ds", Namespace: "kube-flannel", Generation: 1,
			},
			Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{
					Name:    "install-bandwidth-plugin",
					Command: []string{"cp"},
					Args: []string{
						"-f", "/opt/cni/bin/bandwidth", "/host/opt/cni/bin/bandwidth",
					},
					VolumeMounts: []corev1.VolumeMount{{
						Name: "cni-plugin", MountPath: "/host/opt/cni/bin",
					}},
				}},
				Volumes: []corev1.Volume{{
					Name: "cni-plugin",
					VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
						Path: "/opt/cni/bin",
					}},
				}},
			}}},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 1,
				UpdatedNumberScheduled: 1,
				NumberReady:            1,
				ObservedGeneration:     1,
			},
		},
	)
}
