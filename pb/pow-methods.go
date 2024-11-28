package pb

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
)

func (b *PoWBlock) Identifier(h string) string {
	return hex.EncodeToString(b.Hash(h))[:10]
}

func (b *PoWBlock) Hash(h string) []byte {
	if b.BlockHash != nil {
		return b.BlockHash
	}
	return b.ForceHash(h)
}

func (b *PoWBlock) ForceHash(h string) []byte {
	hasher, _ := crypto.HashFactory(h)
	hasher.Write(b.ParentHash)
	hasher.Write(binary.BigEndian.AppendUint64(nil, b.Height))
	hasher.Write(bytes.Join(b.Txs, []byte{}))
	hasher.Write(binary.BigEndian.AppendUint64(nil, b.Nonce))
	b.BlockHash = hasher.Sum(nil)
	return b.BlockHash
}

func (b *PoWBlock) GetDifficulty(h string) *big.Int {
	return new(big.Int).SetBytes(b.Hash(h))
}
