package blockchain_api

import (
	"bytes"
	"encoding/binary"
	dleq "github.com/zzz136454872/upgradeable-consensus/crypto/proof/dleq/bls12381"
	mrpvss "github.com/zzz136454872/upgradeable-consensus/crypto/share/mrpvss/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

type CommitteeConfig struct {
	H                    []byte      // 生成元h
	PK                   []byte      // 委员会公钥
	DPVSSConfigForMember DPVSSConfig // 委员会成员拥有
}

type DPVSSConfig struct {
	Index        uint32   // 自己的位置
	ShareCommits [][]byte // 份额承诺
	Share        []byte   // 份额
	IsLeader     bool     // 标识领导者
}

type RoundShare struct {
	Index uint32
	Piece []byte
	Proof []byte
}

func (r *RoundShare) ToBytes() []byte {
	buffer := bytes.Buffer{}
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, r.Index)
	buffer.Write(buf)
	buffer.Write(r.Piece)
	buffer.Write(r.Proof)
	return buffer.Bytes()
}

func CalcRoundShareOfDPVSS(index uint32, hBytes []byte, cBytes []byte, shareCommitBytes []byte, shareBytes []byte) (roundShareBytes []byte, roundShareProofBytes []byte, err error) {
	h, err := g1.FromCompressed(hBytes)
	if err != nil {
		return nil, nil, err
	}
	c, err := g1.FromCompressed(cBytes)
	if err != nil {
		return nil, nil, err
	}
	shareCommit, err := g1.FromCompressed(shareCommitBytes)
	if err != nil {
		return nil, nil, err
	}
	share := bls12381.NewFr().FromBytes(shareBytes)
	roundShare, roundShareProof, err := mrpvss.CalcRoundShare(index, h, c, shareCommit, share)
	return g1.ToCompressed(roundShare), roundShareProof.ToBytes(), err
}

func VerifyRoundShareOfDPVSS(index uint32, hBytes []byte, cBytes []byte, shareCommitBytes []byte, roundShareBytes []byte, roundShareProofBytes []byte) bool {
	h, err := g1.FromCompressed(hBytes)
	if err != nil {
		return false
	}
	c, err := g1.FromCompressed(cBytes)
	if err != nil {
		return false
	}
	shareCommit, err := g1.FromCompressed(shareCommitBytes)
	if err != nil {
		return false
	}
	roundShare, err := g1.FromCompressed(roundShareBytes)
	if err != nil {
		return false
	}
	roundShareProof, err := new(dleq.Proof).FromBytes(roundShareProofBytes)
	if err != nil {
		return false
	}
	return mrpvss.VerifyRoundShare(index, h, c, shareCommit, roundShare, roundShareProof)
}

func RecoverRoundSecret(threshold uint32, roundShares []*RoundShare) (roundSecretBytes []byte) {
	roundShareNum := len(roundShares)
	indices := make([]uint32, roundShareNum)
	roundSharesInner := make([]*bls12381.PointG1, roundShareNum)
	var err error
	for i := 0; i < roundShareNum; i++ {
		indices[i] = uint32(i + 1)
		roundSharesInner[i], err = g1.FromCompressed(roundShares[i].Piece)
		if err != nil {
			return nil
		}
	}
	roundSecret := mrpvss.RecoverRoundSecret(threshold, indices, roundSharesInner)
	return g1.ToCompressed(roundSecret)
}
