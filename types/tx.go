package types

import (
	"crypto/rand"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
	"math/big"
)

/*
PT Txs o
*/

type Tx struct {
	Data []byte
}

func (t *Tx) Validate() bool {
	return true
}

func (t *Tx) ToProto() *pb.Tx {
	return &pb.Tx{Data: t.Data}
}

func ToTx(tx *pb.Tx) *Tx {
	return &Tx{Data: tx.GetData()}
}

func ToTxs(tx []*pb.Tx) []*Tx {
	txs := make([]*Tx, len(tx))
	for i := 0; i < len(tx); i++ {
		txs[i] = ToTx(tx[i])
	}
	return txs
}

func TestTx() *Tx {
	data := &ExecutedTxData{
		ExecutedHeight: 0,
		TxHash:         crypto.Hash([]byte("Test")),
	}
	txdata, _ := data.EncodeToByte()
	return &Tx{Data: txdata}
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

func (t *Tx) Hash() [crypto.Hashlen]byte {
	hash := crypto.Hash(t.Data)
	return crypto.Convert(hash)
}

func Txs2Bytes(txs []*Tx) [][]byte {
	if txs != nil {
		txsbyte := make([][]byte, len(txs))
		for i := 0; i < len(txs); i++ {
			txsbyte[i] = txs[i].Data
		}
		return txsbyte
	} else {
		return nil
	}
}

func (t *Tx) GetTxType() pb.TxDataType {
	txdata := new(pb.TxData)
	err := proto.Unmarshal(t.Data, txdata)
	if err != nil {
		return txdata.TxDataType
	}
	return 0
}
