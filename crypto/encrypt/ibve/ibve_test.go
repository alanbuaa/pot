package ibve

import (
	"fmt"
	"testing"
)

func TestIBVE(t *testing.T) {
	msg := gt.New().One()
	sk, pk := Keygen()
	g1r, g2r, c2 := Encrypt(pk, msg)
	res, decMsg := Decrypt(sk, pk, g1r, g2r, c2)
	fmt.Println(res && decMsg.Equal(msg))
}
