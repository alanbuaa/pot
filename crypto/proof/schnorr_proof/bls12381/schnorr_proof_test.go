package schnorr_proof

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"testing"
)

func TestVerifyTrue(t *testing.T) {
	x, _ := NewFr().Rand(rand.Reader)
	y := group1.MulScalar(group1.New(), group1.One(), x)
	proof := CreateWitness(group1.One(), y, x)
	assert.True(t, Verify(group1.One(), y, proof))
}

func TestProof_ToBytes_FromBytes(t *testing.T) {
	x, _ := NewFr().Rand(rand.Reader)
	y := group1.MulScalar(group1.New(), group1.One(), x)
	proof := CreateWitness(group1.One(), y, x)

	proofBytes := proof.ToBytes()

	decodeProof, err := new(SchnorrProof).FromBytes(proofBytes)
	assert.Nil(t, err)
	assert.Equal(t, decodeProof, proof)

	assert.Equal(t, Verify(group1.One(), y, decodeProof), true)
}
