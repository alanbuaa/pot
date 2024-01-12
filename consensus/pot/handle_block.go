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
		case header := <-w.peerMsgQueue:
			// TODO: stop accpeting the backup header from other nodes
			// w.stop
			epoch := w.getEpoch()
			if w.storage.HasBlock(header.Hashes) {
				continue
			}
			if header.Height < epoch {
				break
			} else if header.Height == epoch {
				if header.Address == w.self() {
					w.log.Infof("[PoT]\tepoch %d:Receive block from myself, Difficulty %d, with parent %s", epoch, header.Difficulty.Int64(), hex.EncodeToString(header.ParentHash))

					w.mutex.Lock()
					// w.backupBlock = append(w.backupBlock, header)
					err := w.storage.Put(header)
					w.mutex.Unlock()

					err = w.headerBroadcast(header)

					if err != nil {
						w.log.Errorf("[PoT]\tbroadcast header error:%s", err)
					}

				} else {
					w.log.Infof("[PoT]\tepoch %d:Receive block from node %d, Difficulty %d, with parent %s", epoch, header.Address, header.Difficulty.Int64(), hex.EncodeToString(header.ParentHash))
					if header.Difficulty.Cmp(common.Big0) == 0 {
						w.storage.Put(header)
						continue
					}
					go func() {
						err := w.handleCurrentBlock(header)
						if err != nil {
							w.log.Errorf("[PoT]\tepoch %d:handle current block err for %s", epoch, err)
							for true {

								break
							}
						}
					}()

				}
			} else if header.Height > epoch {
				// vdf check
				w.log.Infof("[PoT]\tepoch %d:Receive a epoch %d block %s from node %d", epoch, header.Height, hexutil.Encode(header.Hashes), header.Address)
				go w.handleAdvancedBlock(epoch, header)
			}
		}
	}
}

func (w *Worker) handleCurrentBlock(block *types.Header) error {
	header := block
	// _, _ = w.checkHeader(header)
	if !w.isBehindHeight(block.Height-1, block) {
		if block.ParentHash != nil {
			w.log.Errorf("[PoT]\tfind fork at epoch %d block %s with parents %s,current epoch %d parent %s", block.Height, hexutil.Encode(block.Hashes), hexutil.Encode(block.ParentHash), w.chainReader.GetCurrentHeight(), hexutil.Encode(w.chainReader.GetCurrentBlock().Hashes))
			b, err := w.GetSharedAncestor(block)
			if err != nil {
				w.log.Error(err)
				return err
			}
			c, err := w.chainReader.GetByHeight(b.Height)
			if err != nil {
				w.log.Error(err)
				return err
			}
			w.log.Errorf("[PoT]\tthe shared ancestor of fork is %s at %d,match %t", hexutil.Encode(b.Hashes), b.Height, bytes.Equal(c.Hashes, b.Hashes))
			nowBranch, _, err := w.GetBranch(b, w.chainReader.GetCurrentBlock())
			forkBranch, _, err := w.GetBranch(b, block)
			if err != nil {
				w.log.Error(err)
				return err
			}
			for i := 0; i < len(nowBranch); i++ {
				w.log.Errorf("[PoT]\tthe nowBranch at height %d: %s", nowBranch[i].Height, hexutil.Encode(nowBranch[i].Hashes))
			}
			for i := 0; i < len(forkBranch); i++ {
				w.log.Errorf("[PoT]\tthe fork chain at height %d: %s", forkBranch[i].Height, hexutil.Encode(forkBranch[i].Hashes))
			}

			w1 := w.calculateChainWeight(b, w.chainReader.GetCurrentBlock())
			w2 := w.calculateChainWeight(b, block)
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
			_ = w.storage.Put(header)
			w.mutex.Unlock()

		}
	} else {
		w.mutex.Lock()
		// w.backupBlock = append(w.backupBlock, header)

		w.blockCounter += 1
		err := w.storage.Put(header)
		w.mutex.Unlock()

		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (w *Worker) handleAdvancedBlock(epoch uint64, header *types.Header) {

	ances, err := w.GetSharedAncestor(header)
	if err != nil {
		return
	}
	w.mutex.Lock()
	// w.backupBlock = append(w.backupBlock, header)

	w.blockCounter += 1
	err = w.storage.Put(header)
	w.mutex.Unlock()

	w.log.Infof("[PoT]\tGet shared ancestor of block %s is %s at height %d", hexutil.Encode(header.Hashes), hexutil.Encode(ances.Hashes), ances.Height)

	branch, _, err := w.GetBranch(ances, header)

	if err != nil {
		w.log.Errorf("[PoT]\tGet branch error for: %s", err)
		return
	}

	for i := 0; i < len(branch); i++ {
		w.log.Errorf("[PoT]\tthe nowbranch at height %d: %s", branch[i].Height, hexutil.Encode(branch[i].Hashes))
	}

	err = w.chainResetAdvanced(branch)
	if err != nil {
		w.log.Errorf("[PoT]\tchain reset error for %s", err)
	}
	flag := w.vdf0.Finished
	w.log.Infof("[PoT]\tMinew Work flag: %t", flag)

	if w.isMinerWorking() {
		w.workFlag = false
		close(w.abort)
		w.wg.Wait()
	}

	// if !w.vdf0.IsFinished() {
	//	err := w.vdf0.Abort()
	//	if err != nil {
	//		return
	//	}
	//	w.log.Infof("[PoT]\tepoch %d:VDF0 got abort for advanced chain reset ", epoch)
	// }

	err = w.setVDF0epoch(header.Height - 1)
	if err != nil {
		w.log.Warnf("[PoT]\tset vdf error for %s:", err)
		return
	}
	res := &types.VDF0res{
		Res:   header.PoTProof[0],
		Epoch: header.Height - 1,
	}
	w.vdf0Chan <- res
	w.log.Infof("[PoT]\tepoch %d:set vdf complete. Start from epoch %d with res %s", epoch, header.Height-1, hexutil.Encode(crypto.Hash(res.Res)))
	return
}

func (w *Worker) checkAdvancedBlock(epoch uint64, header *types.Header) (bool, error) {
	if header.Height == epoch+1 {
		// now the parent is at the same height
		headerParent, err := w.getParentBlock(header)
		if err != nil {
			return false, err
		}
		if headerParent.Height == epoch {
			err := w.handleCurrentBlock(headerParent)
			if err != nil {
				return false, err
			}
		}
		flag, err := w.checkHeaderVDF0(header)
		if err != nil && !flag {
			return false, err
		}
		// w.log.Infof("")
	}
	return true, nil
}

func (w *Worker) checkHeaderVDF0(header *types.Header) (bool, error) {
	epoch := w.getEpoch()
	if header.Height == epoch+1 {
		vdfRes, err := w.GetVdf0byEpoch(epoch)
		if err != nil {
			return false, fmt.Errorf("get VDF for epoch %d error for: %s", epoch, err)
		}

		vdfInput := crypto.Hash(vdfRes)
		vdfOutput := header.PoTProof[0]

		if !w.vdfChecker.CheckVDF(vdfInput, vdfOutput) {
			return false, fmt.Errorf("the vdf0 proof of block is wrong")
		}

		return true, nil
	}
	return true, nil
}

func (w *Worker) handleAdvancedHeaderVDF(epoch uint64, header *types.Header) bool {
	epoch0, err := w.storage.GetPoTbyEpoch(epoch)
	if epoch+1 == header.Height {
		// we don't have next epoch vdfRes, but we have now vdfRes
		if err == nil && len(epoch0) != 0 {
			return w.vdfChecker.CheckVDF(epoch0, header.PoTProof[0])
		}
	} else if header.Height > epoch+1 {
		res, err := w.requestPoTResFor(epoch+1, header.Address, header.PeerId)
		if err != nil {
			w.log.Error(err)
			return false
		}
		if w.vdfChecker.CheckVDF(epoch0, res) {
			w.SetVdf0res(epoch+1, res)
			return w.handleAdvancedHeaderVDF(epoch+1, header)
		}
	}
	return false
}

func (w *Worker) headerBroadcast(header *types.Header) error {
	pbHeader := header.ToProto()
	headerByte, err := proto.Marshal(pbHeader)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_Header_Data,
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

func (w *Worker) checkHeader(header *types.Header) (bool, error) {
	if header.ParentHash == nil {
		return true, nil
	}

	parent, err := w.getParentBlock(header)

	if err != nil {
		return false, err
	}
	if parent.Hashes != nil {
		return true, nil
	}
	return true, nil
}
