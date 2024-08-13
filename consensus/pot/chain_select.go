package pot

import (
	"bytes"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"math/big"
	"strconv"
	time2 "time"
)

func (w *Worker) calculateChainWeight(root, leaf *types.Block) *big.Int {
	total := big.NewInt(0)
	if root == nil || leaf == nil {
		return big.NewInt(0)
	}
	pointer := leaf
	for {
		total = new(big.Int).Add(total, pointer.GetHeader().Difficulty)

		if bytes.Equal(pointer.GetHeader().Hashes, root.GetHeader().Hashes) {
			break
		}
		for i := 0; i < len(pointer.GetHeader().UncleHash); i++ {
			hashes := pointer.GetHeader().UncleHash[i]
			ommer, _ := w.blockStorage.Get(hashes)
			if ommer != nil {
				total = new(big.Int).Add(total, pointer.GetHeader().Difficulty)
			}
		}
		if pointer.GetHeader().ParentHash != nil {
			pointer, _ = w.blockStorage.Get(pointer.GetHeader().ParentHash)
		} else {
			return big.NewInt(0)
		}
	}
	return total
	// for pointer := leaf; !bytes.Equal(pointer.Hashes, root.Hashes); pointer, _ = w.storage.Get(pointer.ParentHash) {
	//	if pointer == nil {
	//		return big.NewInt(0)
	//	}
	//	total = new(big.Int).Add(total, pointer.Difficulty)
	// }
	// total = new(big.Int).Add(total, root.Difficulty)
}

func (w *Worker) GetSharedAncestor(forkblock *types.Block, currentblock *types.Block) (*types.Block, error) {

	current := currentblock
	currentheight := currentblock.GetHeader().Height

	fork := forkblock
	header := fork.GetHeader()

	if fork.GetHeader().Height == currentheight {

		if bytes.Equal(current.Hash(), fork.Hash()) {
			return current, nil
		}

		for {
			current, err := w.chainReader.GetByHeight(current.GetHeader().Height - 1)
			if err != nil {
				return nil, err
			}

			if fork.GetHeader().Height == 0 {
				return types.DefaultGenesisBlock(), nil
			}

			fork, err := w.getParentBlock(fork)

			if err != nil {
				return nil, err
			}

			if bytes.Equal(current.Hash(), fork.Hash()) {
				return current, nil
			}
		}
	}

	if header.Height > currentheight {

		for true {
			forkahead, err := w.getParentBlock(fork)
			if err != nil {
				return nil, err
			}

			if forkahead.GetHeader().Height == currentheight {
				if bytes.Equal(current.Hash(), forkahead.Hash()) {
					return current, nil
				}

				for {
					if forkahead.GetHeader().Height == 0 {
						return types.DefaultGenesisBlock(), nil
					}

					headeraheadparent, err := w.getParentBlock(forkahead)
					if err != nil {
						return nil, err
					}

					current, err := w.chainReader.GetByHeight(headeraheadparent.GetHeader().Height)
					if err != nil {
						return nil, err
					}

					if bytes.Equal(current.Hash(), headeraheadparent.Hash()) {
						return current, nil
					}

					forkahead = headeraheadparent
				}
			} else {
				fork = forkahead
			}

		}
	}

	if header.Height < currentheight {
		current, err := w.chainReader.GetByHeight(header.Height)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(current.Hash(), fork.Hash()) {
			return current, nil
		}

		for {
			current, err := w.chainReader.GetByHeight(current.GetHeader().Height - 1)
			if err != nil {
				return nil, err
			}
			//header, err := w.storage.Get(header.ParentHash)
			fork, err := w.getParentBlock(fork)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(current.Hash(), fork.Hash()) {
				return current, nil
			}
		}
	}

	return nil, fmt.Errorf("get ancestor error for unknown end")
}

func (w *Worker) GetBranch(root, leaf *types.Block) ([]*types.Block, [][]*types.Block, error) {
	if root == nil || leaf == nil {
		return nil, nil, fmt.Errorf("branch is nil")
	}

	// rootheight := root.ExecHeight
	// leaftheight := leaf.ExecHeight

	// depth := leaftheight - rootheight
	mainbranch := make([]*types.Block, 1)
	ommerbranch := make([][]*types.Block, 1)

	// h := depth - 1
	mainbranch[0] = leaf
	ommerbranch[0] = make([]*types.Block, 0)
	// var err error

	for i := leaf; !bytes.Equal(i.GetHeader().ParentHash, root.GetHeader().Hashes); {
		// h -= 1
		parentBlock, err := w.getParentBlock(i)
		if err != nil {
			return nil, nil, err
		}
		mainbranch = append(mainbranch, parentBlock)
		// ommerblock := make([]*types.Header, 0)
		for k := 0; k < len(i.GetHeader().UncleHash); k++ {
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

func (w *Worker) chainResetAdvanced(branch []*types.Block) error {
	epoch := w.getEpoch()
	branchlen := len(branch)
	branchstr := ""

	for i := branchlen - 1; i > 0; i-- {

		height := branch[i].GetHeader().Height

		w.chainReader.SetHeight(height, branch[i])
		branchstr = branchstr + "\t" + strconv.Itoa(int(height))
	}
	w.log.Infof("[PoT]\tepoch %d: the chain has been reset by branch %s", epoch, branchstr)

	return nil
}

func (w *Worker) chainreset(branch []*types.Block) error {
	epoch := w.getEpoch()

	branchlen := len(branch)
	branchstr := ""
	for i := branchlen - 1; i >= 0; i-- {

		blocks := branch[i]
		height := blocks.GetHeader().Height

		if height != epoch {
			originblocks, err := w.chainReader.GetByHeight(height)
			if err != nil {
				return fmt.Errorf("get chain block err for %s", err)
			}
			originHeaders := originblocks.GetExecutedHeaders()
			w.mempool.UnMarkByHeader(originHeaders)
			w.chainReader.SetHeight(height, blocks)
			txs := blocks.GetExecutedHeaders()
			w.mempool.MarkProposedByHeader(txs)
			//w.blockStorage.SetVDFres(height, branch[i].GetHeader().PoTProof[0])
			branchstr = branchstr + "\t" + strconv.Itoa(int(height))
		}

	}

	w.log.Infof("[PoT]\tepoch %d: the chain has been reset by branch %s", epoch, branchstr)
	flag := w.IsVDF1Working()
	//w.log.Infof("[PoT]\tflag: %t", flag)
	time := time2.Now()
	if flag {
		w.setWorkFlagFalse()
		w.abort.once.Do(func() {
			close(w.abort.abortchannel)
		})
		w.wg.Wait()
		w.log.Infof("[PoT]\tthe vdf1 work got abort for chain reset, need %d ms", time2.Since(time)/time2.Millisecond)
	}

	return nil
}

func (w *Worker) isBehindHeight(height uint64, block *types.Block) bool {
	b, err := w.chainReader.GetByHeight(height)
	if err != nil {
		return false
	}
	if block.GetHeader().Height != b.GetHeader().Height+1 {
		return false
	}
	if !bytes.Equal(block.GetHeader().ParentHash, b.Hash()) {
		return false
	}
	return true
}
