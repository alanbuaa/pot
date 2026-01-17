package pot

import (
	"blockchain-crypto/vdf/wesolowski_rust"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
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

			if w.blockStorage.HasBlock(header.Hash()) {
				continue
			}

			flag, err := block.Header.BasicVerify()

			if !flag {
				w.log.WithError(err).WithFields(logrus.Fields{
					"epoch":     epoch,
					"from_node": block.GetHeader().Address,
				}).Warn("Received invalid block")
				continue
			}

			if header.Height < epoch {
				break
			} else if header.Height == epoch {
				if header.PeerId == w.self() {

					w.log.WithFields(logrus.Fields{
						"epoch":      epoch,
						"difficulty": header.Difficulty.Int64(),
						"parent":     hex.EncodeToString(header.ParentHash),
					}).Debug("Received block from myself")

					w.mutex.Lock()
					// w.backupBlock = append(w.backupBlock, header)
					//err := w.storage.Put(header)
					b, err := w.blockStorage.PutByte(block)
					w.mutex.Unlock()
					if err != nil {
						w.log.WithError(err).Error("Failed to store block")
					}

					err = w.blockbytebroadcast(b)

					if err != nil {
						w.log.WithError(err).Error("Failed to broadcast block")
					}

				} else {
					w.log.WithFields(logrus.Fields{
						"epoch":            epoch,
						"from_node":        header.Address,
						"difficulty":       header.Difficulty.Int64(),
						"parent":           hex.EncodeToString(header.ParentHash),
						"transfer_time_ms": float64(time.Since(header.Timestamp)) / float64(time.Millisecond),
					}).Trace("Received block from peer %d", header.PeerId)
					//if w.ID == 0 {
					//	transfertime := float64(time.Since(header.Timestamp)) / float64(time.Millisecond)
					//	fill, err := os.OpenFile("transfertime", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
					//	if err != nil {
					//		fmt.Println(err)
					//	}
					//	_, err = fill.WriteString(fmt.Sprintf("%f\Commitees", transfertime))
					//	if err != nil {
					//		fmt.Println(err)
					//	}
					//	fill.Close()
					//}
					if header.Difficulty.Cmp(common.Big0) == 0 {
						//w.storage.Put(header)
						w.blockStorage.Put(block)
						continue
					}

					go func() {
						err := w.handleCurrentBlock(block)
						if err != nil {
							w.log.WithError(err).WithField("epoch", epoch).Warn("Failed to handle current block")
						}
					}()

				}
			} else if header.Height > epoch {
				// vdf check
				w.log.WithFields(logrus.Fields{
					"current_epoch": epoch,
					"block_epoch":   header.Height,
					"block_hash":    hexutil.Encode(header.Hashes),
					"from_node":     header.Address,
				}).Debug("Received advanced block")
				go func() {
					err := w.handleAdvancedBlock(epoch, block)
					if err != nil {
						w.log.WithError(err).WithFields(logrus.Fields{
							"epoch":        epoch,
							"block_height": header.Height,
						}).Warn("Failed to handle advanced block")
					}

				}()
			}
		}
	}
}

func (w *Worker) handleCurrentBlock(block *types.Block) error {
	header := block.GetHeader()
	// _, _ = w.checkblock(header)
	flag, err := w.CheckVDF0Current(header)
	if !flag {
		return err
	}

	half, _ := w.blockStorage.GetVDFHalf(header.Height)
	if len(half) != 0 {
		if !bytes.Equal(half, header.PoTProof[2]) {
			return fmt.Errorf("block half vdf0 is error")
		}
	} else {
		flag := wesolowski_rust.Verify(header.PoTProof[0], w.config.PoT.Vdf1Iteration, header.PoTProof[2])
		if !flag {
			return fmt.Errorf("block half vdf0 is error")
		} else {
			w.blockStorage.SetVDFHalf(header.Height, header.PoTProof[2])
		}
	}

	if !w.isBehindHeight(header.Height-1, block) {

		if header.ParentHash != nil {
			w.log.WithFields(logrus.Fields{
				"block_epoch":   block.GetHeader().Height,
				"block_hash":    hexutil.Encode(block.Hash()),
				"parent_hash":   hexutil.Encode(block.GetHeader().ParentHash),
				"current_epoch": w.chainReader.GetCurrentHeight(),
				"current_hash":  hexutil.Encode(w.chainReader.GetCurrentBlock().Hash()),
			}).Warn("Fork detected")
			w.mutex.Lock()
			if w.chainresetflag {
				w.mutex.Unlock()
				return fmt.Errorf("the worker is handling fork right now")
			}

			w.chainresetflag = true
			defer w.SetChainSelectFlagFalse()
			w.mutex.Unlock()

			done := make(chan struct{})
			var doonce sync.Once
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			go func() {
				select {
				case <-ctx.Done():
					doonce.Do(func() {
						close(done)
						w.log.Error("Fork handling timeout")
					})
					return
				}
			}()

			currentblock := w.chainReader.GetCurrentBlock()
			ances, err := w.GetSharedAncestor(block, currentblock)

			if err != nil {
				w.log.WithError(err).Error("Failed to get shared ancestor")
				return err
			}

			c, err := w.chainReader.GetByHeight(ances.GetHeader().Height)
			if err != nil {
				w.log.WithError(err).Error("Failed to get block by height")
				return err
			}

			w.log.WithFields(logrus.Fields{
				"ancestor_hash":   hexutil.Encode(ances.GetHeader().Hashes),
				"ancestor_height": ances.GetHeader().Height,
				"match":           bytes.Equal(c.GetHeader().Hashes, ances.GetHeader().Hashes),
			}).Debug("Found shared ancestor")
			//nowBranch, _, err := w.GetBranch(ances, currentblock)

			forkBranch, _, err := w.GetBranch(ances, block)
			if err != nil {

				return fmt.Errorf("get Branch error for %s", err.Error())
			}
			nowBranch, _, err := w.GetBranch(ances, currentblock)

			flag, err := w.CheckVDF0ForBranch(forkBranch)
			if flag {
				w.log.Trace("VDF verification passed for fork branch")
			} else {
				return err
			}

			for i := 0; i < len(nowBranch); i++ {
				w.log.WithFields(logrus.Fields{
					"height": nowBranch[i].GetHeader().Height,
					"hash":   utils.EncodeShortPrint(nowBranch[i].Hash()),
				}).Trace("Current chain branch block")
			}
			for i := 0; i < len(forkBranch); i++ {
				w.log.WithFields(logrus.Fields{
					"height": forkBranch[i].GetHeader().Height,
					"hash":   utils.EncodeShortPrint(forkBranch[i].Hash()),
				}).Trace("Fork chain branch block")
			}

			w1 := w.calculateChainWeight(ances, currentblock)
			w2 := w.calculateChainWeight(ances, block)
			w.log.WithFields(logrus.Fields{
				"current_weight": w1.Int64(),
				"fork_weight":    w2.Int64(),
			}).Info("Comparing chain weights")

			if w1.Int64() > w2.Int64() {
				w.log.Info("Fork chain weight insufficient, keeping current chain")
				return nil
			}

			flag, err = w.handleForkTx(nowBranch, forkBranch)
			if err != nil {
				w.log.WithError(err).Error("Failed to handle fork transactions")

			}

			err = w.chainreset(forkBranch)
			if err != nil {
				w.log.WithError(err).Error("Failed to reset chain")
			}

			_ = w.blockStorage.Put(block)

			err = w.workReset(block.GetHeader().Height, block)
			if err != nil {
				w.log.WithError(err).Error("Failed to reset work")
			}
			doonce.Do(func() {
				w.log.Info("Fork handling complete")
				close(done)
			})

			<-done
			return nil
			//txs := block.GetExcutedTxs()
			//w.mempool.MarkProposed(txs)
		}
	} else {
		flag, err = header.BasicVerify()
		if !flag {
			return err
		}
		flag, err = w.CheckBlockTxs(block)
		if !flag && err != nil {
			return err
		}

		//if !w.uponReceivedBlock(block.GetHeader().Height, block) {
		//	return fmt.Errorf("block %s fails the cryptoelement check", hexutil.Encode(block.Hash()))
		//}

		err := w.blockStorage.Put(block)

		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (w *Worker) handleAdvancedBlock(epoch uint64, block *types.Block) error {
	current := w.chainReader.GetCurrentBlock()
	w.mutex.Lock()
	if w.chainresetflag {
		w.mutex.Unlock()
		return fmt.Errorf("the worker is handling fork right now")
	}

	w.chainresetflag = true
	defer w.SetChainSelectFlagFalse()
	w.mutex.Unlock()

	done := make(chan struct{})
	var doonce sync.Once
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
			doonce.Do(func() {
				close(done)
				w.log.Error("Advanced block handling timeout")
				return
			})
			return
		}
	}()
	ances, err := w.GetSharedAncestor(block, current)
	if err != nil {
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	w.log.WithFields(logrus.Fields{
		"block_hash":      hexutil.Encode(block.Hash()),
		"ancestor_hash":   hexutil.Encode(ances.Hash()),
		"ancestor_height": ances.GetHeader().Height,
	}).Debug("Found shared ancestor for advanced block")

	nowbranch, _, err := w.GetBranch(ances, current)
	if err != nil {
		w.log.WithError(err).Error("Failed to get current branch")
		doonce.Do(func() {
			close(done)
		})
		return err
	}
	branch, _, err := w.GetBranch(ances, block)
	if err != nil {
		w.log.WithError(err).Error("Failed to get fork branch")
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	flag, err := w.CheckVDF0ForBranch(branch)
	if flag {
		w.log.Trace("VDF verification passed for advanced block")
	}
	if err != nil {
		w.log.WithError(err).Error("VDF verification failed for branch")
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	weightnow := w.calculateChainWeight(ances, current)
	weightadvanced := w.calculateChainWeight(ances, block)

	if weightnow.Cmp(weightadvanced) > 0 {
		w.log.WithFields(logrus.Fields{
			"current_weight": weightnow.Int64(),
			"fork_weight":    weightadvanced.Int64(),
		}).Info("Current chain weight greater, keeping current chain")
		doonce.Do(func() {
			close(done)
		})
		return fmt.Errorf("the current chain weight %d is greater than the fork chain weight %d", weightnow.Int64(), weightadvanced.Int64())
	}

	w.log.WithFields(logrus.Fields{
		"current_weight": weightnow,
		"fork_weight":    w.calculateChainWeight(ances, block).Int64(),
	}).Info("Switching to fork chain")

	//for i := 0; i < len(branch); i++ {
	//	w.log.Infof("[PoT]\tthe nowbranch at height %d: %s", branch[i].GetHeader().ExecHeight, hexutil.Encode(branch[i].Hash()))
	//}
	flag, err = w.handleForkTx(nowbranch, branch)
	if err != nil {
		w.log.WithError(err).Error("Failed to handle fork transactions")
		return err
	}
	err = w.chainResetAdvanced(branch)
	if err != nil {
		w.log.WithError(err).Error("Failed to reset chain for advanced block")
		doonce.Do(func() {
			close(done)
		})
		return err
	}
	finishflag := w.vdf0.Finished
	w.log.WithField("vdf0_finished", finishflag).Debug("Checked VDF0 work status")

	if w.IsVDF1Working() {
		w.abort.once.Do(func() {
			close(w.abort.abortchannel)
		})
		w.wg.Wait()
		w.setWorkFlagFalse()
	}

	header := block.GetHeader()
	if len(header.UncleHash) != 0 {
		_, err := w.getUncleBlock(block)
		if err != nil {
			w.log.WithError(err).Error("Failed to get uncle block")
			doonce.Do(func() {
				close(done)
			})
			return err
		}
	}
	w.mutex.Lock()
	// w.backupBlock = append(w.backupBlock, block)

	w.blockCounter += 1
	//err = w.storage.Put(block)
	err = w.blockStorage.Put(block)
	w.mutex.Unlock()

	err = w.setVDF0epoch(block.GetHeader().Height)
	if err != nil {
		w.log.WithError(err).WithField("epoch", epoch).Warn("Failed to set VDF epoch")
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	res := &types.VDF0res{
		Res:   block.GetHeader().PoTProof[2],
		Epoch: block.GetHeader().Height,
	}
	w.vdfhalfchan <- res
	w.log.WithFields(logrus.Fields{
		"current_epoch": epoch,
		"start_epoch":   block.GetHeader().Height,
		"vdf_res":       hexutil.Encode(crypto.Hash(res.Res)),
	}).Debug("VDF epoch set complete")
	//w.mutex.Lock()
	//w.log.Error(w.chainresetflag)
	//w.mutex.Unlock()
	doonce.Do(func() {
		close(done)
		w.log.Debug("Advanced block handling complete")
	})

	<-done
	return nil

}

func (w *Worker) blockbytebroadcast(blockbyte []byte) error {
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_Block_Data,
		MsgByte: blockbyte,
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

func (w *Worker) blockbroadcast(block *types.Block) error {

	pbblock := block.ToProto()
	blockbyte, err := proto.Marshal(pbblock)
	if err != nil {
		return err
	}
	message := &pb.PoTMessage{
		MsgType: pb.MessageType_Block_Data,
		MsgByte: blockbyte,
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
	}

	return true, nil
}

func (w *Worker) CheckVDF0ForBranch(branch []*types.Block) (bool, error) {

	for i := len(branch) - 1; i >= 0; i-- {
		storageheight := w.blockStorage.GetVDFHeight()
		header := branch[i].GetHeader()
		if header.Height <= storageheight {
			//if header.Height == 0 {
			//	return true, nil
			//}
			vdfres, err := w.blockStorage.GetVDFresbyEpoch(header.Height)

			if err != nil {
				return false, err
			}
			if !bytes.Equal(vdfres, header.PoTProof[0]) {
				fmt.Println(hexutil.Encode(vdfres), hexutil.Encode(header.PoTProof[0]))
				return false, fmt.Errorf("the vdf0 proof is wrong of height %d", header.Height)
			}

		} else if header.Height == storageheight+1 {
			// vdfres, err := w.blockStorage.GetVDFresbyEpoch(storageheight)
			// if err != nil {
			// 	return false, err
			// }
			// vdfinput := crypto.Hash(vdfres)
			// vdfoutput := header.PoTProof[0]
			// times := time.Now()
			// if !w.vdfChecker.CheckVDF(vdfinput, vdfoutput) {
			// 	return false, fmt.Errorf("the vdf0 proof is wrong for height %d", header.Height)
			// }
			// w.log.Infof("[PoT]\tVDF Check need %d ms", time.Since(times)/time.Millisecond)
			// w.blockStorage.SetVDFres(header.Height, header.PoTProof[0])
			vdfhalfch := make(chan []byte, 2048)
			go func(epoch uint64) []byte {
				for {
					b, err := w.blockStorage.GetVDFHalf(epoch)
					if err == nil && b != nil {
						vdfhalfch <- b
					} else {
						time.Sleep(5 * time.Second)
					}
				}
			}(storageheight)

			vdfhalf := <-vdfhalfch
			vdfinput := crypto.Hash(vdfhalf)
			vdfout := header.PoTProof[0]
			if !wesolowski_rust.Verify(vdfinput, w.config.PoT.Vdf0Iteration-w.config.PoT.Vdf1Iteration, vdfout) {
				return false, fmt.Errorf("the vdf0 proof is wrong for height %d", header.Height)
			}
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
		times := time.Now()
		if !w.vdfChecker.CheckVDF(vdfInput, vdfOutput) {
			return false, fmt.Errorf("the vdf0 proof of block is wrong")
		}
		w.log.WithField("duration_ms", time.Since(times)/time.Millisecond).Trace("VDF verification complete")
		return true, nil
	}
	return true, nil
}

func (w *Worker) CheckHeaderVDF1(block *types.Block) (bool, error) {
	b := block.GetHeader()
	mixdigest := b.Mixdigest
	noncebyte := new(big.Int).SetInt64(b.Nonce).Bytes()

	if b.PoTProof == nil {
		return false, fmt.Errorf("the block without pot proof")
	}
	vdf0res := b.PoTProof[0]
	input := bytes.Join([][]byte{noncebyte, vdf0res, mixdigest}, []byte(""))
	hashinput := crypto.Hash(input)
	output := b.PoTProof[1]
	flag := wesolowski_rust.Verify(hashinput, w.config.PoT.Vdf1Iteration, output)
	if !flag {
		return false, fmt.Errorf("the block vdf1 proof is wrong")
	}

	target := new(big.Int).Set(bigD)
	target = target.Div(bigD, b.Difficulty)
	tmp := new(big.Int).SetBytes(output)
	if tmp.Cmp(target) >= 0 {
		return false, fmt.Errorf("the difficulty check fail")
	}
	return true, nil
}

func (w *Worker) checkHeaderVDFHalf(block *types.Block) (bool, error) {
	header := block.GetHeader()
	height := header.Height
	if len(header.PoTProof) != 3 {
		return false, fmt.Errorf("the block vdf proof is less than 3")
	}

	b, err := w.blockStorage.GetVDFHalf(height)

	if err != nil {
		flag := wesolowski_rust.Verify(header.PoTProof[0], w.config.PoT.Vdf1Iteration, header.PoTProof[2])
		if !flag {
			return false, fmt.Errorf("the vdf1 proof is wrong")
		}
	}
	if !bytes.Equal(b, header.PoTProof[2]) {
		return false, fmt.Errorf("the vdf half proof is wrong")
	}
	return true, nil
}

func (w *Worker) workReset(epoch uint64, block *types.Block) error {
	header := block.GetHeader()

	parentblock, err := w.blockStorage.Get(header.ParentHash)
	if err != nil {
		return err
	}

	uncleblock := make([]*types.Block, 0)
	for i := 0; i < len(header.UncleHash); i++ {
		b, err := w.blockStorage.Get(header.UncleHash[i])
		if err != nil {
			return err
		}
		uncleblock = append(uncleblock, b)
	}

	res0, err := w.blockStorage.GetVDFresbyEpoch(epoch)
	if err != nil {
		return err
	}

	difficulty := header.Difficulty
	if w.IsVDF1Working() {
		w.setWorkFlagFalse()
		w.abort.once.Do(func() {
			close(w.abort.abortchannel)
		})
		w.wg.Wait()

	}
	w.startWorking()
	w.abort = NewAbortcontrol()
	w.wg.Add(cpuCounter)

	vdf0rescopy := make([]byte, len(res0))
	copy(vdf0rescopy, res0)
	exeblocks := w.GetExecutedBlockFromMempool()
	rawtxs := w.mempool.GetRawTx()
	Bcirewards := w.mempool.GetAllBciRewards()

	for i := 0; i < cpuCounter; i++ {
		emptyblock := w.createBlockWithoutKey(epoch+1, parentblock, uncleblock, difficulty, exeblocks, rawtxs)
		go w.mine(epoch, vdf0rescopy, rand.Int63(), i, w.abort, difficulty, parentblock, uncleblock, w.wg, emptyblock, Bcirewards)
	}

	return nil
}

func (w *Worker) CheckBlockNumEnough(block *types.Block) bool {
	if block.GetHeader().Height == 0 {
		return true
	}
	cnt := 0
	if block.GetHeader().ParentHash != nil {
		cnt += 1
	}
	if block.GetHeader().UncleHash != nil {
		cnt += len(block.GetHeader().UncleHash)
	}

	low := w.config.PoT.Snum / 2
	if int64(cnt) < low {
		return false
	} else {
		return true
	}
}

func (w *Worker) handleForkTx(current []*types.Block, fork []*types.Block) (bool, error) {

	for i := 0; i < len(current); i++ {
		curblock := current[i]
		if flag, _ := w.chainReader.IsBlockOnChain(curblock); flag {
			err := w.chainReader.TryResetTxForBlock(curblock)
			w.log.WithField("height", curblock.GetHeader().Height).Trace("Resetting transactions for block")
			if err != nil {
				w.log.WithError(err).WithField("height", curblock.GetHeader().Height).Warn("Transaction reset attempt failed")
				return false, err
			}
			err = w.chainReader.ResetTxForBlock(curblock)
			if err != nil {
				w.log.WithError(err).WithField("height", curblock.GetHeader().Height).Error("Transaction reset failed")
				for j := i; j >= 0; j-- {
					block := current[j]
					err := w.chainReader.TryUpdateTxForBlock(block)
					if err != nil {
						return false, err
					}
					w.log.WithField("height", block.Header.Height).Debug("Rolling back to previous height")
					err = w.chainReader.UpdateTxForBlock(block)
					if err != nil {
						return false, err
					}
				}
				return false, err
			}
		}
	}

	for i := len(fork) - 1; i >= 1; i-- {
		updateblock := fork[i]
		_, err := w.CheckBlockTxs(updateblock)
		if err != nil {
			for j := i + 1; j < len(fork); j++ {
				err = w.chainReader.ResetTxForBlock(fork[j])
				if err != nil {
					return false, err
				}
			}
			for j := len(current) - 1; j >= 0; j-- {

			}

			return false, err
		}

		err = w.chainReader.TryUpdateTxForBlock(updateblock)
		if err != nil {
			for j := i + 1; j < len(fork); j++ {
				err = w.chainReader.ResetTxForBlock(fork[j])
				if err != nil {
					return false, err
				}
			}

			for j := len(current) - 1; j >= 0; j-- {
				_ = w.chainReader.UpdateTxForBlock(current[j])

			}
			return false, err
		}
		err = w.chainReader.UpdateTxForBlock(updateblock)
		if err != nil {
			for j := i + 1; j < len(fork); j++ {
				err = w.chainReader.ResetTxForBlock(fork[j])
				if err != nil {
					return false, err
				}
			}

			for j := len(current) - 1; j >= 0; j-- {
				_ = w.chainReader.UpdateTxForBlock(current[j])

			}
			return false, err
		}
	}
	return true, nil
}
