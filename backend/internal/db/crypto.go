package db

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encPrefix = "enc:v1:"

type fieldCrypter struct {
	aead cipher.AEAD
}

func newFieldCrypter(key []byte) (fieldCrypter, error) {
	if len(key) == 0 {
		return fieldCrypter{}, nil
	}
	if len(key) != 32 {
		return fieldCrypter{}, fmt.Errorf("token encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fieldCrypter{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fieldCrypter{}, err
	}
	return fieldCrypter{aead: aead}, nil
}

func (c fieldCrypter) encrypt(plaintext string) (string, error) {
	if c.aead == nil || plaintext == "" {
		return plaintext, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("encrypt: nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (c fieldCrypter) decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, encPrefix) {
		return stored, nil
	}
	if c.aead == nil {
		return "", fmt.Errorf("decrypt: value is encrypted but TOKEN_ENCRYPTION_KEY is not set")
	}
	raw, err := base64.StdEncoding.DecodeString(stored[len(encPrefix):])
	if err != nil {
		return "", fmt.Errorf("decrypt: base64: %w", err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("decrypt: ciphertext too short")
	}
	plaintext, err := c.aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: open: %w", err)
	}
	return string(plaintext), nil
}

func (db *DB) SetEncryptionKey(key []byte) error {
	c, err := newFieldCrypter(key)
	if err != nil {
		return err
	}
	db.crypter = c
	return nil
}

func (db *DB) GetOrCreateEncryptedKV(ctx context.Context, key, value string) (string, error) {
	enc, err := db.crypter.encrypt(value)
	if err != nil {
		return "", err
	}
	stored, err := db.GetOrCreateKV(ctx, key, enc)
	if err != nil {
		return "", err
	}
	return db.crypter.decrypt(stored)
}
