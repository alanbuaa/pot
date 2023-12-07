package pot

import (
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"math/big"
	"strconv"
	time2 "time"
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

			header, err := w.getParentBlock(header)
			if err != nil {
				return nil, err
			}

			if bytes.Equal(current.Hashes, header.Hashes) {
				return current, nil
			}
		}
	}

	if header.Height > currentheight {
		//headerahead, err := w.storage.Get(header.ParentHash)
		//if err == leveldb.ErrNotFound {
		//	headerahead, err = w.getParentBlock(header)
		//	if err != nil {
		//		return nil, err
		//	}
		//} else if err != nil {
		//	return nil, err
		//}
		//return w.GetSharedAncestor(headerahead)
		for true {
			headerahead, err := w.getParentBlock(header)
			if err != nil {
				return nil, err
			}

			if headerahead.Height == currentheight {
				if bytes.Equal(current.Hashes, headerahead.Hashes) {
					return current, nil
				}

				for {
					current, err := w.chainreader.GetByHeight(current.Height - 1)
					if err != nil {
						return nil, err
					}

					headerahead, err = w.getParentBlock(headerahead)
					if err != nil {
						return nil, err
					}

					if bytes.Equal(current.Hashes, headerahead.Hashes) {
						return current, nil
					}
				}
			} else {
				header = headerahead
			}

		}
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

	return mainbranch, ommerbranch, nil
}

func (w *Worker) chainResetAdvanced(branch []*types.Header) error {
	epoch := w.getEpoch()
	branchlen := len(branch)
	branchstr := ""

	for i := branchlen - 1; i > 0; i-- {

		height := branch[i].Height

		w.chainreader.SetHeight(height, branch[i])
		branchstr = branchstr + "\t" + strconv.Itoa(int(height))
	}
	w.log.Infof("[PoT]\tepoch %d: the chain has been reset by branch %s", epoch, branchstr)

	return nil
}

func (w *Worker) chainreset(branch []*types.Header) error {
	epoch := w.getEpoch()

	branchlen := len(branch)
	branchstr := ""
	for i := branchlen - 1; i >= 0; i-- {

		height := branch[i].Height
		if height != epoch {
			w.chainreader.SetHeight(height, branch[i])
			branchstr = branchstr + "\t" + strconv.Itoa(int(height))
		}
	}

	w.log.Infof("[PoT]\tepoch %d: the chain has been reset by branch %s", epoch, branchstr)
	flag := w.isMinerWorking()
	w.log.Infof("[PoT]\tflag: %t", flag)
	time := time2.Now()
	if flag {
		close(w.abort)
		w.wg.Wait()
		w.log.Infof("[PoT]\tthe vdf1 work got abort for chain reset, need %d ms", time2.Since(time)/time2.Millisecond)
		w.workflag = false
	}

	return nil
}

func (w *Worker) isBehindHeight(height uint64, block *types.Header) bool {
	b, err := w.chainreader.GetByHeight(height)
	if err != nil {
		return false
	}
	if block.Height != b.Height+1 {
		return false
	}
	if !bytes.Equal(block.ParentHash, b.Hashes) {
		return false
	}
	return true
}
