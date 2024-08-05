package pot

import (
	"container/list"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"

	"sync"
)

type WrappedExcutedTx struct {
	executedBlock *types.ExecutedBlock
	proposed      bool
}

type WrappedRawTx struct {
	rawtx    *types.RawTx
	proposed bool
}

type DciReward struct {
	Address []byte
	Amount  int64
	Proof   DciProof
	ChainID int64
	weight  float64
}
type DciProof struct {
	Epoch  int64
	Height int64
	Hash   []byte
}

func (d *DciReward) ToProto() *pb.DciReward {
	return &pb.DciReward{
		Address: d.Address,
		Amount:  d.Amount,
		ChainID: d.ChainID,
		DciProof: &pb.DciProof{
			Epoch:  d.Proof.Epoch,
			Height: d.Proof.Height,
			Hash:   d.Proof.Hash,
		},
	}
}

func ToDciReward(proof *pb.DciReward) *DciReward {
	return &DciReward{
		Address: proof.GetAddress(),
		Amount:  proof.GetAmount(),
		ChainID: proof.GetChainID(),
		Proof: DciProof{
			Epoch:  proof.GetDciProof().GetEpoch(),
			Height: proof.GetDciProof().GetHeight(),
			Hash:   proof.GetDciProof().GetHash(),
		},
	}
}

type Mempool struct {
	mutex         *sync.RWMutex
	DciRewardPool map[string]*DciReward
	execorder     *list.List
	execset       map[[crypto.Hashlen]byte]*list.Element
	raworder      *list.List
	rawset        map[[crypto.Hashlen]byte]*list.Element
}

func NewMempool() *Mempool {
	c := &Mempool{
		mutex:         new(sync.RWMutex),
		DciRewardPool: make(map[string]*DciReward),
		execorder:     new(list.List),
		execset:       make(map[[crypto.Hashlen]byte]*list.Element),
		raworder:      new(list.List),
		rawset:        make(map[[crypto.Hashlen]byte]*list.Element),
	}
	c.execorder.Init()
	c.raworder.Init()
	return c
}

func (c *Mempool) Has(blocks *types.ExecutedBlock) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	e, ok := c.execset[blocks.Hash()]
	block := e.Value.(*WrappedExcutedTx).executedBlock
	if block.Txs == nil {
		block.Txs = blocks.Txs
	}
	return ok
}

func (c *Mempool) Add(blocks ...*types.ExecutedBlock) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, block := range blocks {
		txHash := block.Hash()
		// avoid duplication

		if _, ok := c.execset[txHash]; ok {
			continue
		}
		e := c.execorder.PushBack(&WrappedExcutedTx{
			executedBlock: block,
			proposed:      false,
		})
		c.execset[txHash] = e
	}
}

// Remove commands from execset and list
func (c *Mempool) Remove(blocks []types.ExecutedBlock) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, block := range blocks {
		txHash := block.Hash()
		if e, ok := c.execset[txHash]; ok {
			c.execorder.Remove(e)
			delete(c.execset, txHash)
		}
	}
}

// GetFirstN return the top n unused commands from the list
func (c *Mempool) GetFirstN(n int) []*types.ExecutedBlock {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.execset) == 0 {
		return nil
	}
	txs := make([]*types.ExecutedBlock, 0, n)
	i := 0
	// get the first element of list
	e := c.execorder.Front()
	for i < n {
		if e == nil {
			break
		}
		if wrtx := e.Value.(*WrappedExcutedTx); !wrtx.proposed {
			txs = append(txs, wrtx.executedBlock)
			i++
		}
		e = e.Next()
	}
	return txs
}

func (c *Mempool) IsProposed(tx *types.ExecutedBlock) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if e, ok := c.execset[tx.Hash()]; ok {
		return e.Value.(*WrappedExcutedTx).proposed
	}
	return false
}

// MarkProposed will mark the given commands as proposed and move them to the back of the queue
func (c *Mempool) MarkProposed(txs []*types.ExecutedBlock) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		txHash := tx.Hash()
		if e, ok := c.execset[txHash]; ok {
			e.Value.(*WrappedExcutedTx).proposed = true
			// Move to back so that it's not immediately deleted by a call to TrimToLen()
			c.execorder.MoveToBack(e)
		} else {
			// new executedBlock, store it to back
			e := c.execorder.PushBack(&WrappedExcutedTx{executedBlock: tx, proposed: true})
			c.execset[txHash] = e
		}
	}
}

// MarkProposedByHeader will mark the given commands as proposed and move them to the back of the queue
func (c *Mempool) MarkProposedByHeader(txs []*types.ExecuteHeader) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		txHash := tx.Hash()
		if e, ok := c.execset[txHash]; ok {
			e.Value.(*WrappedExcutedTx).proposed = true
			// Move to back so that it's not immediately deleted by a call to TrimToLen()
			c.execorder.MoveToBack(e)
		} else {
			// new executedBlock, store it to back
			e := c.execorder.PushBack(&WrappedExcutedTx{executedBlock: &types.ExecutedBlock{
				Header: tx,
				Txs:    nil,
			}, proposed: true})
			c.execset[txHash] = e
		}
	}
}

func (c *Mempool) UnMarkByHeader(headers []*types.ExecuteHeader) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range headers {
		if e, ok := c.execset[tx.Hash()]; ok {
			e.Value.(*WrappedExcutedTx).proposed = false
			c.execorder.MoveToFront(e)
		}
	}
}

func (c *Mempool) UnMark(txs []*types.ExecutedBlock) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		if e, ok := c.execset[tx.Hash()]; ok {
			block := e.Value.(*WrappedExcutedTx)
			e.Value.(*WrappedExcutedTx).proposed = false
			if block.executedBlock.Txs == nil {
				block.executedBlock.Txs = tx.Txs
			}
			c.execorder.MoveToFront(e)
		}
	}
}

func (c *Mempool) HasRawTx(tx *types.RawTx) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.rawset[tx.Hash()]
	return ok
}

func (c *Mempool) AddRawTx(txs ...*types.RawTx) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, tx := range txs {
		txHash := tx.Hash()
		// avoid duplication

		if _, ok := c.rawset[txHash]; ok {
			continue
		}
		e := c.raworder.PushBack(&WrappedRawTx{
			rawtx:    tx,
			proposed: false,
		})
		c.rawset[txHash] = e
	}
}

func (c *Mempool) RemoveRawTx(txs []*types.RawTx) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, tx := range txs {
		txHash := tx.Hash()
		if e, ok := c.rawset[txHash]; ok {
			c.raworder.Remove(e)
			delete(c.rawset, txHash)
		}
	}
}

func (c *Mempool) GetRawTx() []*types.RawTx {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.rawset) == 0 {
		return nil
	}
	txs := make([]*types.RawTx, 0)
	e := c.raworder.Front()
	for true {
		if e == nil {
			break
		}
		if wrtx := e.Value.(*WrappedRawTx); !wrtx.proposed {
			txs = append(txs, wrtx.rawtx)
		}
		e = e.Next()
	}
	return txs
}

func (c *Mempool) IsRawTxProposed(tx *types.RawTx) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if e, ok := c.rawset[tx.Hash()]; ok {
		return e.Value.(*WrappedRawTx).proposed
	}
	return false
}
func (c *Mempool) MarkRawTxProposed(txs []*types.RawTx) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		txHash := tx.Hash()
		if e, ok := c.rawset[txHash]; ok {
			e.Value.(*WrappedRawTx).proposed = true
			// Move to back so that it's not immediately deleted by a call to TrimToLen()
			c.raworder.MoveToBack(e)
		} else {
			// new executedBlock, store it to back
			e := c.raworder.PushBack(&WrappedRawTx{rawtx: tx, proposed: true})
			c.rawset[txHash] = e
		}
	}
}

func (c *Mempool) UnmarkRawTx(txs []*types.RawTx) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		if e, ok := c.rawset[tx.Hash()]; ok {
			e.Value.(*WrappedRawTx).proposed = false
			c.raworder.MoveToFront(e)
		}
	}
}

func (c *Mempool) HasDciReward(reward *DciReward) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	strings := fmt.Sprintf(hexutil.Encode(reward.Address)+"-%d", reward.Amount)
	_, ok := c.DciRewardPool[strings]
	return ok
}

func (c *Mempool) AddDciReward(rewards ...*DciReward) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, reward := range rewards {
		strings := fmt.Sprintf(hexutil.Encode(reward.Address)+"-%d", reward.Amount)
		_, ok := c.DciRewardPool[strings]
		if ok {
			continue
		} else {
			c.DciRewardPool[strings] = reward
		}

	}

}

func (c *Mempool) RemoveDciReward(rewards ...*DciReward) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, reward := range rewards {
		strings := fmt.Sprintf(hexutil.Encode(reward.Address)+"-%d", reward.Amount)
		if _, ok := c.DciRewardPool[strings]; ok {
			delete(c.DciRewardPool, strings)
		}
	}
}

func (c *Mempool) GetAllDciRewards() []*DciReward {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	rewards := make([]*DciReward, 0)
	for _, reward := range c.DciRewardPool {
		rewards = append(rewards, reward)
	}

	c.DciRewardPool = make(map[string]*DciReward)
	return rewards
}
