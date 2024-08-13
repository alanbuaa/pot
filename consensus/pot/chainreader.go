package pot

import (
	"bytes"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"log"
	"sync"
)

/*
ChainReader is used to store chain state for the pot chain
*/

type ChainReader struct {
	storage *types.BlockStorage
	chain   map[uint64]*types.Block
	height  uint64
	sync    *sync.RWMutex
}

func NewChainReader(storage *types.BlockStorage) *ChainReader {
	c := &ChainReader{
		storage: storage,
		chain:   make(map[uint64]*types.Block),
		height:  0,
		sync:    new(sync.RWMutex),
	}
	c.chain[0] = types.DefaultGenesisBlock()
	return c
}

func (c *ChainReader) SetHeight(height uint64, block *types.Block) {
	c.sync.Lock()
	defer c.sync.Unlock()
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
	if !bytes.Equal(current.GetHeader().Hashes, block.GetHeader().ParentHash) {
		return false
	}
	return true
}

func (c *ChainReader) FindUnspentTransactions(address []byte) []*types.RawTx {
	unspentTxs := make([]*types.RawTx, 0)
	spentsUTXOs := make(map[[32]byte][]int64)
	height := c.GetCurrentHeight()
	for height >= 0 {
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
				if output.CanBeUnlockWith(address) {
					unspentTxs = append(unspentTxs, rawtx)
				}
			}

			if rawtx.IsCoinBase() == false {
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

func (c *ChainReader) FindUTXO(address []byte) []types.TxOutput {
	utxos := make([]types.TxOutput, 0)
	//unspenttransactions := c.FindUnspentTransactions(address)
	//
	//for _, unspenttransaction := range unspenttransactions {
	//	for _, output := range unspenttransaction.TxOutput {
	//		if output.CanBeUnlockWith(address) {
	//			utxos = append(utxos, output)
	//		}
	//	}
	//}
	db := c.GetBoltDb()
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		c := b.Cursor()
		count := 0
		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := types.DecodeByte2Outputs(v)

			for _, out := range outs {
				if out.IsLockedWithKey(address) {
					utxos = append(utxos, out)
				}
			}
			count += 1
		}
		//fmt.Println(count)
		return nil
	})
	if err != nil {
		return nil
	}

	return utxos
}

func (c *ChainReader) GetBalance(address []byte) int64 {
	balance := int64(0)

	utxos := c.FindUTXO(address)

	for _, utxo := range utxos {
		balance += utxo.Value
	}
	return balance

}

func (c *ChainReader) FindAllUnspentOutputs() map[[32]byte]types.TxOutputs {
	UTXO := make(map[[32]byte]types.TxOutputs)
	spentTXOs := make(map[[32]byte][]int64)

	height := c.GetCurrentHeight()

	for height >= 0 {
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

			if tx.IsCoinBase() == false {
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
				if out.CanBeUnlockWith(pubkey) && accumulated < amount {
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
		if err != nil && err != bolt.ErrBucketNotFound {
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
	txs := block.GetRawTx()
	db := c.GetBoltDb()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))

		for _, tx := range txs {

			err := b.Delete(tx.Txid[:])
			if err != nil {
				return err
			}

			if !tx.IsCoinBase() {
				for _, input := range tx.TxInput {
					updateOut := types.TxOutputs{}
					txid := input.Txid
					height := block.Header.Height
					for height >= 0 {
						heighttxs := block.GetRawTx()
						for _, heighttx := range heighttxs {
							if txid == heighttx.Txid {
								out := heighttx.TxOutput[input.Voutput]
								updateOut = append(updateOut, out)
								break
							}
						}
					}
					outsBytes := b.Get(input.Txid[:])
					outs := types.DecodeByte2Outputs(outsBytes)
					for _, out := range outs {
						updateOut = append(updateOut, out)
					}
					if len(updateOut) == 0 {
						err := b.Delete(input.Txid[:])
						if err != nil {
							return err
						}
					} else {
						err := b.Put(input.Txid[:], updateOut.EncodeTxOutputs2Byte())
						if err != nil {
							return err
						}
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

func (c *ChainReader) UpdateTxForBlock(block *types.Block) error {
	db := c.GetBoltDb()
	txs := block.GetRawTx()

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, rawTx := range txs {
			if rawTx.IsCoinBase() == false {
				for _, input := range rawTx.TxInput {
					var updateOuts types.TxOutputs
					outsBytes := b.Get(input.Txid[:])
					outs := types.DecodeByte2Outputs(outsBytes)

					for i, out := range outs {
						if int64(i) != input.Voutput {
							updateOuts = append(updateOuts, out)
						}
					}

					if len(updateOuts) == 0 {
						err := b.Delete(input.Txid[:])
						if err != nil {
							return err
						}
					} else {
						err := b.Put(input.Txid[:], updateOuts.EncodeTxOutputs2Byte())
						if err != nil {
							return err
						}
					}

				}
			}
			newouts := make(types.TxOutputs, 0)

			for _, output := range rawTx.TxOutput {
				//fmt.Println(hexutil.Encode(output.Address), output.Value)
				//fmt.Println(output.Address)
				newouts = append(newouts, output)
			}
			err := b.Put(rawTx.Txid[:], newouts.EncodeTxOutputs2Byte())
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

func (c *ChainReader) UpdateBlock(block *types.Block) error {
	txs := block.GetRawTx()
	db := c.GetBoltDb()
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(types.UTXOBucket))
		for _, tx := range txs {
			if tx.IsCoinBase() == false {
				for _, input := range tx.TxInput {
					updateOut := types.TxOutputs{}
					outsBytes := b.Get(input.Txid[:])

					outs := types.DecodeByte2Outputs(outsBytes)

					for i, out := range outs {
						if int64(i) != input.Voutput {
							updateOut = append(updateOut, out)
						}
					}

					if len(updateOut) == 0 {
						err := b.Delete(input.Txid[:])
						if err != nil {
							return err
						}
					} else {
						err := b.Put(input.Txid[:], updateOut.EncodeTxOutputs2Byte())
						if err != nil {
							return err
						}
					}
				}
			}

			newoutputs := types.TxOutputs{}
			for _, output := range tx.TxOutput {
				newoutputs = append(newoutputs, output)
			}

			err := b.Put(tx.Txid[:], newoutputs.EncodeTxOutputs2Byte())
			if err != nil {
				return err
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
