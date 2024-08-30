package poly

import (
	. "blockchain-crypto/types/curve/bls12381"
	"crypto/rand"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/big"
	math_rand "math/rand"
	"testing"
)

func TestEval(t *testing.T) {
	coeffs := make([]*Fr, 3)
	coeffs[0] = FrFromInt(3)
	coeffs[1] = FrFromInt(-4)
	coeffs[2] = NewFr().One()
	poly := FromSlice(coeffs)
	x := FrFromInt(4)
	fmt.Println(poly.Eval(x))
}

func TestUVPolynomial_Equal(t *testing.T) {
	zeroPoly := Zero()
	onePoly := One()
	poly := FromSlice([]*Fr{NewFr().One(), NewFr(), NewFr().One()})
	assert.True(t, poly.Equal(poly.Add(zeroPoly)))
	assert.True(t, poly.Equal(poly.Sub(zeroPoly)))
	assert.True(t, poly.Equal(poly.Mul(onePoly)))
	// assert.True(t, poly.Equal(poly.DivWithoutRem(onePoly)))
}

func TestUVPolynomial_IsZero(t *testing.T) {
	zeroPoly := Zero()
	assert.True(t, zeroPoly.IsZero())

	onePoly := One()
	assert.True(t, !onePoly.IsZero())

	poly := FromSlice([]*Fr{NewFr().One(), NewFr(), NewFr().One()})

	subPoly := poly.Sub(poly)
	assert.True(t, subPoly.IsZero())

	mulZeroPoly := poly.Mul(zeroPoly)
	assert.True(t, mulZeroPoly.IsZero())

	mulScalarZeroPoly := poly.MulScalar(NewFr())
	assert.True(t, mulScalarZeroPoly.IsZero())
}

func TestUVPolynomial_Add(t *testing.T) {
	poly1 := FromSlice([]*Fr{NewFr().One(), NewFr(), NewFr().One()})
	poly2 := FromSlice([]*Fr{NewFr().One(), NewFr().One(), NewFr().One(), NewFr().One()})
	poly3 := FromSlice([]*Fr{FrFromInt(2), NewFr().One(), FrFromInt(2), NewFr().One()})
	assert.True(t, poly3.Equal(poly2.Add(poly1)))

	r, _ := NewFr().Rand(rand.Reader)
	rNeg := NewFr().Neg(r)
	poly4 := FromSlice([]*Fr{r})
	poly5 := FromSlice([]*Fr{rNeg})
	assert.True(t, Zero().Equal(poly4.Add(poly5)))
}

func TestUVPolynomial_Sub(t *testing.T) {
	poly1 := FromSlice([]*Fr{NewFr().One(), NewFr(), NewFr().One()})
	poly2 := FromSlice([]*Fr{NewFr().One(), NewFr().One(), NewFr().One(), NewFr().One()})
	poly3 := FromSlice([]*Fr{FrFromInt(2), NewFr().One(), FrFromInt(2), NewFr().One()})
	assert.True(t, poly2.Equal(poly3.Sub(poly1)))

	// r, _ := NewFr().Rand(rand.Reader)
	r := NewFr().One()
	poly4 := FromSlice([]*Fr{r})
	poly5 := FromSlice([]*Fr{r})
	assert.True(t, Zero().Equal(poly4.Sub(poly5)))
}

func TestUVPolynomial_MulScalar(t *testing.T) {
	r := NewFr()
	for {
		if !r.IsZero() {
			break
		}
		r, _ = NewFr().Rand(rand.Reader)
	}
	rInv := NewFr()
	rInv.Inverse(r)
	poly1 := FromSlice([]*Fr{rInv, NewFr(), NewFr().One()})
	poly2 := FromSlice([]*Fr{NewFr().One(), NewFr(), r})
	assert.True(t, poly2.Equal(poly1.MulScalar(r)))
	assert.True(t, Zero().Equal(poly1.MulScalar(NewFr())))
}

func TestUVPolynomial_DivScalar(t *testing.T) {
	r := NewFr()
	for {
		if !r.IsZero() {
			break
		}
		r, _ = NewFr().Rand(rand.Reader)
	}
	rInv := NewFr()
	rInv.Inverse(r)
	poly1 := FromSlice([]*Fr{rInv, NewFr(), NewFr().One()})
	poly2 := FromSlice([]*Fr{NewFr().One(), NewFr(), r})
	assert.True(t, poly1.Equal(poly2.DivScalar(r)))
	assert.True(t, poly2.DivScalar(NewFr()) == nil)
}

func TestUVPolynomial_Mul(t *testing.T) {
	poly1 := FromSlice([]*Fr{FrFromInt(-1), NewFr().One()})
	poly2 := FromSlice([]*Fr{NewFr().One(), NewFr().One(), NewFr().One()})
	poly3 := FromSlice([]*Fr{FrFromInt(-1), NewFr(), NewFr(), NewFr().One()})
	assert.True(t, poly3.Equal(poly2.Mul(poly1)))
	assert.True(t, poly3.Equal(poly1.Mul(poly2)))
}

func TestUVPolynomial_DivWithoutRem(t *testing.T) {
	poly1 := FromSlice([]*Fr{FrFromInt(-1), NewFr().One()})
	poly2 := FromSlice([]*Fr{NewFr().One(), NewFr().One(), NewFr().One()})
	poly3 := FromSlice([]*Fr{FrFromInt(-1), NewFr(), NewFr(), NewFr().One()})
	assert.True(t, poly1.Equal(poly3.DivWithoutRem(poly2)))
	assert.True(t, poly2.Equal(poly3.DivWithoutRem(poly1)))
}

func TestInterpolate(t *testing.T) {
	size := math_rand.Intn(128)
	coeffs := make([]*Fr, size)
	for i := range coeffs {
		coeffs[i], _ = NewFr().Rand(rand.Reader)
	}
	poly := FromSlice(coeffs)
	n := poly.Degree() + 1
	xs := make([]*Fr, n)
	ys := make([]*Fr, n)
	for i := uint32(0); i < n; i++ {
		iFr := FrFromUInt32(i + 1)
		xs[i] = iFr
		ys[i] = poly.Eval(iFr)
	}
	polyInt := Interpolate(xs, ys)
	assert.True(t, poly.Equal(polyInt))
}

func TestInterpolationAndEval(t *testing.T) {
	xs := make([]*Fr, 3)
	xs[0] = FrFromBig(big.NewInt(1))
	xs[1] = FrFromBig(big.NewInt(2))
	xs[2] = FrFromBig(big.NewInt(4))
	ys := make([]*Fr, 3)
	ys[0] = FrFromBig(big.NewInt(2))
	ys[1] = FrFromBig(big.NewInt(5))
	ys[2] = FrFromBig(big.NewInt(17))
	fmt.Println(InterpolationAndEval(NewFr().Zero(), xs, ys))
	fmt.Println(InterpolationAndEval(NewFr().One(), xs, ys))
	fmt.Println(InterpolationAndEval(FrFromBig(big.NewInt(3)), xs, ys))
}
