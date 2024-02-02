package types

import (
	"crypto/rand"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"math/big"
)

/*
Tx 用于定义交易一系列结构与操作
*/

type Tx struct {
	TxHash     []byte
	ExecHeight uint64
}

func (t *Tx) Validate() bool {
	return true
}

func (t *Tx) ToProto() *pb.Tx {
	return &pb.Tx{TxHash: t.TxHash, Height: t.ExecHeight}
}

func ToTx(tx *pb.Tx) *Tx {
	return &Tx{TxHash: tx.GetTxHash(), ExecHeight: tx.GetHeight()}
}

func ToTxs(tx []*pb.Tx) []*Tx {
	txs := make([]*Tx, len(tx))
	for i := 0; i < len(tx); i++ {
		txs[i] = ToTx(tx[i])
	}
	return txs
}

func TestTx() *Tx {
	return &Tx{TxHash: []byte("test")}
}

func TestTxs() []*Tx {
	txs := make([]*Tx, 0)
	txs = append(txs, TestTx())
	return txs
}

func TestExecuteBlock(start uint64) []*pb.ExecuteBlock {
	length, _ := rand.Int(rand.Reader, big.NewInt(10))
	res := make([]*pb.ExecuteBlock, length.Int64())
	l := length.Int64()
	txs := make([]*pb.ExecutedTx, 0)
	txs = append(txs, &pb.ExecutedTx{
		Data:   RandByte(),
		TxHash: crypto.Hash(RandByte()),
	})
	for i := 0; i < int(l); i++ {
		res[i] = &pb.ExecuteBlock{
			Header: &pb.ExecuteHeader{Height: start + uint64(i)},
			Txs:    txs,
		}
	}
	return res
}

func (t *Tx) Hash() [crypto.HashLen]byte {
	return crypto.Convert(t.TxHash)
}
