package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

const aesKeyLength = 32

// 默认密钥：由于 AES 校验严格，这里凑齐了 32 位字符
const defaultSecretKey = "CRATER_SYSTEM_SECRET_KEY_DEFAULT" // #nosec G101

// 通过匿名函数初始化变量，避免使用 init 函数 (gochecknoinits)
var secretKey = func() []byte {
	key := os.Getenv("SYSTEM_SECRET_KEY")
	if key == "" {
		key = defaultSecretKey
	}

	keyBytes := []byte(key)

	// 此时如果长度不符合 AES 要求，进行截断或补齐处理（保证健壮性）
	if len(keyBytes) == aesKeyLength {
		return keyBytes
	}

	// 自动补齐或截断到 32 位
	finalKey := make([]byte, aesKeyLength)
	copy(finalKey, keyBytes)
	return finalKey
}()

// Encrypt AES-GCM 加密
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	block, err := aes.NewCipher(secretKey)
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
	if encryptedText == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secretKey)
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
