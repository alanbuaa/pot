package schnorr_proof

import (
	"bytes"
	"crypto/rand"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

type SchnorrProof struct {
	// rand commit T = G^r
	T *PointG1
	// response s = r + cx mod q
	Response *Fr
}

func (p *SchnorrProof) ToBytes() []byte {
	group1 := NewG1()
	buffer := bytes.Buffer{}
	buffer.Write(group1.ToCompressed(p.T))
	buffer.Write(p.Response.ToBytes())
	return buffer.Bytes()
}
func (p *SchnorrProof) FromBytes(data []byte) (*SchnorrProof, error) {
	group1 := NewG1()
	pointG1Buf := make([]byte, 48)
	frBuf := make([]byte, 32)
	buffer := bytes.Buffer{}
	buffer.Write(data)

	_, err := buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	T, err := group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}

	_, err = buffer.Read(frBuf)
	if err != nil {
		return nil, err
	}
	R := NewFr().FromBytes(frBuf)
	return &SchnorrProof{
		T:        T,
		Response: R,
	}, nil
}

// CreateWitness base point g ,product point h = ^x
func CreateWitness(g *PointG1, h *PointG1, x *Fr) *SchnorrProof {
	group1 := NewG1()
	r, _ := NewFr().Rand(rand.Reader)
	t := group1.MulScalar(group1.New(), g, r)
	// challenge c = H(g,h,T)
	c := HashToFr(append(append(group1.ToBytes(g), group1.ToBytes(h)...), group1.ToBytes(t)...))
	// s = r + cx mod q
	res := NewFr().Mul(c, x)
	res.Add(res, r)
	return &SchnorrProof{
		T:        t,
		Response: res,
	}
}

func Verify(g *PointG1, h *PointG1, proof *SchnorrProof) bool {
	group1 := NewG1()
	// challenge c = H(g,h,T)
	c := HashToFr(append(append(group1.ToBytes(g), group1.ToBytes(h)...), group1.ToBytes(proof.T)...))
	// G^s = T · H^c
	left := group1.MulScalar(group1.New(), g, proof.Response)
	right := group1.Add(group1.New(), proof.T, group1.MulScalar(group1.New(), h, c))
	return group1.Equal(left, right)
}
