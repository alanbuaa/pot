package kzg_ped_linkability

import (
	"fmt"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	roots_of_unity "github.com/zzz136454872/upgradeable-consensus/crypto/types/domain/bls12_381"
	poly "github.com/zzz136454872/upgradeable-consensus/crypto/types/poly/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"
	"testing"
	"time"
)

func TestVerifyProof(t *testing.T) {
	group1 := NewG1()
	quota := uint32(4)

	g1Degree, g2Degree := utils.CalcMinLimitedDegree(4, quota)
	fmt.Println("start gen srs")
	start := time.Now()
	s, _ := srs.NewSRS(g1Degree, g2Degree)
	fmt.Println("gen srs time:", time.Since(start))

	generator := s.G1PowerOf(0)

	vVector := []*Fr{FrFromInt(4), FrFromInt(2), FrFromInt(3), FrFromInt(1)}
	h := group1.Affine(group1.MulScalar(group1.New(), generator, FrFromInt(123)))
	r := FrFromInt(3)

	secretKeys := make([]*Fr, quota)
	publicKeys := make([]*PointG1, quota)
	for i := uint32(0); i < quota; i++ {
		secretKeys[i] = FrFromUInt32(i + 1)
		publicKeys[i] = group1.Affine(group1.MulScalar(group1.New(), generator, secretKeys[i]))
	}

	rootsOfUnity, err := roots_of_unity.CalcRootsOfUnity(quota)
	if err != nil {
		fmt.Println(err)
		return
	}
	APoly := poly.Interpolate(rootsOfUnity, vVector)

	kzgCM := group1.Zero()
	for i := uint32(0); i < quota; i++ {
		group1.Add(kzgCM, kzgCM, group1.MulScalar(group1.New(), s.G1PowerOf(i), APoly.Coeffs[i]))
	}
	kzgCM = group1.Affine(kzgCM)
	fmt.Println("kzgCM:", kzgCM)

	// h^r \prod h_i^(v_i)
	pedCM := group1.MulScalar(group1.New(), h, r)
	for i := uint32(0); i < quota; i++ {
		group1.Add(pedCM, pedCM, group1.MulScalar(group1.New(), publicKeys[i], vVector[i]))
	}
	group1.Affine(pedCM)
	fmt.Println("pedCM:", pedCM)
	lnkProof, err := CreateProof(s, publicKeys, h, kzgCM, pedCM, vVector, r)
	if err != nil {
		fmt.Println(err)
		return
	}
	res := VerifyProof(s, publicKeys, h, lnkProof)
	fmt.Println(res)
}
func TestVectorLinkProof_FromBytes_ToBytes(t *testing.T) {
	group1 := NewG1()
	quota := uint32(4)

	g1Degree, g2Degree := utils.CalcMinLimitedDegree(4, quota)
	fmt.Println("start gen srs")
	start := time.Now()
	s, _ := srs.NewSRS(g1Degree, g2Degree)
	fmt.Println("gen srs time:", time.Since(start))

	generator := s.G1PowerOf(0)

	vVector := []*Fr{FrFromInt(4), FrFromInt(2), FrFromInt(3), FrFromInt(1)}
	h := group1.Affine(group1.MulScalar(group1.New(), generator, FrFromInt(123)))
	r := FrFromInt(3)

	secretKeys := make([]*Fr, quota)
	publicKeys := make([]*PointG1, quota)
	for i := uint32(0); i < quota; i++ {
		secretKeys[i] = FrFromUInt32(i + 1)
		publicKeys[i] = group1.Affine(group1.MulScalar(group1.New(), generator, secretKeys[i]))
	}

	rootsOfUnity, err := roots_of_unity.CalcRootsOfUnity(quota)
	if err != nil {
		fmt.Println(err)
		return
	}
	APoly := poly.Interpolate(rootsOfUnity, vVector)

	kzgCM := group1.Zero()
	for i := uint32(0); i < quota; i++ {
		group1.Add(kzgCM, kzgCM, group1.MulScalar(group1.New(), s.G1PowerOf(i), APoly.Coeffs[i]))
	}
	group1.Affine(kzgCM)
	fmt.Println("kzgCM:", kzgCM)

	// h^r \prod h_i^(v_i)
	pedCM := group1.MulScalar(group1.New(), h, r)
	for i := uint32(0); i < quota; i++ {
		group1.Add(pedCM, pedCM, group1.MulScalar(group1.New(), publicKeys[i], vVector[i]))
	}
	group1.Affine(pedCM)
	fmt.Println("pedCM:", pedCM)
	lnkProof, err := CreateProof(s, publicKeys, h, kzgCM, pedCM, vVector, r)
	if err != nil {
		fmt.Println(err)
		return
	}

	lnkProofBytes := lnkProof.ToBytes()

	decodedProof, err := new(VectorLinkProof).FromBytes(lnkProofBytes)
	if err != nil {
		return
	}
	res := VerifyProof(s, publicKeys, h, decodedProof)
	fmt.Println(res)
}
