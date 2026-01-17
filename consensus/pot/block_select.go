// Package pot implements the Proof of Time (PoT) consensus algorithm worker.
// This file contains block selection and validation logic.
package pot

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// CheckParentBlockEnough validates if there are sufficient parent blocks for the given height
func (w *Worker) CheckParentBlockEnough(height uint64) (bool, error) {
	if height == 0 {
		return true, nil
	}

	backupblock, err := w.blockStorage.GetbyHeight(height)

	if err != nil {
		return false, err
	}

	if len(backupblock) == 0 {
		return false, nil
	}

	current, err := w.chainReader.GetByHeight(height - 1)
	if err != nil {
		return false, err
	}
	/* */
	count := int64(0)
	for _, block := range backupblock {
		blockheader := block.GetHeader()
		if bytes.Equal(current.GetHeader().Hashes, blockheader.ParentHash) {
			count += 1
		}
	}
	if count*2 < w.config.PoT.Snum {
		return false, fmt.Errorf("not enough parent block for height %d", height)
	}

	return true, nil
}

// blockSelection selects the parent block and uncle blocks from a list of candidate blocks
func (w *Worker) blockSelection(blocks []*types.Block, vdf0res []byte, height uint64) (parent *types.Block, uncle []*types.Block) {
	if height == 0 {
		return types.DefaultGenesisBlock(), nil
	}
	sr := crypto.Hash(vdf0res)
	maxweight := big.NewInt(0)
	max := -1

	current, err := w.chainReader.GetByHeight(height - 1)

	if err != nil {
		return nil, nil
	}

	if len(blocks) == 0 {
		return nil, nil
	}

	readyblocks := make([]*types.Block, 0)
	for _, block := range blocks {
		blockheader := block.GetHeader()
		if (bytes.Equal(current.GetHeader().Hashes, blockheader.ParentHash) && block.GetHeader().Difficulty.Cmp(common.Big0) != 0) || blockheader.ParentHash == nil {

			if len(block.Header.PoTProof) >= 2 {
				vdf1res := blockheader.PoTProof[1]

				readyblocks = append(readyblocks, block)
				hashinput := append(vdf1res, sr...)

				tmp := new(big.Int).Mul(bigD, blockheader.Difficulty)
				weight := new(big.Int).Div(tmp, new(big.Int).SetBytes(crypto.Hash(hashinput)))
				if weight.Cmp(maxweight) > 0 {
					max = len(readyblocks)
					maxweight.Set(weight)
				}
			}
		}
	}
	if max == -1 {
		return nil, nil
	}

	parent = readyblocks[max-1]
	uncle = append(readyblocks[:max-1], readyblocks[max:]...)
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.chainReader.SetHeight(parent.GetHeader().Height, parent)

	return parent, uncle
}
