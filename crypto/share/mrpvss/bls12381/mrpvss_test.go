package mrpvss

import (
	"crypto/rand"
	"fmt"
	dleq "github.com/zzz136454872/upgradeable-consensus/crypto/proof/dleq/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"math/big"
	"testing"
)

func TestSplitFr(t *testing.T) {
	fr := &Fr{18054189570020830075, 8221007895056591497, 12699603262867178032, 3107037372339810918}
	frBytes := fr.ToBytes()
	fmt.Println("fr:", fr)
	fmt.Println("frBytes:", frBytes)
	frSum := NewFr()
	fr256 := FrFromInt(256)
	for i := 0; i < 32; i++ {
		term1 := FrFromInt(int(frBytes[31-i]))
		fmt.Println("term1:", term1)
		term2 := NewFr().Exp(fr256, big.NewInt(int64(i)))
		fmt.Println("term2:", term2)
		tmp := NewFr().Mul(term1, term2)
		frSum.Add(frSum, tmp)
	}
	fmt.Println("frSum:", frSum)
	// frSplits := make([]*Fr, 32)
	// for i := 0; i < 32; i++ {
	// 	frSplits[i] = NewFr().
	// }

}

func TestBruteForceFindExp(t *testing.T) {
	group1 := NewG1()
	g := group1.One()
	bias := NewFr().Exp(FrFromInt(256), big.NewInt(33))
	expected := NewFr().Mul(FrFromInt(123), bias)
	target := group1.MulScalar(group1.New(), g, expected)
	res, err := bruteForceFindExp(g, target, bias, 256)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(bias)
	fmt.Println(expected)
	fmt.Println(res)
}

func TestMRPVSS(t *testing.T) {
	group1 := NewG1()
	n := 15
	threshold := uint32(7)

	// 生成 g, h
	g := group1.One()
	// random, _ := NewFr().Rand(rand.Reader)
	random := FrFromInt(5)
	h := group1.Affine(group1.MulScalar(group1.New(), g, random))

	// 轮ID
	c := group1.Affine(group1.MulScalar(group1.New(), g, FrFromInt(123)))

	// 秘密
	s := FrFromInt(123)

	// 公私钥
	privKeyList := make([]*Fr, n)
	pubKeyList := make([]*PointG1, n)
	for i := 0; i < n; i++ {
		// random, _ = NewFr().Rand(rand.Reader)
		random = FrFromInt(i + 1)
		privKeyList[i] = random
		pubKeyList[i] = group1.Affine(group1.MulScalar(group1.New(), g, random))
	}
	// 分发份额
	shareCommitments, coeffCommits, encShares, _, err := EncShares(g, h, pubKeyList, s, threshold)
	if err != nil {
		fmt.Println(err)
		return
	}
	// 验证加密份额
	verifyEncShareRes := VerifyEncShares(uint32(n), threshold, g, h, pubKeyList, shareCommitments, coeffCommits, encShares)
	fmt.Println("verifyEncShareRes:", verifyEncShareRes)

	// 解密份额并计算轮份额
	shares := make([]*Fr, n)
	roundShares := make([]*PointG1, n)
	roundShareProofs := make([]*dleq.Proof, n)
	for i := 0; i < n; i++ {
		// g^(s_ij)  = (B_ij)/(A_i^sk_i)
		shares[i] = DecryptShare(g, privKeyList[i], encShares[i])
		if shares[i] == nil {
			fmt.Println("shares is nil")
			return
		}
		// c, c^(s_i), h, S_i
		roundShares[i], roundShareProofs[i], err = CalcRoundShare(uint32(i+1), h, c, shareCommitments[i], shares[i])
		if err != nil {
			return
		}
	}

	// 验证轮份额
	for i := 0; i < n; i++ {
		index := i + 1
		verifyRoundShareRes := VerifyRoundShare(uint32(index), h, c, shareCommitments[i], roundShares[i], roundShareProofs[i])
		if !verifyRoundShareRes {
			fmt.Println("fail to verify Round Share at:", index)
		}
	}

	// 重构秘密
	indices := make([]uint32, n)
	for i := 0; i < n; i++ {
		indices[i] = uint32(i + 1)
	}
	recoveredSecret := RecoverRoundSecret(threshold, indices, roundShares)
	fmt.Println("s: ", s[0])
	fmt.Println("c^s: ", group1.Affine(group1.MulScalar(group1.One(), c, s)))
	fmt.Println("recovered: ", recoveredSecret)
}

func TestDPVSS(t *testing.T) {
	group1 := NewG1()
	n := 15
	threshold := uint32(7)
	m := 3

	// 生成 g, h
	g := group1.One()
	random, _ := NewFr().Rand(rand.Reader)
	// random := FrFromInt(5)
	h := group1.Affine(group1.MulScalar(group1.New(), g, random))

	// 轮ID
	c := group1.Affine(group1.MulScalar(group1.New(), g, FrFromInt(123)))

	// 秘密
	sList := []*Fr{FrFromInt(2), FrFromInt(3), FrFromInt(5)}

	aggrS := NewFr().Zero()
	for i := 0; i < m; i++ {
		aggrS.Add(aggrS, sList[i])
	}

	// 公私钥
	privKeyList := make([]*Fr, n)
	pubKeyList := make([]*PointG1, n)
	for i := 0; i < n; i++ {
		random, _ = NewFr().Rand(rand.Reader)
		// random = FrFromInt(i + 1)
		privKeyList[i] = random
		pubKeyList[i] = group1.Affine(group1.MulScalar(group1.New(), g, random))
	}

	// 加密份额
	encsharesList := make([][]*EncShare, m)
	shareCommitmentsList := make([][]*PointG1, m)
	coeffCommitsList := make([][]*PointG1, m)

	for k := 0; k < m; k++ {
		// 分发份额
		var err error
		shareCommitmentsList[k], coeffCommitsList[k], encsharesList[k], _, err = EncShares(g, h, pubKeyList, sList[k], threshold)
		if err != nil {
			fmt.Println(err)
			return
		}
		// 验证加密份额
		verifyEncShareRes := VerifyEncShares(uint32(n), threshold, g, h, pubKeyList, shareCommitmentsList[k], coeffCommitsList[k], encsharesList[k])
		fmt.Println("verifyEncShareRes:", verifyEncShareRes)
	}

	// 聚合份额承诺
	aggrShareCommits := make([]*PointG1, n)
	for i := 0; i < n; i++ {
		shareCommitsForI := make([]*PointG1, m)
		for k := 0; k < m; k++ {
			shareCommitsForI[k] = shareCommitmentsList[k][i]
		}
		aggrShareCommits[i] = AggregateShareCommits(shareCommitsForI)
	}

	// 聚合份额
	aggEncSharesList := make([]*EncShare, n)
	for i := 0; i < n; i++ {
		encSharesForI := make([]*EncShare, m)
		for k := 0; k < m; k++ {
			encSharesForI[k] = encsharesList[k][i]
		}
		aggEncSharesList[i] = AggregateEncShareList(encSharesForI)
	}

	// 解密份额并计算轮份额
	shares := make([]*Fr, n)
	roundShares := make([]*PointG1, n)
	roundShareProofs := make([]*dleq.Proof, n)
	for i := 0; i < n; i++ {
		// g^(s_ij)  = (B_ij)/(A_i^sk_i)
		// TODO
		shares[i] = DecryptAggregateShare(g, privKeyList[i], aggEncSharesList[i], uint32(m))
		if shares[i] == nil {
			fmt.Println("shares is nil")
			return
		}
		// c, c^(s_i), h, S_i
		var err error
		roundShares[i], roundShareProofs[i], err = CalcRoundShare(uint32(i+1), h, c, aggrShareCommits[i], shares[i])
		if err != nil {
			return
		}
	}

	// 验证轮份额
	for i := 0; i < n; i++ {
		index := i + 1
		verifyRoundShareRes := VerifyRoundShare(uint32(index), h, c, aggrShareCommits[i], roundShares[i], roundShareProofs[i])
		if !verifyRoundShareRes {
			fmt.Println("fail to verify Round Share at:", index)
		}
	}
	indices := make([]uint32, n)
	for i := 0; i < n; i++ {
		indices[i] = uint32(i + 1)
	}
	// 重构秘密
	recoveredSecret := RecoverRoundSecret(threshold, indices, roundShares)
	fmt.Println("s: ", aggrS[0])
	fmt.Println("c^s: ", group1.Affine(group1.MulScalar(group1.One(), c, aggrS)))
	fmt.Println("recovered: ", recoveredSecret)
}
