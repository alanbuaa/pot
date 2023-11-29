package pot

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/syndtr/goleveldb/leveldb"
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
	return total
	//for pointer := leaf; !bytes.Equal(pointer.Hashes, root.Hashes); pointer, _ = w.storage.Get(pointer.ParentHash) {
	//	if pointer == nil {
	//		return big.NewInt(0)
	//	}
	//	total = new(big.Int).Add(total, pointer.Difficulty)
	//}
	//total = new(big.Int).Add(total, root.Difficulty)
}

func (w *Worker) GetSharedAncestor(block *types.Header) (*types.Header, error) {
	current := w.chainreader.GetCurrentBlock()
	currentheight := w.chainreader.GetCurrentHeight()

	header := block

	if header.Height == currentheight {

		if bytes.Equal(current.Hashes, header.Hashes) {
			return current, nil
		}

		for {
			current, err := w.chainreader.GetByHeight(current.Height - 1)
			if err != nil {
				return nil, err
			}

			header, err := w.storage.Get(header.ParentHash)
			if err == leveldb.ErrNotFound {
				header, err = w.getParentBlock(header)
				if err != nil {
					return nil, err
				}
			} else if err != nil {
				return nil, err
			}

			if bytes.Equal(current.Hashes, header.Hashes) {
				return current, nil
			}
		}
	}

	if header.Height > currentheight {
		headerahead, err := w.storage.Get(header.ParentHash)
		if err == leveldb.ErrNotFound {
			headerahead, err = w.getParentBlock(header)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		return w.GetSharedAncestor(headerahead)
	}

	if header.Height < currentheight {
		current, err := w.chainreader.GetByHeight(header.Height)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(current.Hashes, header.Hashes) {
			return current, nil
		}

		for {
			current, err := w.chainreader.GetByHeight(current.Height - 1)
			if err != nil {
				return nil, err
			}
			header, err := w.storage.Get(header.ParentHash)
			if err == leveldb.ErrNotFound {
				header, err = w.getParentBlock(header)
				if err != nil {
					return nil, err
				}
			} else if err != nil {
				return nil, err
			}
			if bytes.Equal(current.Hashes, header.Hashes) {
				return current, nil
			}
		}
	}

	return nil, fmt.Errorf("get ancestor error for unknown end")
}

func (w *Worker) GetBranch(root, leaf *types.Header) ([]*types.Header, [][]*types.Header, error) {
	if root == nil || leaf == nil {
		return nil, nil, fmt.Errorf("branch is nil")
	}

	//rootheight := root.Height
	//leaftheight := leaf.Height

	//depth := leaftheight - rootheight
	mainbranch := make([]*types.Header, 1)
	ommerbranch := make([][]*types.Header, 1)

	//h := depth - 1
	mainbranch[0] = leaf
	ommerbranch[0] = make([]*types.Header, 0)
	//var err error

	for i := leaf; !bytes.Equal(i.ParentHash, root.Hashes); {
		//h -= 1
		parentBlock, err := w.getParentBlock(i)
		if err != nil {
			return nil, nil, err
		}
		mainbranch = append(mainbranch, parentBlock)
		//ommerblock := make([]*types.Header, 0)
		for k := 0; k < len(i.UncleHash); k++ {
			ommer, err := w.getUncleBlock(i)
			if err != nil {
				return nil, nil, err
			}
			ommerbranch = append(ommerbranch, ommer)
		}
		i = parentBlock
	}
	n := len(mainbranch) / 2
	for i := 0; i < len(mainbranch)/2; i++ {
		mainbranch[i], mainbranch[n-i-1] = mainbranch[n-i-1], mainbranch[i]
		ommerbranch[i], ommerbranch[n-i-1] = ommerbranch[n-i-1], ommerbranch[i]
	}
	return mainbranch, ommerbranch, nil
}

func (w *Worker) chainreset(branch []*types.Header) error {
	epoch := w.getEpoch()

	branchlen := len(branch)
	branchstr := ""
	for i := 0; i < branchlen; i++ {

		height := branch[i].Height
		if height != epoch {
			w.chainreader.SetHeight(height, branch[i])
			branchstr = branchstr + "\t" + hexutil.Encode(branch[i].Hash())
		} else {
			if i+1 != branchlen {
				w.chainreader.SetHeight(height, branch[i])
			}
		}
	}

	w.log.Infof("[PoT]\tthe chain has been reset by branch %s", branchstr)
	if w.isMinerWorking() {
		abort := w.abort
		if abort != nil {
			close(abort)
			w.wg.Wait()
			w.log.Infof("[PoT]\tthe work got abort for chain reset")
		} else {
			w.log.Warnf("[PoT]\twithout worker working")
		}
	}
	return nil
}
