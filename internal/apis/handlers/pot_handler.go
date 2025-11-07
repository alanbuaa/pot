package handlers

import (
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
)

// PotHandler handles HTTP requests for PoT consensus
type PotHandler struct {
	service model.PotConsensusService
	log     *logrus.Entry
}

// NewPotHandler creates a new PoT handler instance
func NewPotHandler(service model.PotConsensusService, log *logrus.Entry) *PotHandler {
	return &PotHandler{
		service: service,
		log:     log,
	}
}

// RegisterRoutes registers all PoT-specific routes to the router group
func (h *PotHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/createlocktransaction", h.HandleCreateLockTransaction)
	group.POST("/locktransfertransaction", h.HandleLockTransferTransaction)
	group.POST("/nonlocktransfertransaction", h.HandleNonLockTransferTransaction)
	group.POST("/devastatetransaction", h.HandleDevastateTransaction)
	group.GET("/getblockheight", h.HandleGetBlockHeight)
	group.POST("/hello", h.HandleHello)
}

// HandleCreateLockTransaction handles create lock transaction requests
func (h *PotHandler) HandleCreateLockTransaction(c *gin.Context) {
	var request model.RequestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  "decode request body error",
		})
		h.log.Error("error for: decode request body error")
		return
	}

	h.log.Infof("Received create lock transaction, type: %s, txid: %s", request.Type, request.Transaction.Txid)

	tx, err := model.HttpTx2Tx(&request.Transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("HttpTx2Tx error: %v", err)
		return
	}

	h.log.Debugf("Converted transaction: %s", hexutil.Encode(tx.Txid[:]))

	// Validate transaction
	err = h.service.CheckLockTransaction(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("CheckLockTransaction error: %v", err)
		return
	}

	// Broadcast transaction
	err = h.service.BroadcastClientTransaction(tx, pb.TxType_CreateLockTransaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("BroadcastClientTransaction error: %v", err)
		return
	}

	// Add to mempool
	h.service.AddRawTxToMempool(tx)

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
	})
}

// HandleLockTransferTransaction handles lock transfer transaction requests
func (h *PotHandler) HandleLockTransferTransaction(c *gin.Context) {
	var request model.RequestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  "decode request body error",
		})
		h.log.Error("error for: decode request body error")
		return
	}

	h.log.Infof("Received lock transfer transaction")

	tx, err := model.HttpTx2Tx(&request.Transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("HttpTx2Tx error: %v", err)
		return
	}

	err = h.service.CheckLockTransferTransaction(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("CheckLockTransferTransaction error: %v", err)
		return
	}

	err = h.service.BroadcastClientTransaction(tx, pb.TxType_LockTransferTranscation)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("BroadcastClientTransaction error: %v", err)
		return
	}

	h.service.AddRawTxToMempool(tx)

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
	})
}

// HandleNonLockTransferTransaction handles non-lock transfer transaction requests
func (h *PotHandler) HandleNonLockTransferTransaction(c *gin.Context) {
	var request model.RequestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  "decode request body error",
		})
		h.log.Error("error for: decode request body error")
		return
	}

	h.log.Infof("Received non lock transfer transaction")

	tx, err := model.HttpTx2Tx(&request.Transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("HttpTx2Tx error: %v", err)
		return
	}

	err = h.service.CheckNonLockTransferTransaction(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("CheckNonLockTransferTransaction error: %v", err)
		return
	}

	err = h.service.BroadcastClientTransaction(tx, pb.TxType_NonLockTransferTranscation)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("BroadcastClientTransaction error: %v", err)
		return
	}

	h.service.AddRawTxToMempool(tx)

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
	})
}

// HandleDevastateTransaction handles devastate transaction requests
func (h *PotHandler) HandleDevastateTransaction(c *gin.Context) {
	var request model.RequestData

	if err := c.ShouldBindBodyWithJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  "decode request body error",
		})
		h.log.Error("error for: decode request body error")
		return
	}

	h.log.Infof("Received devastate transaction")

	tx, err := model.HttpTx2Tx(&request.Transaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("HttpTx2Tx error: %v", err)
		return
	}

	err = h.service.CheckDevastateTransaction(tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("CheckDevastateTransaction error: %v", err)
		return
	}

	err = h.service.BroadcastClientTransaction(tx, pb.TxType_DevasteTransaction)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 500,
			Msg:  err.Error(),
		})
		h.log.Errorf("BroadcastClientTransaction error: %v", err)
		return
	}

	h.service.AddRawTxToMempool(tx)

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
	})
}

// HandleGetBlockHeight handles get block height requests
func (h *PotHandler) HandleGetBlockHeight(c *gin.Context) {
	height := h.service.GetCurrentHeight()
	c.JSON(http.StatusOK, model.BlockHeightResponse{
		Code:   200,
		Msg:    "success",
		Height: height,
	})
}

// HandleHello handles hello test requests
func (h *PotHandler) HandleHello(c *gin.Context) {
	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
	})
}
