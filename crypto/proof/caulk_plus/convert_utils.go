package caulk_plus

import (
	"github.com/zzz136454872/upgradeable-consensus/crypto/pb"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

var (
	group1 = NewG1()
	group2 = NewG2()
)

func ConvertFrToProtoFr(v *Fr) *pb.Fr {
	tmp := NewFr().Set(v).ToRed()
	return &pb.Fr{
		E1: tmp[0],
		E2: tmp[1],
		E3: tmp[2],
		E4: tmp[3],
	}
}

func ConvertProtoFrToFr(v *pb.Fr) *Fr {
	return (&Fr{v.E1, v.E2, v.E3, v.E4}).FromRed()
}

func ConvertFeToProtoFq(v *Fe) *pb.Fq {
	return &pb.Fq{
		E1: v[0],
		E2: v[1],
		E3: v[2],
		E4: v[3],
		E5: v[4],
		E6: v[5],
	}
}

func ConvertProtoFqToFe(v *pb.Fq) *Fe {
	return &Fe{v.E1, v.E2, v.E3, v.E4, v.E5, v.E6}
}

func ConvertFe2ToProtoFq2(v *Fe2) *pb.Fq2 {
	return &pb.Fq2{
		C1: ConvertFeToProtoFq(&v[0]),
		C2: ConvertFeToProtoFq(&v[1]),
	}
}

func ConvertProtoFq2ToFe2(v *pb.Fq2) *Fe2 {
	return &Fe2{
		*ConvertProtoFqToFe(v.C1),
		*ConvertProtoFqToFe(v.C2),
	}
}

func ConvertPointG1ToProtoG1Affine(p *PointG1) *pb.G1Affine {
	tmp := group1.New().Set(p)
	if !tmp.IsAffine() {
		tmp = group1.Affine(tmp)
	}
	return &pb.G1Affine{
		X:        ConvertFeToProtoFq(&tmp[0]),
		Y:        ConvertFeToProtoFq(&tmp[1]),
		Infinity: group1.IsZero(tmp),
	}
}

func ConvertG1AffineToPointG1(p *pb.G1Affine) *PointG1 {
	return &PointG1{*ConvertProtoFqToFe(p.X), *ConvertProtoFqToFe(p.Y), *NewFe().One()}
}

func ConvertPointG2ToProtoG2Affine(p *PointG2) *pb.G2Affine {
	tmp := group2.New().Set(p)
	if !tmp.IsAffine() {
		tmp = group2.Affine(tmp)
	}
	return &pb.G2Affine{
		X:        ConvertFe2ToProtoFq2(&tmp[0]),
		Y:        ConvertFe2ToProtoFq2(&tmp[1]),
		Infinity: group2.IsZero(tmp),
	}
}

func ConvertG2AffineToPointG2(p *pb.G2Affine) *PointG2 {
	return &PointG2{*ConvertProtoFq2ToFe2(p.X), *ConvertProtoFq2ToFe2(p.Y), *NewFe2().One()}
}
