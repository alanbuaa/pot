package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

/*
用于存储和查找区块信息
*/
type BlockStorage interface {
	Put(block *pb.Block) error
	Get(hash []byte) (*pb.Block, error)
	UpdateState(block *pb.Block) error
	BlockOf(cert *pb.QuorumCert) (*pb.Block, error)
	ParentOf(block *pb.Block) (*pb.Block, error)
	GetLastBlockHash() []byte
	RestoreStatus()
	Close()
}

func Hash(block *pb.Block) []byte {
	// 防止重复生成哈希
	if block.Hash != nil {
		return block.Hash
	}
	hasher := sha256.New()

	hasher.Write(block.ParentHash)

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, block.Height)
	hasher.Write(height)

	for _, tx := range block.Txs {
		hasher.Write(tx)
	}

	qcByte, _ := proto.Marshal(block.Justify)
	hasher.Write(qcByte)
	blockHash := hasher.Sum(nil)
	return blockHash
}

func String(block *pb.Block) string {
	return fmt.Sprintf("\n[BLOCK]\nParentHash: %s\nHash: %s\nHeight: %d\n",
		hex.EncodeToString(block.ParentHash), hex.EncodeToString(block.Hash), block.Height)
}

type BlockStorageImpl struct {
	db  *leveldb.DB
	Tip []byte
}

func NewBlockStorageImpl(id string) *BlockStorageImpl {
	db, err := leveldb.OpenFile("dbfile/node"+id, nil)
	if err != nil {
		fmt.Println("output", err)
		panic(err)
	}

	return &BlockStorageImpl{
		db:  db,
		Tip: nil,
	}
}

func (bsi *BlockStorageImpl) Put(block *pb.Block) error {
	marshal, _ := proto.Marshal(block)
	err := bsi.db.Put(block.Hash, marshal, nil)
	if err != nil {
		return err
	}
	err = bsi.db.Put([]byte("l"), block.Hash, nil)
	bsi.Tip = block.Hash
	return err
}

func (bsi *BlockStorageImpl) Get(hash []byte) (*pb.Block, error) {
	blockByte, err := bsi.db.Get(hash, nil)
	if err != nil {
		return nil, err
	}
	block := &pb.Block{}
	err = proto.Unmarshal(blockByte, block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (bsi *BlockStorageImpl) BlockOf(cert *pb.QuorumCert) (*pb.Block, error) {
	blockBytes, err := bsi.db.Get(cert.BlockHash, nil)
	if err != nil {
		return nil, err
	}
	b := &pb.Block{}
	err = proto.Unmarshal(blockBytes, b)
	if err != nil {
		return nil, err
	}
	return b, err
}

func (bsi *BlockStorageImpl) ParentOf(block *pb.Block) (*pb.Block, error) {
	bytes, err := bsi.db.Get(block.ParentHash, nil)
	if err != nil {
		return nil, err
	}
	parentBlock := &pb.Block{}
	err = proto.Unmarshal(bytes, parentBlock)
	if err != nil {
		return nil, err
	}

	return parentBlock, err
}

func (bsi *BlockStorageImpl) GetLastBlockHash() []byte {
	return bsi.Tip
}

func (bsi *BlockStorageImpl) RestoreStatus() {
	latestBlockHash, _ := bsi.db.Get([]byte("l"), nil)
	if latestBlockHash == nil {
		return
	}
	bsi.Tip = latestBlockHash

}

func (bsi *BlockStorageImpl) UpdateState(block *pb.Block) error {
	if block == nil || block.Hash == nil {
		return errors.New("block is null")
	}
	block.Committed = true
	marshal, _ := proto.Marshal(block)
	err := bsi.db.Put(block.Hash, marshal, nil)
	if err != nil {
		return err
	}
	return nil
}

func (bsi *BlockStorageImpl) Close() {
	bsi.db.Close()
}
