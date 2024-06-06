package pot

import (
	"github.com/zzz136454872/upgradeable-consensus/types"
	"math/big"
)

func (w *Worker) GenerateCoinbaseTx(pubkeybyte []byte) *types.Tx {
	coinbasetx := &types.RawTx{
		ChainID:    big.NewInt(0),
		Nonce:      0,
		GasPrice:   big.NewInt(0),
		Gas:        0,
		To:         pubkeybyte,
		Data:       big.NewInt(0).Bytes(),
		Value:      big.NewInt(0),
		V:          big.NewInt(0),
		R:          big.NewInt(0),
		S:          big.NewInt(0),
		Accesslist: []byte(""),
	}
	txdata, _ := coinbasetx.EncodeToByte()
	return &types.Tx{Data: txdata}
}

func (w *Worker) ReceiveDCITx() {

}

func (w *Worker) VerifyDCITx() bool {
	return true
}
