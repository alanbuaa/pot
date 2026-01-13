package pot

import (
	"container/list"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
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

type WrappedBciReward struct {
	Bcireward *BciReward
	proposed  bool
}

type BciReward struct {
	Address []byte
	Amount  int64
	Proof   BciProof
	BciType int32
	weight  float64
	DoDraw  bool
}

type BciProof struct {
	Height    uint64
	BlockHash []byte
	TxHash    []byte
}

func (d *BciReward) ToProto() *pb.BciReward {
	return &pb.BciReward{
		Address: d.Address,
		Amount:  d.Amount,
		ChainID: d.BciType,
		BciProof: &pb.BciProof{
			Height:    d.Proof.Height,
			BlockHash: d.Proof.BlockHash,
			TxHash:    d.Proof.TxHash,
		},
		BciType: d.BciType,
	}
}

func ToBciReward(proof *pb.BciReward) *BciReward {
	return &BciReward{
		Address: proof.GetAddress(),
		Amount:  proof.GetAmount(),
		BciType: proof.GetBciType(),
		Proof: BciProof{
			Height:    proof.GetBciProof().GetHeight(),
			BlockHash: proof.GetBciProof().GetBlockHash(),
			TxHash:    proof.GetBciProof().GetTxHash(),
		},
	}
}

type Mempool struct {
	mutex         *sync.RWMutex
	BciRewardPool map[string]*WrappedBciReward
	execorder     *list.List
	execset       map[[crypto.Hashlen]byte]*list.Element // 存放已执行的区块
	raworder      *list.List
	rawset        map[[crypto.Hashlen]byte]*list.Element
	rawmap        map[[crypto.Hashlen]byte][]byte
}

func NewMempool() *Mempool {
	c := &Mempool{
		mutex:         new(sync.RWMutex),
		BciRewardPool: make(map[string]*WrappedBciReward),
		execorder:     new(list.List),
		execset:       make(map[[crypto.Hashlen]byte]*list.Element),
		raworder:      new(list.List),
		rawset:        make(map[[crypto.Hashlen]byte]*list.Element),
		rawmap:        make(map[[crypto.Hashlen]byte][]byte),
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

func (c *Mempool) GetBlockByHash(txHash [crypto.Hashlen]byte) *types.ExecutedBlock {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if e, ok := c.execset[txHash]; ok {
		return e.Value.(*WrappedExcutedTx).executedBlock
	} else {
		return nil
	}
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

func (c *Mempool) RemoveALl() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for hash, element := range c.execset {
		c.execorder.Remove(element)
		delete(c.execset, hash)
	}
}

// GetFirstN return the top Commitees unused commands from the list
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

func (c *Mempool) HasBciReward(reward *BciReward) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	strings := fmt.Sprintf(hexutil.Encode(reward.Address)+"-%d", reward.Amount)
	_, ok := c.BciRewardPool[strings]
	return ok
}

func (c *Mempool) HasBciRewardByCoinbaseProof(coinbaseproof *types.CoinbaseProof) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	strings := fmt.Sprintf("%s", hexutil.Encode(coinbaseproof.TxHash)+"-"+hexutil.Encode(coinbaseproof.Address))
	_, ok := c.BciRewardPool[strings]
	return ok
}

func (c *Mempool) AddBciReward(rewards ...*BciReward) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, reward := range rewards {
		strings := hexutil.Encode(reward.Proof.TxHash) + "-" + hexutil.Encode(reward.Address)

		_, ok := c.BciRewardPool[strings]
		if ok {
			continue
		} else {
			c.BciRewardPool[strings] = &WrappedBciReward{
				Bcireward: reward,
				proposed:  false,
			}
		}

	}

}

func (c *Mempool) MarkBciRewardProposed(coinbaseProofs []types.CoinbaseProof) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, proof := range coinbaseProofs {
		strings := hexutil.Encode(proof.TxHash) + "-" + hexutil.Encode(proof.Address)
		if _, ok := c.BciRewardPool[strings]; ok {
			c.BciRewardPool[strings].proposed = true
		}
	}
}

func (c *Mempool) RemoveBciRewardByTxHash(hash []byte, Address []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	str := hexutil.Encode(hash) + "-" + hexutil.Encode(Address)

	delete(c.BciRewardPool, str)
}

func (c *Mempool) GetAllBciRewards() []*BciReward {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	rewards := make([]*BciReward, 0)
	for _, reward := range c.BciRewardPool {
		if !reward.proposed {
			rewards = append(rewards, reward.Bcireward)
		}
	}

	//c.BciRewardPool = make(map[string]*BciReward)
	return rewards
}

// GetSize returns the total size and marked transaction count
func (c *Mempool) GetSize() (int, int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	totalSize := c.raworder.Len()
	markedCount := 0
	for e := c.raworder.Front(); e != nil; e = e.Next() {
		wrappedTx := e.Value.(*WrappedRawTx)
		if wrappedTx.proposed {
			markedCount++
		}
	}
	return totalSize, markedCount
}

// GetRecentTxs returns the most recent N transactions
func (c *Mempool) GetRecentTxs(n int) []*types.RawTx {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	txs := make([]*types.RawTx, 0, n)
	count := 0
	for e := c.raworder.Back(); e != nil && count < n; e = e.Prev() {
		wrappedTx := e.Value.(*WrappedRawTx)
		txs = append(txs, wrappedTx.rawtx)
		count++
	}
	return txs
}

// GetAllTxs returns all transactions in the mempool
func (c *Mempool) GetAllTxs() []*types.RawTx {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	txs := make([]*types.RawTx, 0, c.raworder.Len())
	for e := c.raworder.Front(); e != nil; e = e.Next() {
		wrappedTx := e.Value.(*WrappedRawTx)
		txs = append(txs, wrappedTx.rawtx)
	}
	return txs
}
