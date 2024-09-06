package verifiable_draw

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/crypto/pb"
	"github.com/zzz136454872/upgradeable-consensus/crypto/proof/caulk_plus"
	dleq "github.com/zzz136454872/upgradeable-consensus/crypto/proof/dleq/bls12381"
	kzg_ped_linkability "github.com/zzz136454872/upgradeable-consensus/crypto/proof/kzg_ped_linkability/kzg_vector"
	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"
	"google.golang.org/protobuf/proto"
)

var (
	group1 = NewG1()
)

type DrawProof struct {
	RCommit         *PointG1    // prevRCommit^r
	DLEQProof       *dleq.Proof // DLEQ(prevRCommit RCommit, D' C')
	SelectedPubKeys []*PointG1
	B               *Fr
	DBlind          *PointG1
	ACommit         *PointG1 // kzg commitment of {a_i}
	VCommit         *PointG1 // kzg commitment of {θ_i}
	CaulkPlusProof  *pb.MultiProof
	SchnorrProof    *schnorr_proof.SchnorrProof
	LnkProof        *kzg_ped_linkability.VectorLinkProof
}

func (d *DrawProof) ToBytes() ([]byte, error) {
	uint32Bytes := make([]byte, 4)
	uint64Bytes := make([]byte, 8)
	buffer := bytes.Buffer{}
	// RCommit
	buffer.Write(group1.ToCompressed(d.RCommit))
	// DLEQProof
	buffer.Write(d.DLEQProof.ToBytes())
	// SelectedPubKeys Size
	binary.BigEndian.PutUint32(uint32Bytes, uint32(len(d.SelectedPubKeys)))
	buffer.Write(uint32Bytes)
	// SelectedPubKeys
	for _, p := range d.SelectedPubKeys {
		buffer.Write(group1.ToCompressed(p))
	}
	// B
	buffer.Write(d.B.ToBytes())
	// DBlind
	buffer.Write(group1.ToCompressed(d.DBlind))
	// ACommit
	buffer.Write(group1.ToCompressed(d.ACommit))
	// VCommit
	buffer.Write(group1.ToCompressed(d.VCommit))
	// CaulkPlusProof
	caulkPlusProofData, err := proto.Marshal(d.CaulkPlusProof)
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint64(uint64Bytes, uint64(len(caulkPlusProofData)))
	buffer.Write(uint64Bytes)
	buffer.Write(caulkPlusProofData)
	// SchnorrProof
	buffer.Write(d.SchnorrProof.ToBytes())
	// LnkProof
	buffer.Write(d.LnkProof.ToBytes())
	return buffer.Bytes(), nil
}

func (d *DrawProof) FromBytes(data []byte) (*DrawProof, error) {
	pointG1Buf := make([]byte, 48)
	frBuf := make([]byte, 32)
	DLEQBuf := make([]byte, 128)
	uint32Buf := make([]byte, 4)
	uint64Buf := make([]byte, 8)
	SchnorrBuf := make([]byte, 80)
	buffer := bytes.NewBuffer(data)
	// RCommit
	_, err := buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	d.RCommit, err = group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	// DLEQProof
	_, err = buffer.Read(DLEQBuf)
	if err != nil {
		return nil, err
	}
	d.DLEQProof, err = new(dleq.Proof).FromBytes(DLEQBuf)
	if err != nil {
		return nil, err
	}
	// SelectedPubKeys Size
	_, err = buffer.Read(uint32Buf)
	if err != nil {
		return nil, err
	}
	SelectedPubKeysSize := binary.BigEndian.Uint32(uint32Buf)
	// SelectedPubKeys
	d.SelectedPubKeys = make([]*PointG1, SelectedPubKeysSize)
	for i := uint32(0); i < SelectedPubKeysSize; i++ {
		_, err = buffer.Read(pointG1Buf)
		if err != nil {
			return nil, err
		}
		d.SelectedPubKeys[i], err = group1.FromCompressed(pointG1Buf)
		if err != nil {
			return nil, err
		}
	}
	// B
	_, err = buffer.Read(frBuf)
	if err != nil {
		return nil, err
	}
	d.B = NewFr().FromBytes(frBuf)
	//	DBlind
	_, err = buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	d.DBlind, err = group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	//	ACommit
	_, err = buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	d.ACommit, err = group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	//	VCommit
	_, err = buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	d.VCommit, err = group1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	//	CaulkPlusProof *pb.MultiProof
	_, err = buffer.Read(uint64Buf)
	if err != nil {
		return nil, err
	}
	CaulkPlusProofSize := binary.BigEndian.Uint64(uint64Buf)
	CaulkPlusProofBuf := make([]byte, CaulkPlusProofSize)
	_, err = buffer.Read(CaulkPlusProofBuf)
	if err != nil {
		return nil, err
	}
	d.CaulkPlusProof = new(pb.MultiProof)
	err = proto.Unmarshal(CaulkPlusProofBuf, d.CaulkPlusProof)
	if err != nil {
		return nil, err
	}
	// SchnorrProof
	_, err = buffer.Read(SchnorrBuf)
	if err != nil {
		return nil, err
	}
	d.SchnorrProof, err = new(schnorr_proof.SchnorrProof).FromBytes(SchnorrBuf)
	if err != nil {
		return nil, err
	}
	// LnkProof
	d.LnkProof, err = new(kzg_ped_linkability.VectorLinkProof).FromBytes(buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return d, nil
}

func Draw(s *srs.SRS, candidatesNum uint32, candidatesPubKey []*PointG1, quota uint32, secretVector []uint32, prevRCommit *PointG1) (*DrawProof, error) {
	// blind factor
	r, _ := NewFr().Rand(rand.Reader)

	selectedPubKeys := make([]*PointG1, quota)
	var selectedPubKeyBytes []byte
	for i := uint32(0); i < quota; i++ {
		selectedPubKeys[i] = group1.Affine(group1.MulScalar(group1.New(), candidatesPubKey[secretVector[i]-1], r))
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
		if err == nil && !NewFr().Mul(b, r).IsOne() {
			break
		}
	}
	B := NewFr().Mul(b, r)

	// h
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
	lnkProof, err := kzg_ped_linkability.CreateProof(s, candidatesPubKey, h, vCommit, DBlind, vVector, b)
	if err != nil {
		return nil, err
	}

	rCommit := group1.MulScalar(group1.New(), prevRCommit, r)
	// DLEQ(prevRCommit RCommit, D' C')
	dleqInput := dleq.DLEQ{
		Index: 0,
		G1:    prevRCommit,
		H1:    rCommit,
		G2:    DBlind,
		H2:    CBlind,
	}
	dleqInput.Prove(r)
	return &DrawProof{
		RCommit: rCommit,
		// DLEQ(prevRCommit RCommit, D' C')
		DLEQProof:       dleqInput.Prove(r),
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

func Verify(s *srs.SRS, candidatesNum uint32, candidatesPubKey []*PointG1, quota uint32, prevRCommit *PointG1, drawProof *DrawProof) bool {
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

	// h
	domain := []byte("BLS12381G1_XMD:SHA-256_SSWU_RO_TESTGEN")
	h, err := group1.HashToCurve(candidatesPubKeyBytes, domain)
	if err != nil {
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

	// verify DLEQ(prevRCommit RCommit, D' C')
	if !dleq.Verify(0, prevRCommit, drawProof.RCommit, drawProof.DBlind, CBlind, drawProof.DLEQProof) {
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
