package pot

import (
	"bytes"
	"math/big"

	"github.com/zzz136454872/upgradeable-consensus/types"
)

func (w *Worker) calculateChainWeight(root, leaf *types.Header) *big.Int {
	total := big.NewInt(0)
	if root == nil || leaf == nil {
		return big.NewInt(0)
	}
	pointer := leaf
	for {
		total = new(big.Int).Add(total, pointer.Difficulty)

		if bytes.Equal(pointer.Hashes, root.Hashes) {
			break
		}
		for i := 0; i < len(pointer.UncleHash); i++ {
			hashes := pointer.UncleHash[i]
			ommer, _ := w.storage.Get(hashes)
			if ommer != nil {
				total = new(big.Int).Add(total, pointer.Difficulty)
			}
		}
		if pointer.ParentHash != nil {
			pointer, _ = w.storage.Get(pointer.ParentHash)
		} else {
			return big.NewInt(0)
		}
	}

	//for pointer := leaf; !bytes.Equal(pointer.Hashes, root.Hashes); pointer, _ = w.storage.Get(pointer.ParentHash) {
	//	if pointer == nil {
	//		return big.NewInt(0)
	//	}
	//	total = new(big.Int).Add(total, pointer.Difficulty)
	//}
	//total = new(big.Int).Add(total, root.Difficulty)
	return total
}
