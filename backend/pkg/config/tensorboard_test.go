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
