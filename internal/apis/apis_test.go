package apis_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// MockPotService is a mock implementation of PotConsensusService for testing
type MockPotService struct {
	ValidateError  error
	BroadcastError error
	CurrentHeight  uint64
	AddedTxs       []*types.RawTx
}

func (m *MockPotService) CheckLockTransaction(rawtx *types.RawTx) error {
	return m.ValidateError
}

func (m *MockPotService) CheckLockTransferTransaction(rawtx *types.RawTx) error {
	return m.ValidateError
}

func (m *MockPotService) CheckNonLockTransferTransaction(rawtx *types.RawTx) error {
	return m.ValidateError
}

func (m *MockPotService) CheckDevastateTransaction(rawtx *types.RawTx) error {
	return m.ValidateError
}

func (m *MockPotService) BroadcastClientTransaction(rawtx *types.RawTx, txType pb.TxType) error {
	return m.BroadcastError
}

func (m *MockPotService) AddRawTxToMempool(rawtx *types.RawTx) {
	m.AddedTxs = append(m.AddedTxs, rawtx)
}

func (m *MockPotService) GetCurrentHeight() uint64 {
	return m.CurrentHeight
}

// TestApiServerCreation tests the creation of ApiServer
func TestApiServerCreation(t *testing.T) {
	logger := logrus.New()
	log := logger.WithField("test", "api")

	config := &apis.Config{Port: 18025}
	server := apis.NewApiServer(config, log)

	assert.NotNil(t, server)
}

// TestPotServiceRegistration tests registering a PoT service
func TestPotServiceRegistration(t *testing.T) {
	logger := logrus.New()
	log := logger.WithField("test", "api")

	config := &apis.Config{Port: 18025}
	server := apis.NewApiServer(config, log)

	mockService := &MockPotService{
		CurrentHeight: 100,
	}

	// This should not panic
	server.RegisterPotService(mockService)
}

// TestGetBlockHeightEndpoint tests the get block height endpoint
func TestGetBlockHeightEndpoint(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	logger := logrus.New()
	log := logger.WithField("test", "api")

	config := &apis.Config{Port: 18025}
	server := apis.NewApiServer(config, log)

	mockService := &MockPotService{
		CurrentHeight: 12345,
	}
	server.RegisterPotService(mockService)

	// Start server setup (without actually running it)
	server.Start()

	// Create a test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/getblockheight", nil)

	// We would need to get the engine from the server to test properly
	// For now, this shows the structure
	server.Engine.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	// Add more assertions as needed
}

// TestHttpTx2TxConversion tests HTTP transaction conversion
func TestHttpTx2TxConversion(t *testing.T) {
	httpTx := &model.HTTPTransaction{
		Txid:           "0x0000000000000000000000000000000000000000000000000000000000000001",
		TxInputs:       []model.HTTPTxInput{},
		TxOutputs:      []model.HTTPTxOutput{},
		TransactionFee: "100",
	}

	rawTx, err := model.HttpTx2Tx(httpTx)

	assert.NoError(t, err)
	assert.NotNil(t, rawTx)
	assert.Equal(t, int64(100), rawTx.TransactionFee)
}

// TestHttpTx2TxConversionError tests error handling in conversion
func TestHttpTx2TxConversionError(t *testing.T) {
	// Test with invalid txid
	httpTx := &model.HTTPTransaction{
		Txid:           "invalid_hex",
		TxInputs:       []model.HTTPTxInput{},
		TxOutputs:      []model.HTTPTxOutput{},
		TransactionFee: "100",
	}

	_, err := model.HttpTx2Tx(httpTx)
	assert.Error(t, err)

	// Test with empty txid
	httpTx2 := &model.HTTPTransaction{
		Txid:           "",
		TxInputs:       []model.HTTPTxInput{},
		TxOutputs:      []model.HTTPTxOutput{},
		TransactionFee: "100",
	}

	_, err2 := model.HttpTx2Tx(httpTx2)
	assert.Error(t, err2)
}

// TestRequestDataSerialization tests JSON serialization
func TestRequestDataSerialization(t *testing.T) {
	reqData := model.RequestData{
		Transaction: model.HTTPTransaction{
			Txid:           "0x0000000000000000000000000000000000000000000000000000000000000001",
			TxInputs:       []model.HTTPTxInput{},
			TxOutputs:      []model.HTTPTxOutput{},
			TransactionFee: "100",
		},
		Type: "createlock",
	}

	data, err := json.Marshal(reqData)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test deserialization
	var decoded model.RequestData
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, reqData.Type, decoded.Type)
}

// TestResponseDataFormat tests response data structure
func TestResponseDataFormat(t *testing.T) {
	resp := model.ResponseData{
		Code: 200,
		Msg:  "success",
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, float64(200), decoded["code"])
	assert.Equal(t, "success", decoded["msg"])
}

// BenchmarkHttpTx2Tx benchmarks the transaction conversion
func BenchmarkHttpTx2Tx(b *testing.B) {
	httpTx := &model.HTTPTransaction{
		Txid: "0x0000000000000000000000000000000000000000000000000000000000000001",
		TxInputs: []model.HTTPTxInput{
			{
				Txid:      "0x0000000000000000000000000000000000000000000000000000000000000002",
				Voutput:   "0",
				ScriptSig: "0x00",
				Value:     "1000",
				Address:   "0x0000000000000000000000000000000000000001",
				BciType:   "0",
			},
		},
		TxOutputs: []model.HTTPTxOutput{
			{
				Address:  "0x0000000000000000000000000000000000000002",
				Value:    "900",
				Interest: "0",
				Proof:    "0x00",
				LockTime: "0",
				BciType:  "0",
				Data:     "",
				BurnLock: "0",
				Rate:     "0.0",
			},
		},
		TransactionFee: "100",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.HttpTx2Tx(httpTx)
	}
}

// Example demonstrating how to use the API server
func ExampleApiServer() {
	logger := logrus.New()
	log := logger.WithField("module", "api")

	// Create API server
	config := &apis.Config{Port: 18025}
	server := apis.NewApiServer(config, log)

	// Create mock service
	mockService := &MockPotService{
		CurrentHeight: 100,
	}

	// Register service
	server.RegisterPotService(mockService)

	// Start server (in production, handle the error)
	_ = server.Start()

	// Server is now running...
	// Output: (no output in this example)
}
