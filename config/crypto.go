package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/user"
	"strings"
)

const encPrefix = "enc:v1:"

func getMachineKey() []byte {
	username := "unknown"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}
	hostname := "unknown"
	if h, err := os.Hostname(); err == nil {
		hostname = h
	}

	seed := username + hostname + "tui-notes-secure-salt-12345"
	hash := sha256.Sum256([]byte(seed))
	return hash[:]
}

// Encrypt encrypts a string using AES-GCM and a machine-derived key.
func Encrypt(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	if strings.HasPrefix(text, encPrefix) {
		return text, nil // Already encrypted
	}

	key := getMachineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encPrefix + encoded, nil
}

// Decrypt decrypts a string using AES-GCM and a machine-derived key.
func Decrypt(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	if !strings.HasPrefix(text, encPrefix) {
		return text, nil // Not encrypted
	}

	encoded := strings.TrimPrefix(text, encPrefix)
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	key := getMachineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err // Could not decrypt (wrong machine or corrupted)
	}

	return string(plaintext), nil
}
