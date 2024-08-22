package utils

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"testing"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

func TestFrDiv(t *testing.T) {
	fr1 := FrFromInt(12)
	fr2 := FrFromInt(24)
	fr2.Inverse(fr2)
	fr1.Mul(fr1, fr2)
	fmt.Println(fr1)
}

func TestRandReader(t *testing.T) {
	b := make([]byte, 4)
	// ReadFull从rand.Reader精确地读取len(b)字节数据填充进b
	// rand.Reader是一个全局、共享的密码用强随机数生成器
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(b)
}

func TestFromBig(t *testing.T) {
	v := big.NewInt(123)
	fmt.Println(FrFromBig(v))
}

func TestFrRand(t *testing.T) {
	fr, err := NewFr().Rand(rand.Reader)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(fr)
	var fr2 *Fr
	_, err2 := fr2.Rand(rand.Reader)
	if err2 != nil {
		fmt.Println(err2)
	}
	fmt.Println(fr2)
}

func frSet(fr *Fr) *Fr {
	f := fr
	return f
}

func TestFrSet(t *testing.T) {
	fr, err := NewFr().Rand(rand.Reader)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(fr)
	fr2 := frSet(fr)
	fmt.Println(fr2)
}

func TestPointEqual(t *testing.T) {
	g1 := NewG1()
	g11 := g1.One()
	g1.MulScalar(g11, g11, FrFromInt(3))
	g12 := g1.One()
	g1.MulScalar(g12, g12, FrFromInt(3))

	fmt.Println(g1.Equal(g11, g12))
}

func TestFr(t *testing.T) {
	fr1 := FrFromInt(3)
	fr2 := fr1
	fr1.Mul(fr1, fr1)
	fmt.Println(fr1)
	fmt.Println(fr2)

}

func TestFrInverse(t *testing.T) {
	fr1 := FrFromInt(4)
	fr2 := FrFromInt(2)
	fr2.Inverse(fr2)
	fmt.Println(fr2)
	fr3 := NewFr().Mul(fr1, fr2)
	fmt.Println(fr3)
}
