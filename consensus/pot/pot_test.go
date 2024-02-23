package pot

import (
	"testing"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
)

func TestPrivkey(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Log(crypto.GenerateKey())
	}
}
