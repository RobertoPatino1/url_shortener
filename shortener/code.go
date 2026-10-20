package shortener

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// NewCode genera un código base62 aleatorio de la longitud indicada.
func NewCode(length int) (string, error) {
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
