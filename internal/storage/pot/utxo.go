package pot

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const UTXOBucket = "chainstate"

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
