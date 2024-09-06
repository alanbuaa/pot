package crypto

import (
	"crypto/rand"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

const (
	PrivateKeyLen = 32
)

var (
	g1Group = bls12381.NewG1()
)

type PrivateKey struct {
	priv []byte
	pub  bls12381.PointG1
}

func GenerateKey() *PrivateKey {
	privKey, err := bls12381.NewFr().Rand(rand.Reader)
	if err != nil {
		return nil
	}

	return &PrivateKey{
		priv: privKey.ToBytes(),
		pub:  *g1Group.MulScalar(g1Group.New(), g1Group.One(), privKey),
	}
}

func (k *PrivateKey) Private() []byte {
	return k.priv
}

func (k *PrivateKey) PublicKey() bls12381.PointG1 {
	return k.pub
}
func (k *PrivateKey) PublicKeyBytes() []byte {
	return g1Group.ToBytes(&k.pub)
}
