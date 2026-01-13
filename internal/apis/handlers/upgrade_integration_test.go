package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
)

// TestUpgradeWorkflow tests the complete upgrade workflow from proposal to completion
func TestUpgradeWorkflow(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	var proposalID string

	t.Run("Step1_CreateProposal", func(t *testing.T) {
		reqData := model.ProposeUpgradeRequest{
			TargetConsensus:      "hotstuff",
			CandidateStartHeight: 100,
			SwitchHeight:         200,
			ForkHeight:           50,
			Description:          "End-to-end test upgrade",
			CDLYaml: `name: hotstuff
type: consensus
version: 1.0.0
parameters:
  block_time: 1.0`,
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, 200, response.Code)

		proposalData := response.Data.(map[string]interface{})
		proposalID = proposalData["proposal_id"].(string)
		require.NotEmpty(t, proposalID)

		t.Logf("Created proposal: %s", proposalID)
	})

	t.Run("Step2_GetProposal", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/upgrade/proposals/"+proposalID, nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.Equal(t, 200, response.Code)

		proposalData := response.Data.(map[string]interface{})
		assert.Equal(t, "hotstuff", proposalData["target_consensus"])
	})

	t.Run("Step3_ListProposals", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/upgrade/proposals", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		proposals := response.Data.([]interface{})
		assert.GreaterOrEqual(t, len(proposals), 1)
	})

	t.Run("Step4_CheckStatus_BeforeStart", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/upgrade/status", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		statusData := response.Data.(map[string]interface{})
		assert.False(t, statusData["started"].(bool))
	})

	t.Run("Step5_ValidateCDL", func(t *testing.T) {
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

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		validationData := response.Data.(map[string]interface{})
		assert.True(t, validationData["valid"].(bool))
	})

	t.Run("Step6_CheckHealth", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/health", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		healthData := response.Data.(map[string]interface{})
		assert.Equal(t, "ok", healthData["status"])
		assert.True(t, healthData["persistence"].(bool))
	})

	t.Run("Step7_CheckMetrics", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/metrics/current", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
	})

	t.Run("Step8_Rollback", func(t *testing.T) {
		reqData := model.RollbackRequest{
			Reason: "End-to-end test rollback",
			Force:  false,
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/upgrade/rollback", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 200, response.Code)
	})
}

// TestConcurrentProposals tests creating multiple proposals concurrently
func TestConcurrentProposals(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	numProposals := 5
	results := make(chan bool, numProposals)

	for i := 0; i < numProposals; i++ {
		go func(index int) {
			reqData := model.ProposeUpgradeRequest{
				TargetConsensus:      "hotstuff",
				CandidateStartHeight: uint64(100 + index*100),
				SwitchHeight:         uint64(200 + index*100),
				Description:          fmt.Sprintf("Concurrent proposal %d", index),
			}

			body, _ := json.Marshal(reqData)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			results <- w.Code == http.StatusOK
		}(i)
	}

	// Wait for all to complete
	successCount := 0
	for i := 0; i < numProposals; i++ {
		if <-results {
			successCount++
		}
	}

	assert.Equal(t, numProposals, successCount, "All concurrent proposals should succeed")

	// Verify all proposals are listed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/upgrade/proposals", nil)
	router.ServeHTTP(w, req)

	var response model.ResponseData
	json.Unmarshal(w.Body.Bytes(), &response)
	proposals := response.Data.([]interface{})
	assert.GreaterOrEqual(t, len(proposals), numProposals)
}

// TestPersistenceReload tests that proposals persist across manager reloads
func TestPersistenceReload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logrus.New().WithField("test", "persistence_reload")

	// Create temporary directory
	dbDir := "/tmp/test_persistence_" + uuid.New().String()
	err := os.MkdirAll(dbDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	var proposalID string

	// Phase 1: Create proposal and save
	{
		persistence, err := upgrade.NewBoltDBPersistence(dbDir, log)
		require.NoError(t, err)

		_, err = upgrade.NewUpgradeManagerWithPersistence(nil, nil, nil, persistence, log)
		require.NoError(t, err)

		// Create proposal
		proposalUUID := uuid.New()
		var pID [32]byte
		copy(pID[:], proposalUUID[:])

		proposal := &upgrade.UpgradeProposal{
			ProposalID:         pID,
			TargetConsensus:    "hotstuff",
			PreexecStartHeight: 100,
			SwitchHeight:       200,
			Description:        "Persistence test",
			Timestamp:          time.Now(),
		}

		err = persistence.SaveProposal(proposal)
		require.NoError(t, err)

		proposalID = proposalUUID.String()

		persistence.Close()
	}

	// Phase 2: Reload and verify
	{
		persistence, err := upgrade.NewBoltDBPersistence(dbDir, log)
		require.NoError(t, err)
		defer persistence.Close()

		// Parse proposal ID
		proposalUUID, err := uuid.Parse(proposalID)
		require.NoError(t, err)

		var pID [32]byte
		copy(pID[:], proposalUUID[:])

		// Load proposal
		loadedProposal, err := persistence.LoadProposal(pID)
		require.NoError(t, err)
		require.NotNil(t, loadedProposal)

		assert.Equal(t, "hotstuff", loadedProposal.TargetConsensus)
		assert.Equal(t, uint64(100), loadedProposal.PreexecStartHeight)
		assert.Equal(t, uint64(200), loadedProposal.SwitchHeight)
		assert.Equal(t, "Persistence test", loadedProposal.Description)
	}
}

// TestAPIErrorCases tests various error scenarios
func TestAPIErrorCases(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		reqData := model.ProposeUpgradeRequest{
			Description: "Missing required fields",
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NonExistentProposal", func(t *testing.T) {
		fakeID := uuid.New().String()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/upgrade/proposals/"+fakeID, nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("InvalidProposalIDFormat", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/upgrade/proposals/invalid-id-format", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("StartWithInvalidProposal", func(t *testing.T) {
		reqData := model.StartUpgradeRequest{
			ProposalID: uuid.New().String(),
		}

		body, _ := json.Marshal(reqData)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/upgrade/start", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestMetricsAndEvents tests metrics collection and event querying
func TestMetricsAndEvents(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	// Create a proposal to generate events
	reqData := model.ProposeUpgradeRequest{
		TargetConsensus:      "hotstuff",
		CandidateStartHeight: 100,
		SwitchHeight:         200,
		Description:          "Metrics test",
	}

	body, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/upgrade/propose", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	t.Run("GetCurrentMetrics", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/metrics/current", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})

	t.Run("GetMetricsHistory", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/metrics/history?start=0&end=1000", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})

	t.Run("QueryEvents", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/events?limit=100", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ResponseData
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})
}
