package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	errKeyLength     = errors.New("encryption keys must contain exactly 32 bytes") //nolint:forbidigo // Stable package error.
	errDecryptFailed = errors.New(                                                 //nolint:forbidigo // Stable package error.
		"ciphertext cannot be decrypted with the configured keys",
	)
)

// ValidateAES256Key checks a caller-provided AES-256 key without reading global configuration.
func ValidateAES256Key(key string) error {
	if len(key) != aesKeyLength {
		return errKeyLength
	}
	return nil
}

// EncryptWithKey encrypts plaintext with an explicit AES-256 key.
func EncryptWithKey(plainText, key string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	if err := ValidateAES256Key(key); err != nil {
		return "", err
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithKeys tries explicit AES-256 keys in order without modifying stored data.
func DecryptWithKeys(encryptedText string, keys ...string) (string, error) {
	if encryptedText == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}
	for _, key := range keys {
		if err := ValidateAES256Key(key); err != nil {
			return "", err
		}
		block, cipherErr := aes.NewCipher([]byte(key))
		if cipherErr != nil {
			return "", cipherErr
		}
		aesGCM, cipherErr := cipher.NewGCM(block)
		if cipherErr != nil {
			return "", cipherErr
		}
		if len(data) < aesGCM.NonceSize() {
			return "", errors.New("ciphertext too short") //nolint:forbidigo // Invalid encrypted payload.
		}
		if plaintext, openErr := aesGCM.Open(
			nil,
			data[:aesGCM.NonceSize()],
			data[aesGCM.NonceSize():],
			nil,
		); openErr == nil {
			return string(plaintext), nil
		}
	}
	return "", errDecryptFailed
}
