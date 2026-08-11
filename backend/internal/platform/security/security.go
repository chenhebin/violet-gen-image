package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

func RandomToken(byteLength int) (string, error) {
	if byteLength < 16 {
		return "", errors.New("token length must be at least 16 bytes")
	}
	buffer := make([]byte, byteLength)
	if _, err := io.ReadFull(rand.Reader, buffer); err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func HMACDigest(value, pepper string) string {
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func EqualDigest(left, right string) bool {
	return hmac.Equal([]byte(left), []byte(right))
}

func Encrypt(plaintext []byte, keyMaterial string) ([]byte, error) {
	key := sha256.Sum256([]byte(keyMaterial))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate encryption nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte, keyMaterial string) ([]byte, error) {
	key := sha256.Sum256([]byte(keyMaterial))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext is too short")
	}
	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, errors.New("decrypt protected value")
	}
	return plaintext, nil
}

func NormalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NormalizeRedemptionCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func MaskSecret(value string) string {
	runes := []rune(value)
	if len(runes) <= 8 {
		return "********"
	}
	return string(runes[:4]) + strings.Repeat("*", min(12, len(runes)-8)) + string(runes[len(runes)-4:])
}

func HashMetadata(value, pepper string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return HMACDigest(value, pepper)
}
