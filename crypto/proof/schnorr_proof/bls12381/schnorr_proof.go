package schnorr_proof

import (
	"crypto/rand"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

var (
	group1 = NewG1()
)

type SchnorrProof struct {
	// rand commit T = G^r
	T *PointG1
	// response s = r + cx mod q
	Response *Fr
}

// CreateWitness base point g ,product point h = ^x
func CreateWitness(g *PointG1, h *PointG1, x *Fr) *SchnorrProof {
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
	// challenge c = H(g,h,T)
	c := HashToFr(append(append(group1.ToBytes(g), group1.ToBytes(h)...), group1.ToBytes(proof.T)...))
	// G^s = T · H^c
	left := group1.MulScalar(group1.New(), g, proof.Response)
	right := group1.Add(group1.New(), proof.T, group1.MulScalar(group1.New(), h, c))
	return group1.Equal(left, right)
}
