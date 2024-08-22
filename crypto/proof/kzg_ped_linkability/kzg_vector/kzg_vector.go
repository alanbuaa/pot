package kzg_ped_linkability

import (
	"crypto/rand"
	"errors"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	roots_of_unity "github.com/zzz136454872/upgradeable-consensus/crypto/types/domain/bls12_381"
	poly "github.com/zzz136454872/upgradeable-consensus/crypto/types/poly/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
)

var (
	group1 = NewG1()
)

type VectorLinkProof struct {
	KZGCommit *PointG1
	PedCommit *PointG1
	A         *PointG1
	AHat      *PointG1
	ZVector   []*Fr
	Omega     *Fr
}

func CreateProof(s *srs.SRS, hBasePoints []*PointG1, h *PointG1, kzgCommit *PointG1, pedCommit *PointG1, values []*Fr, r *Fr) (*VectorLinkProof, error) {

	size := len(hBasePoints)
	if size == 0 || size != len(values) {
		return nil, errors.New("invalid parameter")
	}

	rVector := make([]*Fr, size)
	for i := 0; i < size; i++ {
		rVector[i], _ = NewFr().Rand(rand.Reader)
	}
	// r'
	rDot, _ := NewFr().Rand(rand.Reader)

	rootsOfUnity, err := roots_of_unity.CalcRootsOfUnity(uint32(size))
	if err != nil {
		return nil, err
	}
	APoly := poly.Interpolate(rootsOfUnity, rVector)

	// A
	A := group1.Zero()
	for i := 0; i < size; i++ {
		group1.Add(A, A, group1.MulScalar(group1.New(), s.G1Power(uint32(i)), APoly.Coeffs[i]))
	}
	AHat := group1.MulScalar(group1.New(), h, rDot)
	for i := 0; i < size; i++ {
		group1.Add(AHat, AHat, group1.MulScalar(group1.New(), hBasePoints[i], rDot))
	}

	// calculate challenge
	challenge := HashToFr(append(group1.ToBytes(A), group1.ToBytes(AHat)...))

	// calculate {z_i}
	zVector := make([]*Fr, size)
	for i := 0; i < size; i++ {
		zVector[i] = NewFr().Add(values[i], NewFr().Mul(challenge, rVector[i]))
	}
	omega := NewFr().Add(r, NewFr().Mul(challenge, rDot))
	return &VectorLinkProof{
		KZGCommit: kzgCommit,
		PedCommit: pedCommit,
		A:         A,
		AHat:      AHat,
		ZVector:   zVector,
		Omega:     omega,
	}, nil
}

func VerifyProof(s *srs.SRS, hBasePoints []*PointG1, h *PointG1, vectorLinkProof *VectorLinkProof) bool {
	size := uint32(len(vectorLinkProof.ZVector))
	// calculate challenge
	challenge := HashToFr(append(group1.ToBytes(vectorLinkProof.A), group1.ToBytes(vectorLinkProof.AHat)...))

	left1 := group1.Add(group1.New(), vectorLinkProof.KZGCommit, group1.MulScalar(group1.New(), vectorLinkProof.A, challenge))

	rootsOfUnity, err := roots_of_unity.CalcRootsOfUnity(size)
	if err != nil {
		return false
	}
	right1Poly := poly.Interpolate(rootsOfUnity, vectorLinkProof.ZVector)

	right1 := group1.Zero()
	for i := uint32(0); i < size; i++ {
		group1.Add(right1, right1, group1.MulScalar(group1.New(), s.G1Power(i), right1Poly.Coeffs[i]))
	}
	if !group1.Equal(left1, right1) {
		return false
	}

	left2 := group1.Add(group1.New(), vectorLinkProof.PedCommit, group1.MulScalar(group1.New(), vectorLinkProof.AHat, challenge))
	right2 := group1.MulScalar(group1.New(), h, vectorLinkProof.Omega)
	for i := uint32(0); i < size; i++ {
		group1.Add(right2, right2, group1.MulScalar(group1.New(), hBasePoints[i], vectorLinkProof.ZVector[i]))
	}
	return group1.Equal(left2, right2)
}
