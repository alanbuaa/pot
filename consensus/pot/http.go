package pot

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

var port = 18025

type requestData struct {
	Transaction HTTPTransaction `json:"transaction"`
	Type        string          `json:"type"`
}

type HTTPTransaction struct {
	Txid           string         `json:"txid"`
	TxInputs       []HTTPTxInput  `json:"txInputs"`
	TxOutputs      []HTTPTxOutput `json:"txOutputs"`
	TransactionFee string         `json:"transactionFee"`
}

type HTTPTxInput struct {
	Txid      string `json:"txid"`
	Voutput   string `json:"voutput"`
	ScriptSig string `json:"scriptSig"`
	Value     string `json:"value"`
	Address   string `json:"address"`
	BciType   string `json:"bciType"`
}

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

func HttpTx2Tx(tx *HTTPTransaction) (*types.RawTx, error) {
	txInputs := make([]types.TxInput, 0)
	for _, txInput := range tx.TxInputs {
		txinput, err := HttpTxInput2TxInput(txInput)
		if err != nil {
			return nil, err
		}
		txInputs = append(txInputs, txinput)
	}
	txOutputs := make([]types.TxOutput, 0)
	for _, txOutput := range tx.TxOutputs {
		txoutput, err := HttpTxOutput2TxOutput(txOutput)
		if err != nil {
			return nil, err
		}
		txOutputs = append(txOutputs, txoutput)
	}
	txid, err := hexutil.Decode(tx.Txid)
	if err != nil {
		return nil, err
	}
	transactionfee, err := strconv.ParseInt(tx.TransactionFee, 10, 64)
	if err != nil {
		return nil, err
	}

	return &types.RawTx{
		Txid:           crypto.Convert(txid),
		TxInput:        txInputs,
		TxOutput:       txOutputs,
		TransactionFee: transactionfee,
	}, nil

}

func HttpTxInput2TxInput(txinput HTTPTxInput) (types.TxInput, error) {
	b, err := hexutil.Decode(txinput.Txid)
	if err != nil {
		return types.TxInput{}, err
	}
	voutput, err := strconv.ParseInt(txinput.Voutput, 10, 64)
	if err != nil {
		return types.TxInput{}, err
	}
	scriptsig, err := hexutil.Decode(txinput.ScriptSig)
	if err != nil {
		return types.TxInput{}, err
	}
	value, err := strconv.ParseInt(txinput.Value, 10, 64)
	if err != nil {
		return types.TxInput{}, err
	}
	addr, err := hexutil.Decode(txinput.Address)
	if err != nil {
		return types.TxInput{}, err
	}
	bcitype, err := strconv.ParseInt(txinput.BciType, 10, 64)
	if err != nil {
		return types.TxInput{}, err
	}
	fmt.Println("http bcitype ", txinput.BciType)
	fmt.Println("turn into bcitype: ", bcitype)
	return types.TxInput{
		Txid:      crypto.Convert(b),
		Voutput:   voutput,
		Scriptsig: scriptsig,
		Value:     value,
		Address:   addr,
		BciType:   int32(bcitype),
	}, nil
}

func HttpTxOutput2TxOutput(txoutput HTTPTxOutput) (types.TxOutput, error) {
	addr, err := hexutil.Decode(txoutput.Address)
	if err != nil {
		return types.TxOutput{}, err
	}
	value, err := strconv.ParseInt(txoutput.Value, 10, 64)
	if err != nil {
		return types.TxOutput{}, err
	}
	interest, err := strconv.ParseInt(txoutput.Interest, 10, 64)
	if err != nil {
		return types.TxOutput{}, err
	}
	proof, err := hexutil.Decode(txoutput.Proof)
	if err != nil {
		return types.TxOutput{}, err
	}
	locktime, err := strconv.ParseUint(txoutput.LockTime, 10, 64)
	if err != nil {
		return types.TxOutput{}, err
	}
	bciType, err := strconv.ParseInt(txoutput.BciType, 10, 64)
	if err != nil {
		return types.TxOutput{}, err
	}
	data, err := hexutil.Decode(txoutput.Data)
	if err != nil {
		return types.TxOutput{}, err
	}
	burnlock, err := strconv.ParseUint(txoutput.BurnLock, 10, 64)
	if err != nil {
		return types.TxOutput{}, err
	}
	rate, err := strconv.ParseFloat(txoutput.Rate, 64)
	if err != nil {
		return types.TxOutput{}, err

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

type ResponseData struct {
}

func setHTTPservice(w *Worker) *gin.Engine {
	r := gin.Default()
	// gin.SetMode(gin.ReleaseMode)
	createTransaction := r.Group("/api")
	{
		createTransaction.POST("/createlocktransaction", func(c *gin.Context) { handlerCreateLockTransaction(c, w) })
		createTransaction.POST("/locktransfertransaction", func(ctx *gin.Context) { handlerLockTransferTransaction(ctx, w) })
		createTransaction.POST("/nonlocktransfertransaction", func(ctx *gin.Context) { handlerNonLockTransferTransaction(ctx, w) })
		createTransaction.POST("/devastatetransaction", func(ctx *gin.Context) { handlerDevastateTransaction(ctx, w) })
		createTransaction.GET("/getblockheight", func(ctx *gin.Context) { handlerGetBlockHeight(ctx, w) })
		createTransaction.POST("/hello", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"code": 200,
				"msg":  "success",
			})
		})
	}
	return r
}

func startHTTPserve(w *Worker) *gin.Engine {
	r := setHTTPservice(w)
	go r.Run(fmt.Sprintf("0.0.0.0:%d", port))
	return r
}

func handlerCreateLockTransaction(c *gin.Context, w *Worker) {
	var request requestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "decode request body error",
		})
		fmt.Println("error for :decode request body error")
		return
	}
	fmt.Println(request.Type)
	fmt.Println(request.Transaction.Txid)

	transaction := request.Transaction
	tx, err := HttpTx2Tx(&transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		fmt.Println("error for :", err.Error())
		return
	}
	fmt.Println("test:", hexutil.Encode(tx.Txid[:]))
	err = w.checkLockTransaction(tx)
	fmt.Printf("input bcitype %d, output bci type %d", tx.TxInput[0].BciType, tx.TxOutput[0].BciType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		fmt.Println("error for :", err.Error())
		return
	}
	err = w.broadcastClientTransaction(tx, pb.TxType_CreateLockTransaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		fmt.Println("error for :", err.Error())
		return
	}
	w.mempool.AddRawTx(tx)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
	})
}

func handlerLockTransferTransaction(c *gin.Context, w *Worker) {
	var request requestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "decode request body error",
		})
		w.log.Error("error for :decode request body error")
	}

	w.log.Infof("receive lock transfer transaction")

	transaction := request.Transaction
	tx, err := HttpTx2Tx(&transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}

	err = w.CheckLockTransferTransaction(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}

	err = w.broadcastClientTransaction(tx, pb.TxType_LockTransferTranscation)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}

	w.mempool.AddRawTx(tx)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
	})

}

func handlerNonLockTransferTransaction(c *gin.Context, w *Worker) {
	var request requestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "decode request body error",
		})
		w.log.Error("error for :decode request bodyerror")
	}

	w.log.Infof("receive non lock transfer transaction")
	transaction := request.Transaction
	tx, err := HttpTx2Tx(&transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}
	err = w.CheckNonLockTransferTransaction(tx)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}
	err = w.broadcastClientTransaction(tx, pb.TxType_NonLockTransferTranscation)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}

	w.mempool.AddRawTx(tx)

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
	})
	return
}

func handlerDevastateTransaction(c *gin.Context, w *Worker) {
	var request requestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  "decode request body error",
		})
		w.log.Error("error for :decode request body error")
	}
	w.log.Infof("receive devastate transaction")
	transaction := request.Transaction
	tx, err := HttpTx2Tx(&transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}
	err = w.CheckDevastateTransaction(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}
	err = w.broadcastClientTransaction(tx, pb.TxType_DevasteTransaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 500,
			"msg":  err.Error(),
		})
		w.log.Error("error for :", err.Error())
		return
	}
	w.mempool.AddRawTx(tx)
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
	})
	return
}

func handlerGetBlockHeight(c *gin.Context, w *Worker) {
	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"msg":    "success",
		"height": w.chainReader.GetCurrentHeight(),
	})
	return
}
