package pot

import (
	"container/list"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"sync"
)

type WrappedTx struct {
	executedTxData *types.ExecutedTxData
	proposed       bool
}

type Mempool struct {
	mutex *sync.RWMutex
	order *list.List
	set   map[[crypto.Hashlen]byte]*list.Element
}

func NewMempool() *Mempool {
	c := &Mempool{
		mutex: new(sync.RWMutex),
		order: new(list.List),
		set:   make(map[[crypto.Hashlen]byte]*list.Element),
	}
	c.order.Init()
	return c
}

func (c *Mempool) Has(tx *types.ExecutedTxData) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, ok := c.set[tx.Hash()]
	return ok
}

func (c *Mempool) Add(txs ...*types.ExecutedTxData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, tx := range txs {
		txHash := tx.Hash()
		// avoid duplication

		if _, ok := c.set[txHash]; ok {
			continue
		}
		e := c.order.PushBack(&WrappedTx{
			executedTxData: tx,
			proposed:       false,
		})
		c.set[txHash] = e
	}
}

// Remove commands from set and list
func (c *Mempool) Remove(txs []types.ExecutedTxData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, tx := range txs {
		txHash := tx.Hash()
		if e, ok := c.set[txHash]; ok {
			c.order.Remove(e)
			delete(c.set, txHash)
		}
	}
}

// GetFirst return the top n unused commands from the list
func (c *Mempool) GetFirstN(n int) []*types.ExecutedTxData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.set) == 0 {
		return nil
	}
	txs := make([]*types.ExecutedTxData, 0, n)
	i := 0
	// get the first element of list
	e := c.order.Front()
	for i < n {
		if e == nil {
			break
		}
		if wrtx := e.Value.(*WrappedTx); !wrtx.proposed {
			txs = append(txs, wrtx.executedTxData)
			i++
		}
		e = e.Next()
	}
	return txs
}

func (c *Mempool) IsProposed(tx *types.ExecutedTxData) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if e, ok := c.set[tx.Hash()]; ok {
		return e.Value.(*WrappedTx).proposed
	}
	return false
}

// MarkProposed will mark the given commands as proposed and move them to the back of the queue
func (c *Mempool) MarkProposed(txs []*types.ExecutedTxData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		txHash := tx.Hash()
		if e, ok := c.set[txHash]; ok {
			e.Value.(*WrappedTx).proposed = true
			// Move to back so that it's not immediately deleted by a call to TrimToLen()
			c.order.MoveToBack(e)
		} else {
			// new executedTxData, store it to back
			e := c.order.PushBack(&WrappedTx{executedTxData: tx, proposed: true})
			c.set[txHash] = e
		}
	}
}

func (c *Mempool) UnMark(txs []*types.ExecutedTxData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, tx := range txs {
		if e, ok := c.set[tx.Hash()]; ok {
			e.Value.(*WrappedTx).proposed = false
			c.order.MoveToFront(e)
		}
	}
}
