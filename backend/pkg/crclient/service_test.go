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
