package model

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// HTTPTransaction represents a transaction in HTTP API format
type HTTPTransaction struct {
	Txid           string         `json:"txid"`
	TxInputs       []HTTPTxInput  `json:"txInputs"`
	TxOutputs      []HTTPTxOutput `json:"txOutputs"`
	TransactionFee string         `json:"transactionFee"`
}

// HTTPTxInput represents a transaction input in HTTP API format
type HTTPTxInput struct {
	Txid      string `json:"txid"`
	Voutput   string `json:"voutput"`
	ScriptSig string `json:"scriptSig"`
	Value     string `json:"value"`
	Address   string `json:"address"`
	BciType   string `json:"bciType"`
}

// HTTPTxOutput represents a transaction output in HTTP API format
type HTTPTxOutput struct {
	Address  string `json:"address"`
	Value    string `json:"value"`
	Interest string `json:"interest"`
	Proof    string `json:"proof"`
	LockTime string `json:"lockTime"`
	BciType  string `json:"bciType"`
	Data     string `json:"data"`
	BurnLock string `json:"burnLock"`
	Rate     string `json:"rate"`
}

// RequestData represents the generic request wrapper
type RequestData struct {
	Transaction HTTPTransaction `json:"transaction"`
	Type        string          `json:"type"`
}

// ResponseData represents the generic response wrapper
type ResponseData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// BlockHeightResponse represents the block height response
type BlockHeightResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Height uint64 `json:"height"`
}

// HttpTx2Tx converts HTTPTransaction to RawTx
func HttpTx2Tx(tx *HTTPTransaction) (*types.RawTx, error) {
	txInputs := make([]types.TxInput, 0)
	for i, txInput := range tx.TxInputs {
		txinput, err := HttpTxInput2TxInput(txInput)
		if err != nil {
			return nil, fmt.Errorf("HttpTxInput2TxInput error at index %d: %v", i, err)
		}
		txInputs = append(txInputs, txinput)
	}
	txOutputs := make([]types.TxOutput, 0)
	for i, txOutput := range tx.TxOutputs {
		txoutput, err := HttpTxOutput2TxOutput(txOutput)
		if err != nil {
			return nil, fmt.Errorf("HttpTxOutput2TxOutput error at index %d: %v", i, err)
		}
		txOutputs = append(txOutputs, txoutput)
	}

	if tx.Txid == "" {
		return nil, fmt.Errorf("transaction txid is empty")
	}
	txid, err := hexutil.Decode(tx.Txid)
	if err != nil {
		return nil, fmt.Errorf("decode transaction txid error: %v, txid: %s", err, tx.Txid)
	}
	transactionfee, err := strconv.ParseInt(tx.TransactionFee, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse transaction fee error: %v, fee: %s", err, tx.TransactionFee)
	}

	return &types.RawTx{
		Txid:           crypto.Convert(txid),
		TxInput:        txInputs,
		TxOutput:       txOutputs,
		TransactionFee: transactionfee,
	}, nil
}

// HttpTxInput2TxInput converts HTTPTxInput to TxInput
func HttpTxInput2TxInput(txinput HTTPTxInput) (types.TxInput, error) {
	if txinput.Txid == "" {
		return types.TxInput{}, fmt.Errorf("txinput txid is empty")
	}
	b, err := hexutil.Decode(txinput.Txid)
	if err != nil {
		return types.TxInput{}, fmt.Errorf("decode txinput txid error: %v, txid: %s", err, txinput.Txid)
	}
	voutput, err := strconv.ParseInt(txinput.Voutput, 10, 64)
	if err != nil {
		return types.TxInput{}, fmt.Errorf("parse voutput error: %v, voutput: %s", err, txinput.Voutput)
	}

	// ScriptSig can be empty for some transaction types
	var scriptsig []byte
	if txinput.ScriptSig != "" {
		scriptsig, err = hexutil.Decode(txinput.ScriptSig)
		if err != nil {
			return types.TxInput{}, fmt.Errorf("decode scriptsig error: %v, scriptsig: %s", err, txinput.ScriptSig)
		}
	}

	value, err := strconv.ParseInt(txinput.Value, 10, 64)
	if err != nil {
		return types.TxInput{}, fmt.Errorf("parse value error: %v, value: %s", err, txinput.Value)
	}

	if txinput.Address == "" {
		return types.TxInput{}, fmt.Errorf("txinput address is empty")
	}
	addr, err := hexutil.Decode(txinput.Address)
	if err != nil {
		return types.TxInput{}, fmt.Errorf("decode address error: %v, address: %s", err, txinput.Address)
	}

	bcitype, err := strconv.ParseInt(txinput.BciType, 10, 64)
	if err != nil {
		return types.TxInput{}, fmt.Errorf("parse bcitype error: %v, bcitype: %s", err, txinput.BciType)
	}

	return types.TxInput{
		Txid:      crypto.Convert(b),
		Voutput:   voutput,
		Scriptsig: scriptsig,
		Value:     value,
		Address:   addr,
		BciType:   int32(bcitype),
	}, nil
}

// HttpTxOutput2TxOutput converts HTTPTxOutput to TxOutput
func HttpTxOutput2TxOutput(txoutput HTTPTxOutput) (types.TxOutput, error) {
	if txoutput.Address == "" {
		return types.TxOutput{}, fmt.Errorf("txoutput address is empty")
	}
	addr, err := hexutil.Decode(txoutput.Address)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("decode address error: %v, address: %s", err, txoutput.Address)
	}

	value, err := strconv.ParseInt(txoutput.Value, 10, 64)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("parse value error: %v, value: %s", err, txoutput.Value)
	}

	interest, err := strconv.ParseInt(txoutput.Interest, 10, 64)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("parse interest error: %v, interest: %s", err, txoutput.Interest)
	}

	// Proof can be empty for some transaction types
	var proof []byte
	if txoutput.Proof != "" {
		proof, err = hexutil.Decode(txoutput.Proof)
		if err != nil {
			return types.TxOutput{}, fmt.Errorf("decode proof error: %v, proof: %s", err, txoutput.Proof)
		}
	}

	locktime, err := strconv.ParseUint(txoutput.LockTime, 10, 64)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("parse locktime error: %v, locktime: %s", err, txoutput.LockTime)
	}

	bciType, err := strconv.ParseInt(txoutput.BciType, 10, 64)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("parse bcitype error: %v, bcitype: %s", err, txoutput.BciType)
	}

	// Data can be empty for some transaction types
	var data []byte
	if txoutput.Data != "" {
		data, err = hexutil.Decode(txoutput.Data)
		if err != nil {
			return types.TxOutput{}, fmt.Errorf("decode data error: %v, data: %s", err, txoutput.Data)
		}
	}

	burnlock, err := strconv.ParseUint(txoutput.BurnLock, 10, 64)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("parse burnlock error: %v, burnlock: %s", err, txoutput.BurnLock)
	}

	rate, err := strconv.ParseFloat(txoutput.Rate, 64)
	if err != nil {
		return types.TxOutput{}, fmt.Errorf("parse rate error: %v, rate: %s", err, txoutput.Rate)
	}

	return types.TxOutput{
		Address:  addr,
		Value:    value,
		Interest: interest,
		Proof:    proof,
		LockTime: locktime,
		BciType:  int32(bciType),
		Data:     data,
		BurnLock: burnlock,
		Rate:     rate,
	}, nil
}
