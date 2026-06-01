package types

import (
	"errors"
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type StorageData interface {
	Hash(h string) []byte
	// Encode() ([]byte, error)
	protoreflect.ProtoMessage
}

/*
 * This is a generic storage class that can be used to store any type of data that implements StorageData interface
 */
type Storage struct {
	db     *leveldb.DB
	lock   *sync.Mutex
	hasher string
}

func NewStorage(name string, h string) *Storage {
	db, err := leveldb.OpenFile("dbfile/"+name, nil)
	if err != nil {
		fmt.Println("create storage err:", err)
		panic(err)
	}
	return &Storage{
		db:     db,
		lock:   new(sync.Mutex),
		hasher: h,
	}
}

func (s *Storage) PutIfNotExists(block StorageData) error {
	hash := block.Hash(s.hasher)
	s.lock.Lock()
	defer s.lock.Unlock()
	_, err := s.db.Get(hash, nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			b, err := proto.Marshal(block)
			if err != nil {
				return err
			}
			return s.db.Put(block.Hash(s.hasher), b, nil)
		}
	}
	return nil
}

func (s *Storage) Put(block StorageData) error {
	b, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.db.Put(block.Hash(s.hasher), b, nil)
}

func (s *Storage) Get(hash []byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.db.Get(hash, nil)
}
