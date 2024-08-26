package caulk_plus

import (
	"crypto/rand"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/crypto/pb"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"testing"
)

func TestConvertBetweenFrAndProtoFr(t *testing.T) {
	v := bls12381.FrFromInt(1)
	fmt.Println(v)
	res := ConvertFrToProtoFr(v)
	// [8589934590, 6378425256633387010, 11064306276430008309, 1739710354780652911]
	fmt.Println(res)

	protoFr := &pb.Fr{
		E1: 8589934590,
		E2: 6378425256633387010,
		E3: 11064306276430008309,
		E4: 1739710354780652911,
	}
	fmt.Println(protoFr)
	fr := ConvertProtoFrToFr(protoFr)
	fmt.Println(fr)
}

func TestConvertBetweenFeAndProtoFq(t *testing.T) {
	v := &bls12381.Fe{123124, 123125, 123126, 123127, 123128, 123129}
	fmt.Println(v)
	res := ConvertFeToProtoFq(v)
	fmt.Println(res)

	protoFq := &pb.Fq{
		E1: 12321,
		E2: 214134,
		E3: 1413431,
		E4: 23421343117,
		E5: 3124543,
		E6: 1234535453534,
	}
	fmt.Println(protoFq)
	fe := ConvertProtoFqToFe(protoFq)
	fmt.Println(fe)

}

func TestConvertBetweenFe2AndProtoFq2(t *testing.T) {
	v, _ := bls12381.NewFe2().Rand(rand.Reader)
	fmt.Println(v)
	res := ConvertFe2ToProtoFq2(v)
	fmt.Println(res)
	protoFq2 := &pb.Fq2{
		C1: &pb.Fq{
			E1: 346457,
			E2: 23546456,
			E3: 2344254,
			E4: 1,
			E5: 34,
			E6: 2,
		},
		C2: &pb.Fq{
			E1: 3,
			E2: 235555,
			E3: 33333333333,
			E4: 1,
			E5: 34,
			E6: 2,
		},
	}
	fmt.Println(protoFq2)
	fe2 := ConvertProtoFq2ToFe2(protoFq2)
	fmt.Println(fe2)
}

func TestConvertBetweenG1AffineAndProtoG1Affine(t *testing.T) {
	v := group1.MulScalar(group1.New(), group1.One(), bls12381.FrFromInt(1234))
	v = group1.Affine(v)
	fmt.Println(v)
	res := ConvertPointG1ToProtoG1Affine(v)
	fmt.Println(res)
	fmt.Println()

	x, _ := bls12381.NewFe().Rand(rand.Reader)
	y, _ := bls12381.NewFe().Rand(rand.Reader)

	protoG1 := &pb.G1Affine{
		X:        ConvertFeToProtoFq(x),
		Y:        ConvertFeToProtoFq(y),
		Infinity: false,
	}
	fmt.Println(protoG1)
	g1 := ConvertG1AffineToPointG1(protoG1)
	fmt.Println(g1)
}

func TestConvertBetweenG2AffineAndProtoG2Affine(t *testing.T) {
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
