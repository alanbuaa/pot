package types

import (
	"bytes"
	"fmt"
	"sync"
)

/*
ChainReader is used to store chain state for the pot chain
*/

type ChainReader struct {
	storage *BlockStorage
	chain   map[uint64]*Block
	height  uint64
	sync    *sync.RWMutex
}

func NewChainReader(storage *BlockStorage) *ChainReader {
	c := &ChainReader{
		storage: storage,
		chain:   make(map[uint64]*Block),
		height:  0,
		sync:    new(sync.RWMutex),
	}
	c.chain[0] = DefaultGenesisBlock()
	return c
}

func (c *ChainReader) SetHeight(height uint64, block *Block) {
	c.sync.Lock()
	defer c.sync.Unlock()
	c.chain[height] = block
	if height > c.height {
		c.height = height
	}
}

func (c *ChainReader) GetByHeight(height uint64) (*Block, error) {
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

func (c *ChainReader) GetCurrentBlock() *Block {
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

func (c *ChainReader) ValidateBlock(block *Header) bool {

	return true
}

func (c *ChainReader) IsBehindCurrent(block *Block) bool {
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
