package types

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

const blocksBucket = "blocks"
const UTXOBucket = "chainstate"
const ExecutedBucket = "Executed"
const ClientBucket = "client"

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

func NewBlockStorage(id int64, cid int64) *BlockStorage {
	sint := fmt.Sprintf("node%d-cid%d", id, cid)

	boltdbname := fmt.Sprintf("boltdbfile/" + sint)
	if dbExists(boltdbname) {
		err := os.Remove(boltdbname)
		if err != nil {
			panic(err)
		}
	}
	boltdb, err := bolt.Open("boltdbfile/"+sint, 0600, nil)
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

func GetExistedBlockStorage(id int64, cid int64) *BlockStorage {
	sint := fmt.Sprintf("node%d-cid%d", id, cid)

	boltdbname := fmt.Sprintf("boltdbfile/" + sint)
	if !dbExists(boltdbname) {
		return NewBlockStorage(id, cid)
	}
	boltdb, err := bolt.Open("boltdbfile/"+sint, 0600, nil)
	if err != nil {
		fmt.Println("create boltdb err for ", err)
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
		key := []byte("highest:")
		err = s.boltdb.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			err := b.Put(key, []byte(strconv.Itoa(int(epoch))))
			if err != nil {
				return err
			}
			return nil
		})

	}
	return err
}

func (s *BlockStorage) GetHighestVDFRes() (uint64, []byte, error) {
	var epoch int
	err := s.boltdb.View(func(tx *bolt.Tx) error {
		key := []byte("highest:")
		b := tx.Bucket([]byte(blocksBucket))
		epochbyte := b.Get(key)
		var err error
		if len(epochbyte) == 0 {
			return fmt.Errorf("not existed highest vdfres")
		}
		epoch, err = strconv.Atoi(string(epochbyte))

		return err
	})

	if err != nil {
		return 0, nil, err
	}

	vdfres, err := s.GetVDFresbyEpoch(uint64(epoch))
	if err != nil {
		return 0, nil, err
	}

	return uint64(epoch), vdfres, nil
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

func (s *BlockStorage) SetVDFHalf(epoch uint64, vdfres []byte) error {
	key := []byte(fmt.Sprintf("epoch half: %d", epoch))
	err := s.boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
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

type UTXOs [][]byte

func SerializeUTXOs(utxos [][]byte) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(utxos)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DeserializeUTXOs(data []byte) ([][]byte, error) {
	var utxos [][]byte
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&utxos)
	if err != nil {
		return nil, err
	}
	return utxos, nil
}
func (s *BlockStorage) PutClientUTXO(address []byte, utxo []byte) error {

	err := s.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ClientBucket))
		b := bucket.Get(address)

		if b == nil {
			utxos := make([][]byte, 0)
			utxos = append(utxos, utxo)
			data, err := SerializeUTXOs(utxos)
			if err != nil {
				return err
			}
			err = bucket.Put(address, data)
			if err != nil {
				return err
			}
		} else {
			utxos, err := DeserializeUTXOs(b)
			if err != nil {
				return err
			}
			utxos = append(utxos, utxo)
			data, err := SerializeUTXOs(utxos)
			if err != nil {
				return err
			}
			err = bucket.Put(address, data)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *BlockStorage) GetClientUTXO(address []byte) ([][]byte, error) {
	var utxos [][]byte
	var err error
	err = s.boltdb.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ClientBucket))
		b := bucket.Get(address)
		if b == nil {
			return nil
		}
		utxos, err = DeserializeUTXOs(b)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return utxos, nil
}

func (s *BlockStorage) DeleteClientUTXO(address []byte, utxo []byte) error {
	err := s.boltdb.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(ClientBucket))
		b := bucket.Get(address)
		if b == nil {
			return fmt.Errorf("can't not find utxo for address %s", hexutil.Encode(address))
		}

		utxos, err := DeserializeUTXOs(b)
		if err != nil {
			return err
		}

		for i, utxo := range utxos {
			if bytes.Equal(utxo, utxo) {
				utxos = append(utxos[:i], utxos[i+1:]...)
				data, err := SerializeUTXOs(utxos)
				if err != nil {
					return err
				}
				err = bucket.Put(address, data)
				if err != nil {
					return err
				}
				return nil
			}
		}

		return fmt.Errorf("can't not find utxo %s for address %s", hexutil.Encode(utxo), hexutil.Encode(address))
	})
	if err != nil {
		return err

	}
	return nil
}
