package crypto

import (
	bls12382 "blockchain-crypto/types/curve/bls12381"
	"bytes"
	"crypto/rand"

	"blockchain-crypto/pqcgo"
)

const (
	PrivateKeyLen = 32
)

var (
	PqcScheme = int(0)
)

type PrivateKey struct {
	priv []byte
	pub  bls12382.PointG1
}

func GenerateKey() *PrivateKey {
	privKey, err := bls12382.NewFr().Rand(rand.Reader)
	if err != nil {
		return nil
	}
	group1 := bls12382.NewG1()
	return &PrivateKey{
		priv: privKey.ToBytes(),
		pub:  *group1.MulScalar(group1.New(), group1.One(), privKey),
	}
}

func GenerateCommiteeKeyWithPqcKey(pqcKey *PqcKey) (*PrivateKey, []byte, error) {
	pqcbyte := pqcKey.PublicKeyBytes()
	randseed := make([]byte, 32)
	_, err := rand.Read(randseed)
	if err != nil {
		return nil, nil, err
	}

	hashkey := bytes.Join([][]byte{pqcbyte, randseed}, []byte{})
	privKey := bls12382.NewFr().FromBytes(Hash((hashkey)))
	group1 := bls12382.NewG1()
	return &PrivateKey{
		priv: privKey.ToBytes(),
		pub:  *group1.MulScalar(group1.New(), group1.One(), privKey),
	}, randseed, nil
}

func (k *PrivateKey) Private() []byte {
	return k.priv
}

func (k *PrivateKey) PublicKey() bls12382.PointG1 {
	return k.pub
}
func (k *PrivateKey) PublicKeyBytes() []byte {
	group1 := bls12382.NewG1()
	return group1.ToBytes(&k.pub)
}

type PqcKey struct {
	privkey []byte
	pubkey  []byte
	scheme  int
}

func GeneratePqcKey() (*PqcKey, error) {
	randseed := make([]byte, 32)
	_, err := rand.Read(randseed)

	if err != nil {
		return nil, err
	}
	pk, sk, err := pqcgo.KeyGenWithSeed(PqcScheme, randseed)
	if err != nil {
		return nil, err
	}
	return &PqcKey{sk, pk, PqcScheme}, nil
}

func (k *PqcKey) PrivateKeyBytes() []byte {
	return k.privkey
}

func (k *PqcKey) PublicKeyBytes() []byte {
	return k.pubkey
}

func (k *PqcKey) SignatureScheme() int {
	return k.scheme
}

func (k *PqcKey) Sign(message []byte) ([]byte, error) {
	sig, err := pqcgo.Sign(k.scheme, message, k.privkey)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func VerifySig(message []byte, sig []byte, pk []byte) (bool, error) {
	return pqcgo.Verify(PqcScheme, sig, message, pk)
}
