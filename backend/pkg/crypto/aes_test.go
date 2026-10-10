package crypto

import "testing"

const (
	testCurrentKey  = "0123456789abcdef0123456789abcdef"
	testPreviousKey = "abcdef0123456789abcdef0123456789"
)

func TestValidateKeyConfigurationRequiresConfiguredKey(t *testing.T) {
	t.Setenv(secretKeyEnv, "")
	if err := ValidateKeyConfiguration(); err == nil {
		t.Fatal("expected missing SYSTEM_SECRET_KEY to fail validation")
	}
	if _, err := Encrypt("secret"); err == nil {
		t.Fatal("expected encryption without SYSTEM_SECRET_KEY to fail")
	}
}

func TestDecryptWithRotationUsesPreviousKey(t *testing.T) {
	t.Setenv(secretKeyEnv, testPreviousKey)
	legacyCiphertext, err := Encrypt("secret")
	if err != nil {
		t.Fatalf("encrypt with previous key: %v", err)
	}

	t.Setenv(secretKeyEnv, testCurrentKey)
	t.Setenv(previousSecretKeyEnv, testPreviousKey)
	plainText, needsRotation, err := DecryptWithRotation(legacyCiphertext)
	if err != nil {
		t.Fatalf("decrypt with previous key: %v", err)
	}
	if plainText != "secret" || !needsRotation {
		t.Fatalf("decryption result = %q/%t, want secret/true", plainText, needsRotation)
	}
}

func TestDecryptWithRotationMigratesLegacyDefault(t *testing.T) {
	legacyCiphertext, err := encryptWithKey("legacy-secret", []byte(legacyDefaultKey))
	if err != nil {
		t.Fatalf("encrypt with legacy key: %v", err)
	}
	t.Setenv(secretKeyEnv, testCurrentKey)
	t.Setenv(previousSecretKeyEnv, "")

	plainText, needsRotation, err := DecryptWithRotation(legacyCiphertext)
	if err != nil {
		t.Fatalf("decrypt with legacy key: %v", err)
	}
	if plainText != "legacy-secret" || !needsRotation {
		t.Fatalf("decryption result = %q/%t, want legacy-secret/true", plainText, needsRotation)
	}
}
