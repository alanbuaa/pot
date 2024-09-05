package dleq

import (
	"github.com/stretchr/testify/assert"
	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bn254"
	"testing"
)

func TestDLEQTrue(t *testing.T) {
	g1 := g1curve.MulScalar(g1curve.New(), g1curve.One(), FrFromInt(3))
	g2 := g1curve.MulScalar(g1curve.New(), g1curve.One(), FrFromInt(5))
	// alpha, _ := NewFr().Rand(rand.Reader)
	alpha := FrFromInt(3)
	h1 := g1curve.MulScalar(g1curve.New(), g1, alpha)
	h2 := g1curve.MulScalar(g1curve.New(), g2, alpha)

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
	g1 := g1curve.MulScalar(g1curve.New(), g1curve.One(), FrFromInt(3))
	g2 := g1curve.MulScalar(g1curve.New(), g1curve.One(), FrFromInt(5))
	alpha := FrFromInt(3)

	h1 := g1curve.MulScalar(g1curve.New(), g1, alpha)
	h2 := g1curve.MulScalar(g1curve.New(), g2, alpha)

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
