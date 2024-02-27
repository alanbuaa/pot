package types

import (
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
	"math/big"
)

const (
	RawTxType     = 0x01
	ExcutedTxType = 0x02
)

type RawTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasPrice   *big.Int // wei per gas
	Gas        uint64   // gas limit
	To         []byte
	Data       []byte   // contract invocation input Data
	Value      *big.Int // wei amount
	V, R, S    *big.Int // signature values
	Accesslist []byte
}

func (r *RawTx) ToProto() *pb.RawTxData {
	pbrawtx := &pb.RawTxData{
		ChainID:    r.ChainID.Bytes(),
		Nonce:      r.Nonce,
		GasPrice:   r.GasPrice.Bytes(),
		Gas:        r.Gas,
		To:         r.To,
		Data:       r.Data,
		Value:      r.Value.Bytes(),
		V:          r.V.Bytes(),
		R:          r.R.Bytes(),
		S:          r.S.Bytes(),
		Accesslist: r.Accesslist,
	}
	return pbrawtx
}

func (t *RawTx) EncodeToByte() ([]byte, error) {
	pbrawtx := t.ToProto()
	rawtxbyte, err := proto.Marshal(pbrawtx)
	if err != nil {
		return nil, err
	}
	pbtx := &pb.TxData{
		TxDataType: pb.TxDataType_RawTx,
		TxData:     rawtxbyte,
	}
	b, err := proto.Marshal(pbtx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type ExecutedTxData struct {
	ExecutedHeight uint64
	TxHash         []byte
}

func (e *ExecutedTxData) ToProto() *pb.ExecutedTxData {
	return &pb.ExecutedTxData{
		ExecutedHeight: e.ExecutedHeight,
		TxHash:         e.TxHash,
	}
}

func (e *ExecutedTxData) EncodeToByte() ([]byte, error) {
	pbexTx := e.ToProto()
	pbbyte, err := proto.Marshal(pbexTx)
	if err != nil {
		return nil, err
	}
	pbtx := &pb.TxData{
		TxDataType: pb.TxDataType_ExcutedTx,
		TxData:     pbbyte,
	}
	b, err := proto.Marshal(pbtx)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (e *ExecutedTxData) Hash() [crypto.Hashlen]byte {
	if e.TxHash != nil {
		return crypto.Convert(e.TxHash)
	} else {
		return crypto.Convert(crypto.NilTxsHash)
	}
}
