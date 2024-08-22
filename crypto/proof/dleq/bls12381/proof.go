package dleq

import (
	"crypto/rand"
	"encoding/binary"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

var (
	g1curve = NewG1()
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

func (d *DLEQ) Prove(alpha *Fr) *Proof {
	w, _ := NewFr().Rand(rand.Reader)
	a1 := g1curve.MulScalar(g1curve.New(), d.G1, w)
	a2 := g1curve.MulScalar(g1curve.New(), d.G2, w)
	// g1, h_1, g2, h_2, a_1, a_2; i
	challenge := g1curve.ToBytes(d.G1)
	challenge = append(challenge, g1curve.ToBytes(d.H1)...)
	challenge = append(challenge, g1curve.ToBytes(d.G2)...)
	challenge = append(challenge, g1curve.ToBytes(d.H2)...)
	challenge = append(challenge, g1curve.ToBytes(a1)...)
	challenge = append(challenge, g1curve.ToBytes(a2)...)
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
	challenge := g1curve.ToBytes(d.G1)
	challenge = append(challenge, g1curve.ToBytes(d.H1)...)
	challenge = append(challenge, g1curve.ToBytes(d.G2)...)
	challenge = append(challenge, g1curve.ToBytes(d.H2)...)
	challenge = append(challenge, g1curve.ToBytes(proof.A1)...)
	challenge = append(challenge, g1curve.ToBytes(proof.A2)...)
	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, d.Index)
	challenge = append(challenge, indexBytes...)
	c := HashToFr(challenge)
	a1 := g1curve.Add(g1curve.New(), g1curve.MulScalar(g1curve.New(), d.G1, proof.R),
		g1curve.MulScalar(g1curve.New(), d.H1, c))
	a2 := g1curve.Add(g1curve.New(), g1curve.MulScalar(g1curve.New(), d.G2, proof.R),
		g1curve.MulScalar(g1curve.New(), d.H2, c))
	if !g1curve.Equal(a1, proof.A1) {
		return false
	} else if !g1curve.Equal(a2, proof.A2) {
		return false
	}
	return true
}

func Verify(Index uint32, G1 *PointG1, H1 *PointG1, G2 *PointG1, H2 *PointG1, proof *Proof) bool {
	// g1, h_1, g2, h_2, a_1, a_2
	challenge := g1curve.ToBytes(G1)
	challenge = append(challenge, g1curve.ToBytes(H1)...)
	challenge = append(challenge, g1curve.ToBytes(G2)...)
	challenge = append(challenge, g1curve.ToBytes(H2)...)
	challenge = append(challenge, g1curve.ToBytes(proof.A1)...)
	challenge = append(challenge, g1curve.ToBytes(proof.A2)...)
	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, Index)
	challenge = append(challenge, indexBytes...)
	c := HashToFr(challenge)
	a1 := g1curve.Add(g1curve.New(), g1curve.MulScalar(g1curve.New(), G1, proof.R),
		g1curve.MulScalar(g1curve.New(), H1, c))
	a2 := g1curve.Add(g1curve.New(), g1curve.MulScalar(g1curve.New(), G2, proof.R),
		g1curve.MulScalar(g1curve.New(), H2, c))
	if !g1curve.Equal(a1, proof.A1) {
		return false
	} else if !g1curve.Equal(a2, proof.A2) {
		return false
	}
	return true
}
