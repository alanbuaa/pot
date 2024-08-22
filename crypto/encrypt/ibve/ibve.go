package ibve

import (
	"crypto/rand"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

var (
	g1 = NewG1()
	g2 = NewG2()
	gt = NewGT()
)

func Keygen() (*Fr, *PointG1) {
	sk, _ := NewFr().Rand(rand.Reader)
	pk := g1.MulScalar(g1.New(), g1.One(), sk)
	return sk, pk
}

func Encrypt(pk *PointG1, msg *E) (g1r *PointG1, g2r *PointG2, c2 *E) {
	// 生成随机数 r
	r, _ := NewFr().Rand(rand.Reader)
	// c1 = g2 ^ r
	// ID = (g1^r, g2^r)
	g1r = g1.MulScalar(g1.New(), g1.One(), r)
	g2r = g2.MulScalar(g2.New(), g2.One(), r)
	// c2 = e(y^r, c1) * m
	c2 = NewPairingEngine().AddPair(g1.MulScalar(g1.New(), pk, r), g2r).Result()
	gt.Mul(c2, c2, msg)
	return g1r, g2r, c2
}

func Decrypt(sk *Fr, pk *PointG1, g1r *PointG1, g2r *PointG2, c2 *E) (res bool, msg *E) {
	// sigma = c1 ^ x
	sigma := g2.MulScalar(g2.New(), g2r, sk)

	// m = c2 / e(g1^r, σ)
	eInv := gt.New()
	gt.Inverse(eInv, NewPairingEngine().AddPair(g1r, sigma).Result())
	msg = gt.New()
	gt.Mul(msg, c2, eInv)

	// 验证 e(σ, g2) = e(y, g2r)
	return NewPairingEngine().AddPair(g1.One(), sigma).AddPairInv(pk, g2r).Check(), msg
}
