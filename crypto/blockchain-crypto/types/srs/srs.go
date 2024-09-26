package srs

import (
	"blockchain-crypto/proof/schnorr_proof/bls12381"
	. "blockchain-crypto/types/curve/bls12381"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const BinFileName = "srs.binary"

type SRS struct {
	g1Degree uint32
	g2Degree uint32
	// g1, g1^tau, g1^tau^2, ... g1^tau^d
	// [1]₁,[τ]₁,...,[τᵈ]₁
	g1Powers []*PointG1
	// g2, g2^tau, g2^tau^2, ... g2^tau^d
	// [1]₂,[τ]₂,...,[τᵈ]₂
	g2Powers []*PointG2
}

func TrivialSRS(g1Degree, g2Degree uint32) *SRS {
	group1 := NewG1()
	group2 := NewG2()
	// get generators
	g1PowersSize := g1Degree + 1
	g2PowersSize := g2Degree + 1
	g1Powers := make([]*PointG1, g1PowersSize)
	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		g1Powers[i] = group1.New().Set(group1.One())
	}
	for i := uint32(0); i < g2PowersSize; i++ {
		g2Powers[i] = group2.New().Set(group2.One())
	}
	return &SRS{
		g1Degree: g1Degree,
		g2Degree: g2Degree,
		g1Powers: g1Powers,
		g2Powers: g2Powers,
	}
}

func NewSRS(g1Degree, g2Degree uint32) (*SRS, *schnorr_proof.SchnorrProof) {
	x, _ := NewFr().Rand(rand.Reader)
	srs := TrivialSRS(g1Degree, g2Degree)
	return srs.Update(x)
}

func (s *SRS) Set(s1 *SRS) *SRS {
	group1 := NewG1()
	group2 := NewG2()
	s.g1Degree = s1.g1Degree
	s.g2Degree = s1.g2Degree
	g1PowersSize := s.g1Degree + 1
	g2PowersSize := s.g2Degree + 1
	s.g1Powers = make([]*PointG1, g1PowersSize)
	s.g2Powers = make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		s.g1Powers[i] = group1.New().Set(s1.g1Powers[i])
	}
	for i := uint32(0); i < g2PowersSize; i++ {
		s.g2Powers[i] = group2.New().Set(s1.g2Powers[i])
	}
	return s
}

func (s *SRS) Update(r *Fr) (*SRS, *schnorr_proof.SchnorrProof) {
	group1 := NewG1()
	group2 := NewG2()
	rPower := NewFr().Set(r)
	g1PowersSize := len(s.g1Powers)
	g2PowersSize := len(s.g2Powers)
	newSRS := &SRS{
		g1Degree: s.g1Degree,
		g2Degree: s.g2Degree,
		g1Powers: make([]*PointG1, g1PowersSize),
		g2Powers: make([]*PointG2, g2PowersSize),
	}
	newSRS.g1Powers[0] = s.g1Powers[0]
	newSRS.g2Powers[0] = s.g2Powers[0]
	if g1PowersSize > g2PowersSize {
		for i := 1; i < g2PowersSize; i++ {
			newSRS.g1Powers[i] = group1.MulScalar(group1.New(), s.g1Powers[i], rPower)
			newSRS.g2Powers[i] = group2.MulScalar(group2.New(), s.g2Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
		for i := g2PowersSize; i < g1PowersSize; i++ {
			newSRS.g1Powers[i] = group1.MulScalar(group1.New(), s.g1Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
	} else {
		for i := 1; i < g1PowersSize; i++ {
			newSRS.g1Powers[i] = group1.MulScalar(group1.New(), s.g1Powers[i], rPower)
			newSRS.g2Powers[i] = group2.MulScalar(group2.New(), s.g2Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
		for i := g1PowersSize; i < g2PowersSize; i++ {
			newSRS.g2Powers[i] = group2.MulScalar(group2.New(), s.g2Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 1: updated setup builds on the work of the preceding participants
	// fmt.Println(newSRS)
	return newSRS, schnorr_proof.CreateWitness(s.g1Powers[1], newSRS.g1Powers[1], r)
}

func Verify(s *SRS, prevSRSG1FirstElem *PointG1, proof *schnorr_proof.SchnorrProof) error {
	group1 := NewG1()
	group2 := NewG2()
	if s == nil {
		return fmt.Errorf("srs is nil")
	}
	if proof == nil {
		return fmt.Errorf("proof is nil")
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 3: updated setup is non-degenerative
	if group1.IsZero(s.g1Powers[1]) || group1.IsZero(s.g1Powers[1]) {
		return fmt.Errorf("pdated setup is degenerative")
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 1: updated setup builds on the work of the preceding participants
	if !schnorr_proof.Verify(prevSRSG1FirstElem, s.g1Powers[1], proof) {
		return fmt.Errorf("updated setup is not build on the work of the preceding participants")
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 2: updated setup is well-formed
	// challenge
	rho1, _ := NewFr().Rand(rand.Reader)
	rho2, _ := NewFr().Rand(rand.Reader)
	// compute common term of g1
	g1CommonTerm := group1.Zero()
	rho1Power := NewFr().One()
	for i := uint32(1); i < s.g1Degree; i++ {
		group1.Add(g1CommonTerm, g1CommonTerm, group1.MulScalar(group1.New(), s.g1Powers[i], rho1Power))
		rho1Power.Mul(rho1Power, rho1)
	}
	// compute common term of g2
	g2CommonTerm := group2.Zero()
	rho2Power := NewFr().One()
	for i := uint32(1); i < s.g2Degree; i++ {
		group2.Add(g2CommonTerm, g2CommonTerm, group2.MulScalar(group2.New(), s.g2Powers[i], rho2Power))
		rho2Power.Mul(rho2Power, rho2)
	}
	leftG1Elem := group1.Add(group1.New(), g1CommonTerm, group1.MulScalar(group1.New(), s.g1Powers[s.g1Degree], rho1Power))
	rightG1Elem := group1.Add(group1.New(), s.g1Powers[0], group1.MulScalar(group1.New(), g1CommonTerm, rho1))
	leftG2Elem := group2.Add(group2.New(), s.g2Powers[0], group2.MulScalar(group2.New(), g2CommonTerm, rho2))
	rightG2Elem := group2.Add(group2.New(), g2CommonTerm, group2.MulScalar(group2.New(), s.g2Powers[s.g2Degree], rho2Power))

	if !NewPairingEngine().AddPair(leftG1Elem, leftG2Elem).AddPairInv(rightG1Elem, rightG2Elem).Check() {
		return fmt.Errorf("pairing failed, updated setup is not well-formed")
	}
	return nil
}

// Trim srs_proof to desired degree (inclusive)
func (s *SRS) Trim(g1Degree, g2Degree uint32) {
	if g1Degree >= s.g1Degree || g2Degree >= s.g2Degree {
		return
	}
	s.g1Degree = g1Degree
	s.g2Degree = g2Degree
	s.g1Powers = s.g1Powers[:g1Degree+1]
	s.g2Powers = s.g2Powers[:g2Degree+1]
}

func (s *SRS) ToBinaryFile() error {
	compressedBytes, err := s.ToCompressedBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(BinFileName, compressedBytes, 0644)
}

func FromBinaryFile() (*SRS, error) {
	group1 := NewG1()
	group2 := NewG2()
	f, err := os.Open(BinFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// for degree
	degreeBuf := make([]byte, 4)
	// for g1 power
	g1PowerBuf := make([]byte, 48)
	// for g2 power
	g2PowerBuf := make([]byte, 96)

	_, err = f.Read(degreeBuf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	g1Degree := binary.BigEndian.Uint32(degreeBuf)
	_, err = f.Read(degreeBuf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	g2Degree := binary.BigEndian.Uint32(degreeBuf)
	g1PowersSize := g1Degree + 1
	g2PowersSize := g2Degree + 1
	g1Powers := make([]*PointG1, g1PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		if _, err = f.Read(g1PowerBuf); err != nil && err != io.EOF {
			return nil, err
		}
		g1Powers[i], err = group1.FromCompressed(g1PowerBuf)
		if err != nil {
			return nil, err
		}
	}
	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g2PowersSize; i++ {
		if _, err = f.Read(g2PowerBuf); err != nil && err != io.EOF {
			return nil, err
		}
		g2Powers[i], err = group2.FromCompressed(g2PowerBuf)
		if err != nil {
			return nil, err
		}
	}
	return &SRS{
		g1Powers: g1Powers,
		g2Powers: g2Powers,
		g1Degree: g1Degree,
		g2Degree: g2Degree,
	}, nil
}

func FromCompressedBytes(input []byte) (*SRS, error) {
	group1 := NewG1()
	group2 := NewG2()
	buffer := bytes.NewBuffer(input)
	uint32Buf := make([]byte, 4)
	pointG1Buf := make([]byte, 48)
	pointG2Buf := make([]byte, 96)
	// degree
	_, err := buffer.Read(uint32Buf)
	if err != nil {
		return nil, err
	}
	g1Degree := binary.BigEndian.Uint32(uint32Buf)
	_, err = buffer.Read(uint32Buf)
	if err != nil {
		return nil, err
	}
	g2Degree := binary.BigEndian.Uint32(uint32Buf)
	g1PowersSize := g1Degree + 1
	g2PowersSize := g2Degree + 1
	// for g1 power
	g1Powers := make([]*PointG1, g1PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		_, err = buffer.Read(pointG1Buf)
		if err != nil {
			return nil, err
		}
		g1Powers[i], err = group1.FromCompressed(pointG1Buf)
		if err != nil {
			fmt.Printf("FCB: G1 at %v : %v, uncompressed: %v, error: %v\n",
				i,
				pointG1Buf,
				g1Powers[i],
				err,
			)
			return nil, fmt.Errorf("failed to deserialize pointG1 at %v: %v", i, err)

		}
	}
	// for g2 power
	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g2PowersSize; i++ {
		_, err = buffer.Read(pointG2Buf)
		if err != nil {
			return nil, err
		}
		g2Powers[i], err = group2.FromCompressed(pointG2Buf)
		if err != nil {
			fmt.Printf("FCB: G2 at %v : %v, uncompressed: %v, error: %v\n",
				i,
				pointG2Buf,
				g2Powers[i],
				err,
			)
			return nil, fmt.Errorf("failed to deserialize pointG2 at %v: %v", i, err)
		}
	}
	return &SRS{
		g1Powers: g1Powers,
		g2Powers: g2Powers,
		g1Degree: g1Degree,
		g2Degree: g2Degree,
	}, nil
}

func (s *SRS) ToCompressedBytes() ([]byte, error) {
	group1 := NewG1()
	group2 := NewG2()
	buf := bytes.NewBuffer([]byte{})
	if err := binary.Write(buf, binary.BigEndian, s.g1Degree); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, s.g2Degree); err != nil {
		return nil, err
	}
	g1PowersSize := s.g1Degree + 1
	g2PowersSize := s.g2Degree + 1
	for i := uint32(0); i < g1PowersSize; i++ {
		pointBytes := group1.ToCompressed(s.g1Powers[i])
		// recoveredPoint, err := group1.FromCompressed(pointBytes)
		// fmt.Printf("TCB: G1 at %v: %v, curve?= %v, group?= %v, compressed: %v, curve?= %v, group?= %v, error: %v, eq?=%v\n",
		// 	i,
		// 	s.g1Powers[i],
		// 	group1.IsOnCurve(s.g1Powers[i]),
		// 	group1.InCorrectSubgroup(s.g1Powers[i]),
		// 	pointBytes,
		// 	group1.IsOnCurve(recoveredPoint),
		// 	group1.InCorrectSubgroup(recoveredPoint),
		// 	err,
		// 	group1.Equal(recoveredPoint, s.g1Powers[i]),
		// )
		buf.Write(pointBytes)
	}
	for i := uint32(0); i < g2PowersSize; i++ {
		pointBytes := group2.ToCompressed(s.g2Powers[i])
		// recoveredPoint, err := group2.FromCompressed(pointBytes)
		// fmt.Printf("TCB: G2 at %v : %v, curve?= %v, group?= %v, compressed: %v curve?= %v, group?= %v, error: %v, eq?=%v\n",
		// 	i,
		// 	s.g2Powers[i],
		// 	group2.IsOnCurve(s.g2Powers[i]),
		// 	group2.InCorrectSubgroup(s.g2Powers[i]),
		// 	pointBytes,
		// 	group2.IsOnCurve(recoveredPoint),
		// 	group2.InCorrectSubgroup(recoveredPoint),
		// 	err,
		// 	group2.Equal(recoveredPoint, s.g2Powers[i]))
		buf.Write(pointBytes)
	}
	return buf.Bytes(), nil
}

func FromBinaryFileWithDegree(neededG1Degree uint32, neededG2Degree uint32) (*SRS, error) {
	group1 := NewG1()
	group2 := NewG2()
	f, err := os.Open(BinFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// for degree
	degreeBuf := make([]byte, 4)
	// for g1 power
	g1PowerBuf := make([]byte, 48)
	// for g2 power
	g2PowerBuf := make([]byte, 96)

	_, err = f.Read(degreeBuf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	g1Degree := binary.BigEndian.Uint32(degreeBuf)
	if neededG1Degree > g1Degree {
		return nil, errors.New("degree of g1 is low")
	}

	_, err = f.Read(degreeBuf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	g2Degree := binary.BigEndian.Uint32(degreeBuf)
	if neededG2Degree > g2Degree {
		return nil, errors.New("degree of g2 is low")
	}

	g1PowersSize := neededG1Degree + 1
	g2PowersSize := neededG2Degree + 1

	g1Powers := make([]*PointG1, g1PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		if _, err = f.Read(g1PowerBuf); err != nil && err != io.EOF {
			return nil, err
		}
		g1Powers[i], err = group1.FromCompressed(g1PowerBuf)
		if err != nil {
			fmt.Println("g1", i)
			return nil, err
		}
	}
	for i := neededG1Degree; i < g1Degree; i++ {
		if _, err = f.Read(g1PowerBuf); err != nil && err != io.EOF {
			return nil, err
		}
	}

	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g2PowersSize; i++ {
		if _, err = f.Read(g2PowerBuf); err != nil && err != io.EOF {
			return nil, err
		}
		g2Powers[i], err = group2.FromCompressed(g2PowerBuf)
		if err != nil {
			fmt.Println("g2", i)
			return nil, err
		}
	}
	for i := neededG2Degree; i < g2Degree; i++ {
		if _, err = f.Read(g2PowerBuf); err != nil && err != io.EOF {
			return nil, err
		}
	}
	return &SRS{
		g1Powers: g1Powers,
		g2Powers: g2Powers,
		g1Degree: g1Degree,
		g2Degree: g2Degree,
	}, nil
}

func (s *SRS) Degrees() (uint32, uint32) {
	return s.g1Degree, s.g2Degree
}

func (s *SRS) G1Degree() uint32 {
	return s.g1Degree
}

func (s *SRS) G2Degree() uint32 {
	return s.g2Degree
}

func (s *SRS) G1Powers() []*PointG1 {
	return s.g1Powers
}

func (s *SRS) G1PowerOf(i uint32) *PointG1 {
	if i > s.g1Degree {
		return nil
	}
	return s.g1Powers[i]
}

func (s *SRS) G2PowerOf(i uint32) *PointG2 {
	if i > s.g2Degree {
		return nil
	}
	return s.g2Powers[i]
}

func (s *SRS) G2Powers() []*PointG2 {
	return s.g2Powers
}

func (s *SRS) G1Generator() *PointG1 {
	group1 := NewG1()
	return group1.New().Set(s.g1Powers[0])
}

func (s *SRS) G2Generator() *PointG2 {
	group2 := NewG2()
	return group2.New().Set(s.g2Powers[0])
}
