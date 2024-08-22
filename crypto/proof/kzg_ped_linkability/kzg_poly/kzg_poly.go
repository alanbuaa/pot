package kzg_ped_linkability

import (
	"crypto/rand"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	poly "github.com/zzz136454872/upgradeable-consensus/crypto/types/poly/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
)

var (
	group1 = NewG1()
)

type PolyLinkProof struct {
	KZGCommit *PointG1
	PedCommit *PointG1
	A         *PointG1
	AHat      *PointG1
	ZVector   []*Fr
	Omega     *Fr
}

func CreateProof(s *srs.SRS, hBasePoints []*PointG1, h *PointG1, kzgCommit *PointG1, pedCommit *PointG1, p *poly.UVPolynomial, r *Fr) *PolyLinkProof {
	size := p.Degree() + 1
	rVector := make([]*Fr, size)
	for i := uint32(0); i < size; i++ {
		// TODO
		rVector[i], _ = NewFr().Rand(rand.Reader)
	}
	// r'
	rDot, _ := NewFr().Rand(rand.Reader)

	// A
	A := group1.Zero()
	for i := uint32(0); i < size; i++ {
		group1.Add(A, A, group1.MulScalar(group1.New(), s.G1Power(i), rVector[i]))
	}
	AHat := group1.MulScalar(group1.New(), h, rDot)
	for i := uint32(0); i < size; i++ {
		group1.Add(AHat, AHat, group1.MulScalar(group1.New(), hBasePoints[i], rDot))
	}

	// calculate challenge
	challenge := HashToFr(append(group1.ToBytes(A), group1.ToBytes(AHat)...))

	// calculate {z_i}
	zVector := make([]*Fr, size)
	for i := uint32(0); i < size; i++ {
		zVector[i] = NewFr().Add(p.Coeffs[i], NewFr().Mul(challenge, rVector[i]))
	}
	omega := NewFr().Add(r, NewFr().Mul(challenge, rDot))
	return &PolyLinkProof{
		KZGCommit: kzgCommit,
		PedCommit: pedCommit,
		A:         A,
		AHat:      AHat,
		ZVector:   zVector,
		Omega:     omega,
	}
}

func VerifyProof(s *srs.SRS, hBasePoints []*PointG1, h *PointG1, polyLinkProof *PolyLinkProof) bool {
	size := uint32(len(polyLinkProof.ZVector))
	// calculate challenge
	challenge := HashToFr(append(group1.ToBytes(polyLinkProof.A), group1.ToBytes(polyLinkProof.AHat)...))

	left1 := group1.Add(group1.New(), polyLinkProof.KZGCommit, group1.MulScalar(group1.New(), polyLinkProof.A, challenge))
	right1 := group1.Zero()
	for i := uint32(0); i < size; i++ {
		group1.Add(right1, right1, group1.MulScalar(group1.New(), s.G1Power(i), polyLinkProof.ZVector[i]))
	}
	if !group1.Equal(left1, right1) {
		return false
	}
	left2 := group1.Add(group1.New(), polyLinkProof.PedCommit, group1.MulScalar(group1.New(), polyLinkProof.AHat, challenge))
	right2 := group1.MulScalar(group1.New(), h, polyLinkProof.Omega)
	for i := uint32(0); i < size; i++ {
		group1.Add(right2, right2, group1.MulScalar(group1.New(), hBasePoints[i], polyLinkProof.ZVector[i]))
	}
	return group1.Equal(left2, right2)
}
