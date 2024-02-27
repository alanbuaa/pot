package pot

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

func (w *Worker) handleBlock() {
	for {
		select {
		case block := <-w.peerMsgQueue:
			// TODO: stop accpeting the backup header from other nodes
			// w.stop
			epoch := w.getEpoch()
			header := block.GetHeader()

			if w.blockStorage.HasBlock(header.Hashes) {
				continue
			}

			flag, err := block.Header.BasicVerify()
			if !flag {
				w.log.Errorf("[PoT]\tepoch %d:Receive error block from node %d for %s", epoch, block.GetHeader().Address, err)
				continue
			}

			if header.Height < epoch {
				break
			} else if header.Height == epoch {
				if header.PeerId == w.self() {
					w.log.Infof("[PoT]\tepoch %d:Receive block from myself, Difficulty %d, with parent %s", epoch, header.Difficulty.Int64(), hex.EncodeToString(header.ParentHash))

					w.mutex.Lock()
					// w.backupBlock = append(w.backupBlock, header)
					//err := w.storage.Put(header)
					err := w.blockStorage.Put(block)
					w.mutex.Unlock()

					err = w.blockbroadcast(block)

					if err != nil {
						w.log.Errorf("[PoT]\tbroadcast header error:%s", err)
					}

				} else {
					w.log.Infof("[PoT]\tepoch %d:Receive block from node %d, Difficulty %d, with parent %s", epoch, header.Address, header.Difficulty.Int64(), hex.EncodeToString(header.ParentHash))
					if header.Difficulty.Cmp(common.Big0) == 0 {
						//w.storage.Put(header)
						w.blockStorage.Put(block)
						continue
					}

					go func() {
						err := w.handleCurrentBlock(block)
						if err != nil {
							w.log.Errorf("[PoT]\tepoch %d:handle current block err for %s", epoch, err)
						}
					}()

				}
			} else if header.Height > epoch {
				// vdf check
				w.log.Infof("[PoT]\tepoch %d:Receive a epoch %d block %s from node %d", epoch, header.Height, hexutil.Encode(header.Hashes), header.Address)
				go w.handleAdvancedBlock(epoch, block)
			}
		}
	}
}

func (w *Worker) handleCurrentBlock(block *types.Block) error {
	header := block.Header
	// _, _ = w.checkblock(header)
	flag, err := w.CheckVDF0Current(block.GetHeader())
	if !flag {
		return err
	}

	if !w.isBehindHeight(header.Height-1, block) {
		if header.ParentHash != nil {
			w.log.Errorf("[PoT]\tfind fork at epoch %d block %s with parents %s,current epoch %d %s", block.GetHeader().Height, hexutil.Encode(block.Hash()), hexutil.Encode(block.GetHeader().ParentHash), w.chainReader.GetCurrentHeight(), hexutil.Encode(w.chainReader.GetCurrentBlock().Hash()))

			currentblock := w.chainReader.GetCurrentBlock()
			ances, err := w.GetSharedAncestor(block, currentblock)

			if err != nil {
				w.log.Error(err)
				return err
			}
			c, err := w.chainReader.GetByHeight(ances.GetHeader().Height)
			if err != nil {
				w.log.Error(err)
				return err
			}

			w.log.Errorf("[PoT]\tthe shared ancestor of fork is %s at %d,match %t", hexutil.Encode(ances.GetHeader().Hashes), ances.GetHeader().Height, bytes.Equal(c.GetHeader().Hashes, ances.GetHeader().Hashes))
			//nowBranch, _, err := w.GetBranch(ances, currentblock)

			forkBranch, _, err := w.GetBranch(ances, block)
			if err != nil {
				w.log.Error(err)
				return err
			}

			flag, err := w.CheckVDF0ForBranch(forkBranch)
			if flag {
				w.log.Infof("[PoT]\tPass VDF Check")
			} else {
				return err
			}

			//for i := 0; i < len(nowBranch); i++ {
			//	w.log.Errorf("[PoT]\tthe nowBranch at height %d: %s", nowBranch[i].GetHeader().ExecHeight, hexutil.Encode(nowBranch[i].Hash()))
			//}
			//for i := 0; i < len(forkBranch); i++ {
			//	w.log.Errorf("[PoT]\tthe fork chain at height %d: %s", forkBranch[i].GetHeader().ExecHeight, hexutil.Encode(forkBranch[i].Hash()))
			//}

			w1 := w.calculateChainWeight(ances, currentblock)
			w2 := w.calculateChainWeight(ances, block)
			w.log.Errorf("[PoT]\tthe chain weight %d, the fork chain weight %d", w1.Int64(), w2.Int64())

			if w1.Int64() < w2.Int64() {
				err := w.chainreset(forkBranch)
				if err != nil {
					w.log.Errorf("[PoT]\tchain reset error for %s", err)
				}
			}
			w.mutex.Lock()
			// w.backupBlock = append(w.backupBlock, header)

			w.blockCounter += 1
			//_ = w.storage.Put(header)
			_ = w.blockStorage.Put(block)
			w.mutex.Unlock()

			//txs := block.GetExcutedTxs()
			//w.mempool.MarkProposed(txs)
		}
	} else {
		w.mutex.Lock()
		// w.backupBlock = append(w.backupBlock, header)

		w.blockCounter += 1
		//err := w.storage.Put(header)
		err := w.blockStorage.Put(block)
		w.mutex.Unlock()

		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (w *Worker) handleAdvancedBlock(epoch uint64, block *types.Block) error {
	current := w.chainReader.GetCurrentBlock()

	if epoch+100 < block.GetHeader().Height {
		err := w.setVDF0epoch(block.GetHeader().Height - 1)
		if err != nil {
			w.log.Warnf("[PoT]\tset vdf error for %s:", err)
			return err
		}
		res := &types.VDF0res{
			Res:   block.GetHeader().PoTProof[0],
			Epoch: block.GetHeader().Height - 1,
		}
		w.vdf0Chan <- res
		w.log.Infof("[PoT]\tepoch %d:set vdf complete. Start from epoch %d with res %s", epoch, block.GetHeader().Height-1, hexutil.Encode(crypto.Hash(res.Res)))
		return nil
	}

	ances, err := w.GetSharedAncestor(block, current)
	if err != nil {
		return err
	}
	w.mutex.Lock()
	// w.backupBlock = append(w.backupBlock, block)

	w.blockCounter += 1
	//err = w.storage.Put(block)
	err = w.blockStorage.Put(block)
	w.mutex.Unlock()

	w.log.Infof("[PoT]\tGet shared ancestor of block %s is %s at height %d", hexutil.Encode(block.Hash()), hexutil.Encode(ances.Hash()), ances.GetHeader().Height)

	branch, _, err := w.GetBranch(ances, block)
	flag, err := w.CheckVDF0ForBranch(branch)
	if flag {
		w.log.Infof("[PoT]\tPass VDF Check")
	}
	if err != nil {
		w.log.Errorf("[PoT]\tGet branch error for: %s", err)
		return err
	}

	//for i := 0; i < len(branch); i++ {
	//	w.log.Infof("[PoT]\tthe nowbranch at height %d: %s", branch[i].GetHeader().ExecHeight, hexutil.Encode(branch[i].Hash()))
	//}

	err = w.chainResetAdvanced(branch)
	if err != nil {
		w.log.Errorf("[PoT]\tchain reset error for %s", err)
		return err
	}
	finishflag := w.vdf0.Finished
	w.log.Infof("[PoT]\tMinew Work flag: %t", finishflag)

	if w.isMinerWorking() {
		close(w.abort)
		w.workFlag = false
		w.wg.Wait()
	}

	err = w.setVDF0epoch(block.GetHeader().Height - 1)
	if err != nil {
		w.log.Warnf("[PoT]\tset vdf error for %s:", err)
		return err
	}
	res := &types.VDF0res{
		Res:   block.GetHeader().PoTProof[0],
		Epoch: block.GetHeader().Height - 1,
	}
	w.vdf0Chan <- res
	w.log.Infof("[PoT]\tepoch %d:set vdf complete. Start from epoch %d with res %s", epoch, block.GetHeader().Height-1, hexutil.Encode(crypto.Hash(res.Res)))
	return nil
}

func (w *Worker) blockbroadcast(block *types.Block) error {

	pbHeader := block.ToProto()
	headerByte, err := proto.Marshal(pbHeader)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_Block_Data,
		MsgByte: headerByte,
	}
	messageByte, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	err = w.Engine.Broadcast(messageByte)
	if err != nil {
		return err
	}
	return nil
}

func (w *Worker) checkblock(block *types.Block) (bool, error) {

	flag, err := block.GetHeader().BasicVerify()
	if !flag {
		return false, err
	}
	return true, nil
}

func (w *Worker) CheckVDF0Current(header *types.Header) (bool, error) {
	vdfres, err := w.blockStorage.GetVDFresbyEpoch(header.Height)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(vdfres, header.PoTProof[0]) {
		return false, fmt.Errorf("the vdf0 proof is wrong ")
	} else {
		return true, nil
	}
}

func (w *Worker) CheckVDF0ForBranch(branch []*types.Block) (bool, error) {

	for i := len(branch) - 1; i >= 0; i-- {
		storageheight := w.blockStorage.GetVDFHeight()
		header := branch[i].GetHeader()
		if header.Height <= storageheight {
			vdfres, err := w.blockStorage.GetVDFresbyEpoch(header.Height)
			if err != nil {
				return false, err
			}
			if !bytes.Equal(vdfres, header.PoTProof[0]) {
				return false, fmt.Errorf("the vdf0 proof is wrong ")
			}
		} else if header.Height == storageheight+1 {
			vdfres, err := w.blockStorage.GetVDFresbyEpoch(storageheight)
			if err != nil {
				return false, err
			}
			vdfinput := crypto.Hash(vdfres)
			vdfoutput := header.PoTProof[0]

			//times := time.Now()
			if !w.vdfChecker.CheckVDF(vdfinput, vdfoutput) {
				return false, fmt.Errorf("the vdf0 proof is wrong ")
			}
			//w.log.Infof("[PoT]\tVDF Check need %d ms", time.Since(times)/time.Millisecond)
			w.blockStorage.SetVDFres(header.Height, header.PoTProof[0])
		}
	}
	return true, nil
}

func (w *Worker) checkHeaderVDF0(block *types.Block) (bool, error) {
	epoch := w.getEpoch()
	if block.GetHeader().Height == epoch+1 {
		vdfRes, err := w.GetVdf0byEpoch(epoch)
		if err != nil {
			return false, fmt.Errorf("get VDF for epoch %d error for: %s", epoch, err)
		}

		vdfInput := crypto.Hash(vdfRes)
		vdfOutput := block.GetHeader().PoTProof[0]

		if !w.vdfChecker.CheckVDF(vdfInput, vdfOutput) {
			return false, fmt.Errorf("the vdf0 proof of block is wrong")
		}

		return true, nil
	}
	return true, nil
}
