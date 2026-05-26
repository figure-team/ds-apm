package secretbox_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/stretchr/testify/require"
)

// newTestCipher creates a Cipher backed by a freshly generated 32-byte key.
func newTestCipher(t *testing.T) *secretbox.Cipher {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	c, err := secretbox.New(base64.StdEncoding.EncodeToString(key))
	require.NoError(t, err)
	return c
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	c := newTestCipher(t)
	plaintext := "sk-supersecretapikey-1234567890"

	enc, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	require.NotEmpty(t, enc)
	require.NotEqual(t, plaintext, enc)

	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	require.Equal(t, plaintext, dec)
}

func TestEncrypt_EmptyPlaintext_ReturnsEmpty(t *testing.T) {
	c := newTestCipher(t)

	enc, err := c.Encrypt("")
	require.NoError(t, err)
	require.Equal(t, "", enc)

	dec, err := c.Decrypt("")
	require.NoError(t, err)
	require.Equal(t, "", dec)
}

func TestEncrypt_DistinctNonces(t *testing.T) {
	c := newTestCipher(t)
	plaintext := "same-plaintext-encrypted-twice"

	enc1, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	enc2, err := c.Encrypt(plaintext)
	require.NoError(t, err)

	// Different nonces → different ciphertexts even for identical input.
	require.NotEqual(t, enc1, enc2)

	// Both must still decrypt correctly.
	dec1, err := c.Decrypt(enc1)
	require.NoError(t, err)
	require.Equal(t, plaintext, dec1)

	dec2, err := c.Decrypt(enc2)
	require.NoError(t, err)
	require.Equal(t, plaintext, dec2)
}

func TestDecrypt_TamperedCiphertext_Errors(t *testing.T) {
	c := newTestCipher(t)

	enc, err := c.Encrypt("sensitive-key")
	require.NoError(t, err)

	// Decode, flip a byte in the ciphertext portion, re-encode.
	raw, err := base64.StdEncoding.DecodeString(enc)
	require.NoError(t, err)
	// Flip the last byte (authtag area).
	raw[len(raw)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(raw)

	_, err = c.Decrypt(tampered)
	require.Error(t, err)
}

func TestNew_RejectsShortKey(t *testing.T) {
	key := make([]byte, 16) // AES-128, not AES-256
	_, err := rand.Read(key)
	require.NoError(t, err)

	_, err = secretbox.New(base64.StdEncoding.EncodeToString(key))
	require.Error(t, err)
}

func TestNew_RejectsBadBase64(t *testing.T) {
	_, err := secretbox.New("this is not valid base64!!!")
	require.Error(t, err)
}

func TestNew_RejectsEmptyKey(t *testing.T) {
	_, err := secretbox.New("")
	require.ErrorIs(t, err, secretbox.ErrMissingKey)
}

func TestPlaintextCipher_Roundtrip(t *testing.T) {
	c := secretbox.PlaintextCipher()

	// Non-empty value: Encrypt returns the input unchanged.
	enc, err := c.Encrypt("hello")
	require.NoError(t, err)
	require.Equal(t, "hello", enc)

	dec, err := c.Decrypt("hello")
	require.NoError(t, err)
	require.Equal(t, "hello", dec)

	// Empty value: both return "" without error.
	enc, err = c.Encrypt("")
	require.NoError(t, err)
	require.Equal(t, "", enc)

	dec, err = c.Decrypt("")
	require.NoError(t, err)
	require.Equal(t, "", dec)
}

func TestFromEnv_NoEnv_ReturnsPlaintext(t *testing.T) {
	t.Setenv("DS_APM_AI_CONFIG_ENCRYPTION_KEY", "")

	c, insecure, err := secretbox.FromEnv()
	require.NoError(t, err)
	require.True(t, insecure, "expected insecure=true when env is unset")
	require.NotNil(t, c)

	// PlaintextCipher must work as identity.
	enc, err := c.Encrypt("test-value")
	require.NoError(t, err)
	require.Equal(t, "test-value", enc)
}

func TestFromEnv_ValidEnv_ReturnsRealCipher(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	t.Setenv("DS_APM_AI_CONFIG_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))

	c, insecure, err := secretbox.FromEnv()
	require.NoError(t, err)
	require.False(t, insecure, "expected insecure=false when valid key is set")
	require.NotNil(t, c)

	// Verify it is a real AES cipher: output must differ from input.
	enc, err := c.Encrypt("api-key-value")
	require.NoError(t, err)
	require.NotEqual(t, "api-key-value", enc)

	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	require.Equal(t, "api-key-value", dec)
}

func TestFromEnv_BadBase64_Errors(t *testing.T) {
	t.Setenv("DS_APM_AI_CONFIG_ENCRYPTION_KEY", "!!!not-base64!!!")

	_, _, err := secretbox.FromEnv()
	require.Error(t, err)
}
