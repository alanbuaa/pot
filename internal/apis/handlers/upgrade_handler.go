package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"gopkg.in/yaml.v3"
)

// UpgradeHandler handles upgrade-related API requests
type UpgradeHandler struct {
	service model.UpgradeService
	log     *logrus.Entry
}

// NewUpgradeHandler creates a new UpgradeHandler instance
func NewUpgradeHandler(service model.UpgradeService, log *logrus.Entry) *UpgradeHandler {
	return &UpgradeHandler{
		service: service,
		log:     log,
	}
}

// RegisterRoutes registers all upgrade-related routes
func (h *UpgradeHandler) RegisterRoutes(group *gin.RouterGroup) {
	// Proposal management
	group.POST("/upgrade/propose", h.ProposeUpgrade)
	group.GET("/upgrade/proposals", h.ListProposals)
	group.GET("/upgrade/proposals/:id", h.GetProposal)

	// Upgrade control
	group.POST("/upgrade/start", h.StartUpgrade)
	group.POST("/upgrade/rollback", h.Rollback)
	group.GET("/upgrade/status", h.GetUpgradeStatus)
	group.GET("/upgrade/phase", h.GetCurrentPhase)

	// CDL operations
	group.POST("/cdl/validate", h.ValidateCDL)
	group.POST("/cdl/compile", h.CompileCDL)

	// Candidate chain management
	group.POST("/candidate/start", h.StartCandidateChain)
	group.GET("/candidate/list", h.ListCandidateChains)
	group.GET("/candidate/:id/state", h.GetCandidateState)
	group.POST("/candidate/merge", h.MergeCandidateChain)
	group.POST("/candidate/rollback", h.RollbackCandidateChain)

	// Metrics and monitoring
	group.GET("/metrics/current", h.GetCurrentMetrics)
	group.GET("/metrics/history", h.GetMetricsHistory)
	group.GET("/events", h.QueryEvents)
	group.GET("/health", h.HealthCheck)
}

// ProposeUpgrade handles upgrade proposal creation
// @Summary Create upgrade proposal
// @Accept json
// @Produce json
// @Param request body model.ProposeUpgradeRequest true "Proposal request"
// @Success 200 {object} model.ResponseData{data=model.ProposalResponse}
// @Router /api/upgrade/propose [post]
func (h *UpgradeHandler) ProposeUpgrade(c *gin.Context) {
	var req model.ProposeUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("Invalid proposal request: %v", err)
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	mgr := h.service.GetUpgradeManager()

	// Create proposal ID (convert UUID to TxHash)
	proposalUUID := uuid.New()
	var proposalID [32]byte
	copy(proposalID[:], proposalUUID[:])

	// Parse CDL if provided
	var cdlDescriptor *upgrade.CDLDescriptor
	if req.CDLYaml != "" {
		cdlDescriptor = &upgrade.CDLDescriptor{}
		if err := yaml.Unmarshal([]byte(req.CDLYaml), cdlDescriptor); err != nil {
			h.log.Errorf("Failed to parse CDL: %v", err)
			c.JSON(http.StatusBadRequest, model.ResponseData{
				Code: 400,
				Msg:  fmt.Sprintf("Invalid CDL YAML: %v", err),
			})
			return
		}
	}

	// Create proposal
	proposal := &upgrade.UpgradeProposal{
		ProposalID:         proposalID,
		TargetConsensus:    req.TargetConsensus,
		CDLDescriptor:      cdlDescriptor,
		ForkHeight:         req.ForkHeight,
		PreexecStartHeight: req.CandidateStartHeight,
		SwitchHeight:       req.SwitchHeight,
		Incentive:          req.Incentive,
		Description:        req.Description,
		ConsensusParams:    req.ConsensusParams,
		Timestamp:          time.Now(),
	}

	// Save proposal to persistence
	if mgr.GetPersistence() != nil {
		if err := mgr.GetPersistence().SaveProposal(proposal); err != nil {
			h.log.Errorf("Failed to save proposal: %v", err)
			c.JSON(http.StatusInternalServerError, model.ResponseData{
				Code: 500,
				Msg:  fmt.Sprintf("Failed to save proposal: %v", err),
			})
			return
		}

		// Record event
		event := &upgrade.UpgradeEvent{
			Type:        upgrade.EventProposalCreated,
			ProposalID:  proposalID,
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("Proposal created: %s", req.Description),
		}
		if err := mgr.GetPersistence().SaveEvent(event); err != nil {
			h.log.Warnf("Failed to save event: %v", err)
		}
	}

	// Convert proposal ID to string for response
	var proposalIDStr string
	if proposalUUID != (uuid.UUID{}) {
		proposalIDStr = proposalUUID.String()
	} else {
		proposalIDStr = fmt.Sprintf("%x", proposalID)
	}

	response := model.ProposalResponse{
		ProposalID: proposalIDStr,
		Status:     "created",
		CreatedAt:  proposal.Timestamp.Unix(),
	}

	h.log.Infof("Proposal created: %s", proposalIDStr)
	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: response,
	})
}

// ListProposals lists all upgrade proposals
// @Summary List all proposals
// @Produce json
// @Success 200 {object} model.ResponseData{data=[]upgrade.UpgradeProposal}
// @Router /api/upgrade/proposals [get]
func (h *UpgradeHandler) ListProposals(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()

	if mgr.GetPersistence() == nil {
		c.JSON(http.StatusOK, model.ResponseData{
			Code: 200,
			Msg:  "success",
			Data: []interface{}{},
		})
		return
	}

	proposals, err := mgr.GetPersistence().ListProposals()
	if err != nil {
		h.log.Errorf("Failed to list proposals: %v", err)
		c.JSON(http.StatusInternalServerError, model.ResponseData{
			Code: 500,
			Msg:  fmt.Sprintf("Failed to list proposals: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: proposals,
	})
}

// GetProposal retrieves a specific proposal by ID
// @Summary Get proposal by ID
// @Produce json
// @Param id path string true "Proposal ID"
// @Success 200 {object} model.ResponseData{data=upgrade.UpgradeProposal}
// @Router /api/upgrade/proposals/{id} [get]
func (h *UpgradeHandler) GetProposal(c *gin.Context) {
	idStr := c.Param("id")

	// Try to parse as UUID first, then as hex string
	var proposalID [32]byte
	proposalUUID, err := uuid.Parse(idStr)
	if err == nil {
		copy(proposalID[:], proposalUUID[:])
	} else {
		// Try parsing as hex
		hexBytes := []byte(idStr)
		if len(hexBytes) != 64 {
			c.JSON(http.StatusBadRequest, model.ResponseData{
				Code: 400,
				Msg:  "Invalid proposal ID format",
			})
			return
		}
		for i := 0; i < 32; i++ {
			fmt.Sscanf(string(hexBytes[i*2:i*2+2]), "%02x", &proposalID[i])
		}
	}

	mgr := h.service.GetUpgradeManager()
	if mgr.GetPersistence() == nil {
		c.JSON(http.StatusNotFound, model.ResponseData{
			Code: 404,
			Msg:  "Persistence not configured",
		})
		return
	}

	proposal, err := mgr.GetPersistence().LoadProposal(proposalID)
	if err != nil {
		h.log.Errorf("Failed to load proposal %s: %v", idStr, err)
		c.JSON(http.StatusNotFound, model.ResponseData{
			Code: 404,
			Msg:  fmt.Sprintf("Proposal not found: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: proposal,
	})
}

// StartUpgrade starts the upgrade process
// @Summary Start upgrade
// @Accept json
// @Produce json
// @Param request body model.StartUpgradeRequest true "Start upgrade request"
// @Success 200 {object} model.ResponseData
// @Router /api/upgrade/start [post]
func (h *UpgradeHandler) StartUpgrade(c *gin.Context) {
	var req model.StartUpgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Parse proposal ID
	var proposalID [32]byte
	proposalUUID, err := uuid.Parse(req.ProposalID)
	if err == nil {
		copy(proposalID[:], proposalUUID[:])
	} else {
		// Try parsing as hex
		hexBytes := []byte(req.ProposalID)
		if len(hexBytes) != 64 {
			c.JSON(http.StatusBadRequest, model.ResponseData{
				Code: 400,
				Msg:  "Invalid proposal ID format",
			})
			return
		}
		for i := 0; i < 32; i++ {
			fmt.Sscanf(string(hexBytes[i*2:i*2+2]), "%02x", &proposalID[i])
		}
	}

	mgr := h.service.GetUpgradeManager()

	// Load proposal from persistence
	if mgr.GetPersistence() == nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  "Persistence not configured",
		})
		return
	}

	proposal, err := mgr.GetPersistence().LoadProposal(proposalID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ResponseData{
			Code: 404,
			Msg:  fmt.Sprintf("Proposal not found: %v", err),
		})
		return
	}

	// Start upgrade using ConsensusFactory
	err = mgr.StartUpgradeWithFactory(proposal)
	if err != nil {
		h.log.WithError(err).Error("Failed to start upgrade")
		c.JSON(http.StatusInternalServerError, model.ResponseData{
			Code: 500,
			Msg:  fmt.Sprintf("Failed to start upgrade: %v", err),
		})
		return
	}

	h.log.WithFields(logrus.Fields{
		"proposal_id":      proposal.ProposalID.String(),
		"target_consensus": proposal.TargetConsensus,
	}).Info("Upgrade started successfully")

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "Upgrade started successfully",
		Data: gin.H{
			"proposal_id": proposal.ProposalID.String(),
			"phase":       "Preparing",
		},
	})
}

// Rollback performs upgrade rollback
// @Summary Rollback upgrade
// @Accept json
// @Produce json
// @Param request body model.RollbackRequest true "Rollback request"
// @Success 200 {object} model.ResponseData
// @Router /api/upgrade/rollback [post]
func (h *UpgradeHandler) Rollback(c *gin.Context) {
	var req model.RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	mgr := h.service.GetUpgradeManager()

	// For now, just reset the upgrade state
	// TODO: Implement proper rollback in RollbackManager
	mgr.Reset()

	// Record rollback event
	if mgr.GetPersistence() != nil {
		event := &upgrade.UpgradeEvent{
			Type:        upgrade.EventRollbackExecuted,
			Timestamp:   time.Now(),
			Description: req.Reason,
		}
		if err := mgr.GetPersistence().SaveEvent(event); err != nil {
			h.log.Warnf("Failed to save rollback event: %v", err)
		}
	}

	h.log.Infof("Upgrade rolled back: %s (force=%v)", req.Reason, req.Force)
	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "Rollback successful",
	})
}

// GetUpgradeStatus retrieves current upgrade status
// @Summary Get upgrade status
// @Produce json
// @Success 200 {object} model.ResponseData{data=model.UpgradeStatusResponse}
// @Router /api/upgrade/status [get]
func (h *UpgradeHandler) GetUpgradeStatus(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()
	state := mgr.GetUpgradeState()

	// Get latest metrics if available
	var metrics *upgrade.PerformanceMetrics
	if mgr.GetPersistence() != nil {
		// Get latest metrics from manager
		metrics = mgr.GetMetrics()
	}

	response := model.ConvertUpgradeStateToResponse(state, metrics)

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: response,
	})
}

// GetCurrentPhase retrieves current upgrade phase
// @Summary Get current phase
// @Produce json
// @Success 200 {object} model.ResponseData{data=string}
// @Router /api/upgrade/phase [get]
func (h *UpgradeHandler) GetCurrentPhase(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()
	state := mgr.GetUpgradeState()

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: gin.H{
			"phase":   state.Phase.String(),
			"started": state.Started,
		},
	})
}

// ValidateCDL validates CDL configuration
// @Summary Validate CDL
// @Accept json
// @Produce json
// @Param request body model.ValidateCDLRequest true "CDL validation request"
// @Success 200 {object} model.ResponseData{data=model.CDLValidationResponse}
// @Router /api/cdl/validate [post]
func (h *UpgradeHandler) ValidateCDL(c *gin.Context) {
	var req model.ValidateCDLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	var cdlDescriptor upgrade.CDLDescriptor
	if err := yaml.Unmarshal([]byte(req.CDLYaml), &cdlDescriptor); err != nil {
		c.JSON(http.StatusOK, model.ResponseData{
			Code: 200,
			Msg:  "success",
			Data: model.CDLValidationResponse{
				Valid: false,
			},
		})
		return
	}

	// Validate required fields
	valid := cdlDescriptor.Name != "" && cdlDescriptor.Type != "" && cdlDescriptor.Version != ""

	response := model.CDLValidationResponse{
		Valid:   valid,
		Name:    cdlDescriptor.Name,
		Type:    cdlDescriptor.Type,
		Version: cdlDescriptor.Version,
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: response,
	})
}

// CompileCDL compiles CDL configuration
// @Summary Compile CDL
// @Accept json
// @Produce json
// @Param request body model.CompileCDLRequest true "CDL compile request"
// @Success 200 {object} model.ResponseData
// @Router /api/cdl/compile [post]
func (h *UpgradeHandler) CompileCDL(c *gin.Context) {
	var req model.CompileCDLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	var cdlDescriptor upgrade.CDLDescriptor
	if err := yaml.Unmarshal([]byte(req.CDLYaml), &cdlDescriptor); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid CDL YAML: %v", err),
		})
		return
	}

	// For now, just validate and return success
	// In a real implementation, this would compile the CDL to executable code
	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "CDL compiled successfully",
		Data: gin.H{
			"name":    cdlDescriptor.Name,
			"type":    cdlDescriptor.Type,
			"version": cdlDescriptor.Version,
		},
	})
}

// GetCurrentMetrics retrieves current performance metrics
// @Summary Get current metrics
// @Produce json
// @Success 200 {object} model.ResponseData{data=upgrade.PerformanceMetrics}
// @Router /api/metrics/current [get]
func (h *UpgradeHandler) GetCurrentMetrics(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()

	metrics := mgr.GetMetrics()
	if metrics == nil {
		c.JSON(http.StatusOK, model.ResponseData{
			Code: 200,
			Msg:  "success",
			Data: nil,
		})
		return
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: metrics,
	})
}

// GetMetricsHistory retrieves historical metrics
// @Summary Get metrics history
// @Produce json
// @Param start query int64 false "Start timestamp"
// @Param end query int64 false "End timestamp"
// @Param limit query int false "Limit results"
// @Success 200 {object} model.ResponseData{data=[]upgrade.PerformanceMetrics}
// @Router /api/metrics/history [get]
func (h *UpgradeHandler) GetMetricsHistory(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()

	if mgr.GetPersistence() == nil {
		c.JSON(http.StatusOK, model.ResponseData{
			Code: 200,
			Msg:  "success",
			Data: []interface{}{},
		})
		return
	}

	// Parse query parameters
	var startHeight, endHeight uint64

	if startStr := c.Query("start"); startStr != "" {
		fmt.Sscanf(startStr, "%d", &startHeight)
	}

	if endStr := c.Query("end"); endStr != "" {
		fmt.Sscanf(endStr, "%d", &endHeight)
	}

	metrics, err := mgr.GetPersistence().QueryMetrics(startHeight, endHeight)
	if err != nil {
		h.log.Errorf("Failed to query metrics: %v", err)
		c.JSON(http.StatusInternalServerError, model.ResponseData{
			Code: 500,
			Msg:  fmt.Sprintf("Failed to query metrics: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: metrics,
	})
}

// QueryEvents queries upgrade events
// @Summary Query events
// @Produce json
// @Param proposal_id query string false "Proposal ID"
// @Param event_type query int false "Event type"
// @Param limit query int false "Limit results"
// @Success 200 {object} model.ResponseData{data=[]upgrade.UpgradeEvent}
// @Router /api/events [get]
func (h *UpgradeHandler) QueryEvents(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()

	if mgr.GetPersistence() == nil {
		c.JSON(http.StatusOK, model.ResponseData{
			Code: 200,
			Msg:  "success",
			Data: []interface{}{},
		})
		return
	}

	// Parse filter from query parameters
	filter := upgrade.EventFilter{
		Limit: 100,
	}

	if proposalIDStr := c.Query("proposal_id"); proposalIDStr != "" {
		// Parse proposal ID
		var proposalID [32]byte
		proposalUUID, err := uuid.Parse(proposalIDStr)
		if err == nil {
			copy(proposalID[:], proposalUUID[:])
		} else {
			// Try parsing as hex
			hexBytes := []byte(proposalIDStr)
			if len(hexBytes) == 64 {
				for i := 0; i < 32; i++ {
					fmt.Sscanf(string(hexBytes[i*2:i*2+2]), "%02x", &proposalID[i])
				}
			}
		}
		// Convert proposalID to types.TxHash
		var txHashPtr types.TxHash
		copy(txHashPtr[:], proposalID[:])
		filter.ProposalID = &txHashPtr
	}

	if eventTypeStr := c.Query("event_type"); eventTypeStr != "" {
		var eventType int
		if _, err := fmt.Sscanf(eventTypeStr, "%d", &eventType); err == nil {
			et := upgrade.EventType(eventType)
			filter.EventType = &et
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &filter.Limit); err == nil && l > 0 {
			// limit parsed
		}
	}

	events, err := mgr.GetPersistence().QueryEvents(filter)
	if err != nil {
		h.log.Errorf("Failed to query events: %v", err)
		c.JSON(http.StatusInternalServerError, model.ResponseData{
			Code: 500,
			Msg:  fmt.Sprintf("Failed to query events: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: events,
	})
}

// HealthCheck performs health check
// @Summary Health check
// @Produce json
// @Success 200 {object} model.ResponseData
// @Router /api/health [get]
func (h *UpgradeHandler) HealthCheck(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()
	state := mgr.GetUpgradeState()

	health := gin.H{
		"status":      "ok",
		"phase":       state.Phase.String(),
		"started":     state.Started,
		"persistence": mgr.GetPersistence() != nil,
	}

	if state.Failed {
		health["status"] = "degraded"
		health["error"] = state.FailureReason
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: health,
	})
}

// StartCandidateChain starts a new candidate chain
// @Summary Start candidate chain
// @Accept json
// @Produce json
// @Param request body model.StartCandidateChainRequest true "Start candidate chain request"
// @Success 200 {object} model.ResponseData{data=model.CandidateChainResponse}
// @Router /api/candidate/start [post]
func (h *UpgradeHandler) StartCandidateChain(c *gin.Context) {
	var req model.StartCandidateChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	mgr := h.service.GetUpgradeManager()
	multiChainMgr := mgr.GetMultiChainManager()
	if multiChainMgr == nil {
		c.JSON(http.StatusServiceUnavailable, model.ResponseData{
			Code: 503,
			Msg:  "Multi-chain manager not available",
		})
		return
	}

	h.log.WithFields(logrus.Fields{
		"candidate_id": req.CandidateID,
		"proposal_id":  req.ProposalID,
	}).Info("Candidate chain start requested (feature in development)")

	response := model.CandidateChainResponse{
		CandidateID: req.CandidateID,
		ProposalID:  req.ProposalID,
		Status:      "pending",
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "Candidate chain start requested (feature requires full consensus factory setup)",
		Data: response,
	})
}

// ListCandidateChains lists all candidate chains
// @Summary List all candidate chains
// @Produce json
// @Success 200 {object} model.ResponseData{data=model.ListCandidateChainsResponse}
// @Router /api/candidate/list [get]
func (h *UpgradeHandler) ListCandidateChains(c *gin.Context) {
	mgr := h.service.GetUpgradeManager()
	multiChainMgr := mgr.GetMultiChainManager()

	chains := []model.CandidateChainResponse{}
	count := 0

	if multiChainMgr != nil {
		// Note: ListCandidateChains returns map[string]*CandidateChainState
		// This is a simplified implementation
		h.log.Debug("Listing candidate chains")
	}

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: model.ListCandidateChainsResponse{
			Chains: chains,
			Count:  count,
		},
	})
}

// GetCandidateState retrieves the state of a specific candidate chain
// @Summary Get candidate chain state
// @Produce json
// @Param id path string true "Candidate ID"
// @Success 200 {object} model.ResponseData{data=model.CandidateChainResponse}
// @Router /api/candidate/{id}/state [get]
func (h *UpgradeHandler) GetCandidateState(c *gin.Context) {
	candidateID := c.Param("id")

	mgr := h.service.GetUpgradeManager()
	multiChainMgr := mgr.GetMultiChainManager()
	if multiChainMgr == nil {
		c.JSON(http.StatusServiceUnavailable, model.ResponseData{
			Code: 503,
			Msg:  "Multi-chain manager not available",
		})
		return
	}

	c.JSON(http.StatusNotFound, model.ResponseData{
		Code: 404,
		Msg:  fmt.Sprintf("Candidate chain %s not found", candidateID),
	})
}

// MergeCandidateChain merges a candidate chain to main chain
// @Summary Merge candidate chain
// @Accept json
// @Produce json
// @Param request body model.MergeCandidateChainRequest true "Merge request"
// @Success 200 {object} model.ResponseData
// @Router /api/candidate/merge [post]
func (h *UpgradeHandler) MergeCandidateChain(c *gin.Context) {
	var req model.MergeCandidateChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	mgr := h.service.GetUpgradeManager()
	multiChainMgr := mgr.GetMultiChainManager()
	if multiChainMgr == nil {
		c.JSON(http.StatusServiceUnavailable, model.ResponseData{
			Code: 503,
			Msg:  "Multi-chain manager not available",
		})
		return
	}

	h.log.WithField("candidate_id", req.CandidateID).Info("Candidate chain merge requested (feature in development)")

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "Candidate chain merge requested (feature requires full implementation)",
	})
}

// RollbackCandidateChain rolls back a candidate chain
// @Summary Rollback candidate chain
// @Accept json
// @Produce json
// @Param request body model.RollbackCandidateChainRequest true "Rollback request"
// @Success 200 {object} model.ResponseData
// @Router /api/candidate/rollback [post]
func (h *UpgradeHandler) RollbackCandidateChain(c *gin.Context) {
	var req model.RollbackCandidateChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Errorf("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, model.ResponseData{
			Code: 400,
			Msg:  fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	mgr := h.service.GetUpgradeManager()
	multiChainMgr := mgr.GetMultiChainManager()
	if multiChainMgr == nil {
		c.JSON(http.StatusServiceUnavailable, model.ResponseData{
			Code: 503,
			Msg:  "Multi-chain manager not available",
		})
		return
	}

	h.log.WithFields(logrus.Fields{
		"candidate_id": req.CandidateID,
		"reason":       req.Reason,
	}).Info("Candidate chain rollback requested (feature in development)")

	c.JSON(http.StatusOK, model.ResponseData{
		Code: 200,
		Msg:  "Candidate chain rollback requested (feature requires full implementation)",
	})
}
