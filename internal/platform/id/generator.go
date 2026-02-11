package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Generator creates opaque IDs suitable for external references.
type Generator interface {
	NewID() (string, error)
}

type RandomGenerator struct{}

func NewRandomGenerator() *RandomGenerator {
	return &RandomGenerator{}
}

func (g *RandomGenerator) NewID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
