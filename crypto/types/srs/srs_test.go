package srs

import (
	"crypto/rand"
	"fmt"
	"github.com/stretchr/testify/assert"
	schnorr_proof "github.com/zzz136454872/upgradeable-consensus/crypto/proof/schnorr_proof/bls12381"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"math/big"
	"testing"
	"time"
)

func TestVerifyTrue1(t *testing.T) {
	s, initProof := NewSRS(128, 16)
	assert.True(t, Verify(s, g1, initProof))
	prevSRSG1FirstElem := s.G1PowerOf(1)
	x, _ := NewFr().Rand(rand.Reader)
	newSrs, proof := s.Update(x)
	assert.True(t, Verify(newSrs, prevSRSG1FirstElem, proof))
}

func TestVerifyTrue2(t *testing.T) {
	N := 10
	startTime := time.Now()
	s, initProof := NewSRS(2<<12, 2<<12)
	assert.True(t, Verify(s, g1, initProof))
	fmt.Println("init time:", time.Since(startTime))
	for i := 0; i < N; i++ {
		x, _ := NewFr().Rand(rand.Reader)

		startTime = time.Now()
		prevSRSG1FirstElem := s.G1PowerOf(1)
		newSrs, proof := s.Update(x)
		fmt.Println("update time:", time.Since(startTime))

		startTime = time.Now()
		assert.True(t, Verify(newSrs, prevSRSG1FirstElem, proof))
		fmt.Println("verify time:", time.Since(startTime))
	}
}

func TestSRS_Update(t *testing.T) {
	g1Degree := uint32(128)
	g2Degree := uint32(128)

	g1Powers := make([]*PointG1, g1Degree)
	g2Powers := make([]*PointG2, g2Degree)
	for i := uint32(0); i < g1Degree; i++ {
		g1Powers[i] = group1.New().Set(g1)
	}
	for i := uint32(0); i < g2Degree; i++ {
		g2Powers[i] = group2.New().Set(g2)
	}
	s1 := &SRS{
		g1Degree: g1Degree,
		g2Degree: g2Degree,
		g1Powers: g1Powers,
		g2Powers: g2Powers,
	}
	prevSRSG1FirstElem := s1.G1PowerOf(1)
	_, proof := s1.Update(FrFromInt(3))
	assert.True(t, schnorr_proof.Verify(prevSRSG1FirstElem, s1.G1PowerOf(1), proof))
	for i := uint32(0); i < g1Degree; i++ {
		assert.True(t, group1.Equal(s1.g1Powers[i], group1.MulScalar(group1.New(), s1.g1Powers[0], NewFr().Exp(FrFromInt(3), big.NewInt(int64(i))))))
	}
	for i := uint32(0); i < g2Degree; i++ {
		assert.True(t, group2.Equal(s1.g2Powers[i], group2.MulScalar(group2.New(), s1.g2Powers[0], NewFr().Exp(FrFromInt(3), big.NewInt(int64(i))))))
	}
}

func TestNewSRS(t *testing.T) {
	g1Degree := uint32(128)
	g2Degree := uint32(128)

	s, _ := NewSRS(g1Degree, g2Degree)
	r, _ := NewFr().Rand(rand.Reader)
	s.Update(r)

	oldG1Powers := make([]*PointG1, g1Degree)
	oldG2Powers := make([]*PointG2, g1Degree)
	for i := uint32(0); i < g1Degree; i++ {
		oldG1Powers[i] = group1.New().Set(s.g1Powers[i])
	}
	r, _ = NewFr().Rand(rand.Reader)
	for i := uint32(0); i < g2Degree; i++ {
		oldG2Powers[i] = group2.New().Set(s.g2Powers[i])
	}
	s.Update(r)
	for i := uint32(0); i < g1Degree; i++ {
		assert.True(t, group1.Equal(s.g1Powers[i], group1.MulScalar(group1.New(), oldG1Powers[i], NewFr().Exp(r, big.NewInt(int64(i))))))
	}
	for i := uint32(0); i < g2Degree; i++ {
		assert.True(t, group2.Equal(s.g2Powers[i], group2.MulScalar(group2.New(), oldG2Powers[i], NewFr().Exp(r, big.NewInt(int64(i))))))
	}
}

func TestUpdateSRSTime(t *testing.T) {
	g1Degree := uint32(2 << 13)
	g2Degree := uint32(64)
	s, _ := NewSRS(g1Degree, g2Degree)
	r1, _ := NewFr().Rand(rand.Reader)
	s.Update(r1)

	r2, _ := NewFr().Rand(rand.Reader)
	// 2^12 1.8s
	// 2^14 5.4s
	startTime := time.Now()
	s.Update(r2)
	fmt.Println(time.Since(startTime))
}

func TestG1ToBytes(t *testing.T) {
	for i := 0; i < 1024; i++ {
		g := group1.Affine(group1.MulScalar(group1.New(), g1, FrFromInt(i)))
		lastByte := group1.ToCompressed(g)[0]
		fmt.Printf("%4v %3v %08b\n", i, lastByte, lastByte)
	}
}

func TestG2ToBytes(t *testing.T) {
	for i := 0; i < 1024; i++ {
		g := group2.Affine(group2.MulScalar(group2.New(), g2, FrFromInt(i)))
		compressedG := group2.ToCompressed(g)
		lastByte := compressedG[0]
		fmt.Printf("%4v %3v %08b\n", i, lastByte, lastByte)
	}
}

func TestSRS_ToBinaryFile(t *testing.T) {
	g1Degree := uint32(2 << 14)
	g2Degree := uint32(2 << 14)

	startTime := time.Now()
	s, _ := NewSRS(g1Degree, g2Degree)
	fmt.Println(time.Since(startTime))

	// r, _ := NewFr().Rand(rand.Reader)
	r := FrFromInt(123)
	startTime = time.Now()
	s.Update(r)
	fmt.Println(time.Since(startTime))

	startTime = time.Now()
	if err := s.ToBinaryFile(); err != nil {
		fmt.Println(err)
	}
	fmt.Println(time.Since(startTime))
	fmt.Println(group1.Affine(s.g1Powers[1]))
	// [[18243118728177801133 14623819927364728506 1825459713259277039 12727253491261430033 12306323223833194323 1453726496009202278]
	// [3689948152682391391 11776693781107060284 17809592519250008748 16315109345568452514 7611191707486024903 438409272077168088]
	// [8505329371266088957 17002214543764226050 6865905132761471162 8632934651105793861 6631298214892334189 1582556514881692819]]
}

func TestSRS_FromBinaryFile(t *testing.T) {
	s := &SRS{}

	startTime := time.Now()
	s, err := FromBinaryFile()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(time.Since(startTime))
	fmt.Println(s.g1Degree)
	fmt.Println(len(s.g1Powers))
	fmt.Println(s.g2Degree)
	fmt.Println(len(s.g2Powers))
	fmt.Println(s.g1Powers[1])
	// [18243118728177801133 14623819927364728506 1825459713259277039 12727253491261430033 12306323223833194323 1453726496009202278]
	// [3689948152682391391 11776693781107060284 17809592519250008748 16315109345568452514 7611191707486024903 438409272077168088]
	// [8505329371266088957 17002214543764226050 6865905132761471162 8632934651105793861 6631298214892334189 1582556514881692819]]
}

func TestFrConvert(t *testing.T) {
	fr := FrFromInt(1234567890123456789)
	fmt.Printf("%X\n", fr)
	fmt.Println(fr.ToBytes())
}

func TestFqConvert(t *testing.T) {
	fe := NewFe().SetBig(big.NewInt(1234567890123456789))
	fmt.Printf("%X\n", fe)
	fmt.Println(fe.Bytes())
}

func TestSRS_ToCompressedBytes(t *testing.T) {
	s, proof := NewSRS(256, 256)
	srsBytes, err := s.ToCompressedBytes()
	if err != nil {
		fmt.Println("to byte", err)
		return
	}

	fmt.Println("from byte")
	recoverSRS, err := FromCompressedBytes(srsBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(Verify(recoverSRS, group1.One(), proof))
}
