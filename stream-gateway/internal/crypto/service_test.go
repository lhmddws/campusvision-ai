package crypto

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

// testKey is a valid 32-byte key for testing.
var testKey = []byte("abcdefghijklmnopqrstuvwxyz123456")

// testKey2 is a different valid 32-byte key for testing wrong-key scenarios.
var testKey2 = []byte("1234567890abcdefghijklmnopqrstuv")

func TestNewServiceWithKey(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatalf("NewServiceWithKey with valid key should not error: %v", err)
	}
	if s == nil {
		t.Fatal("NewServiceWithKey should return non-nil service")
	}
}

func TestNewServiceWithKey_WrongLength(t *testing.T) {
	_, err := NewServiceWithKey([]byte("short"))
	if err == nil {
		t.Fatal("NewServiceWithKey with 5-byte key should error")
	}
	if !strings.Contains(err.Error(), "key must be 32 bytes") {
		t.Fatalf("unexpected error message: %v", err)
	}

	_, err = NewServiceWithKey([]byte("1234567890123456")) // 16 bytes
	if err == nil {
		t.Fatal("NewServiceWithKey with 16-byte key should error")
	}
}

func TestNewService_EnvVar(t *testing.T) {
	const testEnvValue = "test_key_32_bytes_abcdefgh123456"
	os.Setenv(EnvKey, testEnvValue)
	defer os.Unsetenv(EnvKey)

	s, err := NewService()
	if err != nil {
		t.Fatalf("NewService with valid env var should not error: %v", err)
	}
	if s == nil {
		t.Fatal("NewService should return non-nil service")
	}

	plaintext := []byte("password123")
	ct, nonce, err := s.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := s.Decrypt(ct, nonce)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDevModeFallback(t *testing.T) {
	os.Unsetenv(EnvKey)

	s, err := NewService()
	if err != nil {
		t.Fatalf("NewService in dev mode should not error: %v", err)
	}
	if s == nil {
		t.Fatal("NewService should return non-nil service")
	}

	plaintext := []byte("dev-password")
	ct, nonce, err := s.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed in dev mode: %v", err)
	}

	decrypted, err := s.Decrypt(ct, nonce)
	if err != nil {
		t.Fatalf("Decrypt failed in dev mode: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestRoundTrip(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	passwords := []string{"admin123", "", "a", "password with spaces and !@#$%", "very long password that spans multiple blocks because aes blocks are 16 bytes each and we need to test gcm with more than one block of plaintext data to ensure the streaming mode works correctly"}
	for _, pw := range passwords {
		t.Run(pw, func(t *testing.T) {
			if pw == "" {
				t.Run("empty", func(t *testing.T) {
					ct, nonce, err := s.EncryptPassword(pw)
					if err != nil {
						t.Fatalf("EncryptPassword failed: %v", err)
					}
					decrypted, err := s.DecryptPassword(ct, nonce)
					if err != nil {
						t.Fatalf("DecryptPassword failed: %v", err)
					}
					if decrypted != pw {
						t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted, pw)
					}
				})
				return
			}
			ct, nonce, err := s.EncryptPassword(pw)
			if err != nil {
				t.Fatalf("EncryptPassword failed: %v", err)
			}
			if ct == "" || nonce == "" {
				t.Fatal("EncryptPassword returned empty strings")
			}
			decrypted, err := s.DecryptPassword(ct, nonce)
			if err != nil {
				t.Fatalf("DecryptPassword failed: %v", err)
			}
			if decrypted != pw {
				t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted, pw)
			}
		})
	}
}

func TestWrongKey(t *testing.T) {
	s1, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewServiceWithKey(testKey2)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("admin123")
	ct, nonce, err := s1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = s2.Decrypt(ct, nonce)
	if err == nil {
		t.Fatal("Decrypt with wrong key should error")
	}
	if !strings.Contains(err.Error(), "decryption failed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestInvalidBase64(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		ct     string
		nonce  string
		errMsg string
	}{
		{
			name:   "invalid ciphertext base64",
			ct:     "!!!not-base64!!!",
			nonce:  base64.RawStdEncoding.EncodeToString(make([]byte, NonceSize)),
			errMsg: "invalid base64",
		},
		{
			name:   "invalid nonce base64",
			ct:     base64.RawStdEncoding.EncodeToString([]byte("ciphertext")),
			nonce:  "!!!not-base64!!!",
			errMsg: "invalid base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Decrypt(tt.ct, tt.nonce)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Fatalf("expected error containing %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestDecrypt_EmptyCiphertext(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	// Empty string is valid base64 (decodes to empty bytes), so it bypasses
	// the base64 check and hits GCM authentication, which fails.
	nonce := base64.RawStdEncoding.EncodeToString(make([]byte, NonceSize))
	_, err = s.Decrypt("", nonce)
	if err == nil {
		t.Fatal("decrypt empty ciphertext should error")
	}
	if err != ErrDecryptionFailed && !strings.Contains(err.Error(), "decryption failed") {
		t.Fatalf("expected ErrDecryptionFailed, got: %v", err)
	}
}

func TestInvalidNonceLength(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	shortNonce := base64.RawStdEncoding.EncodeToString(make([]byte, 4))  // 4 bytes, not 12
	longNonce := base64.RawStdEncoding.EncodeToString(make([]byte, 16))  // 16 bytes, not 12
	ct := base64.RawStdEncoding.EncodeToString([]byte("some-ciphertext"))

	tests := []struct {
		name  string
		nonce string
	}{
		{"short nonce (4 bytes)", shortNonce},
		{"long nonce (16 bytes)", longNonce},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Decrypt(ct, tt.nonce)
			if err == nil {
				t.Fatal("expected error for invalid nonce length")
			}
			if err != ErrInvalidNonceSize {
				t.Fatalf("expected ErrInvalidNonceSize, got: %v", err)
			}
		})
	}
}

func TestNonceUniqueness(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("same plaintext each time")
	seen := make(map[string]bool)

	for i := 0; i < 10; i++ {
		_, nonce, err := s.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt failed at iteration %d: %v", i, err)
		}
		if seen[nonce] {
			t.Fatalf("duplicate nonce detected at iteration %d: %s", i, nonce)
		}
		seen[nonce] = true
	}
}

func TestEncryptDecryptPasswords(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	ct, nonce, err := s.EncryptPassword("supersecret")
	if err != nil {
		t.Fatalf("EncryptPassword failed: %v", err)
	}
	if ct == "" || nonce == "" {
		t.Fatal("EncryptPassword returned empty result")
	}

	decrypted, err := s.DecryptPassword(ct, nonce)
	if err != nil {
		t.Fatalf("DecryptPassword failed: %v", err)
	}
	if decrypted != "supersecret" {
		t.Fatalf("got %q, want %q", decrypted, "supersecret")
	}
}

func TestEncryptOutputFormat(t *testing.T) {
	s, err := NewServiceWithKey(testKey)
	if err != nil {
		t.Fatal(err)
	}

	ct, nonce, err := s.Encrypt([]byte("test"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify base64 can decode (ensures RawStdEncoding was used, not StdEncoding with padding)
	ctBytes, err := base64.RawStdEncoding.DecodeString(ct)
	if err != nil {
		t.Fatalf("ciphertext not valid RawStdEncoding: %v", err)
	}

	nonceBytes, err := base64.RawStdEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce not valid RawStdEncoding: %v", err)
	}

	// Verify nonce size
	if len(nonceBytes) != NonceSize {
		t.Fatalf("nonce size mismatch: got %d, want %d", len(nonceBytes), NonceSize)
	}

	// Verify ciphertext contains tag (ciphertext should be longer than plaintext due to GCM tag)
	if len(ctBytes) < len("test") {
		t.Fatal("ciphertext too short, missing GCM tag?")
	}

	// Verify StdEncoding would fail (no padding characters)
	if strings.Contains(ct, "=") || strings.Contains(nonce, "=") {
		t.Fatal("base64 padding detected - expected RawStdEncoding")
	}
}
