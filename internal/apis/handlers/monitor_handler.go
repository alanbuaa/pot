package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
)

// MonitorHandler handles HTTP requests for monitoring and visualization
type MonitorHandler struct {
	service model.MonitorService
	log     *logrus.Entry
}

// NewMonitorHandler creates a new monitor handler instance
func NewMonitorHandler(service model.MonitorService, log *logrus.Entry) *MonitorHandler {
	return &MonitorHandler{
		service: service,
		log:     log,
	}
}

// RegisterRoutes registers all monitoring routes to the router group
func (h *MonitorHandler) RegisterRoutes(group *gin.RouterGroup) {
	// System routes
	group.GET("/system/overview", h.HandleSystemOverview)

	// POT consensus routes
	group.GET("/pot/status", h.HandlePOTStatus)
	group.GET("/pot/vdf", h.HandleVDFStatus)

	// Committee routes
	group.GET("/committee/status", h.HandleCommitteeStatus)

	// BCI routes
	group.GET("/bci/status", h.HandleBCIStatus)

	// Mempool routes
	group.GET("/mempool/status", h.HandleMempoolStatus)

	// Network routes
	group.GET("/network/topology", h.HandleNetworkTopology)
}

// HandleSystemOverview handles system overview requests
func (h *MonitorHandler) HandleSystemOverview(c *gin.Context) {
	overview, err := h.service.GetSystemOverview()
	if err != nil {
		h.log.Errorf("Failed to get system overview: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// HandlePOTStatus handles POT consensus status requests
func (h *MonitorHandler) HandlePOTStatus(c *gin.Context) {
	status, err := h.service.GetPOTStatus()
	if err != nil {
		h.log.Errorf("Failed to get POT status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleVDFStatus handles VDF computation status requests
func (h *MonitorHandler) HandleVDFStatus(c *gin.Context) {
	status, err := h.service.GetVDFStatus()
	if err != nil {
		h.log.Errorf("Failed to get VDF status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleCommitteeStatus handles committee status requests
func (h *MonitorHandler) HandleCommitteeStatus(c *gin.Context) {
	status, err := h.service.GetCommitteeStatus()
	if err != nil {
		h.log.Errorf("Failed to get committee status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleBCIStatus handles BCI incentive status requests
func (h *MonitorHandler) HandleBCIStatus(c *gin.Context) {
	status, err := h.service.GetBCIStatus()
	if err != nil {
		h.log.Errorf("Failed to get BCI status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleMempoolStatus handles mempool status requests
func (h *MonitorHandler) HandleMempoolStatus(c *gin.Context) {
	status, err := h.service.GetMempoolStatus()
	if err != nil {
		h.log.Errorf("Failed to get mempool status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// HandleNetworkTopology handles network topology requests
func (h *MonitorHandler) HandleNetworkTopology(c *gin.Context) {
	topology, err := h.service.GetNetworkTopology()
	if err != nil {
		h.log.Errorf("Failed to get network topology: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Internal Server Error",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, topology)
}
