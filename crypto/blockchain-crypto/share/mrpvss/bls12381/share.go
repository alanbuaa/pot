package mrpvss

import (
	dleq "blockchain-crypto/proof/dleq/bls12381"
	. "blockchain-crypto/types/curve/bls12381"
	"bytes"
	"fmt"
	"strings"
)

type EncShare struct {
	// 加密份额
	A     *PointG1
	BList []*PointG1
	// 分发份额证明
	DealProof *dleq.Proof
}

func (e *EncShare) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("EncShare{A: %v, ", e.A[0][0]))
	sb.WriteString("BList: [")
	// 写入 BList 字段的字符串表示
	for i, b := range e.BList {
		sb.WriteString(fmt.Sprintf("%v", b[0][0]))
		if i < len(e.BList)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(fmt.Sprintf("], DealProof: %v}", e.DealProof))
	return sb.String()
}

func (e *EncShare) DeepCopy() *EncShare {
	group1 := NewG1()
	ret := &EncShare{
		A:         nil,
		BList:     nil,
		DealProof: nil,
	}
	if e.A != nil {
		ret.A = e.A
	}
	if e.BList != nil {
		ret.BList = make([]*PointG1, 32)
		for i := 0; i < 32; i++ {
			if e.BList[i] != nil {
				ret.BList[i] = group1.New().Set(e.BList[i])
			}
		}
	}
	if e.DealProof != nil {
		ret.DealProof = e.DealProof.DeepCopy()
	}
	return ret
}

func NewEmptyEncShare() *EncShare {
	group1 := NewG1()
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
	group1 := NewG1()
	buffer := bytes.Buffer{}
	buffer.Write(group1.ToCompressed(e.A))
	for _, b := range e.BList {
		buffer.Write(group1.ToCompressed(b))
	}
	buffer.Write(e.DealProof.ToBytes())
	return buffer.Bytes()
}

func (e *EncShare) FromBytes(data []byte) (*EncShare, error) {
	group1 := NewG1()
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
