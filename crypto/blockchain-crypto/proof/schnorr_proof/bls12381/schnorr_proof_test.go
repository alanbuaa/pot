package schnorr_proof

import (
	. "blockchain-crypto/types/curve/bls12381"
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVerifyTrue(t *testing.T) {
	group1 := NewG1()
	x, _ := NewFr().Rand(rand.Reader)
	y := group1.MulScalar(group1.New(), group1.One(), x)
	proof := CreateWitness(group1.One(), y, x)
	assert.True(t, Verify(group1.One(), y, proof))
}

func TestProof_ToBytes_FromBytes(t *testing.T) {
	group1 := NewG1()
	x, _ := NewFr().Rand(rand.Reader)
	y := group1.MulScalar(group1.New(), group1.One(), x)
	proof := CreateWitness(group1.One(), y, x)

	proofBytes := proof.ToBytes()

	decodeProof, err := new(SchnorrProof).FromBytes(proofBytes)
	assert.Nil(t, err)
	assert.Equal(t, decodeProof, proof)

	assert.Equal(t, Verify(group1.One(), y, decodeProof), true)
}
