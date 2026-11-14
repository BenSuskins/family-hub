package services

import (
	"crypto/rand"
	"encoding/hex"
)

func NewCredentialSecret() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
