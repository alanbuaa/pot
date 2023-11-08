package types

import (
	"encoding/json"

	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

type HeaderStorage struct {
	db        *leveldb.DB
	heightmap map[uint64][][]byte
	vdfmap    map[uint64][]byte
}

func NewHeaderStorage(id int64) *HeaderStorage {
	sint := fmt.Sprintf("%d", id)
	db, err := leveldb.OpenFile("dbfile/node0-"+sint, nil)
	if err != nil {
		fmt.Println("output:", err)
		panic(err)
	}

	return &HeaderStorage{db: db, heightmap: make(map[uint64][][]byte), vdfmap: map[uint64][]byte{}}
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
		return nil, fmt.Errorf("not have block for height %d", height)
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
		headers = append(headers, ToHeader(block))
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
	s.vdfmap[epoch] = vdfres
	return err
}

func (s *HeaderStorage) GetPoTbyEpoch(epoch uint64) ([]byte, error) {
	//key := []byte(fmt.Sprintf("epoch:%d", epoch))
	//value, err := s.db.Get(key, nil)
	//if err != nil {
	//	return nil, err
	//}
	//return value, nil
	v := s.vdfmap[epoch]
	if v != nil {
		return v, nil
	} else {
		return nil, fmt.Errorf("didn't get height %d VDF result", epoch)
	}
}
