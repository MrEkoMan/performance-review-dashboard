package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

func loadEncryptionKey() ([]byte, error) {
	encodedKey := os.Getenv("MANAGER_DASHBOARD_ENCRYPTION_KEY")
	if encodedKey == "" {
		return nil, errors.New(
			"MANAGER_DASHBOARD_ENCRYPTION_KEY is not configured",
		)
	}
	
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, errors.New("Encryption key is not a valid Base64")
	}

	if len(key) != 32 {
		return nil, errors.New("Encryption key must decode to 32 bytes")
	}

	return key, nil
}

func encryptSecret(plaintext string) (string, error) {
	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(
		nonce,
		nonce,
		[]byte(plaintext),
		nil,
	)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptSecret(encodedValue string) (string, error) {
	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encodedValue)
	if err != nil {
		return "", err
	}

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
		return "", errors.New("Encrypted value is invalid")
	}

	nonce := ciphertext[:nonceSize]
	encryptedData := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(
		nil,
		nonce,
		encryptedData,
		nil,
	)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}