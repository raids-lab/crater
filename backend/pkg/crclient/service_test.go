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

package crclient

import "testing"

func TestIngressAnnotationsIncludesCustomAuthentication(t *testing.T) {
	annotations := ingressAnnotations("http", []IngressOptions{{
		Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/auth-url":    "https://$host/api/tensorboard/auth",
			"nginx.ingress.kubernetes.io/auth-method": "GET",
		},
	}})

	if got := annotations[AnnotationKeyPortName]; got != "http" {
		t.Fatalf("port annotation = %q, want http", got)
	}
	if got := annotations["nginx.ingress.kubernetes.io/auth-url"]; got != "https://$host/api/tensorboard/auth" {
		t.Fatalf("auth-url annotation = %q", got)
	}
	if got := annotations["nginx.ingress.kubernetes.io/auth-method"]; got != "GET" {
		t.Fatalf("auth-method annotation = %q, want GET", got)
	}
	if got := annotations["nginx.ingress.kubernetes.io/ssl-redirect"]; got != "true" {
		t.Fatalf("default ssl-redirect annotation = %q, want true", got)
	}
}

func TestIngressAnnotationsAllowsExplicitOverrides(t *testing.T) {
	annotations := ingressAnnotations("http", []IngressOptions{{
		Annotations: map[string]string{
			"nginx.ingress.kubernetes.io/proxy-read-timeout": "600",
		},
	}})

	if got := annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"]; got != "600" {
		t.Fatalf("proxy-read-timeout annotation = %q, want 600", got)
	}
}
