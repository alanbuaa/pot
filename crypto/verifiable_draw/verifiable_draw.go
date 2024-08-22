package verifiable_draw

import (
	"crypto/rand"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/crypto/pb"
	"github.com/zzz136454872/upgradeable-consensus/crypto/proof/caulk_plus"
	kzg_ped_linkability "github.com/zzz136454872/upgradeable-consensus/crypto/proof/kzg_ped_linkability/kzg_vector"
	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"
)

var (
	group1 = NewG1()
)

type DrawProof struct {
	// prevRCommit^r
	RCommit         *PointG1
	SelectedPubKeys []*PointG1
	B               *Fr
	DBlind          *PointG1
	// kzg commitment of {a_i}
	ACommit *PointG1
	// kzg commitment of {θ_i}
	VCommit        *PointG1
	CaulkPlusProof *pb.MultiProof
	SchnorrProof   *schnorr_proof.SchnorrProof
	LnkProof       *kzg_ped_linkability.VectorLinkProof
}

func Draw(s *srs.SRS, candidatesNum uint32, candidatesPubKey []*PointG1, quota uint32, secretVector []uint32, prevRCommit *PointG1) (*DrawProof, error) {
	// blind factor
	r, _ := NewFr().Rand(rand.Reader)
	// r := FrFromInt(123)
	fmt.Println("r:", r)

	selectedPubKeys := make([]*PointG1, quota)
	var selectedPubKeyBytes []byte
	for i := uint32(0); i < quota; i++ {
		selectedPubKeys[i] = group1.Affine(group1.MulScalar(group1.New(), candidatesPubKey[secretVector[i]-1], r))
		fmt.Printf(" pk %d:%v\n", secretVector[i], candidatesPubKey[secretVector[i]-1])
		fmt.Printf("spk %d:%v\n\n", i+1, selectedPubKeys[i])
		selectedPubKeyBytes = append(selectedPubKeyBytes, group1.ToBytes(selectedPubKeys[i])...)
	}
	// TODO
	var candidatesPubKeyBytes []byte
	for i := uint32(0); i < candidatesNum; i++ {
		candidatesPubKeyBytes = append(candidatesPubKeyBytes, group1.ToBytes(candidatesPubKey[i])...)
	}

	preImageBytes := append(candidatesPubKeyBytes, selectedPubKeyBytes...)

	// construct vector a
	aVector := make([]*Fr, quota)
	for i := uint32(0); i < quota; i++ {
		aVector[i] = HashToFr(append(preImageBytes, utils.Uint32ToBytes(i)...))
	}
	// calc kzg commitment and poly
	// aCommit, aPoly, err := kzg_vector_commit.Commit(srs, aVector)
	// if err != nil {
	// 	return nil, err
	// }
	// construct vector v
	vVector := make([]*Fr, candidatesNum)
	for i := uint32(0); i < candidatesNum; i++ {
		vVector[i] = NewFr().Zero()
	}
	for i := uint32(0); i < quota; i++ {
		// θ_s(i) = a_i
		vVector[secretVector[i]-1] = NewFr().Set(aVector[i])
	}
	// construct caulk plus proof of vector a and vector v
	caulkPlusProof, err := caulk_plus.CreateMultiProof(candidatesNum, vVector, quota, aVector)
	if err != nil {
		return nil, err
	}
	aCommit := caulk_plus.ConvertG1AffineToPointG1(caulkPlusProof.ACommit)
	vCommit := caulk_plus.ConvertG1AffineToPointG1(caulkPlusProof.CCommit)

	// generate blind factor b, b != r^(-1)
	b := NewFr()
	for {
		_, err = b.Rand(rand.Reader)
		if err != nil || !NewFr().Mul(b, r).IsOne() {
			break
		}
	}
	B := NewFr().Mul(b, r)
	domain := []byte("BLS12381G1_XMD:SHA-256_SSWU_RO_TESTGEN")
	h, err := group1.HashToCurve(candidatesPubKeyBytes, domain)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// calc C
	C := group1.Zero()
	for i := 0; i < int(quota); i++ {
		C = group1.Add(group1.New(), C, group1.MulScalar(group1.New(), selectedPubKeys[i], aVector[i]))
	}
	// calc C' = h^B · C
	CBlind := group1.Add(group1.New(), C, group1.MulScalar(group1.New(), h, B))

	// calc D
	D := group1.Zero()
	for i := 0; i < int(candidatesNum); i++ {
		D = group1.Add(group1.New(), D, group1.MulScalar(group1.New(), candidatesPubKey[i], vVector[i]))
	}

	// calc D' = h^b · D
	DBlind := group1.Add(group1.New(), D, group1.MulScalar(group1.New(), h, b))

	// generate Schnorr proof of C' = D' ^ r
	schnorrProof := schnorr_proof.CreateWitness(DBlind, CBlind, r)

	// generate kzg and pedersen linkability proof
	lnkProof, err := kzg_ped_linkability.CreateProof(s, candidatesPubKey, h, vCommit, DBlind, vVector, B)
	if err != nil {
		return nil, err
	}

	return &DrawProof{
		RCommit: group1.MulScalar(group1.New(), prevRCommit, r),
		// TODO DLEQ(prevRCommit RCommit, D' C')
		SelectedPubKeys: selectedPubKeys,
		B:               B,
		DBlind:          DBlind,
		ACommit:         aCommit,
		VCommit:         vCommit,
		CaulkPlusProof:  caulkPlusProof,
		SchnorrProof:    schnorrProof,
		LnkProof:        lnkProof,
	}, nil
}

func Verify(s *srs.SRS, candidatesNum uint32, candidatesPubKey []*PointG1, quota uint32, drawProof *DrawProof) bool {
	if quota != uint32(len(drawProof.SelectedPubKeys)) {
		return false
	}

	var selectedPubKeyBytes []byte
	for i := uint32(0); i < quota; i++ {
		selectedPubKeyBytes = append(selectedPubKeyBytes, group1.ToBytes(drawProof.SelectedPubKeys[i])...)
	}

	var candidatesPubKeyBytes []byte
	for i := uint32(0); i < candidatesNum; i++ {
		candidatesPubKeyBytes = append(candidatesPubKeyBytes, group1.ToBytes(candidatesPubKey[i])...)
	}
	preImageBytes := append(candidatesPubKeyBytes, selectedPubKeyBytes...)

	// construct vector a
	aVector := make([]*Fr, quota)
	for i := uint32(0); i < quota; i++ {
		aVector[i] = HashToFr(append(preImageBytes, utils.Uint32ToBytes(i)...))
	}

	domain := []byte("BLS12381G1_XMD:SHA-256_SSWU_RO_TESTGEN")
	h, err := group1.HashToCurve(candidatesPubKeyBytes, domain)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// verify C'
	C := group1.Zero()
	for i := 0; i < int(quota); i++ {
		C = group1.Add(group1.New(), C, group1.MulScalar(group1.New(), drawProof.SelectedPubKeys[i], aVector[i]))
	}
	// calc C' = h^B · C
	CBlind := group1.Add(group1.New(), C, group1.MulScalar(group1.New(), h, drawProof.B))

	// verify caulk plus
	res, err := caulk_plus.VerifyMultiProof(drawProof.CaulkPlusProof)
	if err != nil || !res {
		return false
	}

	// verify schnorr proof of C' = D' ^ r
	if !schnorr_proof.Verify(drawProof.DBlind, CBlind, drawProof.SchnorrProof) {
		return false
	}
	return kzg_ped_linkability.VerifyProof(s, candidatesPubKey, h, drawProof.LnkProof)
}

// IsSelected check y'^(1/x) ?= g^r, y' = y^r
func IsSelected(sk *Fr, rCommit *PointG1, selectedKeys []*PointG1) (bool, *PointG1) {
	skInv := NewFr().Inverse(sk)
	keysLen := len(selectedKeys)
	for i := 0; i < keysLen; i++ {
		if group1.Equal(rCommit, group1.MulScalar(group1.New(), selectedKeys[i], skInv)) {
			return true, selectedKeys[i]
		}
	}
	return false, nil
}
