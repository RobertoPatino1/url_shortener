package shortener

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCode(t *testing.T) {
	code, err := NewCode(7)
	require.NoError(t, err)
	assert.Len(t, code, 7)
	for _, c := range code {
		assert.True(t, strings.ContainsRune(alphabet, c), "char %q fuera del alfabeto", c)
	}
}

func TestNewCodeInvalidLength(t *testing.T) {
	_, err := NewCode(0)
	assert.Error(t, err)
}

func TestNewCodeIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		c, err := NewCode(7)
		require.NoError(t, err)
		seen[c] = struct{}{}
	}
	assert.Len(t, seen, 1000)
}
