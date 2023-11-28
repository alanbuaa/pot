package types

import (
	"bytes"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
)

/*
ChainReader is used to store chain state for the pot chain
*/

type ChainReader struct {
	storage *HeaderStorage
	chain   map[uint64]*Header
	height  uint64
	sync    *sync.RWMutex
}

func NewChainReader(storage *HeaderStorage) *ChainReader {
	c := &ChainReader{
		storage: storage,
		chain:   make(map[uint64]*Header),
		height:  0,
		sync:    new(sync.RWMutex),
	}
	c.chain[0] = DefaultGenesisHeader()
	return c
}

func (c *ChainReader) SetHeight(height uint64, block *Header) {
	c.sync.Lock()
	defer c.sync.Unlock()
	c.chain[height] = block
	if height > c.height {
		c.height = height
	}
}

func (c *ChainReader) GetByHeight(height uint64) (*Header, error) {
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

func (c *ChainReader) GetParentOf(block *Header) (*Header, error) {
	header := block
	height := header.Height
	parent, err := c.GetByHeight(height - 1)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(parent.Hashes, block.ParentHash) {
		return nil, fmt.Errorf("block's parent doesn't match, may get a wrong block")
	}
	return parent, nil
}

func (c *ChainReader) GetCurrentBlock() *Header {
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

func (c *ChainReader) GetSharedAncestor(block *Header) (*Header, error) {
	current := c.GetCurrentBlock()
	currentheight := c.GetCurrentHeight()
	header := block

	if header.Height == currentheight {
		if bytes.Equal(current.Hashes, header.Hashes) {
			return current, nil
		}
		for {
			current, err := c.GetByHeight(current.Height - 1)
			if err != nil {
				return nil, err
			}
			header, err := c.storage.Get(header.ParentHash)
			if err == leveldb.ErrNotFound {
				return nil, err
			}
			if bytes.Equal(current.Hashes, header.Hashes) {
				return current, nil
			}
		}
	}
	if header.Height > currentheight {
		headerahead, err := c.storage.Get(header.ParentHash)
		if err != nil {
			return nil, err
		}
		return c.GetSharedAncestor(headerahead)
	}
	if header.Height < currentheight {
		current, err := c.GetByHeight(header.Height)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(current.Hashes, header.Hashes) {
			return current, nil
		}
		for {
			current, err := c.GetByHeight(current.Height - 1)
			if err != nil {
				return nil, err
			}
			header, err := c.storage.Get(header.ParentHash)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(current.Hashes, header.Hashes) {
				return current, nil
			}
		}
	}
	return nil, fmt.Errorf("get ancestor error for unknown end")
}

func (c *ChainReader) IsBehindCurrent(header *Header) bool {
	currentheight := c.GetCurrentHeight()
	if currentheight == 0 {
		return true
	}
	if currentheight+1 != header.Height {
		return false
	}
	current := c.GetCurrentBlock()
	if !bytes.Equal(current.Hashes, header.ParentHash) {
		return false
	}
	return true
}
