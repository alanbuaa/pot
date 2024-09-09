package dleq

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"testing"
)

func TestDLEQTrue(t *testing.T) {
	group1 := NewG1()
	g1 := group1.MulScalar(group1.New(), group1.One(), FrFromInt(3))
	g2 := group1.MulScalar(group1.New(), group1.One(), FrFromInt(5))
	// alpha, _ := NewFr().Rand(rand.Reader)
	alpha := FrFromInt(3)
	h1 := group1.MulScalar(group1.New(), g1, alpha)
	h2 := group1.MulScalar(group1.New(), g2, alpha)

	dleq := &DLEQ{
		Index: 1,
		G1:    g1,
		H1:    h1,
		G2:    g2,
		H2:    h2,
	}

	proof := dleq.Prove(alpha)

	res := dleq.Verify(proof) && Verify(1, g1, h1, g2, h2, proof)

	assert.Equal(t, res, true)

}

func TestDLEQFalse(t *testing.T) {
	group1 := NewG1()
	g1 := group1.MulScalar(group1.New(), group1.One(), FrFromInt(3))
	g2 := group1.MulScalar(group1.New(), group1.One(), FrFromInt(5))
	alpha, _ := NewFr().Rand(rand.Reader)

	h1 := group1.MulScalar(group1.New(), g1, alpha)
	h2 := group1.MulScalar(group1.New(), g2, alpha)

	dleq := &DLEQ{
		Index: 1,
		G1:    g1,
		H1:    h1,
		G2:    g2,
		H2:    h2,
	}
	dleq2 := &DLEQ{
		Index: 2,
		G1:    g1,
		H1:    h1,
		G2:    g2,
		H2:    h2,
	}

	proof := dleq.Prove(alpha)
	res := dleq2.Verify(proof)

	assert.Equal(t, res, false)
}

func TestProof_ToBytes_FromBytes(t *testing.T) {
	group1 := NewG1()
	g1 := group1.MulScalar(group1.New(), group1.One(), FrFromInt(3))
	g2 := group1.MulScalar(group1.New(), group1.One(), FrFromInt(5))
	// alpha, _ := NewFr().Rand(rand.Reader)
	alpha := FrFromInt(3)
	h1 := group1.MulScalar(group1.New(), g1, alpha)
	h2 := group1.MulScalar(group1.New(), g2, alpha)

	dleq := &DLEQ{
		Index: 1,
		G1:    g1,
		H1:    h1,
		G2:    g2,
		H2:    h2,
	}

	proof := dleq.Prove(alpha)

	proofBytes := proof.ToBytes()

	decodeProof, err := new(Proof).FromBytes(proofBytes)
	assert.Nil(t, err)
	assert.Equal(t, decodeProof, proof)

	assert.Equal(t, Verify(1, g1, h1, g2, h2, decodeProof), true)
}
