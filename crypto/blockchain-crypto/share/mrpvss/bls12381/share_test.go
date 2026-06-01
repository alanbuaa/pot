package mrpvss

import (
	. "blockchain-crypto/types/curve/bls12381"
	"crypto/rand"
	"fmt"
	"testing"
)

func TestEncShare_ToBytes_FromBytes(t *testing.T) {
	n := 15
	threshold := uint32(7)
	group1 := NewG1()
	// 生成 g, h
	g := group1.One()
	random, _ := NewFr().Rand(rand.Reader)
	h := group1.Affine(group1.MulScalar(group1.New(), g, random))
	// 秘密
	s := FrFromInt(123)

	// 公私钥
	privKeyList := make([]*Fr, n)
	pubKeyList := make([]*PointG1, n)
	for i := 0; i < n; i++ {
		random, _ = NewFr().Rand(rand.Reader)
		// random = FrFromInt(i + 1)
		privKeyList[i] = random
		pubKeyList[i] = group1.Affine(group1.MulScalar(group1.New(), g, random))
	}
	// 分发份额
	shareCommitments, coeffCommits, encShares, _, err := EncShares(g, h, pubKeyList, s, threshold)
	if err != nil {
		fmt.Println(err)
		return
	}

	decodeEncShares := make([]*EncShare, n)
	for i := 0; i < n; i++ {
		decodeEncShares[i], err = new(EncShare).FromBytes(encShares[i].ToBytes())
		decodeEncShares[i], err = new(EncShare).FromBytes(decodeEncShares[i].ToBytes())
		decodeEncShares[i], err = new(EncShare).FromBytes(decodeEncShares[i].ToBytes())
		decodeEncShares[i], err = new(EncShare).FromBytes(decodeEncShares[i].ToBytes())
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// 验证加密份额
	verifyEncShareRes := VerifyEncShares(uint32(n), threshold, g, h, pubKeyList, shareCommitments, coeffCommits, decodeEncShares)
	fmt.Println("verifyEncShareRes:", verifyEncShareRes)
}

func TestEncShare_ToBytes_FromBytes2(t *testing.T) {
	n := 15
	threshold := uint32(7)
	group1 := NewG1()
	// 生成 g, h
	g := group1.One()
	random, _ := NewFr().Rand(rand.Reader)
	h := group1.Affine(group1.MulScalar(group1.New(), g, random))
	// 秘密
	s := FrFromInt(123)

	// 公私钥
	privKeyList := make([]*Fr, n)
	pubKeyList := make([]*PointG1, n)
	for i := 0; i < n; i++ {
		random, _ = NewFr().Rand(rand.Reader)
		// random = FrFromInt(i + 1)
		privKeyList[i] = random
		pubKeyList[i] = group1.Affine(group1.MulScalar(group1.New(), g, random))
	}
	// 分发份额
	_, _, encShares, _, err := EncShares(g, h, pubKeyList, s, threshold)
	if err != nil {
		fmt.Println(err)
		return
	}

	share := encShares[1]
	fmt.Println(share.A)
	fmt.Println(group1.Affine(share.A))
	share2, _ := new(EncShare).FromBytes(share.ToBytes())
	fmt.Println(share2.A)
	fmt.Println(group1.Affine(share2.A))
	share3, _ := new(EncShare).FromBytes(share2.ToBytes())
	fmt.Println(share3.A)
	fmt.Println(group1.Affine(share3.A))
}

func TestToCompressedFromCompressed(t *testing.T) {
	group1 := NewG1()
	// r, _ := NewFr().Rand(rand.Reader)
	// p := group1.MulScalar(group1.New(), group1.One(), r)
	p := group1.MulScalar(group1.New(), group1.One(), FrFromInt(2))
	fmt.Println(p)
	fmt.Println(group1.Affine(p))
	p1, err := group1.FromCompressed(group1.ToCompressed(p))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(p1)
	p2, err := group1.FromCompressed(group1.ToCompressed(p1))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(p2)

	fmt.Println(p1)
	p3, err := group1.FromCompressed(group1.ToCompressed(p2))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(p3)
}
