package pot

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/crypto/vdf/wesolowski_rust"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
	"math/big"
	"math/rand"
	"sync"
	"time"
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
					w.log.Infof("[PoT]\tepoch %d:Receive block from node %d, Difficulty %d, with parent %s, transport need %f millseconds", epoch, header.Address, header.Difficulty.Int64(), hex.EncodeToString(header.ParentHash), float64(time.Since(header.Timestamp))/float64(time.Millisecond))
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
							w.log.Errorf("[PoT]\tepoch %d:handle current block err for %s", epoch, err)
						}
					}()

				}
			} else if header.Height > epoch {
				// vdf check
				w.log.Infof("[PoT]\tepoch %d:Receive a epoch %d block %s from node %d", epoch, header.Height, hexutil.Encode(header.Hashes), header.Address)
				go func() {
					err := w.handleAdvancedBlock(epoch, block)
					if err != nil {
						w.log.Errorf("[PoT]\tepoch %d:handle advanced block err for %s", epoch, err)
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

	if !w.isBehindHeight(header.Height-1, block) {

		if header.ParentHash != nil {
			w.log.Warnf("[PoT]\tfind fork at epoch %d block %s with parents %s,current epoch %d %s", block.GetHeader().Height, hexutil.Encode(block.Hash()), hexutil.Encode(block.GetHeader().ParentHash), w.chainReader.GetCurrentHeight(), hexutil.Encode(w.chainReader.GetCurrentBlock().Hash()))
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
						w.log.Errorf("[PoT]\thandle fork timeout")
						return
					})
					return
				}
			}()

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

			w.log.Debugf("[PoT]\tthe shared ancestor of fork is %s at %d,match %t", hexutil.Encode(ances.GetHeader().Hashes), ances.GetHeader().Height, bytes.Equal(c.GetHeader().Hashes, ances.GetHeader().Hashes))
			//nowBranch, _, err := w.GetBranch(ances, currentblock)

			forkBranch, _, err := w.GetBranch(ances, block)
			if err != nil {

				return fmt.Errorf("get Branch error for %s", err.Error())
			}

			flag, err := w.CheckVDF0ForBranch(forkBranch)
			if flag {
				w.log.Debugf("[PoT]\tPass VDF Check")
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
			w.log.Infof("[PoT]\tthe chain weight %d, the fork chain weight %d", w1.Int64(), w2.Int64())

			if w1.Int64() < w2.Int64() {
				err := w.chainreset(forkBranch)
				if err != nil {
					w.log.Errorf("[PoT]\tchain reset error for %s", err)
				}
			}

			_ = w.blockStorage.Put(block)

			err = w.workReset(block.GetHeader().Height, block)
			if err != nil {
				w.log.Errorf("[PoT]\twork reset error for %s", err)
			}
			doonce.Do(func() {
				w.log.Warn("[PoT]\tHandle fork done")
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

	//w.log.Errorf("Start handleing fork at epoch %d block %s", block.GetHeader().Height, hexutil.Encode(block.GetHeader().Hashes))
	//if epoch+100 < block.GetHeader().Height {
	//	err := w.setVDF0epoch(block.GetHeader().Height - 1)
	//	if err != nil {
	//		w.log.Warnf("[PoT]\tepoch %d: execset vdf error for %s", epoch, err)
	//		return err
	//	}
	//
	//	// w.backupBlock = append(w.backupBlock, block)
	//
	//	w.blockCounter += 1
	//	//err = w.storage.Put(block)
	//	err = w.blockStorage.Put(block)
	//
	//	res := &types.VDF0res{
	//		Res:   block.GetHeader().PoTProof[0],
	//		Epoch: block.GetHeader().Height - 1,
	//	}
	//
	//	w.vdf0Chan <- res
	//	w.log.Infof("[PoT]\tepoch %d:execset vdf complete. Start from epoch %d with res %s", epoch, block.GetHeader().Height-1, hexutil.Encode(crypto.Hash(res.Res)))
	//	return nil
	//}
	done := make(chan struct{})
	var doonce sync.Once
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
			doonce.Do(func() {
				close(done)
				w.log.Errorf("[PoT]\thandle fork timeout")
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

	w.log.Infof("[PoT]\tGet shared ancestor of block %s is %s at height %d", hexutil.Encode(block.Hash()), hexutil.Encode(ances.Hash()), ances.GetHeader().Height)

	branch, _, err := w.GetBranch(ances, block)
	if err != nil {
		w.log.Errorf("[PoT]\tGet branch error for: %s", err)
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	flag, err := w.CheckVDF0ForBranch(branch)
	if flag {
		w.log.Infof("[PoT]\tPass VDF Check")
	}
	if err != nil {
		w.log.Errorf("[PoT]\tCheck Branch VDF0 error for: %s", err)
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	weightnow := w.calculateChainWeight(ances, current)

	weightadvanced := w.calculateChainWeight(ances, block)

	if weightnow.Cmp(weightadvanced) > 0 {
		w.log.Infof("[PoT]\tthe current chain weight %d is greater than the fork chain weight %d", weightnow.Int64(), weightadvanced.Int64())
		doonce.Do(func() {
			close(done)
		})
		return fmt.Errorf("the current chain weight %d is greater than the fork chain weight %d", weightnow.Int64(), weightadvanced.Int64())
	}

	w.log.Infof("[PoT]\tthe chain weight %d, the fork chain weight %d", weightnow, w.calculateChainWeight(ances, block).Int64())

	//for i := 0; i < len(branch); i++ {
	//	w.log.Infof("[PoT]\tthe nowbranch at height %d: %s", branch[i].GetHeader().ExecHeight, hexutil.Encode(branch[i].Hash()))
	//}

	err = w.chainResetAdvanced(branch)
	if err != nil {
		w.log.Errorf("[PoT]\tchain reset error for %s", err)
		doonce.Do(func() {
			close(done)
		})
		return err
	}
	finishflag := w.vdf0.Finished
	w.log.Infof("[PoT]\tMinew Work flag: %t", finishflag)

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
			w.log.Errorf("[PoT]\tGet uncle block err for %s", err)
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

	err = w.setVDF0epoch(block.GetHeader().Height - 1)
	if err != nil {
		w.log.Warnf("[PoT]\tepoch %d: execset vdf error for %s:", epoch, err)
		doonce.Do(func() {
			close(done)
		})
		return err
	}

	res := &types.VDF0res{
		Res:   block.GetHeader().PoTProof[0],
		Epoch: block.GetHeader().Height - 1,
	}
	w.vdf0Chan <- res
	w.log.Infof("[PoT]\tepoch %d:execset vdf complete. Start from epoch %d with res %s", epoch, block.GetHeader().Height-1, hexutil.Encode(crypto.Hash(res.Res)))
	//w.mutex.Lock()
	//w.log.Error(w.chainresetflag)
	//w.mutex.Unlock()
	doonce.Do(func() {
		close(done)
		w.log.Debug("[PoT]\tHandle fork done")
	})

	<-done
	return nil

}

func (w *Worker) blockbroadcast(block *types.Block) error {

	pbblock := block.ToProto()
	headerByte, err := proto.Marshal(pbblock)
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
			vdfres, err := w.blockStorage.GetVDFresbyEpoch(storageheight)
			if err != nil {
				return false, err
			}
			vdfinput := crypto.Hash(vdfres)
			vdfoutput := header.PoTProof[0]
			times := time.Now()
			if !w.vdfChecker.CheckVDF(vdfinput, vdfoutput) {
				return false, fmt.Errorf("the vdf0 proof is wrong for height %d", header.Height)
			}
			w.log.Infof("[PoT]\tVDF Check need %d ms", time.Since(times)/time.Millisecond)
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
		w.log.Infof("[PoT]\tVDF Check need %d ms", time.Since(times)/time.Millisecond)
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

func (w *Worker) CheckBlock(block *types.Block) (bool, error) {
	header := block.GetHeader()
	flag, err := header.BasicVerify()
	if err != nil {
		return flag, err
	}
	flag, err = w.CheckHeaderVDF1(block)
	if err != nil {
		return flag, err
	}
	return true, nil
	//excutedtxs := block.GetExecutedHeaders()

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
	for i := 0; i < cpuCounter; i++ {
		go w.mine(epoch, vdf0rescopy, rand.Int63(), i, w.abort, difficulty, parentblock, uncleblock, w.wg)
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
