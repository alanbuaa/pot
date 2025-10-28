package types

import (
	"blockchain-crypto/pqcgo"

	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"

	"google.golang.org/protobuf/proto"
)

const (
	RawTxType     = 0x01
	ExcutedTxType = 0x02
)

var (
	FpByteSize = 48
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
	Txid           [crypto.Hashlen]byte `json:"txid"`
	TxInput        []TxInput            `json:"txinput"`
	TxOutput       []TxOutput           `json:"txoutput"`
	TransactionFee int64                `json:"transactionFee"`
	CoinbaseProofs []CoinbaseProof      `json:"coinbaseProofs"`
}

type TxInput struct {
	Txid      [crypto.Hashlen]byte `json:"txid"`
	Voutput   int64                `json:"voutput"`
	Scriptsig []byte               `json:"scriptsig"`
	Value     int64                `json:"value"`
	Address   []byte               `json:"address"`
	BciType   int32                `json:"bciType"`
}

type TxOutput struct {
	Address     []byte  `json:"address"`
	Value       int64   `json:"value"`
	Interest    int64   `json:"interest"`
	ScriptPk    []byte  `json:"scriptpk"`
	Proof       []byte  `json:"proof"`
	LockTime    uint64  `json:"locktime"`
	BciType     int32   `json:"bciType"`
	Data        []byte  `json:"data"`
	BurnLock    uint64  `json:"burnTime"`
	Rate        float64 `json:"Rate"`
	CreatedAt   uint64  `json:"createdAt"`
	BlockHeight uint64  `json:"blockHeight"` //only for local check
	UseFlag     bool    `json:"useFlag"`     //only for local check
}

type CoinbaseProof struct {
	Address  []byte  `json:"address"`
	Amount   int64   `json:"amount"`
	TxHash   []byte  `json:"txhash"`
	Type     int32   `json:"type"`
	DoDraw   bool    `json:"doDraw"`
	Interest int64   `json:"interest"`
	Height   uint64  `json:"height"`
	Weight   float64 // only for local
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
		TransactionFee: r.TransactionFee,
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
		Txid:      crypto.Convert(input.GetTxID()),
		Voutput:   input.GetVoutput(),
		Scriptsig: input.GetScriptsig(),
		Value:     input.GetValue(),
		Address:   input.GetAddress(),
		BciType:   input.GetBciType(),
	}
}

func (i TxInput) ToProto() *pb.TxInput {
	return &pb.TxInput{
		TxID:      i.Txid[:],
		Voutput:   i.Voutput,
		Scriptsig: i.Scriptsig,
		Value:     i.Value,
		Address:   i.Address,
		BciType:   i.BciType,
	}
}

func ToTxOutput(output *pb.TxOutput) TxOutput {
	return TxOutput{
		Address:   output.GetAddress(),
		Value:     output.GetValue(),
		Interest:  output.GetInterest(),
		ScriptPk:  output.GetScriptPk(),
		Proof:     output.GetProof(),
		LockTime:  output.GetLockTime(),
		BciType:   output.GetBciType(),
		Data:      output.GetData(),
		Rate:      float64(output.GetRate()),
		CreatedAt: output.GetCreatedAt(),
		BurnLock:  output.GetBurnLock(),
	}
}

func (o TxOutput) ToProto() *pb.TxOutput {
	return &pb.TxOutput{
		Address:   o.Address,
		Value:     o.Value,
		Interest:  o.Interest,
		ScriptPk:  o.ScriptPk,
		LockTime:  o.LockTime,
		BciType:   o.BciType,
		Proof:     o.Proof,
		Data:      o.Data,
		Rate:      float32(o.Rate),
		CreatedAt: o.CreatedAt,
		BurnLock:  o.BurnLock,
	}
}

func (o TxOutput) EncodeToByte() []byte {
	databyte, _ := json.Marshal(o)
	return databyte
}

func DecodeByteToTxOutput(data []byte) TxOutput {
	var txout TxOutput
	json.Unmarshal(data, &txout)
	return txout
}

func ToCoinbaseProof(Proof *pb.CoinbaseProof) CoinbaseProof {
	return CoinbaseProof{
		Address: Proof.GetAddress(),
		Amount:  Proof.GetAmount(),
		TxHash:  Proof.GetTxHash(),
		Type:    Proof.GetBciType(),
		DoDraw:  Proof.GetDoDraw(),
		Height:  Proof.GetHeight(),
	}
}

func (p CoinbaseProof) ToProto() *pb.CoinbaseProof {
	return &pb.CoinbaseProof{
		Address: p.Address,
		Amount:  p.Amount,
		TxHash:  p.TxHash,
		BciType: p.Type,
		DoDraw:  p.DoDraw,
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
func (r *RawTx) IsCoinBase() bool {
	return len(r.TxInput) == 1 && r.TxInput[0].Voutput == -1
}

func (r *RawTx) BasicVerify() bool {
	// if r.IsCoinBase() {
	// } else {
	// 	for _, output := range r.TxOutput {
	// 		if output.Address == nil || bytes.Equal(output.Address, []byte{}) {
	// 			return true
	// 		} else if len(r.TxOutput) == 1 {
	// 			return true
	// 		} else {
	// 			return false
	// 		}
	// 	}
	// }
	// return false
	return true
}

func (o TxOutput) IsLockedWithKey(pubkey []byte) bool {
	//fmt.Println(hexutil.Encode(o.Address))
	//fmt.Println(hexutil.Encode(pubkey))
	return bytes.Equal(o.Address, pubkey)
}

func (o TxOutput) CanBeUnlockWith(input TxInput) bool {
	pubkey := o.Address
	if bytes.Equal(o.Address, input.Scriptsig) {
		return true
	}
	if len(pubkey) == pqcgo.PUBLICKEYBYTES[crypto.PqcScheme] {
		sig := input.Scriptsig
		utxokey := fmt.Sprintf("%s:%d", input.Txid, input.Voutput)
		flag, err := crypto.VerifySig([]byte(utxokey), sig, pubkey)
		if err != nil && !flag {
			return false
		} else {
			return true
		}
	} else {

		// sig := input.Scriptsig
		// utxokey := fmt.Sprintf("%s:%d", input.Txid, input.Voutput)
		// flag, err := crypto.VerifySig([]byte(utxokey), sig, pubkey)
		// if err != nil && !flag {
		// 	return false
		// }
		// TODO: check commiteekey
		pqcpubkey := input.Address
		if len(pqcpubkey) != pqcgo.PUBLICKEYBYTES[crypto.PqcScheme] {
			return false
		}

		commiteekeysig := &crypto.CommiteeKeySig{}
		err := json.Unmarshal(input.Scriptsig, commiteekeysig)

		if err != nil {
			return false
		}

		if !crypto.VerifyCommitteePKAPI(commiteekeysig.Alpha, commiteekeysig.Pqcpubkey, commiteekeysig.G, commiteekeysig.CommiteeKey, commiteekeysig.Rcommit, commiteekeysig.AwardKey) {
			return false
		}
		// alpha := sig[0:crypto.Hashlen]
		// ydot := sig[crypto.Hashlen : crypto.Hashlen+FpByteSize]
		// pqcsig := sig[crypto.Hashlen+FpByteSize:]
		// gdot :=

		// if crypto.VerifyCommitteePKAPI(alpha, pqcpubkey)
	}
	return true
}

func DecodeByte2Proof(b []byte) *CoinbaseProof {
	pbproff := &pb.CoinbaseProof{}
	err := proto.Unmarshal(b, pbproff)
	if err != nil {
		return nil
	}

	proof := ToCoinbaseProof(pbproff)
	return &proof

}
