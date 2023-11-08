package pb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestBlockHash(t *testing.T) {
	b := &PoWBlock{
		ParentHash: []byte{1, 2, 3, 4},
		Height:     1234,
	}
	hash1 := b.ForceHash("sha256")
	hash2 := b.ForceHash("sha256")
	t.Log(hash1)
	t.Log(hash2)
	assert.Equal(t, hash1, hash2)
}
