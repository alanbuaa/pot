package pot

import (
	"bytes"
	"math/big"

	"github.com/zzz136454872/upgradeable-consensus/types"
)

func (w *Worker) calculateChainWeight(root, leaf *types.Header) *big.Int {
	total := big.NewInt(0)

	for pointer := leaf; !bytes.Equal(pointer.Hashes, root.Hashes); pointer, _ = w.storage.Get(pointer.ParentHash) {
		total = new(big.Int).Add(total, pointer.Difficulty)

	}

	return total
}
