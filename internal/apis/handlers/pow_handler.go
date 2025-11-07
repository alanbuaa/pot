package handlers

import (
"net/http"

"github.com/gin-gonic/gin"
"github.com/sirupsen/logrus"
"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
)

// PowHandler handles HTTP requests for PoW consensus
type PowHandler struct {
	service model.PowConsensusService
	log     *logrus.Entry
}

// NewPowHandler creates a new PoW handler instance
func NewPowHandler(service model.PowConsensusService, log *logrus.Entry) *PowHandler {
	return &PowHandler{
		service: service,
		log:     log,
	}
}

// RegisterRoutes registers all PoW-specific routes to the router group
func (h *PowHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/getblockheight", h.HandleGetBlockHeight)
	group.POST("/hello", h.HandleHello)
	// TODO: Add more PoW-specific routes
	// group.POST("/submitwork", h.HandleSubmitWork)
	// group.GET("/getdifficulty", h.HandleGetDifficulty)
}

// HandleGetBlockHeight handles get block height requests
func (h *PowHandler) HandleGetBlockHeight(c *gin.Context) {
	height := h.service.GetCurrentHeight()
	c.JSON(http.StatusOK, model.BlockHeightResponse{
Code:   200,
Msg:    "success",
Height: height,
})
}

// HandleHello handles hello test requests
func (h *PowHandler) HandleHello(c *gin.Context) {
	c.JSON(http.StatusOK, model.ResponseData{
Code: 200,
Msg:  "success",
})
}
