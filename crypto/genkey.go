package crypto

import (
	"crypto/rand"
)

const (
	PrivateKeyLen = 32
)

type PrivateKey struct {
	Prikey []byte
}

type PublicKey struct {
	Pubkey []byte
}

func CreatePriKey(salt []byte) *PrivateKey {
	res := make([]byte, 32)
	_, err := rand.Read(res)
	if err == nil {
		return &PrivateKey{Prikey: res}
	} else {
		return nil
	}
}

func (k *PrivateKey) Public() *PublicKey {
	res := k.Prikey
	return &PublicKey{Pubkey: res}
}

func (k *PrivateKey) Bytes() []byte {
	return k.Prikey
}
