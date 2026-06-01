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
	PqcScheme = int(3)
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

func GenerateCommiteeKey(pqckey *PqcKey, seed []byte, randbyte []byte) *PrivateKey {
	pqcbyte := pqckey.PublicKeyBytes()
	hashkey1 := bytes.Join([][]byte{seed, randbyte}, []byte{})
	hash1 := Hash(hashkey1)

	hashkey2 := bytes.Join([][]byte{pqcbyte, hash1}, []byte{})
	privKey := bls12382.NewFr().FromBytes(Hash(hashkey2))
	group1 := bls12382.NewG1()
	return &PrivateKey{
		priv: privKey.ToBytes(),
		pub:  *group1.MulScalar(group1.New(), group1.One(), privKey),
	}

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
	Privkey []byte
	Pubkey  []byte
	Scheme  int
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
	return k.Privkey
}

func (k *PqcKey) PublicKeyBytes() []byte {
	return k.Pubkey
}

func (k *PqcKey) SignatureScheme() int {
	return k.Scheme
}

func (k *PqcKey) Sign(message []byte) ([]byte, error) {
	sig, err := pqcgo.Sign(k.Scheme, message, k.Privkey)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func VerifySig(message []byte, sig []byte, pk []byte) (bool, error) {
	return pqcgo.Verify(PqcScheme, sig, message, pk)
}

func VerifyCommitteePKAPI(alphaBytes, pkBytes, gBytes, yBytes, gDotBytes, yDotBytes []byte) bool {
	group1 := bls12382.NewG1()
	g, err := group1.FromBytes(gBytes)
	if err != nil {
		return false
	}
	y, err := group1.FromBytes(yBytes)
	if err != nil {
		return false
	}
	gDot, err := group1.FromCompressed(gDotBytes)
	if err != nil {
		return false
	}
	yDot, err := group1.FromCompressed(yDotBytes)
	if err != nil {
		return false
	}
	x := bls12382.HashToFr(append(alphaBytes, pkBytes...))
	if !group1.Equal(y, group1.MulScalar(group1.New(), g, x)) {
		return false
	}
	if !group1.Equal(yDot, group1.MulScalar(group1.New(), gDot, x)) {
		return false
	}
	return true
}

type CommiteeKeySig struct {
	Alpha       []byte `json:"alpha"`
	Pqcpubkey   []byte `json:"pqcpubkey"`
	G           []byte `json:"g"`
	CommiteeKey []byte `json:"commiteekey"`
	Rcommit     []byte `json:"rcommit"`
	AwardKey    []byte `json:"awardkey"`
}
