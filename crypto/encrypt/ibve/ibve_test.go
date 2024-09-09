package ibve

import (
	"crypto/rand"
	"fmt"
	"testing"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

func TestIBVE(t *testing.T) {
	g1 := NewG1()
	gt := NewGT()
	x, y := Keygen()
	msg, _ := gt.New().Rand(rand.Reader)
	cipherText := Encrypt(y, msg)
	roundX := g1.MulScalar(g1.New(), cipherText.C1, x)
	decMsg := Decrypt(roundX, cipherText)
	fmt.Println(decMsg.Equal(msg))
	fmt.Println(Verify(roundX, y, cipherText.C2))
}

func TestCipherText_ToBytes_FromBytes(t *testing.T) {
	g1 := NewG1()
	gt := NewGT()
	msg, _ := gt.New().Rand(rand.Reader)
	x, y := Keygen()
	cipherText := Encrypt(y, msg)
	decodeCipherText, err := new(CipherText).FromBytes(cipherText.ToBytes())
	if err != nil {
		fmt.Println(err)
		return
	}
	roundX := g1.MulScalar(g1.New(), cipherText.C1, x)
	decMsg := Decrypt(roundX, decodeCipherText)
	fmt.Println(decMsg.Equal(msg))
	fmt.Println(Verify(roundX, y, cipherText.C2))
}
