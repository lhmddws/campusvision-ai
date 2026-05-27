package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
)

func init() {
	// Auto-read CAMERA_ENCRYPTION_KEY from env, matching stream-gateway's pattern.
	if keyStr := os.Getenv("CAMERA_ENCRYPTION_KEY"); keyStr != "" {
		key := []byte(keyStr)
		if len(key) == 32 {
			encryptionKey = key
			log.Printf("[INFO] crypto: loaded AES-256 key from CAMERA_ENCRYPTION_KEY")
		} else {
			log.Printf("[WARN] crypto: CAMERA_ENCRYPTION_KEY is not 32 bytes (got %d), using dev key", len(key))
		}
	} else {
		log.Printf("[WARN] crypto: CAMERA_ENCRYPTION_KEY not set, using DEV key (INSECURE — for development only)")
	}
}

// Default encryption key (32 bytes for AES-256).
// DEV KEY: Must match stream-gateway's dev key so passwords can be decrypted across modules.
// In production, set via SetEncryptionKey at startup or CAMERA_ENCRYPTION_KEY env var.
var encryptionKey = []byte("01234567890123456789012345678901")

// SetEncryptionKey overrides the default AES-256 key.
// The key must be exactly 32 bytes.
func SetEncryptionKey(key []byte) {
	if len(key) == 32 {
		encryptionKey = key
	}
}

// EncryptedPassword holds the AES-GCM encrypted password and its nonce.
type EncryptedPassword struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
}

// EncryptPassword encrypts a plaintext string using AES-256-GCM.
// Returns base64-encoded ciphertext and nonce.
func EncryptPassword(plaintext string) (*EncryptedPassword, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)

	return &EncryptedPassword{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

// DecryptPassword decrypts a base64-encoded ciphertext and nonce.
func DecryptPassword(ciphertext, nonce string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}

	plaintextBytes, err := aead.Open(nil, nonceBytes, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintextBytes), nil
}
