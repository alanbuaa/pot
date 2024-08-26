package dleq

import (
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"bytes"
	"crypto/rand"
	"encoding/binary"
)

var (
	group1 = NewG1()
	group2 = NewG2()
)

type DLEQ struct {
	Index uint32
	G1    *PointG1
	H1    *PointG1
	G2    *PointG1
	H2    *PointG1
}

type Proof struct {
	A1 *PointG1
	A2 *PointG1
	R  *Fr
}

func (p *Proof) ToBytes() []byte {
	buffer := bytes.Buffer{}
	buffer.Write(group1.ToCompressed(p.A1))
	buffer.Write(group1.ToCompressed(p.A2))
	buffer.Write(p.R.ToBytes())
	return buffer.Bytes()
}

func (p *Proof) FromBytes(data []byte) (*Proof, error) {
	pointG1Buf := make([]byte, 48)
	frBuf := make([]byte, 32)
	buffer := bytes.Buffer{}
	buffer.Write(data)

	_, err := buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	A1, err := group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	_, err = buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	A2, err := group1.FromCompressed(pointG1Buf)
	_, err = buffer.Read(frBuf)
	if err != nil {
		return nil, err
	}
	R := NewFr().FromBytes(frBuf)
	return &Proof{
		A1: A1,
		A2: A2,
		R:  R,
	}, nil
}

func (d *DLEQ) Prove(alpha *Fr) *Proof {
	w, _ := NewFr().Rand(rand.Reader)
	a1 := group1.MulScalar(group1.New(), d.G1, w)
	a2 := group1.MulScalar(group1.New(), d.G2, w)
	// g1, h_1, g2, h_2, a_1, a_2; i
	challenge := group1.ToBytes(d.G1)
	challenge = append(challenge, group1.ToBytes(d.H1)...)
	challenge = append(challenge, group1.ToBytes(d.G2)...)
	challenge = append(challenge, group1.ToBytes(d.H2)...)
	challenge = append(challenge, group1.ToBytes(a1)...)
	challenge = append(challenge, group1.ToBytes(a2)...)
	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, d.Index)
	challenge = append(challenge, indexBytes...)
	c := HashToFr(challenge)
	r := NewFr()
	tmp := NewFr()
	tmp.Mul(alpha, c)
	r.Sub(w, tmp)
	return &Proof{
		A1: a1,
		A2: a2,
		R:  r,
	}
}

func (d *DLEQ) Verify(proof *Proof) bool {
	// g1, h_1, g2, h_2, a_1, a_2
	challenge := group1.ToBytes(d.G1)
	challenge = append(challenge, group1.ToBytes(d.H1)...)
	challenge = append(challenge, group1.ToBytes(d.G2)...)
	challenge = append(challenge, group1.ToBytes(d.H2)...)
	challenge = append(challenge, group1.ToBytes(proof.A1)...)
	challenge = append(challenge, group1.ToBytes(proof.A2)...)
	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, d.Index)
	challenge = append(challenge, indexBytes...)
	c := HashToFr(challenge)
	a1 := group1.Add(group1.New(), group1.MulScalar(group1.New(), d.G1, proof.R),
		group1.MulScalar(group1.New(), d.H1, c))
	a2 := group1.Add(group1.New(), group1.MulScalar(group1.New(), d.G2, proof.R),
		group1.MulScalar(group1.New(), d.H2, c))
	if !group1.Equal(a1, proof.A1) {
		return false
	} else if !group1.Equal(a2, proof.A2) {
		return false
	}
	return true
}

func Verify(Index uint32, G1 *PointG1, H1 *PointG1, G2 *PointG1, H2 *PointG1, proof *Proof) bool {
	// g1, h_1, g2, h_2, a_1, a_2
	challenge := group1.ToBytes(G1)
	challenge = append(challenge, group1.ToBytes(H1)...)
	challenge = append(challenge, group1.ToBytes(G2)...)
	challenge = append(challenge, group1.ToBytes(H2)...)
	challenge = append(challenge, group1.ToBytes(proof.A1)...)
	challenge = append(challenge, group1.ToBytes(proof.A2)...)
	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, Index)
	challenge = append(challenge, indexBytes...)
	c := HashToFr(challenge)
	a1 := group1.Add(group1.New(), group1.MulScalar(group1.New(), G1, proof.R),
		group1.MulScalar(group1.New(), H1, c))
	a2 := group1.Add(group1.New(), group1.MulScalar(group1.New(), G2, proof.R),
		group1.MulScalar(group1.New(), H2, c))
	if !group1.Equal(a1, proof.A1) {
		return false
	} else if !group1.Equal(a2, proof.A2) {
		return false
	}
	return true
}
