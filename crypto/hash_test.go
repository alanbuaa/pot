package crypto

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"testing"
)

func TestHashlen(t *testing.T) {
	hashbyte := Hash([]byte("a"))
	if len(hashbyte) == 32 {
		t.Log("pass")
	}
}

func TestNilHash(t *testing.T) {
	input := make([]byte, 0)
	hash := Hash(input)

	t.Log(input)
	t.Log(hexutil.Encode(hash))

}
