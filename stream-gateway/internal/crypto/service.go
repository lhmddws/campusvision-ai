package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	NonceSize = 12            // AES-GCM standard nonce size (96 bits)
	KeyLength = 32            // AES-256 requires 32-byte key
	EnvKey    = "CAMERA_ENCRYPTION_KEY"
)

var (
	ErrInvalidKeyLength = errors.New("crypto: key must be 32 bytes for AES-256")
	ErrInvalidNonceSize = errors.New("crypto: nonce must be 12 bytes")
	ErrInvalidBase64    = errors.New("crypto: invalid base64 encoding")
	ErrDecryptionFailed = errors.New("crypto: decryption failed")
)

// Service provides AES-256-GCM encryption and decryption for RTSP credentials.
// The zero value is not usable — use NewService or NewServiceWithKey.
type Service struct {
	masterKey []byte
	warnOnce  bool // dev key warning already emitted to avoid log spam
}

// devKey is the fallback key used when CAMERA_ENCRYPTION_KEY is not set.
// WARNING: This is for development only. Both modules now share the same
// dev key (01234567890123456789012345678901) and CAMERA_ENCRYPTION_KEY override.
var devKey = []byte("01234567890123456789012345678901")

// NewService creates a Service by reading CAMERA_ENCRYPTION_KEY from environment.
// In development mode when env var is empty, uses a hardcoded dev key with warning.
func NewService() (*Service, error) {
	keyStr := os.Getenv(EnvKey)
	if keyStr == "" {
		log.Printf("[WARN] crypto: %s not set, using DEV key (INSECURE — for development only)", EnvKey)
		return &Service{masterKey: devKey}, nil
	}
	key := []byte(keyStr)
	if len(key) != KeyLength {
		return nil, fmt.Errorf("%w: got %d bytes, need %d", ErrInvalidKeyLength, len(key), KeyLength)
	}
	return &Service{masterKey: key}, nil
}

// NewServiceWithKey creates a Service with an explicit key (for testing).
func NewServiceWithKey(key []byte) (*Service, error) {
	if len(key) != KeyLength {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKeyLength, len(key))
	}
	return &Service{masterKey: key}, nil
}

// warnIfDevKey logs a warning once if the service is using the hardcoded dev key.
func (s *Service) warnIfDevKey() {
	if !s.warnOnce && bytes.Equal(s.masterKey, devKey) {
		s.warnOnce = true
		log.Printf("[WARN] crypto: using DEV key for encryption/decryption (INSECURE — set %s for production)", EnvKey)
	}
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext and base64-encoded nonce.
// The nonce is randomly generated for each encryption.
func (s *Service) Encrypt(plaintext []byte) (ciphertextBase64, nonceBase64 string, err error) {
	s.warnIfDevKey()
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", "", fmt.Errorf("crypto: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	// ciphertext = gcm_ciphertext || gcm_tag (16 bytes)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
func (s *Service) Decrypt(ciphertextBase64, nonceBase64 string) ([]byte, error) {
	s.warnIfDevKey()
	nonce, err := decodeBase64(nonceBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid nonce: %v", ErrInvalidBase64, err)
	}

	if len(nonce) != NonceSize {
		return nil, ErrInvalidNonceSize
	}

	ciphertext, err := decodeBase64(ciphertextBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ciphertext: %v", ErrInvalidBase64, err)
	}

	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// EncryptPassword encrypts a password string. Convenience wrapper.
func (s *Service) EncryptPassword(password string) (encrypted, nonce string, err error) {
	return s.Encrypt([]byte(password))
}

// DecryptPassword decrypts to a password string. Convenience wrapper.
func (s *Service) DecryptPassword(encrypted, nonce string) (string, error) {
	plaintext, err := s.Decrypt(encrypted, nonce)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// decodeBase64 decodes a base64 string with backward compatibility.
// Prefers StdEncoding (current), falls back to RawStdEncoding (legacy).
func decodeBase64(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}
