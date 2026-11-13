package crypto

import "testing"

func TestExplicitKeyEncryptionSupportsPreviousKeys(t *testing.T) {
	t.Parallel()
	const (
		currentKey  = "0123456789abcdef0123456789abcdef"
		previousKey = "abcdef0123456789abcdef0123456789"
	)

	encrypted, err := EncryptWithKey("personal-api-key", previousKey)
	if err != nil {
		t.Fatalf("encrypt with previous key: %v", err)
	}
	decrypted, err := DecryptWithKeys(encrypted, currentKey, previousKey)
	if err != nil {
		t.Fatalf("decrypt with key ring: %v", err)
	}
	if decrypted != "personal-api-key" {
		t.Fatalf("decrypted value = %q", decrypted)
	}
}

func TestExplicitKeyEncryptionRejectsInvalidKey(t *testing.T) {
	t.Parallel()
	if _, err := EncryptWithKey("personal-api-key", "too-short"); err == nil {
		t.Fatal("expected invalid key length to fail")
	}
}
