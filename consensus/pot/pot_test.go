package pot

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"testing"
)

func TestPrivkey(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Log(hexutil.Encode(crypto.GenerateKey().PublicKeyBytes()))
	}
}
