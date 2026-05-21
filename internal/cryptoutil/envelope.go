package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const (
	HeaderEncryptedKey = "X-Encrypted-Key"
	HeaderNonce        = "X-Encrypted-Nonce"
)

func Encrypt(publicKey *rsa.PublicKey, plaintext []byte) ([]byte, string, string, error) {
	if publicKey == nil {
		return nil, "", "", errors.New("public key is nil")
	}

	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, "", "", err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", "", err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, aesKey, nil)
	if err != nil {
		return nil, "", "", err
	}

	return ciphertext,
		base64.StdEncoding.EncodeToString(encryptedKey),
		base64.StdEncoding.EncodeToString(nonce),
		nil
}

func Decrypt(privateKey *rsa.PrivateKey, ciphertext []byte, encryptedKeyB64 string, nonceB64 string) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}

	encryptedKey, err := base64.StdEncoding.DecodeString(encryptedKeyB64)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, err
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedKey, nil)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}
