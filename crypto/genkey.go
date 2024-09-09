package crypto

import (
	"crypto/rand"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
)

const (
	PrivateKeyLen = 32
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
	group1 := bls12381.NewG1()
	return &PrivateKey{
		priv: privKey.ToBytes(),
		pub:  *group1.MulScalar(group1.New(), group1.One(), privKey),
	}
}

func (k *PrivateKey) Private() []byte {
	return k.priv
}

func (k *PrivateKey) PublicKey() bls12381.PointG1 {
	return k.pub
}
func (k *PrivateKey) PublicKeyBytes() []byte {
	group1 := bls12381.NewG1()
	return group1.ToBytes(&k.pub)
}
