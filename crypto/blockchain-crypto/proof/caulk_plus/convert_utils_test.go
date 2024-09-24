package caulk_plus

import (
	"blockchain-crypto/pb"
	"blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"blockchain-crypto/utils"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestConvertBetweenFrAndProtoFr(t *testing.T) {
	fr123 := bls12381.FrFromInt(123)
	fmt.Printf("fr123: %v\n", fr123)
	fmt.Printf("fr123: %X\n", fr123.ToBig())
	res := ConvertFrToProtoFr(fr123)
	fmt.Println("convert_proto_fr:", res)

	frRand := &bls12381.Fr{12995873202578071081, 16055258479306406044, 12580165684082946707, 5686935703723219923}
	fmt.Println(frRand)
	fmt.Printf("frRand: %X\n", frRand.ToBig())
	res = ConvertFrToProtoFr(frRand)
	fmt.Println(res)

	protoFr1 := &pb.Fr{
		E1: 8589934590,
		E2: 6378425256633387010,
		E3: 11064306276430008309,
		E4: 1739710354780652911,
	}
	fmt.Println(protoFr1)
	fmt.Println(ConvertProtoFrToFr(protoFr1))

	protoFrRand := &pb.Fr{
		E1: 5184521691154252970,
		E2: 3077544774153105709,
		E3: 15024891403566897681,
		E4: 5559556190772113900,
	}
	fmt.Println("protoFrRand:", protoFrRand)
	frRand = ConvertProtoFrToFr(protoFrRand)
	fmt.Println("frRand:", frRand)
	fmt.Printf("%x\n", frRand.ToBig())
	fmt.Println("resFr:", ConvertFrToProtoFr(frRand))

}

func TestConvertBetweenFeAndProtoFq(t *testing.T) {
	group1 := bls12381.NewG1()
	g123 := group1.Affine(group1.MulScalar(group1.New(), group1.One(), bls12381.FrFromInt(123)))
	fmt.Println("fq:", g123[0])
	fmt.Printf("fq: %X\n", g123[0].ToBig())
	protoFq := ConvertFeToProtoFq(&g123[0])
	fmt.Println(protoFq)
	fmt.Println()

	protoFq = &pb.Fq{
		E1: 9643578314753161704,
		E2: 12363969365937116593,
		E3: 17370378380101614273,
		E4: 10525188256555326244,
		E5: 625377410555126400,
		E6: 66496752359416402,
	}
	fmt.Println("protoFq", protoFq)
	// expect
	fe := ConvertProtoFqToFe(protoFq)
	fmt.Println("fe", fe)
}

func TestBytes(t *testing.T) {
	data := []byte{232, 185, 133, 38, 98, 223, 212, 133}
	u64 := binary.LittleEndian.Uint64(data)
	fmt.Println(u64)
}

func TestConvertBetweenFe2AndProtoFq2(t *testing.T) {
	group2 := bls12381.NewG2()
	g123 := group2.Affine(group2.MulScalar(group2.New(), group2.One(), bls12381.FrFromInt(123)))
	fmt.Println("fq2:", g123[0])
	fmt.Printf("fq2: %X, %X\n", g123[0][0].ToBig(), g123[0][1].ToBig())
	protoFq := ConvertFe2ToProtoFq2(&g123[0])
	fmt.Println("protoFq:", protoFq)
	fmt.Println()

	protoFq2 := &pb.Fq2{
		C1: &pb.Fq{
			E1: 1159306823142176013,
			E2: 10447810949987217124,
			E3: 3083478465137856249,
			E4: 5899801630485792087,
			E5: 1445135372160517734,
			E6: 678458133818818382,
		},
		C2: &pb.Fq{
			E1: 5594160341619071078,
			E2: 8955073851498511001,
			E3: 8812007552358678521,
			E4: 17602816475186738576,
			E5: 16890608467270166650,
			E6: 1576694991520513337,
		},
	}
	fmt.Println(protoFq2)
	fe2 := ConvertProtoFq2ToFe2(protoFq2)
	fmt.Println(fe2)
}

func TestConvertBetweenG1AffineAndProtoG1Affine(t *testing.T) {
	group1 := bls12381.NewG1()
	v := group1.MulScalar(group1.New(), group1.One(), bls12381.FrFromInt(123))
	v = group1.Affine(v)
	fmt.Println(v)
	res := ConvertPointG1ToProtoG1Affine(v)
	fmt.Println(res)
	fmt.Println()

	// x, _ := bls12381.NewFe().Rand(rand.Reader)
	// y, _ := bls12381.NewFe().Rand(rand.Reader)
	x := pb.Fq{
		E1: 3869416301305688404,
		E2: 18227701205550856488,
		E3: 13120644821762217922,
		E4: 1434637430955737077,
		E5: 2768977480434347349,
		E6: 903678310187975643,
	}
	y := pb.Fq{
		E1: 17657551983982243557,
		E2: 12173308879208040333,
		E3: 12839781611262482079,
		E4: 9962227492312131996,
		E5: 15906590867200588005,
		E6: 319252717365518799,
	}

	protoG1 := &pb.G1Affine{
		X:        &x,
		Y:        &y,
		Infinity: false,
	}
	fmt.Println(protoG1)
	g1 := ConvertG1AffineToPointG1(protoG1)
	fmt.Println(g1)
	fmt.Println(group1.IsAffine(g1))
	fmt.Println(group1.IsOnCurve(g1))
	fmt.Println(group1.InCorrectSubgroup(g1))
}

func TestConvertBetweenG2AffineAndProtoG2Affine(t *testing.T) {
	group2 := bls12381.NewG2()
	v := group2.MulScalar(group2.New(), group2.One(), bls12381.FrFromInt(1234))
	v = group2.Affine(v)
	fmt.Println(v)
	res := ConvertPointG2ToProtoG2Affine(v)
	fmt.Println(res)
	fmt.Println()

	x, _ := bls12381.NewFe2().Rand(rand.Reader)
	y, _ := bls12381.NewFe2().Rand(rand.Reader)

	protoG1 := &pb.G2Affine{
		X:        ConvertFe2ToProtoFq2(x),
		Y:        ConvertFe2ToProtoFq2(y),
		Infinity: false,
	}
	fmt.Println(protoG1)
	g2 := ConvertG2AffineToPointG2(protoG1)
	fmt.Println(g2)
}

func TestFe(t *testing.T) {
	fe1 := bls12381.NewFe().One()

	fmt.Println("fe1:", fe1)
	fmt.Println("fe1:", fe1.Bytes())

	protoFq1 := ConvertFeToProtoFq(fe1)
	fmt.Println("protoFq1:", protoFq1)

}

func TestConvertProtoMultiProofBetweenMultiProof(t *testing.T) {
	group1 := bls12381.NewG1()
	candidatesNum := uint32(256)
	quota := uint32(32)

	g1Degree, g2Degree := utils.CalcMinLimitedDegree(candidatesNum, quota)
	s, _ := srs.NewSRS(g1Degree, g2Degree)

	generator := s.G1PowerOf(0)

	secretKeys := make([]*bls12381.Fr, candidatesNum)
	candidatesPubKey := make([]*bls12381.PointG1, candidatesNum)
	for i := uint32(0); i < candidatesNum; i++ {
		secretKeys[i], _ = bls12381.NewFr().Rand(rand.Reader)
		// secretKeys[i] = FromUInt32(i + 1)
		candidatesPubKey[i] = group1.Affine(group1.MulScalar(group1.New(), generator, secretKeys[i]))
	}
	// secretVector := indexList[:quota]
	secretVector := []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	fmt.Println("start create draw proof")

	// blind factor
	r, _ := bls12381.NewFr().Rand(rand.Reader)
	// r := FrFromInt(123)
	// fmt.Println("r:", r)

	selectedPubKeys := make([]*bls12381.PointG1, quota)
	var selectedPubKeyBytes []byte
	for i := uint32(0); i < quota; i++ {
		selectedPubKeys[i] = group1.Affine(group1.MulScalar(group1.New(), candidatesPubKey[secretVector[i]-1], r))
		// fmt.Printf(" pk %d:%v\n", secretVector[i], candidatesPubKey[secretVector[i]-1])
		// fmt.Printf("spk %d:%v\n\n", i+1, selectedPubKeys[i])
		selectedPubKeyBytes = append(selectedPubKeyBytes, group1.ToBytes(selectedPubKeys[i])...)
	}
	// TODO
	var candidatesPubKeyBytes []byte
	for i := uint32(0); i < candidatesNum; i++ {
		candidatesPubKeyBytes = append(candidatesPubKeyBytes, group1.ToBytes(candidatesPubKey[i])...)
	}

	preImageBytes := append(candidatesPubKeyBytes, selectedPubKeyBytes...)

	// construct vector a
	aVector := make([]*bls12381.Fr, quota)
	for i := uint32(0); i < quota; i++ {
		aVector[i] = bls12381.HashToFr(append(preImageBytes, utils.Uint32ToBytes(i)...))
	}
	// calc kzg commitment and poly
	// aCommit, aPoly, err := kzg_vector_commit.Commit(srs, aVector)
	// if err != nil {
	// 	return nil, err
	// }
	// construct vector v
	vVector := make([]*bls12381.Fr, candidatesNum)
	for i := uint32(0); i < candidatesNum; i++ {
		vVector[i] = bls12381.NewFr().Zero()
	}
	for i := uint32(0); i < quota; i++ {
		// θ_s(i) = a_i
		vVector[secretVector[i]-1] = bls12381.NewFr().Set(aVector[i])
	}
	// construct caulk plus proof of vector a and vector v
	protoCaulkPlusProof, err := CreateMultiProof(candidatesNum, vVector, quota, aVector)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(protoCaulkPlusProof)
	fmt.Println()

	caulkPlusProof := ConvertProtoMultiProofToMultiProof(protoCaulkPlusProof)
	fmt.Println(caulkPlusProof)
	fmt.Println()

	convertedProtoMultiProof := ConvertMultiProofToProtoMultiProof(caulkPlusProof)
	fmt.Println(convertedProtoMultiProof)
	fmt.Println()

	res, err := VerifyMultiProof(convertedProtoMultiProof)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Verify res", res)
}
