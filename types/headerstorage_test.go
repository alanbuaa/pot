package types

import (
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	//err := os.Chdir("../")
	//utils.PanicOnError(err)
	t.Log(os.Getwd())
}

func TestNewHeaderStorage(t *testing.T) {
	h := &Header{
		Height:     1,
		ParentHash: nil,
		UncleHash:  nil,
		Mixdigest:  []byte("aa"),
		Difficulty: big.NewInt(4),
		Nonce:      10,
		Timestamp:  time.Now(),
		PoTProof:   [][]byte{[]byte("aa"), []byte("aa")},
		Address:    1,
		Hashes:     nil,
	}
	h1 := &Header{
		Height:     1,
		ParentHash: nil,
		UncleHash:  nil,
		Mixdigest:  []byte("aa"),
		Difficulty: big.NewInt(4),
		Nonce:      10,
		Timestamp:  time.Now(),
		PoTProof:   [][]byte{[]byte("aa"), []byte("aa")},
		Address:    2,
		Hashes:     nil,
	}

	h.Hash()
	h1.Hash()
	st := NewHeaderStorage(0)
	value, err := st.getHeightHash(1)
	t.Log(err)
	t.Log(value)
	t.Log(len(value))
	st.Put(h1)
	st.Put(h)
	value, err = st.getHeightHash(1)
	t.Log(err)
	t.Log(value)
	t.Log(len(value))
	_ = os.RemoveAll("dbfile/node0-" + strconv.Itoa(0))

}
