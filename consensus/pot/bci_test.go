package pot

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

func TestTestTx(t *testing.T) {
	txid, _ := hexutil.Decode("0x89e5feab71c9ccd7f3a999605d012eaf208703a6130b91dc6328fd4a9e976d0d")
	addr, _ := hexutil.Decode("0xa47155b42648816998e576ffbfa045ab00000000000000007662a30d4b5875b73a8768fcf01bf52d2326a7660e24033a")
	input := types.TxInput{
		Txid:      crypto.Convert(txid),
		Voutput:   0,
		Scriptsig: addr,
		Address:   addr,
		Value:     32768,
		BciType:   1,
	}
	data, _ := hexutil.Decode("0x00")
	output := types.TxOutput{
		Address:  addr,
		Value:    32768,
		BciType:  1,
		Interest: 10,
		LockTime: 7,
		Data:     data,
		BurnLock: 62560,
		Rate:     0.00,
	}

	tx := types.RawTx{
		TxInput:        []types.TxInput{input},
		TxOutput:       []types.TxOutput{output},
		TransactionFee: 0,
	}
	tx.Txid = tx.Hash()
	fmt.Println(hexutil.Encode(tx.Txid[:]))
	fmt.Println(hexutil.Encode(data))
	fmt.Println(tx.TxOutput[0].Rate)
	t.Log(hexutil.Encode(tx.Txid[:]))
}
