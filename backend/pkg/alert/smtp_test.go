package alert

import "testing"

func TestSMTPAlerterNewDialer(t *testing.T) {
	tests := []struct {
		name               string
		insecureSkipVerify bool
	}{
		{name: "verify TLS certificates"},
		{name: "skip TLS certificate verification", insecureSkipVerify: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &SMTPAlerter{
				host:               "mail.example.com",
				port:               587,
				insecureSkipVerify: tt.insecureSkipVerify,
			}

			d := sa.newDialer()
			if d.SSL {
				t.Fatal("expected STARTTLS instead of implicit TLS")
			}
			if !tt.insecureSkipVerify {
				if d.TLSConfig != nil {
					t.Fatal("expected the default TLS verification configuration")
				}
				return
			}
			if d.TLSConfig == nil {
				t.Fatal("expected a custom TLS configuration")
			}
			if !d.TLSConfig.InsecureSkipVerify {
				t.Fatal("expected TLS certificate verification to be disabled")
			}
			if d.TLSConfig.ServerName != sa.host {
				t.Fatalf("expected TLS server name %q, got %q", sa.host, d.TLSConfig.ServerName)
			}
		})
	}
}
