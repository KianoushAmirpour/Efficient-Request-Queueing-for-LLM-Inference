package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const stateSize = 32

type DefaultStateTokenManager struct{}

func NewDefaultStateTokenManager() *DefaultStateTokenManager {
	return &DefaultStateTokenManager{}
}

func (m *DefaultStateTokenManager) Generate() (string, error) {
	b := make([]byte, stateSize)
	if _, err := rand.Read(b); err != nil {

		return "", fmt.Errorf("generate state param: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
