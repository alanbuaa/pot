package schnorr_proof

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

func TestVerifyTrue(t *testing.T) {
	x, _ := NewFr().Rand(rand.Reader)
	y := group1.MulScalar(group1.New(), group1.One(), x)
	proof := CreateWitness(group1.One(), y, x)
	assert.True(t, Verify(group1.One(), y, proof))
}
