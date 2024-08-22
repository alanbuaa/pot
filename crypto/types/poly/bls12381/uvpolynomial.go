// Package poly implements polynomials in Fr.
package poly

import (
	"crypto/rand"
	"crypto/subtle"
	"strings"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

var (
	group1 = NewG1()
	group2 = NewG2()
)

// UVPolynomial represents a uni-variate polynomial.
type UVPolynomial struct {
	// Coefficients of the polynomial. The leftmost is the constant term.
	Coeffs []*Fr
}

// Coefficients TODO Coefficients return the list of coefficients representing p. This
// information is generally PRIVATE and should not be revealed to a third party
// lightly.
func (p *UVPolynomial) Coefficients() []*Fr {
	return p.Coeffs
}

// FromSlice returns a UVPolynomial based on the given coefficients
func FromSlice(coeffs []*Fr) *UVPolynomial {
	for _, coeff := range coeffs {
		if coeff == nil {
			return nil
		}
	}
	return (&UVPolynomial{coeffs}).trim()
}

func One() *UVPolynomial {
	return &UVPolynomial{Coeffs: []*Fr{NewFr().One()}}
}

func Zero() *UVPolynomial {
	return &UVPolynomial{Coeffs: []*Fr{NewFr().Zero()}}
}

func (p *UVPolynomial) IsZero() bool {
	if p == nil {
		return false
	}
	if len(p.Coeffs) == 0 {
		return true
	}
	for _, coeff := range p.Coeffs {
		if !coeff.IsZero() {
			return false
		}
	}
	return true
}

// Degree return the degree of UVPolynomial.
func (p *UVPolynomial) Degree() uint32 {
	return uint32(len(p.Coeffs) - 1)
}

// Equal checks equality of two secret sharing polynomials p and q.
func (p *UVPolynomial) Equal(q *UVPolynomial) bool {
	if len(p.Coeffs) != len(q.Coeffs) {
		return false
	}
	pLen := len(p.Coeffs)
	for i := 0; i < pLen; i++ {
		if subtle.ConstantTimeCompare(p.Coeffs[i].ToBytes(), q.Coeffs[i].ToBytes()) != 1 {
			return false
		}
	}
	return true
}

// Eval computes the private share v = p(i).
func (p *UVPolynomial) Eval(x *Fr) *Fr {
	value := NewFr().Zero()
	pLen := len(p.Coeffs)
	for i := pLen - 1; i >= 0; i-- {
		value.Mul(value, x)
		value.Add(value, p.Coeffs[i])
	}
	return value
}

func (p *UVPolynomial) trim() *UVPolynomial {
	pLen := len(p.Coeffs)
	for i := pLen - 1; i > 0; i-- {
		if !p.Coeffs[i].IsZero() {
			if i != pLen-1 {
				p.Coeffs = p.Coeffs[:i+1]
			}
			return p
		}
	}
	p.Coeffs = p.Coeffs[:1]
	return p
}

// Add computes the component-wise sum of the polynomials p and q and returns it
// as a new polynomial.
func (p *UVPolynomial) Add(q *UVPolynomial) *UVPolynomial {
	if p == nil || q == nil {
		return nil
	}
	var coeffs []*Fr
	pLen := len(p.Coeffs)
	qLen := len(q.Coeffs)
	if pLen > qLen {
		coeffs = make([]*Fr, pLen)
		for i := 0; i < pLen; i++ {
			coeffs[i] = NewFr().Set(p.Coeffs[i])
		}
		for i := 0; i < qLen; i++ {
			coeffs[i].Add(coeffs[i], q.Coeffs[i])
		}
	} else {
		coeffs = make([]*Fr, qLen)
		for i := 0; i < qLen; i++ {
			coeffs[i] = NewFr().Set(q.Coeffs[i])
		}
		for i := 0; i < pLen; i++ {
			coeffs[i].Add(coeffs[i], p.Coeffs[i])
		}
	}
	return (&UVPolynomial{coeffs}).trim()
}

// Sub computes the component-wise sum of the polynomials p and q and returns it
// as a new polynomial.
func (p *UVPolynomial) Sub(q *UVPolynomial) *UVPolynomial {
	if p == nil || q == nil {
		return nil
	}
	pLen := len(p.Coeffs)
	qLen := len(q.Coeffs)
	var coeffs []*Fr
	if pLen >= qLen {
		coeffs = make([]*Fr, pLen)
		for i := 0; i < pLen; i++ {
			coeffs[i] = NewFr().Set(p.Coeffs[i])
		}
		for i := 0; i < qLen; i++ {
			coeffs[i].Sub(coeffs[i], q.Coeffs[i])
		}
	} else {
		coeffs = make([]*Fr, qLen)
		for i := 0; i < pLen; i++ {
			coeffs[i] = NewFr().Set(p.Coeffs[i])
		}
		for i := pLen; i < qLen; i++ {
			coeffs[i] = NewFr().Zero()
		}
		for i := 0; i < qLen; i++ {
			coeffs[i].Sub(coeffs[i], q.Coeffs[i])
		}
	}
	return (&UVPolynomial{coeffs}).trim()
}

// MulScalar multiples two UVPolynomials p and q together.
func (p *UVPolynomial) MulScalar(scalar *Fr) *UVPolynomial {
	if p == nil || scalar == nil {
		return nil
	}
	if p.IsZero() || scalar.IsZero() {
		return Zero()
	}
	coeffs := make([]*Fr, len(p.Coeffs))
	for i, c := range p.Coeffs {
		coeffs[i] = NewFr().Mul(c, scalar)
	}
	return &UVPolynomial{coeffs}
}

// DivScalar TODO DivScalar multiples two UVPolynomials p and q together.
func (p *UVPolynomial) DivScalar(scalar *Fr) *UVPolynomial {
	if p == nil || scalar == nil || scalar.IsZero() {
		return nil
	}
	if p.IsZero() {
		return Zero()
	}
	scalarInv := NewFr()
	scalarInv.Inverse(scalar)
	coeffs := make([]*Fr, len(p.Coeffs))
	for i, c := range p.Coeffs {
		coeffs[i] = NewFr()
		coeffs[i].Mul(c, scalarInv)
	}
	return &UVPolynomial{coeffs}
}

// Mul TODO Mul multiples two UVPolynomials p and q together.
func (p *UVPolynomial) Mul(q *UVPolynomial) *UVPolynomial {
	if p == nil || q == nil {
		return nil
	}
	if p.IsZero() || q.IsZero() {
		return Zero()
	}
	coeffs := make([]*Fr, len(p.Coeffs)+len(q.Coeffs)-1)
	for i := range coeffs {
		coeffs[i] = NewFr()
	}
	for i, pCoeff := range p.Coeffs {
		for j, qCoeff := range q.Coeffs {
			tmp := NewFr()
			tmp.Mul(pCoeff, qCoeff)
			index := i + j
			coeffs[index].Add(tmp, coeffs[index])
		}
	}
	return (&UVPolynomial{coeffs}).trim()
}

// Div TODO Div multiples two UVPolynomials p and q together.
func (p *UVPolynomial) Div(q *UVPolynomial) (*UVPolynomial, *UVPolynomial) {
	if p == nil || q == nil || q.IsZero() {
		return nil, nil
	}
	if p.IsZero() {
		return Zero(), Zero()
	}
	if p.Degree() < q.Degree() {
		coeffs := make([]*Fr, len(p.Coeffs))
		for i, c := range p.Coeffs {
			coeffs[i] = NewFr().Set(c)
		}
		return Zero(), &UVPolynomial{coeffs}
	}
	qLen := len(q.Coeffs)
	qFirstCoeffInv := NewFr()
	qFirstCoeffInv.Inverse(q.Coeffs[qLen-1])
	r := &UVPolynomial{}
	r.Coeffs = make([]*Fr, len(p.Coeffs)-qLen+1)
	for i := range r.Coeffs {
		r.Coeffs[i] = NewFr()
	}

	coeffs := make([]*Fr, len(p.Coeffs))
	rem := &UVPolynomial{Coeffs: coeffs}
	for i, c := range p.Coeffs {
		coeffs[i] = NewFr().Set(c)
	}
	lastRemLen := len(rem.Coeffs)
	index := len(r.Coeffs) - 1
	for {
		remLen := len(rem.Coeffs)
		if remLen < qLen {
			break
		}
		index -= lastRemLen - remLen
		lastRemLen = remLen
		r.Coeffs[index].Mul(rem.Coeffs[remLen-1], qFirstCoeffInv)
		tmp := q.MulScalar(r.Coeffs[index])
		for i, c := range tmp.Coeffs {
			k := i + remLen - qLen
			rem.Coeffs[k].Sub(rem.Coeffs[k], c)
		}
		rem = rem.trim()
	}
	return r.trim(), rem
}

func (p *UVPolynomial) DivWithoutRem(q *UVPolynomial) *UVPolynomial {
	r, _ := p.Div(q)
	return r
}

// NewSecretPolynomial creates a new secret sharing polynomial using
// the secret sharing threshold t, and the secret to be shared s.
func NewSecretPolynomial(t uint32, s *Fr) *UVPolynomial {
	coeffs := make([]*Fr, t)
	if s == nil {
		coeffs[0], _ = NewFr().Rand(rand.Reader)
	} else {
		coeffs[0] = s
	}
	for i := uint32(1); i < t; i++ {
		coeffs[i], _ = NewFr().Rand(rand.Reader)
	}
	return &UVPolynomial{Coeffs: coeffs}
}

// NewSimplePolynomial only for test
func NewSimplePolynomial(t uint32, s *Fr) *UVPolynomial {
	coeffs := make([]*Fr, t)
	if s == nil {
		coeffs[0] = FrFromInt(123)
	} else {
		coeffs[0] = s
	}
	for i := uint32(1); i < t; i++ {
		coeffs[i] = NewFr()
	}
	coeffs[t-1].One()
	return &UVPolynomial{Coeffs: coeffs}
}

// TODO
func (p *UVPolynomial) String() string {
	var strs = make([]string, len(p.Coeffs))
	// for i, c := range p.Coeffs {
	// strs[i] = c.String()
	// }
	return "[ " + strings.Join(strs, ", ") + " ]"
}

func LagrangePoly() {

}

// TODO
func Interpolate(xs []*Fr, ys []*Fr) *UVPolynomial {
	// f(X) = Σ lᵢ(X) · yᵢ
	// lᵢ(X) = ∏ (X - xᵢ) / (X - xᵢ) / ∏_(j=1,j≠i)^(n) (xᵢ - xⱼ)
	// p(X) = ∏ (X - xᵢ)
	// q = ∏_(j=1,j≠i)^(n) (xᵢ - xⱼ)
	// lᵢ(X) = p(X) / (X - xᵢ) / q
	// p(X) = ∏ (X - xᵢ)
	p := One()
	xNegs := make([]*Fr, len(xs))
	for i, x := range xs {
		xNegs[i] = NewFr().Zero()
		xNegs[i].Sub(xNegs[i], x)
		tmp := FromSlice([]*Fr{xNegs[i], NewFr().One()})
		p = p.Mul(tmp)
	}
	poly := Zero()
	for i, xi := range xs {
		// lᵢ(X) = p(X) / (X - xᵢ) / q
		l := p.DivWithoutRem(&UVPolynomial{[]*Fr{xNegs[i], NewFr().One()}})
		// q = ∏_(j=1,j≠i)^(n) (xᵢ - xⱼ)
		q := NewFr().One()
		for j, xj := range xs {
			if j != i {
				q.Mul(q, NewFr().Sub(xi, xj))
			}
		}
		l = l.DivScalar(q)
		// f(X) = Σ lᵢ(X) · yᵢ
		poly = poly.Add(l.MulScalar(ys[i]))
	}
	return poly
}

// TODO 快速插值
func InterpolationAndEval(x *Fr, xs []*Fr, ys []*Fr) *Fr {
	// x in xs
	for i := range xs {
		if x.Equal(xs[i]) {
			return ys[i]
		}
	}
	eval := NewFr().Zero()
	// p(x) = ∏ (x - xs[i])
	p := NewFr().One()
	for i := range xs {
		p.Mul(p, NewFr().Sub(x, xs[i]))
	}
	for i := range xs {
		// p(x) / (x - xs[i])
		lagrangeBasis := NewFr()

		tmp := NewFr().Sub(x, xs[i])
		tmp.Inverse(tmp)
		lagrangeBasis.Mul(p, tmp)
		// p(X) / (X - 𝜔ⁱ⁻¹) / q
		// q = ∏_(j=1,j≠i)^(n) (xs[i] - xs[j])
		for j := range xs {
			if j != i {
				q := NewFr().Sub(xs[i], xs[j])
				q.Inverse(q)
				lagrangeBasis.Mul(lagrangeBasis, q)
			}
		}
		factor := NewFr().Mul(ys[i], lagrangeBasis)
		eval.Add(eval, factor)
	}
	return eval
}
