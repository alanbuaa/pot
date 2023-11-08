package types

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
)

func TestMain(m *testing.M) {
	err := os.Chdir("../../")
	utils.PanicOnError(err)
	os.Exit(m.Run())
}

func TestHash(t *testing.T) {
	txs := [][]byte{[]byte("ss"), []byte("aaa"), []byte("ddd")}
	parentHash := sha256.Sum256([]byte("parentHash"))

	block := &pb.WhirlyBlock{
		ParentHash: parentHash[:],
		Hash:       nil,
		Height:     1,
		Txs:        txs,
		Justify: &pb.QuorumCert{
			BlockHash: parentHash[:],
			ViewNum:   1,
		},
	}

	blockHash := Hash(block)

	t.Log(hex.EncodeToString(blockHash))
}

func TestBlock_PutAndGet(t *testing.T) {
	txs := [][]byte{[]byte("ss"), []byte("aaa"), []byte("ddd")}
	parentHash := sha256.Sum256([]byte("parentHash"))

	block := &pb.WhirlyBlock{
		ParentHash: parentHash[:],
		Hash:       nil,
		Height:     1,
		Txs:        txs,
		Justify: &pb.QuorumCert{
			BlockHash: parentHash[:],
			ViewNum:   1,
		},
	}

	blockHash := Hash(block)
	block.Hash = blockHash

	impl := NewBlockStorageImpl("1")

	err := impl.Put(block)
	if err != nil {
		t.Fatal(block)
	}

	get, err := impl.Get(block.Hash)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(String(get))
	impl.Close()
}

func TestBlockNilParam(t *testing.T) {
	impl := NewBlockStorageImpl("1")
	get, err := impl.Get([]byte("ssss"))
	if err == leveldb.ErrNotFound {
		t.Log("not found")
	}
	t.Log(get)
	impl.Close()
}
