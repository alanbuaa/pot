package mrpvss

import (
	"bytes"
	dleq "github.com/zzz136454872/upgradeable-consensus/crypto/proof/dleq/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

type EncShare struct {
	// 加密份额
	A     *PointG1
	BList []*PointG1
	// 分发份额证明
	DealProof *dleq.Proof
}

func NewEmptyEncShare() *EncShare {
	bList := make([]*PointG1, 32)
	for i := 0; i < 32; i++ {
		bList[i] = group1.Zero()
	}
	return &EncShare{
		A:         group1.Zero(),
		BList:     bList,
		DealProof: nil,
	}
}

func (e *EncShare) ToBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.Write(group1.ToCompressed(e.A))
	for _, b := range e.BList {
		buffer.Write(group1.ToCompressed(b))
	}
	buffer.Write(e.DealProof.ToBytes())
	return buffer.Bytes()
}

func (e *EncShare) FromBytes(data []byte) (*EncShare, error) {
	pointG1Buf := make([]byte, 48)
	bList := make([]*PointG1, 32)
	buffer := bytes.NewBuffer(data)

	_, err := buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	A, err := group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	for i := 0; i < 32; i++ {
		_, err = buffer.Read(pointG1Buf)
		if err != nil {
			return nil, err
		}
		bList[i], err = group1.FromCompressed(pointG1Buf)
		if err != nil {
			return nil, err
		}
	}
	dleqProof, err := new(dleq.Proof).FromBytes(buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return &EncShare{
		A:         A,
		BList:     bList,
		DealProof: dleqProof,
	}, nil
}
