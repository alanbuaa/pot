package crypto

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashlen(t *testing.T) {
	hashbyte := Hash([]byte("a"))
	if len(hashbyte) == 32 {
		t.Log("pass")
	}
}

// TODO No guarantee of correctness
func TestComputeMerkleRoot(t *testing.T) {
	hashes := make([][]byte, 8)
	for i := 0; i < 8; i++ {
		hashes[i] = Hash([]byte{byte(i)})
	}
	roots := make([][]byte, 8)
	for i := 0; i < 8; i++ {
		roots[i] = ComputeMerkleRoot(hashes[0 : i+1])
	}
	assert.Equal(t, roots[0], hashes[0])
	for i := 0; i < 8; i++ {
		fmt.Println(roots[i])
	}
}
