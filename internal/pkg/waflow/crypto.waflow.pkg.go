package waflow

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
)

// EncryptedRequest represents the incoming encrypted body from WhatsApp Flows
type EncryptedRequest struct {
	EncryptedFlowData string `json:"encrypted_flow_data"`
	EncryptedAESKey   string `json:"encrypted_aes_key"`
	InitialVector     string `json:"initial_vector"`
}

// DecryptedRequest represents the payload after decryption
type DecryptedRequest struct {
	Version   string                 `json:"version"`
	Action    string                 `json:"action"`
	Screen    string                 `json:"screen"`
	Data      map[string]interface{} `json:"data"`
	FlowToken string                 `json:"flow_token"`
}

// FlowResponse represents the response before encryption
type FlowResponse struct {
	Screen string                 `json:"screen,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// LoadPrivateKey loads an RSA private key from a PEM file
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS#8 first, fallback to PKCS#1
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		rsaKey, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key (PKCS#8: %v, PKCS#1: %v)", err, err2)
		}
		return rsaKey, nil
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rsaKey, nil
}

// DecryptRequest decrypts the incoming WhatsApp Flows encrypted request
// Returns the decrypted request, AES key, and IV for use in encrypting the response
func DecryptRequest(privateKey *rsa.PrivateKey, body EncryptedRequest) (*DecryptedRequest, []byte, []byte, error) {
	// 1. Base64 decode all fields
	encryptedAESKey, err := base64.StdEncoding.DecodeString(body.EncryptedAESKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode encrypted_aes_key: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(body.InitialVector)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode initial_vector: %w", err)
	}

	encryptedFlowData, err := base64.StdEncoding.DecodeString(body.EncryptedFlowData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode encrypted_flow_data: %w", err)
	}

	// 2. RSA-OAEP decrypt the AES key
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedAESKey, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decrypt AES key: %w", err)
	}

	// 3. Create AES-GCM cipher with nonce size 16 (WhatsApp uses 128-bit IV)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 4. Decrypt — GCM Open expects ciphertext+tag concatenated, which is what WhatsApp sends
	plaintext, err := gcm.Open(nil, iv, encryptedFlowData, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decrypt flow data: %w", err)
	}

	// 5. JSON unmarshal
	var decrypted DecryptedRequest
	if err := json.Unmarshal(plaintext, &decrypted); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	return &decrypted, aesKey, iv, nil
}

// EncryptResponse encrypts the response to send back to WhatsApp Flows
func EncryptResponse(aesKey []byte, iv []byte, response FlowResponse) (string, error) {
	// 1. JSON marshal response
	plaintextBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	// 2. Flip IV (bitwise NOT)
	flippedIV := make([]byte, len(iv))
	for i := range iv {
		flippedIV[i] = ^iv[i]
	}

	// 3. Create AES-GCM cipher with nonce size 16
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 4. Encrypt — Seal appends auth tag automatically
	sealed := gcm.Seal(nil, flippedIV, plaintextBytes, nil)

	// 5. Base64 encode
	return base64.StdEncoding.EncodeToString(sealed), nil
}
