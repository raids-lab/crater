package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

const aesKeyLength = 32

const (
	secretKeyEnv         = "SYSTEM_SECRET_KEY"
	previousSecretKeyEnv = "SYSTEM_SECRET_KEY_PREVIOUS"
	legacyDefaultKey     = "CRATER_SYSTEM_SECRET_KEY_DEFAULT" // #nosec G101 -- decrypt-only migration key.
)

var (
	errSecretKeyMissing = errors.New("SYSTEM_SECRET_KEY must be configured") //nolint:forbidigo // Low-level package error.
	errSecretKeyLength  = errors.New(                                        //nolint:forbidigo // Low-level package error.
		"encryption keys must contain exactly 32 bytes",
	)
	errDecryptFailed = errors.New( //nolint:forbidigo // Low-level package error.
		"ciphertext cannot be decrypted with the configured keys",
	)
)

// ValidateKeyConfiguration ensures new ciphertext never uses the historical public key.
func ValidateKeyConfiguration() error {
	_, err := parseSecretKey(strings.TrimSpace(os.Getenv(secretKeyEnv)))
	return err
}

// Encrypt AES-GCM 加密
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	key, err := currentSecretKey()
	if err != nil {
		return "", err
	}
	return encryptWithKey(plainText, key)
}

func encryptWithKey(plainText string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
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

// Decrypt AES-GCM 解密
func Decrypt(encryptedText string) (string, error) {
	plainText, _, err := DecryptWithRotation(encryptedText)
	return plainText, err
}

// DecryptWithRotation decrypts with the current key first and reports whether
// the ciphertext should be encrypted again with that key.
func DecryptWithRotation(encryptedText string) (plainText string, needsRotation bool, err error) {
	if encryptedText == "" {
		return "", false, nil
	}
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", false, err
	}
	keys, err := decryptionKeys()
	if err != nil {
		return "", false, err
	}
	for index, key := range keys {
		decrypted, decryptErr := decryptWithKey(data, key)
		if decryptErr == nil {
			return decrypted, index != 0, nil
		}
	}
	return "", false, errDecryptFailed
}

func decryptWithKey(data, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func currentSecretKey() ([]byte, error) {
	return parseSecretKey(strings.TrimSpace(os.Getenv(secretKeyEnv)))
}

func decryptionKeys() ([][]byte, error) {
	keys := make([][]byte, 0, 3)
	if currentValue := strings.TrimSpace(os.Getenv(secretKeyEnv)); currentValue != "" {
		current, err := parseSecretKey(currentValue)
		if err != nil {
			return nil, err
		}
		keys = append(keys, current)
	}
	for previous := range strings.SplitSeq(os.Getenv(previousSecretKeyEnv), ",") {
		previous = strings.TrimSpace(previous)
		if previous == "" {
			continue
		}
		key, parseErr := parseSecretKey(previous)
		if parseErr != nil {
			return nil, parseErr
		}
		keys = appendUniqueKey(keys, key)
	}
	return appendUniqueKey(keys, []byte(legacyDefaultKey)), nil
}

func parseSecretKey(value string) ([]byte, error) {
	if value == "" {
		return nil, errSecretKeyMissing
	}
	key := []byte(value)
	if len(key) != aesKeyLength {
		return nil, errSecretKeyLength
	}
	return key, nil
}

func appendUniqueKey(keys [][]byte, candidate []byte) [][]byte {
	for _, key := range keys {
		if bytes.Equal(key, candidate) {
			return keys
		}
	}
	return append(keys, candidate)
}
