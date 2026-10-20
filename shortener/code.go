package shortener

import (
	"crypto/rand"
	"errors"
	"math/big"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// NewCode genera un código base62 aleatorio (crypto/rand: no predecible).
func NewCode(length int) (string, error) {
	if length < 1 {
		return "", errors.New("code length must be >= 1")
	}
	limit := big.NewInt(int64(len(alphabet)))
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}
