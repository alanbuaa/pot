package types

import (
	"bytes"
	"encoding/gob"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

const (
	RawTxType     = 0x01
	ExcutedTxType = 0x02
)

type ExecutedBlock struct {
	Header *ExecuteHeader
	Txs    []*ExecutedTx
}

func (e *ExecutedBlock) ToProto() *pb.ExecuteBlock {
	pbtxs := make([]*pb.ExecutedTx, 0)
	for _, tx := range e.Txs {
		pbtxs = append(pbtxs, tx.ToProto())
	}
	return &pb.ExecuteBlock{
		Header: e.Header.ToProto(),
		Txs:    pbtxs,
	}
}

func ToExecuteBlock(block *pb.ExecuteBlock) *ExecutedBlock {
	txs := make([]*ExecutedTx, 0)
	pbtxs := block.GetTxs()
	for _, pbtx := range pbtxs {
		tx := ToExecutedTx(pbtx)
		txs = append(txs, tx)
	}

	return &ExecutedBlock{
		Header: &ExecuteHeader{
			Height:    block.GetHeader().GetHeight(),
			BlockHash: block.GetHeader().GetBlockHash(),
			TxsHash:   block.GetHeader().GetTxsHash(),
		},
		Txs: txs,
	}
}

func (e *ExecutedBlock) Hash() [crypto.Hashlen]byte {
	return crypto.Convert(e.Header.BlockHash)
}

type ExecuteHeader struct {
	Height    uint64
	BlockHash []byte
	TxsHash   []byte
}

func (e *ExecuteHeader) ToProto() *pb.ExecuteHeader {
	return &pb.ExecuteHeader{
		Height:    e.Height,
		BlockHash: e.BlockHash,
		TxsHash:   e.TxsHash,
	}
}

func (e *ExecuteHeader) EncodeToByte() ([]byte, error) {
	pbexTx := e.ToProto()
	pbbyte, err := proto.Marshal(pbexTx)
	if err != nil {
		return nil, err
	}
	return pbbyte, nil
}

func (e *ExecuteHeader) Hash() [crypto.Hashlen]byte {
	return crypto.Convert(e.BlockHash)
}

type ExecutedTx struct {
	Height uint64
	TxHash []byte
	Data   []byte
}

func (e *ExecutedTx) ToProto() *pb.ExecutedTx {
	return &pb.ExecutedTx{
		Height: e.Height,
		TxHash: e.TxHash,
		Data:   e.Data,
	}
}

func (e *ExecutedTx) EncodeToByte() ([]byte, error) {
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

func (e *ExecutedTx) Hash() [crypto.Hashlen]byte {
	if e.TxHash != nil {
		return crypto.Convert(e.TxHash)
	} else {
		return crypto.Convert(crypto.NilTxsHash)
	}
}

func ToExecutedTx(tx *pb.ExecutedTx) *ExecutedTx {
	return &ExecutedTx{
		Data:   tx.GetData(),
		Height: tx.GetHeight(),
		TxHash: tx.GetTxHash(),
	}
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

type RawTx struct {
	Txid           [crypto.Hashlen]byte
	TxInput        []TxInput
	TxOutput       []TxOutput
	CoinbaseProofs []CoinbaseProof
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
	LockTime uint64
}

type CoinbaseProof struct {
	Address []byte
	Amount  int64
	TxHash  []byte
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

	pbproof := make([]*pb.CoinbaseProof, 0)
	for _, proof := range r.CoinbaseProofs {
		pbproof = append(pbproof, proof.ToProto())
	}
	pbrawtx := &pb.RawTxData{
		TxID:           r.Txid[:],
		TxInput:        pbinput,
		TxOutput:       pboutput,
		CoinbaseProofs: pbproof,
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
		//Proof:    output.GetProof(),
		LockTime: output.GetLockTime(),
	}
}

func (o TxOutput) ToProto() *pb.TxOutput {
	return &pb.TxOutput{
		Address:  o.Address,
		Value:    o.Value,
		IsSpent:  o.IsSpent,
		ScriptPk: o.ScriptPk,
		LockTime: o.LockTime,
	}
}

func (o TxOutput) EncodeToByte() []byte {
	pboutput := o.ToProto()
	pboutputbyte, err := proto.Marshal(pboutput)
	if err != nil {
		return nil
	}
	return pboutputbyte
}

func DecodeByteToTxOutput(data []byte) TxOutput {
	pboutput := new(pb.TxOutput)
	err := proto.Unmarshal(data, pboutput)
	if err != nil {
		return TxOutput{}
	}
	return ToTxOutput(pboutput)
}

func ToCoinbaseProof(Proof *pb.CoinbaseProof) CoinbaseProof {
	return CoinbaseProof{
		Address: Proof.GetAddress(),
		Amount:  Proof.GetAmount(),
		TxHash:  Proof.GetTxHash(),
	}
}

func (p CoinbaseProof) ToProto() *pb.CoinbaseProof {
	return &pb.CoinbaseProof{
		Address: p.Address,
		Amount:  p.Amount,
		TxHash:  p.TxHash,
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

	pbcoinbaseproofs := tx.GetCoinbaseProofs()
	coinbaseproofs := make([]CoinbaseProof, 0)
	for _, pbcoinbaseproof := range pbcoinbaseproofs {
		coinbaseproofs = append(coinbaseproofs, ToCoinbaseProof(pbcoinbaseproof))
	}

	return &RawTx{
		Txid:           crypto.Convert(tx.GetTxID()),
		TxInput:        txinputs,
		TxOutput:       txoutputs,
		CoinbaseProofs: coinbaseproofs,
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
	return len(r.TxInput) == 1 && r.TxInput[0].Voutput == -1
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
	//fmt.Println(hexutil.Encode(o.Address))
	//fmt.Println(hexutil.Encode(pubkey))
	return bytes.Equal(o.Address, pubkey)
}

func UTXO2Transaction(tx *RawTx) {

	//to := tx.TxOutput[0].Address
	//transacto := common.BytesToAddress(to)
	//
	//transaction := &types.DynamicFeeTx{
	//	ChainID:    big.NewInt(11223344551),
	//	Nonce:      0,
	//	GasTipCap:  nil,
	//	GasFeeCap:  nil,
	//	Gas:        0,
	//	To:         &transacto,
	//	Value:      big.NewInt(2000),
	//	Data:       nil,
	//	AccessList: nil,
	//	V:          nil,
	//	R:          nil,
	//	S:          nil,
	//}
}

func Transaction2Bytes(tx *types.DynamicFeeTx) {

}
