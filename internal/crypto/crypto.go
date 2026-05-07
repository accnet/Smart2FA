package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen  = 16
	keyLen   = 32
	nonceLen = 12
	// Argon2id params — balanced for speed since security is not critical
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
)

// VaultHash returns SHA256(phrase:passcode) used as vault identifier.
func VaultHash(phrase, passcode string) []byte {
	h := sha256.Sum256([]byte(phrase + ":" + passcode))
	return h[:]
}

// NewSalt generates a random salt.
func NewSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("new salt: %w", err)
	}
	return salt, nil
}

// DeriveKey derives a 32-byte key from phrase+passcode using Argon2id.
func DeriveKey(phrase, passcode string, salt []byte) []byte {
	password := []byte(phrase + passcode)
	return argon2.IDKey(password, salt, argonTime, argonMemory, argonThreads, keyLen)
}

// Encrypt encrypts plaintext with AES-256-GCM. Returns nonce||ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts nonce||ciphertext with AES-256-GCM.
func Decrypt(key, data []byte) ([]byte, error) {
	if len(data) < nonceLen {
		return nil, errors.New("ciphertext too short")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
