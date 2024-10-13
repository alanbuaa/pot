// Package mrpvss multi-round pvss
package mrpvss

import (
	dleq "blockchain-crypto/proof/dleq/bls12381"
	. "blockchain-crypto/types/curve/bls12381"
	poly "blockchain-crypto/types/poly/bls12381"
	"errors"
	"fmt"
)

func EncShares(g *PointG1, h *PointG1, holderPKList []*PointG1, secret *Fr, threshold uint32) (shareCommitments []*PointG1, coeffCommits []*PointG1, encShares []*EncShare, y *PointG1, err error) {
	group1 := NewG1()
	if g == nil || h == nil || holderPKList == nil || secret == nil {
		return nil, nil, nil, nil, errors.New("invalid params")
	}
	n := uint32(len(holderPKList))
	if n < threshold {
		return nil, nil, nil, nil, errors.New("threshold is higher than num of public keys")
	}

	// 计算委员会公钥 y = g^s
	y = group1.Affine(group1.MulScalar(group1.New(), g, secret))

	// 生成秘密共享多项式 p, p(0) = secret
	secretPoly := poly.NewSecretPolynomial(threshold, secret)
	// 计算系数承诺 C_i = h^(a_i)
	coeffCommits = make([]*PointG1, threshold)
	for i, coeff := range secretPoly.Coeffs {
		coeffCommits[i] = group1.Affine(group1.MulScalar(group1.New(), h, coeff))
	}
	// 计算份额 s_i = p(i)
	shares := make([]*Fr, n)
	for i := uint32(0); i < n; i++ {
		shares[i] = secretPoly.Eval(FrFromUInt32(i + 1))
	}

	encShares = make([]*EncShare, n)
	// share commitment S_i = h^(s_i)
	shareCommitments = make([]*PointG1, n)
	for i := uint32(0); i < n; i++ {
		index := i + 1
		// share commitment S_i = h^(s_i)
		shareCommitments[i] = group1.Affine(group1.MulScalar(group1.New(), h, shares[i]))

		// 计算加密份额 (A_i, {B_ij})
		// TODO proof of ciphertext form
		A := group1.Affine(group1.MulScalar(group1.New(), g, shares[i]))
		// (pk_i)^(s_i)
		cipherTerm := group1.MulScalar(group1.New(), holderPKList[i], shares[i])
		BList := make([]*PointG1, 32)
		// product of B_ij, for verification
		BProd := group1.Zero()
		// 拆分份额 s_i = Σ_(j=1,32) s_ij
		shareBytes := shares[i].ToBytes()
		fr256 := FrFromInt(256)
		step := NewFr().One()
		fmt.Println("===============================================================")
		fmt.Printf("i = %v, s_i = %v, (pk_i)^(s_i) = %v\n", i, shares[i], group1.Affine(cipherTerm))
		for j := 0; j < 32; j++ {
			// calc share pieces s_ij slice * 2^8^(k-j)
			sharePiece := NewFr().Mul(FrFromInt(int(shareBytes[31-j])), step)
			step.Mul(step, fr256)
			// 	B_ij = g^(s_ij) (pk_i)^(s_i)
			fmt.Printf("j = %v, s_ij = %v, g^(s_ij) = %v\n", j, shareBytes[31-j], group1.Affine(group1.MulScalar(group1.New(), g, sharePiece)))
			BList[j] = group1.Affine(group1.Add(group1.New(), group1.MulScalar(group1.New(), g, sharePiece), cipherTerm))
			group1.Add(BProd, BProd, BList[j])
		}
		fmt.Println("===============================================================")
		// 分发证明 log_(pk_i) Y_i = log_(h) S_i 证明分发的是 s_i
		dleqParams := dleq.DLEQ{
			Index: index,
			G1:    h,
			H1:    shareCommitments[i],
			G2:    group1.Add(group1.New(), g, group1.MulScalar(group1.New(), holderPKList[i], FrFromInt(32))),
			H2:    group1.Affine(BProd),
		}
		encShares[i] = &EncShare{
			A:         A,
			BList:     BList,
			DealProof: dleqParams.Prove(shares[i]),
		}
	}
	return shareCommitments, coeffCommits, encShares, y, nil
}

func AggregateEncShareList(encShares []*EncShare) *EncShare {
	group1 := NewG1()
	n := len(encShares)
	aggrEncShare := &EncShare{
		A:         group1.Zero(),
		BList:     make([]*PointG1, 32),
		DealProof: nil,
	}
	for j := 0; j < 32; j++ {
		aggrEncShare.BList[j] = group1.Zero()
	}
	for i := 0; i < n; i++ {
		group1.Add(aggrEncShare.A, aggrEncShare.A, encShares[i].A)
		for j := 0; j < 32; j++ {
			group1.Add(aggrEncShare.BList[j], aggrEncShare.BList[j], encShares[i].BList[j])
		}
	}
	return aggrEncShare
}

func AggregateEncShares(prevAggrEncShares, encShares *EncShare) *EncShare {
	group1 := NewG1()
	group1.Add(prevAggrEncShares.A, prevAggrEncShares.A, encShares.A)
	for j := 0; j < 32; j++ {
		group1.Add(prevAggrEncShares.BList[j], prevAggrEncShares.BList[j], encShares.BList[j])
	}
	return prevAggrEncShares
}

func VerifyEncShares(n uint32, t uint32, g, h *PointG1, pubKeyList []*PointG1, shareCommits []*PointG1, coeffCommits []*PointG1, encShares []*EncShare) bool {
	// 计算 {S_i} \prod B_ij
	group1 := NewG1()
	BProdList := make([]*PointG1, n)
	for i := uint32(0); i < n; i++ {
		calcShareCommits := group1.Zero()
		exponent := NewFr().One()
		BProdList[i] = group1.Zero()
		for j := uint32(0); j < t; j++ {
			// S_i = Π_j∈[0,t-1] (X_j)^[(i)^j] = h^p(i), i∈[1,t]
			group1.Add(calcShareCommits, calcShareCommits, group1.MulScalar(group1.New(), coeffCommits[j], exponent))
			exponent.Mul(exponent, FrFromUInt32(i+1))
		}
		if !group1.Equal(calcShareCommits, shareCommits[i]) {
			fmt.Println("failed to verify shareCommits")
			return false
		}
		for j := 0; j < 32; j++ {
			// \prod B_ij
			group1.Add(BProdList[i], BProdList[i], encShares[i].BList[j])
		}
	}
	// 验证加密份额 log_h S_i = log_(g · pk_i^k) \prod B_ij
	for i := uint32(0); i < n; i++ {
		base2 := group1.Add(group1.New(), g, group1.MulScalar(group1.New(), pubKeyList[i], FrFromInt(32)))
		if !dleq.Verify(i+1, h, shareCommits[i], base2, BProdList[i], encShares[i].DealProof) {
			return false
		}
	}
	return true
}

func AggregateCommitteePK(prevPK *PointG1, pk *PointG1) *PointG1 {
	group1 := NewG1()
	group1.Add(prevPK, prevPK, pk)
	return prevPK
}

func AggregateShareCommits(shareCommits []*PointG1) *PointG1 {
	group1 := NewG1()
	n := len(shareCommits)
	aggrShareCommit := group1.Zero()
	for i := 0; i < n; i++ {
		group1.Add(aggrShareCommit, aggrShareCommit, shareCommits[i])
	}
	return aggrShareCommit
}

func AggregateShareCommitList(list1 []*PointG1, list2 []*PointG1) ([]*PointG1, error) {
	group1 := NewG1()
	if list1 == nil {
		return nil, fmt.Errorf("list1 is nil")
	}
	if list2 == nil {
		return nil, fmt.Errorf("list2 is nil")
	}
	if len(list1) != len(list2) {
		return nil, fmt.Errorf("size of list2 is not equal to size of list2")
	}
	for i := 0; i < len(list1); i++ {
		if list1[i] == nil {
			return nil, fmt.Errorf("list1 [%v] is nil", i)
		}
		if list2[i] == nil {
			return nil, fmt.Errorf("list2 [%v] is nil", i)
		}
	}
	for i := 0; i < len(list1); i++ {
		group1.Add(list1[i], list1[i], list2[i])
	}
	return list1, nil
}

func DecryptShare(g *PointG1, privKey *Fr, encShare *EncShare) *Fr {
	group1 := NewG1()
	// A_i^sk_i
	denominator := group1.MulScalar(group1.New(), encShare.A, privKey)
	// g^(s_ij)  = (B_ij)/(A_i^sk_i)
	share := NewFr().Zero()
	step := NewFr().One()
	fr256 := FrFromInt(256)
	for i := 0; i < 32; i++ {
		point := group1.Sub(group1.New(), encShare.BList[i], denominator)
		exponent, err := bruteForceFindExp(g, point, step, 256)
		if err != nil {
			return nil
		}
		share.Add(share, exponent)
		step.Mul(step, fr256)
	}
	return share
}

func DecryptAggregateShare(g *PointG1, privKey *Fr, encShare *EncShare, m uint32) *Fr {
	group1 := NewG1()
	// A_i^sk_i
	denominator := group1.MulScalar(group1.New(), encShare.A, privKey)
	// g^(s_ij)  = (B_ij)/(A_i^sk_i)
	share := NewFr().Zero()
	step := NewFr().One()
	fr256 := FrFromInt(256)
	for i := 0; i < 32; i++ {
		point := group1.Sub(group1.New(), encShare.BList[i], denominator)
		exponent, err := bruteForceFindExp(g, point, step, 256*m)
		if err != nil {
			fmt.Printf("cannot find exponent[%v], exp= %v\n", i, exponent)
			return nil
			// exponent = NewFr().Zero()
		}
		share.Add(share, exponent)
		step.Mul(step, fr256)
	}
	return share
}

func CalcRoundShare(index uint32, h *PointG1, c *PointG1, shareCommit *PointG1, share *Fr) (*PointG1, *dleq.Proof, error) {
	group1 := NewG1()
	if c == nil || share == nil {
		return nil, nil, errors.New("invalid parameter")
	}
	roundShare := group1.Affine(group1.MulScalar(group1.New(), c, share))
	dleqParams := dleq.DLEQ{
		Index: index,
		G1:    c,
		H1:    roundShare,
		G2:    h,
		H2:    shareCommit,
	}
	return roundShare, dleqParams.Prove(share), nil
}

func VerifyRoundShare(index uint32, h *PointG1, c *PointG1, shareCommit *PointG1, roundShare *PointG1, roundShareProof *dleq.Proof) bool {
	// log_c c^(s_i) = log_(h) S_i
	return dleq.Verify(index, c, roundShare, h, shareCommit, roundShareProof)
}

func RecoverRoundSecret(threshold uint32, indices []uint32, roundShares []*PointG1) *PointG1 {
	group1 := NewG1()
	t := int(threshold)
	if t > len(roundShares) {
		return nil
	}
	secret := group1.Zero()
	// numerator = ∏ xᵢ
	numerator := NewFr().One()
	for i := uint32(0); i < threshold; i++ {
		numerator.Mul(numerator, FrFromUInt32(indices[i]))
	}
	// ∏_(i∈[1,t]) (c^(s_i))^(λ_i)
	for i := 0; i < t; i++ {
		// λ_i = t! / x_i / [∏_(j∈[1,t],j≠i) (x_j - x_i)]
		lagrangeBasis := NewFr().Mul(numerator, NewFr().Inverse(FrFromUInt32(indices[i])))
		denominator := NewFr().One()
		for j := 0; j < t; j++ {
			if j != i {
				denominator.Mul(denominator, NewFr().Sub(FrFromUInt32(indices[j]), FrFromUInt32(indices[i])))
			}
		}
		lagrangeBasis.Mul(lagrangeBasis, NewFr().Inverse(denominator))
		// ∏_(i∈[1,t]) rs^(λ_i) , rs = c^(s_i)
		group1.Add(secret, secret, group1.MulScalar(group1.New(), roundShares[i], lagrangeBasis))
	}
	return group1.Affine(secret)
}
