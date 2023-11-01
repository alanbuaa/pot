package types

import (
	"bytes"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/config"
)

/*
ChainReader 用于获取有关当前PoT链的一系列状态
*/

type ChainReader struct {
	storage *PoTBlockStorage
	chain   map[uint64]*PoTBlock
	height  uint64
}

func NewChainReader(config *config.Config, storage *PoTBlockStorage) *ChainReader {
	return &ChainReader{
		storage: storage,
		chain:   make(map[uint64]*PoTBlock),
	}
}

func (c *ChainReader) SetHeight(height uint64, block *PoTBlock) {
	c.chain[height] = block
}

func (c *ChainReader) GetByHeight(height uint64) (*PoTBlock, error) {
	if c.chain[height] != nil {
		return c.chain[height], nil
	} else {
		return nil, fmt.Errorf("the height %d haven't set yet", height)
	}
}

func (c *ChainReader) GetParentOf(block *PoTBlock) (*PoTBlock, error) {
	header := block.Header
	height := header.Height
	parent, err := c.GetByHeight(height - 1)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(parent.Header.Hashes, block.Header.ParentHash) {
		return nil, fmt.Errorf("block's parent doesn't match, may get a wrong block")
	}
	return parent, nil
}

func (c *ChainReader) GetCurrentBlock() *PoTBlock {
	parent, err := c.GetByHeight(c.GetCurrentHeight())
	if err != nil {
		return parent
	}
	return nil
}

func (c *ChainReader) GetCurrentHeight() uint64 {
	return c.height

}

func (c *ChainReader) ValidateBlock(block *PoTBlock) bool {

	return true
}
