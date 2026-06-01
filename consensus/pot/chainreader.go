package pot

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

/*
ChainReader is used to store chain state for the pot chain
*/

type ChainReader struct {
	storage      *types.BlockStorage
	chain        map[uint64]*types.Block
	lockUTXO     map[uint64]map[string]*types.TxOutput
	tempLockUTXO map[uint64]map[string]*types.TxOutput
	height       uint64
	sync         *sync.RWMutex
}

func NewChainReader(storage *types.BlockStorage) *ChainReader {
	c := &ChainReader{
		storage:      storage,
		chain:        make(map[uint64]*types.Block),
		lockUTXO:     make(map[uint64]map[string]*types.TxOutput),
		tempLockUTXO: make(map[uint64]map[string]*types.TxOutput),
		height:       0,
		sync:         new(sync.RWMutex),
	}
	c.chain[0] = types.DefaultGenesisBlock()
	return c
}

func (c *ChainReader) SetHeight(height uint64, block *types.Block) {
	c.sync.Lock()
	defer c.sync.Unlock()
	if height != block.GetHeader().Height {
		err := fmt.Errorf("set height error for block height not match to set height")
		panic(err)
	}
	c.chain[height] = block
	if height > c.height {
		c.height = height
	}
}
func (c *ChainReader) GetBoltDb() *bolt.DB {
	return c.storage.GetBoltdb()
}

func (c *ChainReader) GetByHeight(height uint64) (*types.Block, error) {
	c.sync.RLock()
	defer c.sync.RUnlock()
	if height > c.height {
		return nil, fmt.Errorf("the height %d haven't set yet", height)
	}
	if c.chain[height] != nil {
		return c.chain[height], nil
	} else {
		return nil, fmt.Errorf("the height %d haven't set yet", height)
	}
}

func (c *ChainReader) GetCurrentBlock() *types.Block {
	height := c.GetCurrentHeight()
	parent, err := c.GetByHeight(height)
	if err != nil {
		return nil
	}
	return parent
}

func (c *ChainReader) GetCurrentHeight() uint64 {
	return c.height

}

func (c *ChainReader) ValidateBlock(block *types.Header) bool {
	return true
}

func (c *ChainReader) IsBehindCurrent(block *types.Block) bool {
	currentheight := c.GetCurrentHeight()
	if currentheight == 0 {
		return true
	}
	if currentheight+1 != block.GetHeader().Height {
		return false
	}
	current := c.GetCurrentBlock()
	return !bytes.Equal(current.GetHeader().Hashes, block.GetHeader().ParentHash)
}

func (c *ChainReader) IsBlockOnChain(block *types.Block) (bool, error) {
	c.sync.RLock()
	defer c.sync.RUnlock()

	blockheight := block.GetHeader().Height
	chainBlock, err := c.GetByHeight(blockheight)
	if err != nil {
		return false, err
	}
	flag := bytes.Equal(chainBlock.Hash(), block.Hash())
	if !flag {
		return false, fmt.Errorf("the block is not on chain")
	} else {
		return true, nil
	}
}

func (c *ChainReader) FindUnspentTransactions(address []byte) []*types.RawTx {
	unspentTxs := make([]*types.RawTx, 0)
	spentsUTXOs := make(map[[32]byte][]int64)
	height := c.GetCurrentHeight()
	for height > 0 {
		block, err := c.GetByHeight(height)
		if err != nil {
			return unspentTxs
		}
		rawtxs := block.GetRawTx()
		for _, rawtx := range rawtxs {
			//txID := hex.EncodeToString(rawtx.Txid[:])

		Outputs:
			for outidx, output := range rawtx.TxOutput {
				if spentsUTXOs[rawtx.Txid] != nil {
					for _, spentOut := range spentsUTXOs[rawtx.Txid] {
						if spentOut == int64(outidx) {
							continue Outputs
						}
					}
				}
				if bytes.Equal(output.Address, address) {
					unspentTxs = append(unspentTxs, rawtx)
				}
			}

			if !rawtx.IsCoinBase() {
				for _, in := range rawtx.TxInput {
					if in.CanUnlockOutputwith(address) {
						//inTXid := hex.EncodeToString(in.Txid)
						spentsUTXOs[in.Txid] = append(spentsUTXOs[in.Txid], in.Voutput)
					}
				}
			}
		}
	}
	return unspentTxs
}

func (c *ChainReader) FindUTXO(address []byte) (map[string]types.TxOutput, error) {
	utxos := make(map[string]types.TxOutput)

	db := c.GetBoltDb()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		cursor := b.Cursor()
		count := 0
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {

			outs := types.DecodeByteToTxOutput(v)

			if outs.IsLockedWithKey(address) {
				count += 1
				utxokey := string(k)

				parts := strings.Split(utxokey, ":")

				if len(parts) != 2 {
					return fmt.Errorf("invalid utxokey format: %v", utxokey)
				}
				txid := parts[0]
				_, err := hexutil.Decode(txid)
				if err != nil {
					return fmt.Errorf("failed to decode txid: %v", err)
				}
				fmt.Println(utxokey)
				utxos[utxokey] = outs
			}
			b.Put(k, v)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return utxos, nil
}

func (c *ChainReader) FindAllUnspentOutputs() map[[32]byte]types.TxOutputs {
	UTXO := make(map[[32]byte]types.TxOutputs)
	spentTXOs := make(map[[32]byte][]int64)

	height := c.GetCurrentHeight()

	for height > 0 {
		block, err := c.GetByHeight(height)
		if err != nil {
			break
		}

		txs := block.GetRawTx()
		for _, tx := range txs {
			txID := tx.Txid

		Outputs:
			for outIdx, out := range tx.TxOutput {
				// Was the output spent?
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == int64(outIdx) {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs = append(outs, out)
				UTXO[txID] = outs
			}

			if !tx.IsCoinBase() {
				for _, in := range tx.TxInput {
					inTxID := in.Txid
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Voutput)
				}
			}
		}

	}
	return UTXO
}

func (c *ChainReader) FindSpendableOutputs(pubkey []byte, amount int64) (int64, map[[32]byte][]int64) {
	unspentOutputs := make(map[[32]byte][]int64)

	accumulated := int64(0)

	err := c.GetBoltDb().View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := crypto.Convert(k)
			outs := types.DecodeByte2Outputs(v)

			for outIdx, out := range outs {
				if bytes.Equal(out.Address, pubkey) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], int64(outIdx))
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Panic(err)
	}
	return accumulated, unspentOutputs

}

func (c *ChainReader) Reindex() error {
	db := c.GetBoltDb()
	bucketName := []byte(types.UTXOBucket)
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && !errors.Is(err, bolt.ErrBucketNotFound) {
			return err
		}

		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	utxos := c.FindAllUnspentOutputs()
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		for txid, outputs := range utxos {
			err = b.Put(txid[:], outputs.EncodeTxOutputs2Byte())
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil

}

// ResetTxForBlock reset utxo bucket for block reset, need to happen after UpdateBlock
func (c *ChainReader) ResetTxForBlock(block *types.Block) error {
	c.sync.Lock()
	defer c.sync.Unlock()
	txs := block.GetRawTx()
	db := c.GetBoltDb()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))

		for _, rawTx := range txs {

			// delete txoutput of the rawtx
			for i, output := range rawTx.TxOutput {
				utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(rawTx.Txid[:]), i)
				// delete the lockutxo in map
				lockheight := block.GetHeader().Height + output.LockTime
				if _, exist := c.lockUTXO[lockheight][utxokey]; exist {
					delete(c.lockUTXO[lockheight], utxokey)
					if len(c.lockUTXO[lockheight]) == 0 {
						delete(c.lockUTXO, lockheight)
					}
					continue
				}
				// delete the utxo
				outputbyte := b.Get([]byte(utxokey))
				if outputbyte != nil {
					err := b.Delete([]byte(utxokey))
					if err != nil {
						return err
					}
				}
			}

			// add txinput corresponding output back to database
			if !rawTx.IsCoinBase() {
				for _, input := range rawTx.TxInput {
					txid := input.Txid
					voutput := input.Voutput
					height := block.GetHeader().Height - 1
					findflag := false
					for height > 0 && !findflag {
						currBlock, err := c.GetByHeight(height)
						if err != nil {
							return err
						}
						rawTxes := currBlock.GetRawTx()
						for _, t := range rawTxes {
							if txid == t.Txid {
								if len(t.TxOutput) < int(voutput) {
									return fmt.Errorf("reset block for could not find tx output")
								}
								out := t.TxOutput[voutput]
								utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(txid[:]), voutput)
								out.BlockHeight = height
								if block.GetHeader().Height >= height+out.LockTime {
									err = b.Put([]byte(utxokey), out.EncodeToByte())
									if err != nil {
										return err
									}
								} else {
									if _, exist := c.lockUTXO[height+out.LockTime][utxokey]; !exist {
										c.lockUTXO[height+out.LockTime][utxokey] = &out
									}
								}
								findflag = true
								break
							}
						}
						height--
					}
					if !findflag {
						return fmt.Errorf("reset block for could not find corresponding tx output to txinput")
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	} else {
		return nil
	}
}

func (c *ChainReader) TryResetTxForBlock(block *types.Block) error {
	c.sync.Lock()
	defer c.sync.Unlock()
	txs := block.GetRawTx()
	db := c.GetBoltDb()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, rawtx := range txs {
			for i, output := range rawtx.TxOutput {
				utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(rawtx.Txid[:]), i)
				lockheight := block.GetHeader().Height + output.LockTime
				_, exist := c.lockUTXO[lockheight][utxokey]
				outputbyte := b.Get([]byte(utxokey))
				if outputbyte == nil && !exist {
					return fmt.Errorf("utxo %s not exist", utxokey)
				}
				if outputbyte != nil {
					b.Put([]byte(utxokey), output.EncodeToByte())
				}
			}

			if !rawtx.IsCoinBase() {
				for _, input := range rawtx.TxInput {
					txid := input.Txid
					voutput := input.Voutput
					height := block.GetHeader().Height - 1
					findflag := false
					for height > 0 && !findflag {
						currBlock, err := c.GetByHeight(height)
						if err != nil {
							return err
						}
						rawTxes := currBlock.GetRawTx()
						for _, t := range rawTxes {
							if txid == t.Txid {
								if len(t.TxOutput) < int(voutput) {
									return fmt.Errorf("reset block for could not find tx output")
								}
								out := t.TxOutput[voutput]
								_ = fmt.Sprintf("%s:%d", hexutil.Encode(txid[:]), voutput)
								//out.BlockHeight = height
								if block.GetHeader().Height >= height+out.LockTime {
									// err = b.Put([]byte(utxokey), out.EncodeToByte())
									// if err != nil {
									// 	return err
									// }
								} else {
									// if _, exist := c.lockUTXO[height+out.LockTime][utxokey]; !exist {
									// 	c.lockUTXO[height+out.LockTime][utxokey] = &out
									// }
								}
								findflag = true
								break
							}
						}
						height--
					}
					if !findflag {
						return fmt.Errorf("reset block for could not find corresponding tx output to txinput")
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	} else {
		return nil
	}
}

//func (c *ChainReader) copyLockmap()  {
//	if c.tempLockUTXO == nil {
//		c.tempLockUTXO = make(map[uint64]map)
//	}
//
//	for lockheight, outputs := range c.lockUTXO {
//		c.tempLockUTXO[lockheight] = make(map[string]types.TxOutput)
//		for key, output := range outputs {
//			c.
//		}
//	}
//
//}

func (c *ChainReader) UpdateTxForBlock(block *types.Block) error {
	db := c.GetBoltDb()
	txs := block.GetRawTx()
	height := block.GetHeader().Height
	c.sync.Lock()
	defer c.sync.Unlock()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, rawTx := range txs {
			if !rawTx.IsCoinBase() {
				for _, input := range rawTx.TxInput {
					// find if there is corresponding utxo in the utxo
					utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(input.Txid[:]), input.Voutput)
					outsBytes := b.Get([]byte(utxokey))

					if outsBytes == nil {
						return fmt.Errorf("update tx error for can't find corresponding utxo %s ", utxokey)
					}
					// TODO: add script check
					fmt.Printf("use %s\n", utxokey)
					// use utxo and burn
					err := b.Delete([]byte(utxokey))
					if err != nil {
						return err
					}
				}
			}

			// add lock tx to the map
			for i, output := range rawTx.TxOutput {
				utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(rawTx.Txid[:]), i)
				lockheight := height + output.LockTime

				if _, exist := c.lockUTXO[lockheight]; !exist {
					c.lockUTXO[lockheight] = make(map[string]*types.TxOutput)
				}
				output.BlockHeight = height
				c.lockUTXO[lockheight][utxokey] = &output

			}
		}
		// find utxo can be unlocked and add to utxo bucket

		if pendingUtxos, exist := c.lockUTXO[block.Header.Height]; exist {
			for s, output := range pendingUtxos {
				err := b.Put([]byte(s), output.EncodeToByte())
				if err != nil {
					return err
				}
			}
		}
		delete(c.lockUTXO, block.Header.Height)

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *ChainReader) TryUpdateTxForBlock(block *types.Block) error {
	db := c.GetBoltDb()
	txs := block.GetRawTx()
	//height := block.GetHeader().Height
	c.sync.Lock()
	defer c.sync.Unlock()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, rawTx := range txs {
			if !rawTx.IsCoinBase() {
				for _, input := range rawTx.TxInput {
					// find if there is corresponding utxo in the utxo
					utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(rawTx.Txid[:]), input.Voutput)
					outsBytes := b.Get([]byte(utxokey))
					if outsBytes == nil {
						return fmt.Errorf("update tx error for can't find corresponding utxo ")
					} else {
						b.Put([]byte(utxokey), outsBytes)
					}
					// TODO: add script check

					// use utxo and burn

				}
			}

			// add lock tx to the map
			// for i, output := range rawTx.TxOutput {
			// 	utxokey := fmt.Sprintf("%s:%d", rawTx.Txid, i)
			// 	lockheight := height + output.LockTime
			// 	c.sync.Lock()

			// 	if _, exist := c.lockUTXO[lockheight]; !exist {
			// 		c.lockUTXO[lockheight] = make(map[string]*types.TxOutput)
			// 	}
			// 	output.BlockHeight = height
			// 	c.lockUTXO[lockheight][utxokey] = &output
			// 	c.sync.Unlock()
			// }
		}
		// find utxo can be unlocked and add to utxo bucket
		// c.sync.Lock()
		// if _, exist := c.lockUTXO[block.Header.Height]; exist {
		// 	// for s, output := range pendingUtxos {
		// 	// 	err := b.Put([]byte(s), output.EncodeToByte())
		// 	// 	if err != nil {
		// 	// 		return err
		// 	// 	}
		// 	// }
		// }
		// //delete(c.lockUTXO, block.Header.Height)
		// c.sync.Unlock()
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
