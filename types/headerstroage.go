package types

import (
	"encoding/json"
	"sync"

	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

type HeaderStorage struct {
	db        *leveldb.DB
	heightmap map[uint64][][]byte
	vdfmap    map[uint64][]byte
	synclock  *sync.RWMutex
}

func NewHeaderStorage(id int64) *HeaderStorage {
	sint := fmt.Sprintf("%d", id)

	db, err := leveldb.OpenFile("dbfile/node0-"+sint, nil)
	if err != nil {
		fmt.Println("output:", err)
		panic(err)
	}

	return &HeaderStorage{db: db, heightmap: make(map[uint64][][]byte), vdfmap: map[uint64][]byte{}, synclock: new(sync.RWMutex)}
}

func (s *HeaderStorage) Put(header *Header) error {
	protos := header.ToProto()
	b, err := proto.Marshal(protos)
	if err != nil {
		return err
	}

	hash := header.Hash()
	err = s.db.Put(hash, b, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = s.putHeightHash(header.Height, hash)
	return err
}

func (s *HeaderStorage) Get(hash []byte) (*Header, error) {
	blockByte, err := s.db.Get(hash, nil)
	if err != nil {
		return nil, err
	}
	block := &pb.Header{}
	err = proto.Unmarshal(blockByte, block)

	if err != nil {
		return nil, err
	}
	return ToHeader(block), nil
}

func (s *HeaderStorage) HasBlock(hash []byte) bool {
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

func (s *HeaderStorage) GetbyHeight(height uint64) ([]*Header, error) {
	if height == 0 {
		return []*Header{DefaultGenesisHeader()}, nil
	}
	values, exists := s.getHeightHash(height)
	if exists == leveldb.ErrNotFound {
		return nil, fmt.Errorf("[Headerstorage]\tnot have block for height %d", height)
	} else if exists != nil {
		return nil, exists
	}

	headers := make([]*Header, 0)
	for _, hash := range values {
		pbheader, err := s.db.Get(hash, nil)
		if err != nil {
			return nil, err
		}
		block := &pb.Header{}
		err = proto.Unmarshal(pbheader, block)
		if err != nil {
			return nil, err
		}

		header := ToHeader(block)
		headers = append(headers, header)
	}
	return headers, nil
}

func (s *HeaderStorage) getHeightHash(height uint64) ([][]byte, error) {
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

func (s *HeaderStorage) putHeightHash(height uint64, hash []byte) error {
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

func (s *HeaderStorage) SetVdfRes(epoch uint64, vdfres []byte) error {
	key := []byte(fmt.Sprintf("epoch:%d", epoch))
	err := s.db.Put(key, vdfres, nil)
	s.synclock.Lock()
	s.vdfmap[epoch] = vdfres
	defer s.synclock.Unlock()
	return err
}

func (s *HeaderStorage) GetPoTbyEpoch(epoch uint64) ([]byte, error) {
	//key := []byte(fmt.Sprintf("epoch:%d", epoch))
	//value, err := s.db.Get(key, nil)
	//if err != nil {
	//	return nil, err
	//}
	//return value, nil
	s.synclock.RLock()
	v := s.vdfmap[epoch]
	defer s.synclock.RUnlock()
	if v != nil {
		return v, nil
	} else {
		return nil, fmt.Errorf("[Headerstorage]\tdidn't get height %d VDF result", epoch)
	}
}
