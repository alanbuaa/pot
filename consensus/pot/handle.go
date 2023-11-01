package pot

import (
	"encoding/hex"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
	"math/big"
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
					w.log.Infof("[PoT]\tepoch %d:Receive block from myself, Difficulty %d", epoch, header.Difficulty.Int64())

					w.synclock.Lock()
					//w.backupBlock = append(w.backupBlock, header)
					err := w.storage.Put(header)
					w.synclock.Unlock()

					err = w.headerbroadcast(header)

					if err != nil {
						w.log.Errorf("[PoT]\tbroadcast header error", err)
					}
				} else {
					w.log.Infof("[PoT]\tepoch %d:Receive block from node %d", epoch, header.Address)
					// header check
					//vdfin := crypto.Hash(w.getVDF0lastepoch(epoch))

					//vdfout := header.PoTProof[0]
					//tsp := time.Now()
					//if !w.vdfchecker.CheckVDF(vdfin, vdfout) {
					//	w.log.Errorf("[PoT]\tthe header from %d is a bad header", header.Address)
					//	continue
					//} else {
					//	w.log.Infof("[PoT]\tepoch %d:the header from %d pass check", epoch, header.Address)
					//	w.log.Infof("[PoT]\tthe header check need %d ms", time.Since(tsp)/time.Millisecond)
					//}
					// pass check, add to back up header

					w.synclock.Lock()
					//w.backupBlock = append(w.backupBlock, header)
					w.blockcounter += 1
					err := w.storage.Put(header)
					w.synclock.Unlock()

					if err != nil {
						w.log.Errorf("[PoT]\tstore header error", err)
					}
				}
			} else if header.Height > epoch {
				// vdf check
				w.log.Infof("[PoT]\tepoch %d: Receive a epoch %d block from node %d", epoch, header.Height, header.Address)
				go w.handleAdvancedBlock(epoch, header)

				//w.vdf0.Abort()

				// catch up

				// header check

			}
		}
	}
}

func (w *Worker) handleAdvancedBlock(epoch uint64, header *types.Header) {

	_ = w.handleAdvancedHeaderVDF(epoch, header)
	if header.Height == epoch+1 {
		vdfres, err := w.GetVdf0byEpoch(epoch)
		if err != nil {
			w.log.Error(err)
		}
		vdfin := crypto.Hash(vdfres)
		vdfout := header.PoTProof[0]
		//times := time.Now()
		if w.vdfchecker.CheckVDF(vdfin, vdfout) {
			//w.log.Infof("[PoT]\tVDF check need %d ms", time.Since(times)/time.Millisecond)
			w.storage.Put(header)
			//epochnow := w.getEpoch()
			err := w.setVDF0epoch(header.Height - 1)
			if err != nil {
				w.log.Warnf("[PoT]\tset vdf error for %s:", err)
				return
			}
			w.log.Errorf("[PoT]\tAlready set VDF ,start from epoch %d", header.Height-1)
			res := &types.VDF0res{
				Res:   vdfout,
				Epoch: header.Height - 1,
			}
			w.vdf0Chan <- res
			return
		}
	} else if header.Height > epoch+1 {
		w.log.Errorf("[PoT]\tepoch %d: Receive a epoch %d block from node %d", epoch, header.Height, header.Address)
		//w.log.Warn("[PoT]\terror get too high block")

		if header.ParentHash == nil && header.Difficulty.Cmp(big.NewInt(NoParentD)) == 0 {
			w.log.Infof("[PoT]\tthe block is a higher block with no parentblock")
			w.storage.Put(header)
			potres := header.PoTProof[0]
			err := w.setVDF0epoch(header.Height - 1)
			err = w.storage.SetVdfRes(header.Height, potres)
			if err != nil {
				w.log.Errorf("[PoT]\tset vdf error for %s:", err)
				return
			}
			w.log.Errorf("[PoT]\tAlready set VDF ,start from epoch %d", header.Height-1)
			res := &types.VDF0res{
				Res:   potres,
				Epoch: header.Height - 1,
			}
			w.vdf0Chan <- res
			return
		} else {
			parent, err := w.getParentBlock(header)
			if err != nil {
				w.log.Error("[PoT]\tGet block error for ", err)
				return
			}
			if parent != nil {
				w.log.Errorf("[PoT]\tGet header at height %d parent hash is %s", header.Height, hex.EncodeToString(parent.Hash()))
				w.handleAdvancedBlock(epoch, parent)
				w.storage.Put(header)
				potres := header.PoTProof[0]
				err := w.setVDF0epoch(header.Height - 1)
				if err != nil {
					w.log.Errorf("[PoT]\tset vdf error for %s:", err)
					return
				}

				res := &types.VDF0res{
					Res:   potres,
					Epoch: header.Height - 1,
				}
				w.vdf0Chan <- res
				w.log.Errorf("[PoT]\tAlready set VDF ,start from epoch %d", header.Height-1)
				return
			}
		}
	} else {
		w.log.Errorf("[PoT]\terror to handle block")
		return
	}
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
