package caulk_plus

import (
	"blockchain-crypto/pb"
	. "blockchain-crypto/types/curve/bls12381"
	"blockchain-crypto/types/srs"
	"fmt"
)

type MultiProof struct {
	// common input
	// size of domain H (origin)
	N uint32
	// size of domain H (padded)
	N_padded uint32
	// size of domain V (origin)
	M uint32
	// size of domain V (padded)
	M_padded uint32
	// KZG承诺 𝒄 = [C[x]]₁
	CCommit *PointG1
	// KZG承诺 𝒂 = [A(x)]₁
	ACommit *PointG1
	// //////////////////////////////////////////
	// 公开z_I = [Z_I'(x)]₁, c_I = [C_I'(x)]₁, 𝒖 = [U'(x)]₁
	Z_I *PointG1
	C_I *PointG1
	U   *PointG1
	// 公开 w = r₁^(-1)[W₁(x)+𝒳₂W₂(x)]₂-[r₂+r₃x+r₄x²]₂, 𝒉 = [H(x)]₁
	W *PointG2
	H *PointG1
	// 输出 v₁, v₂, π₁, π₂, π_3
	// (v₁, π₁) ← KZG.Open(U'(X), 𝛼)
	V1   *Fr
	Pi_1 *PointG1
	// (v₂, π₂) ← KZG.Open(P₁(X), v₁)
	V2   *Fr
	Pi_2 *PointG1
	// (0, π₃) ← KZG.Open(P₂(X), 𝛼)
	Pi_3 *PointG1
}

func ConvertFrToProtoFr(v *Fr) *pb.Fr {
	return &pb.Fr{
		E1: v[0],
		E2: v[1],
		E3: v[2],
		E4: v[3],
	}
}

func ConvertProtoFrToFr(v *pb.Fr) *Fr {
	return &Fr{v.E1, v.E2, v.E3, v.E4}
}

func ConvertFeToProtoFq(v *Fe) *pb.Fq {
	tmp := NewFe().Set(v)
	tmp.FromMont()
	return &pb.Fq{
		E1: tmp[0],
		E2: tmp[1],
		E3: tmp[2],
		E4: tmp[3],
		E5: tmp[4],
		E6: tmp[5],
	}
}

func ConvertProtoFqToFe(v *pb.Fq) *Fe {
	return (&Fe{v.E1, v.E2, v.E3, v.E4, v.E5, v.E6}).ToMont()
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
	group1 := NewG1()
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
	group2 := NewG2()
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

func ConvertSRSToProtoSRS(s *srs.SRS) *pb.SRS {
	g1PowersLen := s.G1Degree() + 1

	g1Powers := make([]*pb.G1Affine, g1PowersLen)
	for i := uint32(0); i < g1PowersLen; i++ {
		g1Powers[i] = ConvertPointG1ToProtoG1Affine(s.G1PowerOf(i))
	}
	g2PowersLen := s.G2Degree() + 1
	g2Powers := make([]*pb.G2Affine, g2PowersLen)
	for i := uint32(0); i < g2PowersLen; i++ {
		g2Powers[i] = ConvertPointG2ToProtoG2Affine(s.G2PowerOf(i))
	}
	return &pb.SRS{
		G1Degree: s.G1Degree(),
		G2Degree: s.G2Degree(),
		G1Powers: g1Powers,
		G2Powers: g2Powers,
	}
}

func ConvertProtoMultiProofToMultiProof(p *pb.MultiProof) *MultiProof {
	return &MultiProof{
		N:        p.N,
		N_padded: p.NPadded,
		M:        p.M,
		M_padded: p.MPadded,
		CCommit:  ConvertG1AffineToPointG1(p.CCommit),
		ACommit:  ConvertG1AffineToPointG1(p.ACommit),
		Z_I:      ConvertG1AffineToPointG1(p.Z_I),
		C_I:      ConvertG1AffineToPointG1(p.C_I),
		U:        ConvertG1AffineToPointG1(p.U),
		W:        ConvertG2AffineToPointG2(p.W),
		H:        ConvertG1AffineToPointG1(p.H),
		V1:       ConvertProtoFrToFr(p.V1),
		Pi_1:     ConvertG1AffineToPointG1(p.Pi_1),
		V2:       ConvertProtoFrToFr(p.V2),
		Pi_2:     ConvertG1AffineToPointG1(p.Pi_2),
		Pi_3:     ConvertG1AffineToPointG1(p.Pi_3),
	}
}

func ConvertMultiProofToProtoMultiProof(p *MultiProof) *pb.MultiProof {
	return &pb.MultiProof{
		N:       p.N,
		NPadded: p.N_padded,
		M:       p.M,
		MPadded: p.M_padded,
		CCommit: ConvertPointG1ToProtoG1Affine(p.CCommit),
		ACommit: ConvertPointG1ToProtoG1Affine(p.ACommit),
		Z_I:     ConvertPointG1ToProtoG1Affine(p.Z_I),
		C_I:     ConvertPointG1ToProtoG1Affine(p.C_I),
		U:       ConvertPointG1ToProtoG1Affine(p.U),
		W:       ConvertPointG2ToProtoG2Affine(p.W),
		H:       ConvertPointG1ToProtoG1Affine(p.H),
		V1:      ConvertFrToProtoFr(p.V1),
		Pi_1:    ConvertPointG1ToProtoG1Affine(p.Pi_1),
		V2:      ConvertFrToProtoFr(p.V2),
		Pi_2:    ConvertPointG1ToProtoG1Affine(p.Pi_2),
		Pi_3:    ConvertPointG1ToProtoG1Affine(p.Pi_3),
	}
}

// String 实现了fmt.Stringer接口，返回MultiProof的字符串表示
func (mp MultiProof) String() string {
	return fmt.Sprintf(`MultiProof{
	N: %d,
	N_padded: %d,
	M: %d,
	M_padded: %d,
	CCommit: %v,
	ACommit: %v,
	Z_I: %v,
	C_I: %v,
	U: %v,
	W: %v,
	H: %v,
	V1: %v,
	Pi_1: %v,
	V2: %v,
	Pi_2: %v,
	Pi_3: %v,
}`,
		mp.N, mp.N_padded, mp.M, mp.M_padded,
		mp.CCommit, mp.ACommit, mp.Z_I, mp.C_I, mp.U,
		mp.W, mp.H, mp.V1, mp.Pi_1, mp.V2, mp.Pi_2, mp.Pi_3)
}
