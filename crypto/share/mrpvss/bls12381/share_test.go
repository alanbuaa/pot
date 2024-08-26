package mrpvss

import (
	"crypto/rand"
	"fmt"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"testing"
)

func TestEncShare_ToBytes_FromBytes(t *testing.T) {
	n := 15
	threshold := uint32(7)

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
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// 验证加密份额
	verifyEncShareRes := VerifyEncShares(uint32(n), threshold, g, h, pubKeyList, shareCommitments, coeffCommits, decodeEncShares)
	fmt.Println("verifyEncShareRes:", verifyEncShareRes)

}
