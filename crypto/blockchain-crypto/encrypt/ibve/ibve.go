package ibve

import (
	. "blockchain-crypto/types/curve/bls12381"
	"bytes"
	"crypto/rand"
)

var (
	g1 = NewG1()
	g2 = NewG2()
	gt = NewGT()
)

type CipherText struct {
	C1 *PointG1
	C2 *PointG2
	C3 *E
}

func (c *CipherText) ToBytes() []byte {
	buffer := new(bytes.Buffer)
	buffer.Write(g1.ToCompressed(c.C1))
	buffer.Write(g2.ToCompressed(c.C2))
	buffer.Write(gt.ToBytes(c.C3))
	return buffer.Bytes()
}

func (c *CipherText) FromBytes(data []byte) (*CipherText, error) {
	pointG1Buf := make([]byte, 48)
	pointG2Buf := make([]byte, 96)
	pointGTBuf := make([]byte, 576)
	buffer := bytes.NewBuffer(data)
	// c1
	_, err := buffer.Read(pointG1Buf)
	if err != nil {
		return nil, err
	}
	c.C1, err = g1.FromCompressed(pointG1Buf)
	if err != nil {
		return nil, err
	}
	// c2
	_, err = buffer.Read(pointG2Buf)
	if err != nil {
		return nil, err
	}
	c.C2, err = g2.FromCompressed(pointG2Buf)
	if err != nil {
		return nil, err
	}
	// c3
	_, err = buffer.Read(pointGTBuf)
	if err != nil {
		return nil, err
	}
	c.C3, err = gt.FromBytes(pointGTBuf)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func Keygen() (*Fr, *PointG1) {
	x, _ := NewFr().Rand(rand.Reader)
	y := g1.MulScalar(g1.New(), g1.One(), x)
	return x, y
}

func Encrypt(y *PointG1, msg *E) *CipherText {
	// random r
	r, _ := NewFr().Rand(rand.Reader)
	// c1 = g1 ^ r
	c1 := g1.MulScalar(g1.New(), g1.One(), r)
	// c2 = g2 ^ r
	c2 := g2.MulScalar(g2.New(), g2.One(), r)
	// c3 = e(y^r, c2) * m
	c3 := NewPairingEngine().AddPair(g1.MulScalar(g1.New(), y, r), c2).Result()
	gt.Mul(c3, c3, msg)
	return &CipherText{
		C1: c1,
		C2: c2,
		C3: c3,
	}
}

func Decrypt(sigma *PointG1, cipherText *CipherText) (msg *E) {
	// sigma = c1 ^ x
	// m = c3 / e(σ, c2)
	eInv := gt.New()
	gt.Inverse(eInv, NewPairingEngine().AddPair(sigma, cipherText.C2).Result())
	msg = gt.New()
	gt.Mul(msg, cipherText.C3, eInv)
	return msg
}

func Verify(sigma *PointG1, y *PointG1, c2 *PointG2) bool {
	// e(σ, g2) = e(y, c2)
	return NewPairingEngine().AddPair(sigma, g2.One()).AddPairInv(y, c2).Check()
}
