/*
Copyright 2026 The Crater Project Team, RAIDS-Lab

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import "testing"

func TestTensorboardConfigValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     TensorboardConfig
		wantErrors int
	}{
		{name: "omitted", config: TensorboardConfig{}, wantErrors: 0},
		{
			name: "valid",
			config: TensorboardConfig{
				Image:            "registry.example.com/tensorboard:2.20.0",
				ImagePullPolicy:  "IfNotPresent",
				ImagePullSecrets: []ImagePullSecret{{Name: "registry-secret"}},
			},
			wantErrors: 0,
		},
		{
			name:       "missing image",
			config:     TensorboardConfig{ImagePullPolicy: "Always"},
			wantErrors: 1,
		},
		{
			name:       "invalid pull policy",
			config:     TensorboardConfig{Image: "tensorboard:latest", ImagePullPolicy: "Sometimes"},
			wantErrors: 1,
		},
		{
			name: "empty pull secret",
			config: TensorboardConfig{
				Image:            "tensorboard:latest",
				ImagePullPolicy:  "Never",
				ImagePullSecrets: []ImagePullSecret{{}},
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := len(tt.config.validationErrors()); got != tt.wantErrors {
				t.Fatalf("validation error count = %d, want %d", got, tt.wantErrors)
			}
		})
	}
}

func TestTensorboardIngressAuthDefaultsToEnabled(t *testing.T) {
	t.Parallel()

	defaultConfig := TensorboardConfig{}
	if !defaultConfig.IsIngressAuthEnabled() {
		t.Fatal("TensorBoard Ingress authentication must default to enabled")
	}

	disabled := false
	config := TensorboardConfig{IngressAuthEnabled: &disabled}
	if config.IsIngressAuthEnabled() {
		t.Fatal("TensorBoard Ingress authentication should honor an explicit local disable")
	}
}
