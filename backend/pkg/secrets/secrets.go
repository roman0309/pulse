// Package secrets encrypts sensitive values (e.g. SSH credentials) at rest with
// AES-256-GCM, using a key derived from a passphrase.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Box encrypts with the primary key and decrypts with the primary key first,
// falling back to any legacy keys — so secrets written under an older key still
// open while new data uses the current one.
type Box struct{ keys [][32]byte }

// New derives the primary key from passphrase. Any legacy passphrases are used
// for decryption fallback only.
func New(passphrase string, legacy ...string) *Box {
	keys := [][32]byte{sha256.Sum256([]byte(passphrase))}
	for _, l := range legacy {
		if l != "" && l != passphrase {
			keys = append(keys, sha256.Sum256([]byte(l)))
		}
	}
	return &Box{keys: keys}
}

// Encrypt returns base64(nonce || ciphertext), using the primary key.
func (b *Box) Encrypt(plaintext string) (string, error) {
	gcm, err := newGCM(b.keys[0])
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt, trying the primary then any legacy keys.
func (b *Box) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	var lastErr error
	for _, key := range b.keys {
		gcm, err := newGCM(key)
		if err != nil {
			return "", err
		}
		if len(raw) < gcm.NonceSize() {
			return "", errors.New("ciphertext too short")
		}
		nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
		pt, err := gcm.Open(nil, nonce, ct, nil)
		if err == nil {
			return string(pt), nil
		}
		lastErr = err
	}
	return "", lastErr
}

func newGCM(key [32]byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
