package types

import (
	"encoding/json"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
	"sync"
)

type BlockStorage struct {
	db *leveldb.DB

	vdfheight uint64
	rwmutex   *sync.RWMutex
}

func NewBlockStorage(id int64) *BlockStorage {
	sint := fmt.Sprintf("%d", id)
	db, err := leveldb.OpenFile("dbfile/node0-"+sint, nil)
	if err != nil {
		fmt.Println("output:", err)
		panic(err)
	}

	return &BlockStorage{
		db: db,

		vdfheight: 0,
		rwmutex:   new(sync.RWMutex),
	}
}

func (s *BlockStorage) Put(block *Block) error {
	header := block.GetHeader()
	protos := block.ToProto()
	b, err := proto.Marshal(protos)
	if err != nil {
		return err
	}

	hash := header.Hash()
	err = s.db.Put(hash, b, nil)
	if err != nil {
		return err
	}
	err = s.putHeightHash(header.Height, hash)
	return err
}

func (s *BlockStorage) Get(hash []byte) (*Block, error) {
	blockByte, err := s.db.Get(hash, nil)
	if err != nil {
		return nil, err
	}
	block := &pb.Block{}
	err = proto.Unmarshal(blockByte, block)

	if err != nil {
		return nil, err
	}
	return ToBlock(block), nil
}

func (s *BlockStorage) HasBlock(hash []byte) bool {
	blockByte, err := s.db.Get(hash, nil)

	if err != nil {
		return false
	}
	if blockByte != nil {
		return true
	} else {
		return false
	}
}

func (s *BlockStorage) GetbyHeight(height uint64) ([]*Block, error) {
	if height == 0 {
		return []*Block{DefaultGenesisBlock()}, nil
	}
	values, exists := s.getHeightHash(height)
	if exists == leveldb.ErrNotFound {
		return nil, fmt.Errorf("not have block for height %d", height)
	} else if exists != nil {
		return nil, exists
	}

	headers := make([]*Block, 0)
	for _, hash := range values {
		pbheader, err := s.db.Get(hash, nil)
		if err != nil {
			return nil, err
		}
		block := &pb.Block{}
		err = proto.Unmarshal(pbheader, block)
		if err != nil {
			return nil, err
		}
		headers = append(headers, ToBlock(block))
	}
	return headers, nil
}

func (s *BlockStorage) getHeightHash(height uint64) ([][]byte, error) {

	keys := []byte(fmt.Sprintf("height:%d", height))

	values, err := s.db.Get(keys, nil)
	if err != nil {
		return nil, err
	}
	var decodeData [][]byte
	err = json.Unmarshal(values, &decodeData)
	if err != nil {
		return nil, err
	}
	return decodeData, nil
}

func (s *BlockStorage) putHeightHash(height uint64, hash []byte) error {
	data, err := s.getHeightHash(height)
	if err != leveldb.ErrNotFound && err != nil {
		return err
	}
	if len(data) != 0 {
		data = append(data, hash)
	} else {
		data = [][]byte{hash}
	}
	encodedata, err := json.Marshal(data)
	if err != nil {
		return err
	}
	keys := []byte(fmt.Sprintf("height:%d", height))
	err = s.db.Put(keys, encodedata, nil)
	return err
}

func (s *BlockStorage) SetVDFres(epoch uint64, vdfres []byte) error {
	key := []byte(fmt.Sprintf("epoch:%d", epoch))
	err := s.db.Put(key, vdfres, nil)
	if epoch > s.vdfheight {
		s.rwmutex.Lock()
		s.vdfheight = epoch
		s.rwmutex.Unlock()
	}
	return err
}

func (s *BlockStorage) GetVDFresbyEpoch(epoch uint64) ([]byte, error) {
	key := []byte(fmt.Sprintf("epoch:%d", epoch))
	value, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *BlockStorage) GetVDFHeight() uint64 {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()
	return s.vdfheight
}
