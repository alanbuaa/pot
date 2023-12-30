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

					w.synclock.Lock()
					//w.backupBlock = append(w.backupBlock, header)
					err := w.storage.Put(header)
					w.synclock.Unlock()

					err = w.headerbroadcast(header)

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
	//_, _ = w.checkHeader(header)
	if !w.isBehindHeight(block.Height-1, block) {
		if block.ParentHash != nil {
			w.log.Errorf("[PoT]\tfind fork at epoch %d block %s with parents %s,current epoch %d parent %s", block.Height, hexutil.Encode(block.Hashes), hexutil.Encode(block.ParentHash), w.chainreader.GetCurrentHeight(), hexutil.Encode(w.chainreader.GetCurrentBlock().Hashes))
			b, err := w.GetSharedAncestor(block)
			if err != nil {
				w.log.Error(err)
				return err
			}
			c, err := w.chainreader.GetByHeight(b.Height)
			if err != nil {
				w.log.Error(err)
				return err
			}
			w.log.Errorf("[PoT]\tthe shared ancestor of fork is %s at %d,match %t", hexutil.Encode(b.Hashes), b.Height, bytes.Equal(c.Hashes, b.Hashes))
			nowbranch, _, err := w.GetBranch(b, w.chainreader.GetCurrentBlock())
			forkbranch, _, err := w.GetBranch(b, block)
			if err != nil {
				w.log.Error(err)
				return err
			}
			for i := 0; i < len(nowbranch); i++ {
				w.log.Errorf("[PoT]\tthe nowbranch at height %d: %s", nowbranch[i].Height, hexutil.Encode(nowbranch[i].Hashes))
			}
			for i := 0; i < len(forkbranch); i++ {
				w.log.Errorf("[PoT]\tthe fork chain at height %d: %s", forkbranch[i].Height, hexutil.Encode(forkbranch[i].Hashes))
			}

			w1 := w.calculateChainWeight(b, w.chainreader.GetCurrentBlock())
			w2 := w.calculateChainWeight(b, block)
			w.log.Errorf("[PoT]\tthe chain weight %d, the fork chain weight %d", w1.Int64(), w2.Int64())
			if w1.Int64() < w2.Int64() {
				err := w.chainreset(forkbranch)
				if err != nil {
					w.log.Errorf("[PoT]\tchain reset error for %s", err)
				}
			}
			w.synclock.Lock()
			//w.backupBlock = append(w.backupBlock, header)

			w.blockcounter += 1
			_ = w.storage.Put(header)
			w.synclock.Unlock()

		}
	} else {
		w.synclock.Lock()
		//w.backupBlock = append(w.backupBlock, header)

		w.blockcounter += 1
		err := w.storage.Put(header)
		w.synclock.Unlock()

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
	w.synclock.Lock()
	//w.backupBlock = append(w.backupBlock, header)

	w.blockcounter += 1
	err = w.storage.Put(header)
	w.synclock.Unlock()

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
		w.workflag = false
		close(w.abort)
		w.wg.Wait()
	}

	//if !w.vdf0.IsFinished() {
	//	err := w.vdf0.Abort()
	//	if err != nil {
	//		return
	//	}
	//	w.log.Infof("[PoT]\tepoch %d:VDF0 got abort for advanced chain reset ", epoch)
	//}

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
		//now the parent is at the same height
		headerparent, err := w.getParentBlock(header)
		if err != nil {
			return false, err
		}
		if headerparent.Height == epoch {
			err := w.handleCurrentBlock(headerparent)
			if err != nil {
				return false, err
			}
		}
		flag, err := w.checkHeaderVDF0(header)
		if err != nil && !flag {
			return false, err
		}
		//w.log.Infof("")
	}
	return true, nil
}

func (w *Worker) checkHeaderVDF0(header *types.Header) (bool, error) {
	epoch := w.getEpoch()
	if header.Height == epoch+1 {
		vdfres, err := w.GetVdf0byEpoch(epoch)
		if err != nil {
			return false, fmt.Errorf("get VDF for epoch %d error for: %s", epoch, err)
		}

		vdfin := crypto.Hash(vdfres)
		vdfout := header.PoTProof[0]

		if !w.vdfchecker.CheckVDF(vdfin, vdfout) {
			return false, fmt.Errorf("the vdf0 proof of block is wrong")
		}

		return true, nil
	}
	return true, nil
}

func (w *Worker) handleAdvancedHeaderVDF(epoch uint64, header *types.Header) bool {
	epoch0, err := w.storage.GetPoTbyEpoch(epoch)
	if epoch+1 == header.Height {
		//we don't have next epoch vdfres, but we have now vdfres
		if err == nil && len(epoch0) != 0 {
			return w.vdfchecker.CheckVDF(epoch0, header.PoTProof[0])
		}
	} else if header.Height > epoch+1 {
		res, err := w.requestPoTResFor(epoch+1, header.Address, header.PeerId)
		if err != nil {
			w.log.Error(err)
			return false
		}
		if w.vdfchecker.CheckVDF(epoch0, res) {
			w.SetVdf0res(epoch+1, res)
			return w.handleAdvancedHeaderVDF(epoch+1, header)
		}
	}
	return false
}

func (w *Worker) headerbroadcast(header *types.Header) error {
	pbheader := header.ToProto()
	headerbyte, err := proto.Marshal(pbheader)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_Header_Data,
		MsgByte: headerbyte,
	}
	messagebyte, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	err = w.Engine.Broadcast(messagebyte)
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
