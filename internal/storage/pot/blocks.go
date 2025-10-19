package pot

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	. "github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/protobuf/proto"
)

const BlocksBucket = "blocks"
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

func (s *BlockStorage) GetBoltdb() *bolt.DB {
	return s.boltdb
}

func NewBlockStorage(id int64, dir string) *BlockStorage {
	// ensure directory exists
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Panic(err)
	}

	// use a clear filename based on node id
	boltdbPath := filepath.Join(dir, fmt.Sprintf("node-%d.db", id))

	// open (or create) boltdb at the computed path
	boltdb, err := bolt.Open(boltdbPath, 0600, nil)
	if err != nil {
		fmt.Println("open boltdb err:", err)
		log.Panic(err)
	}

	// Create buckets if they don't exist. Use CreateBucketIfNotExists to avoid errors.
	err = boltdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(BlocksBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(UTXOBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(ExecutedBucket)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(ClientBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Println("create bolt bucket err:", err)
		log.Panic(err)
	}
	return &BlockStorage{
		boltdb: boltdb,
		//db:        db,
		vdfheight: 0,
		rwmutex:   new(sync.RWMutex),
	}
}

func (s *BlockStorage) PutByte(block *Block) ([]byte, error) {
	header := block.GetHeader()
	protos := block.ToProto()
	b, err := proto.Marshal(protos)
	if err != nil {
		return []byte{}, err
	}

	hash := header.Hash()
	//err = s.db.Put(hash, b, nil)
	//if err != nil {
	//	return err
	//}

	err = s.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BlocksBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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
	return b, err
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
		bucket := tx.Bucket([]byte(BlocksBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
		blockbyte := b.Get(hash)
		if blockbyte == nil {
			return fmt.Errorf("get block error for block %s is not found", hash)
		}
		return nil
	})
	return err == nil
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
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
		if len(b.Get(key)) != 0 {
			return nil
		}
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
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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

func (s *BlockStorage) SetVDFHalf(epoch uint64, vdfres []byte) error {
	key := []byte(fmt.Sprintf("epoch half: %d", epoch))
	err := s.boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
		if len(b.Get(key)) != 0 {
			return nil
		}
		err := b.Put(key, vdfres)
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

func (s *BlockStorage) GetVDFHalf(epoch uint64) ([]byte, error) {
	key := []byte(fmt.Sprintf("epoch half: %d", epoch))
	var value []byte
	err := s.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BlocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", BlocksBucket)
		}
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
		if b == nil {
			return fmt.Errorf("bucket %s not found", ExecutedBucket)
		}
		blockbyte := b.Get(hash)
		if blockbyte == nil {
			return fmt.Errorf("get block error for block %s is not found", hexutil.Encode(hash))
		}
		//fmt.Println(hexutil.Encode(blockbyte))
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
	//fmt.Println("put block of height: ", pbblock.GetHeader().GetHeight())
	b, err := proto.Marshal(pbblock)
	if err != nil {
		return err
	}
	blockhash := block.Hash()
	err = s.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ExecutedBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", ExecutedBucket)
		}
		blockin := bucket.Get(blockhash[:])
		if blockin != nil {
			//fmt.Println("error for not nil block")
			return fmt.Errorf("already have block for hash %s", hexutil.Encode(blockhash[:]))
		}
		err = bucket.Put(blockhash[:], b)
		//fmt.Println(hexutil.Encode(blockhash[:]), hexutil.Encode(b))
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
