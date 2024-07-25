package types

type DciTx struct {
	Txid     []byte
	TxInput  []TxInput
	TxOutput []TxOutput
}

// TODO: need to finish
func (t *DciTx) Hash() []byte {
	if t.Txid != nil {
		return t.Txid
	}
	hashes := make([]byte, 0)
	return hashes
}

func (t *DciTx) EncodeToByte() []byte {
	hashes := make([]byte, 0)
	return hashes
}

func (t *DciTx) NewCoinbaseTx(to []byte, value int64, data []byte) *DciTx {
	if len(data) == 0 || data == nil {
		data = RandByte()
	}

	txin := TxInput{
		IsCoinbase: false,
		Txid:       [32]byte{},
		Voutput:    -1,
		Scriptsig:  nil,
		Value:      value,
		Address:    to,
	}
	txout := TxOutput{
		Address:  to,
		Value:    value,
		IsSpent:  false,
		ScriptPk: nil,
	}

	tx := &DciTx{
		Txid:     nil,
		TxInput:  []TxInput{txin},
		TxOutput: []TxOutput{txout},
	}
	tx.Txid = tx.Hash()
	return tx
}
