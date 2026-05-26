// Package secretbox provides AES-256-GCM encryption helpers for the DS-APM
// AI config store. API keys persisted to the database are encrypted at rest
// using a 32-byte master key supplied via the environment variable
// DS_APM_AI_CONFIG_ENCRYPTION_KEY (base64-encoded).
//
// Wire-format of an encrypted value:
//
//	base64StdEncoding( nonce(12 bytes) || ciphertext+authtag )
//
// A PlaintextCipher is available for development / demo environments where
// no master key is configured; see PlaintextCipher for caveats.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// nonceSize is the standard GCM nonce length.
	nonceSize = 12

	// envKey is the environment variable that holds the base64-encoded 32-byte
	// master key.
	envKey = "DS_APM_AI_CONFIG_ENCRYPTION_KEY"
)

// ErrMissingKey is returned by New when the supplied master key is empty.
var ErrMissingKey = errors.New("secretbox: master key not configured")

// Cipher holds an AES-256-GCM cipher initialised from a 32-byte master key.
// The zero value is not usable; construct via New, PlaintextCipher, or FromEnv.
// A Cipher is safe for concurrent use by multiple goroutines.
type Cipher struct {
	gcm       cipher.AEAD // nil when plaintext mode is active
	plaintext bool        // true → identity Encrypt/Decrypt (dev/demo only)
}

// New constructs a Cipher backed by AES-256-GCM.
//
// masterKeyBase64 must be the standard base64 encoding of exactly 32 bytes.
// Returns ErrMissingKey if masterKeyBase64 is the empty string.
func New(masterKeyBase64 string) (*Cipher, error) {
	if masterKeyBase64 == "" {
		return nil, ErrMissingKey
	}

	raw, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("secretbox: invalid base64 master key: %w", err)
	}

	if len(raw) != 32 {
		return nil, fmt.Errorf("secretbox: master key must be 32 bytes (AES-256), got %d", len(raw))
	}

	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, fmt.Errorf("secretbox: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: cipher.NewGCM: %w", err)
	}

	return &Cipher{gcm: gcm}, nil
}

// PlaintextCipher returns a no-op Cipher whose Encrypt and Decrypt are
// identity functions.
//
// WARNING: API keys stored through this cipher are persisted to the database
// in plaintext. Use only in development / demo deployments where no master key
// is available. The caller should log a startup warning whenever this fallback
// is active (FromEnv signals this via its bool return value).
//
// PlaintextCipher satisfies the same *Cipher type as New so that downstream
// code (Upsert/Get closures) is uniform regardless of whether encryption is
// enabled.
func PlaintextCipher() *Cipher {
	return &Cipher{plaintext: true}
}

// FromEnv constructs a Cipher from the environment variable
// DS_APM_AI_CONFIG_ENCRYPTION_KEY.
//
//   - If the variable is unset or empty, returns (PlaintextCipher(), true, nil).
//     The bool signals "running insecurely"; the caller should log a warning.
//   - If the variable is set but contains an invalid value (bad base64 or wrong
//     length), returns (nil, false, error).
//   - If the variable is valid, returns (cipher, false, nil).
func FromEnv() (*Cipher, bool, error) {
	val := os.Getenv(envKey)
	if val == "" {
		return PlaintextCipher(), true, nil
	}

	c, err := New(val)
	if err != nil {
		return nil, false, err
	}

	return c, false, nil
}

// Encrypt encrypts plaintext and returns base64(nonce || ciphertext+authtag).
// Empty plaintext returns "" with no error so that callers can store the result
// directly without special-casing empty API keys.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	if c.plaintext {
		return plaintext, nil
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secretbox: rand nonce: %w", err)
	}

	// Seal appends ciphertext+tag to nonce, so the resulting slice is
	// nonce || ciphertext || tag — exactly the wire format we want.
	sealed := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Empty input returns "" with no error.
func (c *Cipher) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	if c.plaintext {
		return ciphertext, nil
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("secretbox: base64 decode: %w", err)
	}

	if len(raw) < nonceSize {
		return "", fmt.Errorf("secretbox: ciphertext too short (%d bytes)", len(raw))
	}

	nonce, data := raw[:nonceSize], raw[nonceSize:]

	plain, err := c.gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", fmt.Errorf("secretbox: decrypt: %w", err)
	}

	return string(plain), nil
}

// EncryptFunc returns a closure with signature func(string) (string, error)
// that delegates to c.Encrypt. Convenient for passing to AIConfigStore
// Upsert helpers.
func (c *Cipher) EncryptFunc() func(string) (string, error) {
	return c.Encrypt
}

// DecryptFunc returns a closure with signature func(string) (string, error)
// that delegates to c.Decrypt. Convenient for passing to AIConfigStore
// Get helpers.
func (c *Cipher) DecryptFunc() func(string) (string, error) {
	return c.Decrypt
}
