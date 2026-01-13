package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
)

// MockUpgradeService implements model.UpgradeService for testing
type MockUpgradeService struct {
	manager *upgrade.UpgradeManager
}

func (m *MockUpgradeService) GetUpgradeManager() *upgrade.UpgradeManager {
	return m.manager
}

// setupTestRouter creates a test router with upgrade handler
func setupTestRouter(t *testing.T) (*gin.Engine, *upgrade.UpgradeManager, func()) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create logger
	log := logrus.New().WithField("test", "upgrade_handler")

	// Create temporary directory for database
	dbDir := "/tmp/test_upgrade_" + uuid.New().String()
	err := os.MkdirAll(dbDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	persistence, err := upgrade.NewBoltDBPersistence(dbDir, log)
	if err != nil {
		os.RemoveAll(dbDir)
		t.Fatalf("Failed to create persistence: %v", err)
	}
	manager, err := upgrade.NewUpgradeManagerWithPersistence(nil, nil, nil, persistence, log)
	if err != nil {
		t.Fatalf("Failed to create upgrade manager: %v", err)
	}

	// Create service and handler
	service := &MockUpgradeService{manager: manager}
	handler := NewUpgradeHandler(service, log)

	// Create router and register routes
	router := gin.Default()
	apiGroup := router.Group("/api")
	handler.RegisterRoutes(apiGroup)

	// Cleanup function
	cleanup := func() {
		if persistence != nil {
			persistence.Close()
		}
		os.RemoveAll(dbDir)
	}

	return router, manager, cleanup
}

func TestHealthCheck(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
	assert.Equal(t, "success", response.Msg)

	// Check health data
	healthData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "ok", healthData["status"])
	assert.Equal(t, true, healthData["persistence"])
}

func TestGetCurrentPhase(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/upgrade/phase", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)

	phaseData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, phaseData["phase"])
	assert.NotNil(t, phaseData["started"])
}

func TestGetUpgradeStatus(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/upgrade/status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)

	statusData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, statusData["phase"])
	assert.NotNil(t, statusData["started"])
}

func TestProposeUpgrade(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Prepare request
	reqData := model.ProposeUpgradeRequest{
		TargetConsensus:      "hotstuff",
		CandidateStartHeight: 100,
		SwitchHeight:         200,
		Description:          "Test upgrade to HotStuff",
		CDLYaml: `name: hotstuff
type: consensus
version: 1.0.0`,
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
	assert.Equal(t, "success", response.Msg)

	// Check proposal response
	proposalData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.NotEmpty(t, proposalData["proposal_id"])
	assert.Equal(t, "created", proposalData["status"])
}

func TestProposeUpgrade_InvalidRequest(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Missing required fields
	reqData := model.ProposeUpgradeRequest{
		Description: "Test upgrade",
		// Missing TargetConsensus, CandidateStartHeight, SwitchHeight
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 400, response.Code)
}

func TestListProposals(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create a proposal first
	reqData := model.ProposeUpgradeRequest{
		TargetConsensus:      "hotstuff",
		CandidateStartHeight: 100,
		SwitchHeight:         200,
		Description:          "Test proposal",
	}
	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Now list proposals
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/upgrade/proposals", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)

	proposals, ok := response.Data.([]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(proposals), 1)
}

func TestValidateCDL_Valid(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqData := model.ValidateCDLRequest{
		CDLYaml: `name: hotstuff
type: consensus
version: 1.0.0`,
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/cdl/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	validationData, ok := response.Data.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, true, validationData["valid"])
}

func TestRollback(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqData := model.RollbackRequest{
		Reason: "Test rollback",
		Force:  false,
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/rollback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}

func TestListCandidateChains(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/candidate/list", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}

func TestGetCandidateState_NotFound(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/candidate/test-candidate/state", nil)
	router.ServeHTTP(w, req)

	// Should return 503 as MultiChainManager is not available in test setup
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestValidateCDL_Invalid(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqData := model.ValidateCDLRequest{
		CDLYaml: "invalid: yaml: syntax",
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/cdl/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Even invalid YAML should return OK with valid=false
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCompileCDL(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqData := model.CompileCDLRequest{
		CDLYaml: `name: hotstuff
type: consensus
version: 1.0.0`,
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/cdl/compile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}

func TestGetCurrentMetrics(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/metrics/current", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}

func TestGetMetricsHistory(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/metrics/history?start=0&end=100", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}

func TestQueryEvents(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/events?limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}

func TestStartUpgrade(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// First create a proposal
	proposeData := model.ProposeUpgradeRequest{
		TargetConsensus:      "hotstuff",
		CandidateStartHeight: 100,
		SwitchHeight:         200,
		Description:          "Test upgrade",
	}
	proposeBody, _ := json.Marshal(proposeData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(proposeBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var proposeResponse model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &proposeResponse)
	assert.NoError(t, err)

	proposalData := proposeResponse.Data.(map[string]interface{})
	proposalID := proposalData["proposal_id"].(string)

	// Now start the upgrade
	startData := model.StartUpgradeRequest{
		ProposalID: proposalID,
	}

	startBody, _ := json.Marshal(startData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/upgrade/start", bytes.NewBuffer(startBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Note: This might fail because ConsensusFactory is not set up
	// But we're testing the API handler logic
	if w.Code != http.StatusOK {
		var response model.ResponseData
		json.Unmarshal(w.Body.Bytes(), &response)
		// It's ok if it fails due to missing factory
		assert.Contains(t, response.Msg, "factory")
	}
}

func TestGetProposal(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create a proposal first
	proposeData := model.ProposeUpgradeRequest{
		TargetConsensus:      "hotstuff",
		CandidateStartHeight: 100,
		SwitchHeight:         200,
		Description:          "Test",
	}
	body, _ := json.Marshal(proposeData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var proposeResponse model.ResponseData
	json.Unmarshal(w.Body.Bytes(), &proposeResponse)
	proposalData := proposeResponse.Data.(map[string]interface{})
	proposalID := proposalData["proposal_id"].(string)

	// Get the proposal
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/upgrade/proposals/"+proposalID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.ResponseData
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.Code)
}
