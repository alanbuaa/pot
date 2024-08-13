package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
	"sync"
)

const blocksBucket = "blocks"
const UTXOBucket = "chainstate"
const ExecutedBucket = "Executed"

var (
	ErrHeightHashNotFound = errors.New("height hash not found")
)

type BlockStorage struct {
	//db        *leveldb.DB
	boltdb    *bolt.DB
	vdfheight uint64
	rwmutex   *sync.RWMutex
}

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (s *BlockStorage) GetBoltdb() *bolt.DB {
	return s.boltdb
}
func NewBlockStorage(id int64) *BlockStorage {
	sint := fmt.Sprintf("%d", id)

	boltdbname := fmt.Sprintf("boltdbfile/node0-" + sint)
	if dbExists(boltdbname) {
		err := os.Remove(boltdbname)
		if err != nil {
			panic(err)
		}
	}
	boltdb, err := bolt.Open("boltdbfile/node0-"+sint, 0600, nil)
	if err != nil {
		fmt.Println("create boltdb err for ", err)
		log.Panic(err)
	}

	err = boltdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte(UTXOBucket))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte(ExecutedBucket))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Println("create bolt bucket err for ", err)
		log.Panic(err)
	}
	return &BlockStorage{
		boltdb: boltdb,
		//db:        db,
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
	//err = s.db.Put(hash, b, nil)
	//if err != nil {
	//	return err
	//}

	err = s.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(blocksBucket))
		blockInb := bucket.Get(hash)
		if blockInb != nil {
			return nil
		}

		err = bucket.Put(hash, b)
		if err != nil {
			return err
		}

		return nil
	})

	err = s.putHeightHash(header.Height, hash)
	return err
}

func (s *BlockStorage) Get(hash []byte) (*Block, error) {
	block := &pb.Block{}
	//blockByte, err := s.db.Get(hash, nil)
	//if err != nil {
	//	return nil, err
	//}

	//err = proto.Unmarshal(blockByte, block)
	//
	//if err != nil {
	//	return nil, err
	//}
	err := s.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockbyte := b.Get(hash)
		if blockbyte == nil {
			return fmt.Errorf("get block error for block %s is not found", hash)
		}

		err := proto.Unmarshal(blockbyte, block)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return ToBlock(block), nil
}

func (s *BlockStorage) HasBlock(hash []byte) bool {

	err := s.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockbyte := b.Get(hash)
		if blockbyte == nil {
			return fmt.Errorf("get block error for block %s is not found", hash)
		}
		return nil
	})
	if err != nil {
		return false
	}
	return true
}

func (s *BlockStorage) GetbyHeight(height uint64) ([]*Block, error) {
	if height == 0 {
		return []*Block{DefaultGenesisBlock()}, nil
	}
	values, exists := s.getHeightHash(height)
	if exists == ErrHeightHashNotFound {
		return nil, fmt.Errorf("not have block for height %d", height)
	} else if exists != nil {
		return nil, exists
	}

	headers := make([]*Block, 0)
	for _, hash := range values {
		block, err := s.Get(hash)
		if err != nil {
			return nil, err
		}
		headers = append(headers, block)
	}
	return headers, nil
}

func (s *BlockStorage) getHeightHash(height uint64) ([][]byte, error) {

	keys := []byte(fmt.Sprintf("height:%d", height))

	//values, err := s.db.Get(keys, nil)
	//if err != nil {
	//	return nil, err
	//}
	var decodeData [][]byte
	err := s.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blocksbyte := b.Get(keys)
		if blocksbyte == nil {
			return ErrHeightHashNotFound
		}
		err := json.Unmarshal(blocksbyte, &decodeData)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return decodeData, nil
}

func (s *BlockStorage) putHeightHash(height uint64, hash []byte) error {
	data, err := s.getHeightHash(height)
	if err != ErrHeightHashNotFound && err != nil {
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
	err = s.boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err = b.Put(keys, encodedata)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func (s *BlockStorage) SetVDFres(epoch uint64, vdfres []byte) error {
	key := []byte(fmt.Sprintf("epoch:%d", epoch))
	err := s.boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		err := b.Put(key, vdfres)
		if err != nil {
			return err
		}
		return nil
	})

	//err := s.db.Put(key, vdfres, nil)
	if epoch > s.vdfheight {
		s.rwmutex.Lock()
		s.vdfheight = epoch
		s.rwmutex.Unlock()
	}
	return err
}

func (s *BlockStorage) GetVDFresbyEpoch(epoch uint64) ([]byte, error) {
	key := []byte(fmt.Sprintf("epoch:%d", epoch))
	//value, err := s.db.Get(key, nil)
	var value []byte
	err := s.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		value = b.Get(key)
		if value == nil {
			return fmt.Errorf("not found vdf res for height %d", epoch)
		}
		return nil
	})
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

func (s *BlockStorage) GetExcutedBlock(hash []byte) (*ExecutedBlock, error) {
	block := &pb.ExecuteBlock{}
	err := s.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ExecutedBucket))
		blockbyte := b.Get(hash)
		if blockbyte != nil {
			return fmt.Errorf("get block error for block %s is not found", hash)
		}
		err := proto.Unmarshal(blockbyte, block)
		if err != nil {
			return err
		}
		return nil
	},
	)
	if err != nil {
		return nil, err
	}
	exeblock := ToExecuteBlock(block)
	return exeblock, nil
}

func (s *BlockStorage) PutExcutedBlock(block *ExecutedBlock) error {
	pbblock := block.ToProto()
	b, err := proto.Marshal(pbblock)
	if err != nil {
		return err
	}
	blockhash := block.Hash()
	err = s.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ExecutedBucket))
		blockin := bucket.Get(blockhash[:])
		if blockin != nil {
			return nil
		}
		err = bucket.Put(blockhash[:], b)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
