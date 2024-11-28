package blockchain_api

import (
	"blockchain-crypto/encrypt/ibve"
	mrpvss "blockchain-crypto/share/mrpvss/bls12381"
	"blockchain-crypto/types/curve/bls12381"
	"crypto/rand"
	"fmt"
	"testing"
)

func TestIBVE(t *testing.T) {
	// 委员会公私钥 x , y=g^x
	x, _ := bls12381.NewFr().Rand(rand.Reader)
	y := g1.MulScalar(g1.New(), g1.One(), x)
	msgBytes, _ := GenTxKey()
	_, cipherTextBytes, err := EncryptIBVE(g1.ToCompressed(y), msgBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	// c^x
	cipherText, err := new(ibve.CipherText).FromBytes(cipherTextBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	roundSK := g1.MulScalar(g1.New(), cipherText.C1, x)
	decMsg, err := DecryptIBVE(g1.ToCompressed(roundSK), cipherTextBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(msgBytes)
	fmt.Println(decMsg)
	fmt.Println(VerifyIBVE(g1.ToCompressed(roundSK), g1.ToCompressed(y), cipherTextBytes))
}

func TestDPVSSAndIBVE(t *testing.T) {
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// PoT部分
	n := 15
	threshold := uint32(7)
	m := 3

	// 生成 g, h
	g := g1.One()
	random, _ := bls12381.NewFr().Rand(rand.Reader)
	// random := FrFromInt(5)
	h := g1.Affine(g1.MulScalar(g1.New(), g, random))

	// 秘密
	sList := []*bls12381.Fr{bls12381.FrFromInt(2), bls12381.FrFromInt(3), bls12381.FrFromInt(5)}

	aggrS := bls12381.NewFr().Zero()
	for i := 0; i < m; i++ {
		aggrS.Add(aggrS, sList[i])
	}

	// 公私钥
	privKeyList := make([]*bls12381.Fr, n)
	pubKeyList := make([]*bls12381.PointG1, n)
	for i := 0; i < n; i++ {
		random, _ = bls12381.NewFr().Rand(rand.Reader)
		// random = FrFromInt(i + 1)
		privKeyList[i] = random
		pubKeyList[i] = g1.Affine(g1.MulScalar(g1.New(), g, random))
	}

	// 加密份额
	encsharesList := make([][]*mrpvss.EncShare, m)
	shareCommitmentsList := make([][]*bls12381.PointG1, m)
	coeffCommitsList := make([][]*bls12381.PointG1, m)
	aggrY := g1.Zero()
	for k := 0; k < m; k++ {
		// 分发份额
		var err error
		var y *bls12381.PointG1
		shareCommitmentsList[k], coeffCommitsList[k], encsharesList[k], y, err = mrpvss.EncShares(g, h, pubKeyList, sList[k], threshold)
		if err != nil {
			fmt.Println(err)
			return
		}
		// 验证加密份额
		verifyEncShareRes := mrpvss.VerifyEncShares(uint32(n), threshold, g, h, pubKeyList, shareCommitmentsList[k], coeffCommitsList[k], encsharesList[k])
		// 聚合委员会公钥
		if verifyEncShareRes {
			mrpvss.AggregateLeaderPK(aggrY, y)
		} else {
			fmt.Println("VerifyEncShareRes is false")
		}
	}

	// 聚合份额承诺
	aggrShareCommits := make([]*bls12381.PointG1, n)
	for i := 0; i < n; i++ {
		shareCommitsForI := make([]*bls12381.PointG1, m)
		for k := 0; k < m; k++ {
			shareCommitsForI[k] = shareCommitmentsList[k][i]
		}
		aggrShareCommits[i] = mrpvss.AggregateShareCommits(shareCommitsForI)
	}

	// 聚合份额
	aggEncSharesList := make([]*mrpvss.EncShare, n)
	for i := 0; i < n; i++ {
		encSharesForI := make([]*mrpvss.EncShare, m)
		for k := 0; k < m; k++ {
			encSharesForI[k] = encsharesList[k][i]
		}
		aggEncSharesList[i] = mrpvss.AggregateEncShareList(encSharesForI)
	}

	// 解密份额
	shares := make([]*bls12381.Fr, n)
	for i := 0; i < n; i++ {
		// g^(s_ij)  = (B_ij)/(A_i^sk_i)
		// TODO
		shares[i] = mrpvss.DecryptAggregateShare(g, privKeyList[i], aggEncSharesList[i], uint32(m))
		if shares[i] == nil {
			fmt.Println("shares is nil")
			return
		}
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 委员会共识部分
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 公开输入
	userInput := &CommitteeConfig{
		H:  g1.ToCompressed(h),
		PK: g1.ToCompressed(aggrY),
	}
	// PoT节点传递给委员会共识节点的结构
	committeeMemberInputList := make([]*CommitteeConfig, n)
	for i := 0; i < n; i++ {
		aggrShareCommitsBytes := make([][]byte, n)
		for j := 0; j < n; j++ {
			aggrShareCommitsBytes[j] = g1.ToCompressed(aggrShareCommits[j])
		}
		committeeMemberInputList[i] = &CommitteeConfig{
			H:  g1.ToCompressed(h),
			PK: g1.ToCompressed(aggrY),
			DPVSSConfigForMember: DPVSSConfig{
				ShareCommits: aggrShareCommitsBytes,
				Share:        shares[i].ToBytes()},
		}
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// 流程
	// 用户生成交易
	txBytes := []byte{232, 185, 133, 38, 98, 223, 212, 133, 177, 85, 126, 150, 161, 164, 149, 171, 193, 250, 20, 192, 26, 251, 15, 241, 36, 219, 23, 90, 244, 250, 16, 146, 128, 34, 177, 6, 113, 201, 173, 8, 82, 162, 25, 167, 113, 62, 236, 0}
	// 用户随机选择交易密钥
	txKeyBytes, err1 := GenTxKey()
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	// 用户对称加密交易
	cipherTx, err1 := EncryptTx(txKeyBytes, txBytes)
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	// 用户使用委员会公钥加密交易密钥，得到门限加密密文
	c1, cipherTxKeyBytes, err2 := EncryptIBVE(userInput.PK, txKeyBytes)
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	// 用户发送抗审查交易
	txCR := &TxCR{
		C:           c1,
		CipherTxKey: cipherTxKeyBytes,
		CipherTx:    cipherTx,
	}

	// 委员会成员计算轮份额
	roundShareList := make([]*RoundShare, n)
	var err error
	for i := 0; i < n; i++ {
		roundShare, roundShareProof, err := CalcRoundShareOfDPVSS(
			uint32(i+1),
			committeeMemberInputList[i].H,
			txCR.C,
			committeeMemberInputList[i].DPVSSConfigForMember.ShareCommits[i],
			committeeMemberInputList[i].DPVSSConfigForMember.Share,
		)
		if err != nil {
			fmt.Println(err)
		}
		roundShareList[i] = &RoundShare{
			Index: uint32(i + 1),
			Piece: roundShare,
			Proof: roundShareProof,
		}
	}
	// 委员会成员发送轮份额和证明

	// 领导者验证轮份额
	leaderInput := committeeMemberInputList[0] // 实际中需要根据IsLeader字段判断
	leaderInput.DPVSSConfigForMember.IsLeader = true
	var validRoundSharesBytes []*RoundShare
	for i := 0; i < n; i++ {
		verifyRoundShareRes := VerifyRoundShareOfDPVSS(
			roundShareList[i].Index,
			leaderInput.H,
			txCR.C,
			leaderInput.DPVSSConfigForMember.ShareCommits[i],
			roundShareList[i].Piece, roundShareList[i].Proof)
		// 有效则使用
		if verifyRoundShareRes {
			validRoundSharesBytes = append(validRoundSharesBytes, roundShareList[i])
		}
	}

	// 领导者重构门限加密密文
	roundSecretBytes := RecoverRoundSecret(threshold, validRoundSharesBytes)
	if roundSecretBytes == nil {
		// 共识失败，丢弃交易
		return
	}

	// 领导者解密门限加密的交易密钥密文，得到交易密钥
	decryptedTxKeyBytes, err := DecryptIBVE(roundSecretBytes, txCR.CipherTxKey)
	if err != nil {
		fmt.Println(err)
	}
	// 领导者对称解密交易密文，得到交易明文
	decryptedTx, err := DecryptTx(decryptedTxKeyBytes, txCR.CipherTx)
	if err != nil {
		fmt.Println(err)
	}
	// 领导者记录交易明文
	txCR.Tx = decryptedTx
	// 领导者公开txCR
	// 委员会验证交易明文解密正确
	for i := 0; i < n; i++ {
		res := VerifyIBVE(roundSecretBytes, committeeMemberInputList[i].PK, txCR.CipherTxKey)
		if !res {
			fmt.Println("VerifyIBVE error")
		}
	}
	fmt.Println("tx: ", txKeyBytes)
	fmt.Println("decryptedTxKeyBytes: ", decryptedTxKeyBytes)
	fmt.Println("tx: ", txBytes)
	fmt.Println("decryptedTx: ", decryptedTx)
}
