package crypto

import "testing"

func TestHashlen(t *testing.T) {
	hashbyte := Hash([]byte("a"))
	if len(hashbyte) == 32 {
		t.Log("pass")
	}
}
