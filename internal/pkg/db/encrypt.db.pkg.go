package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type cursorCrypto struct {
	cipher cipher.AEAD
}

func newCursorCrypto(secretKey []byte) (*cursorCrypto, error) {
	if len(secretKey) < 32 {
		paddedKey := make([]byte, 32)
		copy(paddedKey, secretKey)
		secretKey = paddedKey
	} else if len(secretKey) > 32 {
		secretKey = secretKey[:32]
	}
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &cursorCrypto{cipher: gcm}, nil
}

func (cc *cursorCrypto) encrypt(value string) (string, error) {
	nonce := make([]byte, cc.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	encrypted := cc.cipher.Seal(nil, nonce, []byte(value), nil)
	nonce = append(nonce, encrypted...)
	return base64.URLEncoding.EncodeToString(nonce), nil
}

func (cc *cursorCrypto) decrypt(cursor string) (string, error) {
	if cursor == "" {
		return "", nil
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("invalid cursor format")
	}

	nonceSize := cc.cipher.NonceSize()
	if len(decoded) < nonceSize {
		return "", fmt.Errorf("invalid cursor size")
	}

	nonce := decoded[:nonceSize]
	ciphertext := decoded[nonceSize:]

	plaintext, err := cc.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("cursor decryption failed")
	}

	return string(plaintext), nil
}
