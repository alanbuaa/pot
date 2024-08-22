package srs

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

var (
	group1 = NewG1()
	g1     = group1.One()
	group2 = NewG2()
	g2     = group2.One()
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

func NewSRS(g1Degree, g2Degree uint32) (*SRS, *schnorr_proof.SchnorrProof) {
	// get generators
	g1PowersSize := g1Degree + 1
	g2PowersSize := g2Degree + 1
	g1Powers := make([]*PointG1, g1PowersSize)
	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		g1Powers[i] = group1.New().Set(g1)
	}
	for i := uint32(0); i < g2PowersSize; i++ {
		g2Powers[i] = group2.New().Set(g2)
	}
	x, _ := NewFr().Rand(rand.Reader)
	srs := &SRS{
		g1Degree: g1Degree,
		g2Degree: g2Degree,
		g1Powers: g1Powers,
		g2Powers: g2Powers,
	}
	return srs.Update(x)
}

func (s *SRS) Update(r *Fr) (*SRS, *schnorr_proof.SchnorrProof) {
	prevSRSG1FirstElem := s.g1Powers[1]
	rPower := NewFr().Set(r)
	g1PowersSize := len(s.g1Powers)
	g2PowersSize := len(s.g2Powers)
	if g1PowersSize > g2PowersSize {
		for i := 1; i < g2PowersSize; i++ {
			s.g1Powers[i] = group1.MulScalar(group1.New(), s.g1Powers[i], rPower)
			s.g2Powers[i] = group2.MulScalar(group2.New(), s.g2Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
		for i := g2PowersSize; i < g1PowersSize; i++ {
			s.g1Powers[i] = group1.MulScalar(group1.New(), s.g1Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
	} else {
		for i := 1; i < g1PowersSize; i++ {
			s.g1Powers[i] = group1.MulScalar(group1.New(), s.g1Powers[i], rPower)
			s.g2Powers[i] = group2.MulScalar(group2.New(), s.g2Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
		for i := g1PowersSize; i < g2PowersSize; i++ {
			s.g2Powers[i] = group2.MulScalar(group2.New(), s.g2Powers[i], rPower)
			rPower.Mul(rPower, r)
		}
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 1: updated setup builds on the work of the preceding participants
	return s, schnorr_proof.CreateWitness(prevSRSG1FirstElem, s.g1Powers[1], r)
}

func Verify(s *SRS, prevSRSG1FirstElem *PointG1, proof *schnorr_proof.SchnorrProof) bool {
	if s == nil || proof == nil {
		return false
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 3: updated setup is non-degenerative
	if group1.IsZero(s.g1Powers[0]) || group1.IsZero(s.g1Powers[1]) {
		return false
	}
	// /////////////////////////////////////////////////////////////////////////////////////////////////
	// check 1: updated setup builds on the work of the preceding participants
	if !schnorr_proof.Verify(prevSRSG1FirstElem, s.g1Powers[1], proof) {
		return false
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

	return NewPairingEngine().AddPair(leftG1Elem, leftG2Elem).AddPairInv(rightG1Elem, rightG2Elem).Check()
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
	buf := bytes.NewBuffer([]byte{})
	if err := binary.Write(buf, binary.BigEndian, s.g1Degree); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.BigEndian, s.g2Degree); err != nil {
		return err
	}
	g1PowersSize := s.g1Degree + 1
	g2PowersSize := s.g2Degree + 1
	for i := uint32(0); i < g1PowersSize; i++ {
		if err := binary.Write(buf, binary.BigEndian, group1.ToCompressed(s.g1Powers[i])); err != nil {
			return err
		}
	}
	for i := uint32(0); i < g2PowersSize; i++ {
		if err := binary.Write(buf, binary.BigEndian, group2.ToCompressed(s.g2Powers[i])); err != nil {
			return err
		}
	}
	return os.WriteFile(BinFileName, buf.Bytes(), 0644)
}

func (s *SRS) FromBinaryFile() error {
	f, err := os.Open(BinFileName)
	if err != nil {
		return err
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
		return err
	}
	g1Degree := binary.BigEndian.Uint32(degreeBuf)
	_, err = f.Read(degreeBuf)
	if err != nil && err != io.EOF {
		return err
	}
	g2Degree := binary.BigEndian.Uint32(degreeBuf)
	g1PowersSize := g1Degree + 1
	g2PowersSize := g2Degree + 1
	g1Powers := make([]*PointG1, g1PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		if _, err = f.Read(g1PowerBuf); err != nil && err != io.EOF {
			return err
		}
		g1Powers[i], err = group1.FromCompressed(g1PowerBuf)
		if err != nil {
			return err
		}
	}
	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g2PowersSize; i++ {
		if _, err = f.Read(g2PowerBuf); err != nil && err != io.EOF {
			return err
		}
		g2Powers[i], err = group2.FromCompressed(g2PowerBuf)
		if err != nil {
			return err
		}
	}
	s.g1Degree = g1Degree
	s.g2Degree = g2Degree
	s.g1Powers = g1Powers
	s.g2Powers = g2Powers
	return nil
}

func (s *SRS) FromBinaryFileWithDegree(neededG1Degree uint32, neededG2Degree uint32) error {
	f, err := os.Open(BinFileName)
	if err != nil {
		return err
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
		return err
	}
	g1Degree := binary.BigEndian.Uint32(degreeBuf)
	if neededG1Degree > g1Degree {
		return errors.New("degree of g1 is low")
	}

	_, err = f.Read(degreeBuf)
	if err != nil && err != io.EOF {
		return err
	}
	g2Degree := binary.BigEndian.Uint32(degreeBuf)
	if neededG2Degree > g2Degree {
		return errors.New("degree of g2 is low")
	}

	g1PowersSize := neededG1Degree + 1
	g2PowersSize := neededG2Degree + 1

	g1Powers := make([]*PointG1, g1PowersSize)
	for i := uint32(0); i < g1PowersSize; i++ {
		if _, err = f.Read(g1PowerBuf); err != nil && err != io.EOF {
			return err
		}
		g1Powers[i], err = group1.FromCompressed(g1PowerBuf)
		if err != nil {
			fmt.Println("g1", i)
			return err
		}
	}
	for i := neededG1Degree; i < g1Degree; i++ {
		if _, err = f.Read(g1PowerBuf); err != nil && err != io.EOF {
			return err
		}
	}

	g2Powers := make([]*PointG2, g2PowersSize)
	for i := uint32(0); i < g2PowersSize; i++ {
		if _, err = f.Read(g2PowerBuf); err != nil && err != io.EOF {
			return err
		}
		g2Powers[i], err = group2.FromCompressed(g2PowerBuf)
		if err != nil {
			fmt.Println("g2", i)
			return err
		}
	}
	for i := neededG2Degree; i < g2Degree; i++ {
		if _, err = f.Read(g2PowerBuf); err != nil && err != io.EOF {
			return err
		}
	}
	s.g1Degree = neededG1Degree
	s.g2Degree = neededG2Degree
	s.g1Powers = g1Powers
	s.g2Powers = g2Powers
	return nil
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

func (s *SRS) G1Power(i uint32) *PointG1 {
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
	return group1.New().Set(s.g1Powers[0])
}

func (s *SRS) G2Generator() *PointG2 {
	return group2.New().Set(s.g2Powers[0])
}
