package utils

import (
	"crypto/sha256"
	"errors"
	"hash"
)

// HashFactory creates a hash function based on the given algorithm name
func HashFactory(h string) (hash.Hash, error) {
	switch h {
	case "sha256":
		return sha256.New(), nil
	default:
		return nil, errors.New("hash function does not exists: " + h)
	}
}
