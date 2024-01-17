package types

import "github.com/zzz136454872/upgradeable-consensus/pb"

/*
Tx 用于定义交易一系列结构与操作
*/

type Tx struct {
	Data []byte
}

func (t *Tx) Validate() bool {
	return true
}

func (t *Tx) ToProto() *pb.Tx {
	return &pb.Tx{TxData: t.Data}
}

func ToTx(tx *pb.Tx) *Tx {
	return &Tx{Data: tx.GetTxData()}
}

func ToTxs(tx []*pb.Tx) []*Tx {
	txs := make([]*Tx, len(tx))
	for i := 0; i < len(tx); i++ {
		txs[i] = ToTx(tx[i])
	}
	return txs
}

func TestTx() *Tx {
	return &Tx{Data: []byte("test")}
}

func TestTxs() []*Tx {
	txs := make([]*Tx, 0)
	txs = append(txs, TestTx())
	return txs
}
