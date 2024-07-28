package types

import (
	"bytes"
	"encoding/gob"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

const (
	RawTxType     = 0x01
	ExcutedTxType = 0x02
)

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

type RawTx struct {
	Txid     [crypto.Hashlen]byte
	TxInput  []TxInput
	TxOutput []TxOutput
}

type TxInput struct {
	IsCoinbase bool
	Txid       [crypto.Hashlen]byte
	Voutput    int64
	Scriptsig  []byte
	Value      int64
	Address    []byte
}

type TxOutput struct {
	Address  []byte
	Value    int64
	IsSpent  bool
	ScriptPk []byte
	Proof    []byte
}

func (r *RawTx) ToProto() *pb.RawTxData {
	pbinput := make([]*pb.TxInput, 0)
	for _, input := range r.TxInput {
		pbinput = append(pbinput, input.ToProto())
	}

	pboutput := make([]*pb.TxOutput, 0)
	for _, output := range r.TxOutput {
		pboutput = append(pboutput, output.ToProto())
	}
	pbrawtx := &pb.RawTxData{
		//TxID:     r.Txid[:],
		TxInput:  pbinput,
		TxOutput: pboutput,
	}
	return pbrawtx
}

func (r *RawTx) EncodeToByte() ([]byte, error) {
	pbrawtx := r.ToProto()
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

func DecodeByteToRawTx(data []byte) *RawTx {
	pbtx := new(pb.TxData)
	err := proto.Unmarshal(data, pbtx)
	if err != nil {
		return nil
	}

	pbrawtx := new(pb.RawTxData)
	err = proto.Unmarshal(pbtx.GetTxData(), pbrawtx)

	if err != nil {
		return nil
	}

	return ToRawTx(pbrawtx)
}

// TODO: need to modify later
func (r *RawTx) Hash() [crypto.Hashlen]byte {
	if r.Txid != [32]byte{} {
		return r.Txid
	}

	b, err := r.EncodeToByte()

	if err != nil {
		return [32]byte{}
	} else {
		hash := crypto.Convert(crypto.Hash(b))
		r.Txid = hash
		return hash
	}

}

func ToTxInput(input *pb.TxInput) TxInput {
	return TxInput{
		IsCoinbase: input.IsCoinbase,
		Txid:       crypto.Convert(input.GetTxID()),
		Voutput:    input.GetVoutput(),
		Scriptsig:  input.GetScriptsig(),
		Value:      input.GetValue(),
		Address:    input.GetAddress(),
	}
}

func (i TxInput) ToProto() *pb.TxInput {
	return &pb.TxInput{
		IsCoinbase: i.IsCoinbase,
		TxID:       i.Txid[:],
		Voutput:    i.Voutput,
		Scriptsig:  i.Scriptsig,
		Value:      i.Value,
		Address:    i.Address,
	}
}

func ToTxOutput(output *pb.TxOutput) TxOutput {
	return TxOutput{
		Address:  output.GetAddress(),
		Value:    output.GetValue(),
		IsSpent:  output.GetIsSpent(),
		ScriptPk: output.GetScriptPk(),
	}
}

func (o TxOutput) ToProto() *pb.TxOutput {
	return &pb.TxOutput{
		Address:  o.Address,
		Value:    o.Value,
		IsSpent:  o.IsSpent,
		ScriptPk: o.ScriptPk,
	}
}

func ToRawTx(tx *pb.RawTxData) *RawTx {
	pbtxintputs := tx.GetTxInput()
	txinputs := make([]TxInput, 0)
	for _, pbtxintput := range pbtxintputs {
		txinputs = append(txinputs, ToTxInput(pbtxintput))
	}

	pbtxoutputs := tx.GetTxOutput()
	txoutputs := make([]TxOutput, 0)
	for _, pbtxoutput := range pbtxoutputs {
		txoutputs = append(txoutputs, ToTxOutput(pbtxoutput))
	}

	return &RawTx{
		Txid:     crypto.Convert(tx.GetTxID()),
		TxInput:  txinputs,
		TxOutput: txoutputs,
	}
}

type TxOutputs []TxOutput

func DecodeByte2Outputs(data []byte) TxOutputs {
	var outputs TxOutputs
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		return nil
	}
	return outputs
}

func (output TxOutputs) EncodeTxOutputs2Byte() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(output)
	if err != nil {
		return nil
	}
	return buff.Bytes()
}

// TODO:
func (i TxInput) CanUnlockOutputwith(address []byte) bool {
	return true
	//
}
func (o TxOutput) CanBeUnlockWith(address []byte) bool {
	return true
	//return bytes.Equal(o.ScriptPk,address)
}
func (r *RawTx) IsCoinBase() bool {
	return len(r.TxInput) == 1 && r.TxInput[0].Txid == [32]byte{} && r.TxInput[0].Voutput == -1
}

func (r *RawTx) BasicVerify() bool {
	if r.IsCoinBase() {
		return true
	} else {
		for _, output := range r.TxOutput {
			if output.Address == nil || bytes.Equal(output.Address, []byte{}) {
				return true
			} else if len(r.TxOutput) == 1 {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func (o TxOutput) IsLockedWithKey(pubkey []byte) bool {
	return bytes.Equal(o.Address, pubkey)
}
