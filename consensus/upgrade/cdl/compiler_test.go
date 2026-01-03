package cdl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuntimeStateMachine_Transition 测试状态转换
func TestRuntimeStateMachine_Transition(t *testing.T) {
	sm := &RuntimeStateMachine{
		States:     []string{"IDLE", "RUNNING", "STOPPED"},
		StateIndex: map[string]int{"IDLE": 0, "RUNNING": 1, "STOPPED": 2},
		TransitionTable: map[string]map[string]*RuntimeTransition{
			"IDLE": {
				"start": &RuntimeTransition{
					From:   "IDLE",
					To:     "RUNNING",
					Event:  "start",
					Action: "run()",
				},
			},
			"RUNNING": {
				"stop": &RuntimeTransition{
					From:   "RUNNING",
					To:     "STOPPED",
					Event:  "stop",
					Action: "cleanup()",
				},
			},
		},
		CurrentState: "IDLE",
	}

	// 测试有效转换
	err := sm.Transition("start")
	assert.NoError(t, err)
	assert.Equal(t, "RUNNING", sm.GetCurrentState())

	// 测试另一个有效转换
	err = sm.Transition("stop")
	assert.NoError(t, err)
	assert.Equal(t, "STOPPED", sm.GetCurrentState())

	// 测试无效转换
	err = sm.Transition("start")
	assert.Error(t, err)
}

// TestRuntimeStateMachine_CanTransition 测试检查转换
func TestRuntimeStateMachine_CanTransition(t *testing.T) {
	sm := &RuntimeStateMachine{
		States:     []string{"IDLE", "RUNNING"},
		StateIndex: map[string]int{"IDLE": 0, "RUNNING": 1},
		TransitionTable: map[string]map[string]*RuntimeTransition{
			"IDLE": {
				"start": &RuntimeTransition{From: "IDLE", To: "RUNNING", Event: "start"},
			},
		},
		CurrentState: "IDLE",
	}

	// 可以转换
	assert.True(t, sm.CanTransition("start"))

	// 不能转换
	assert.False(t, sm.CanTransition("stop"))
	assert.False(t, sm.CanTransition("unknown"))
}

// TestRuntimeStateMachine_GetAvailableEvents 测试获取可用事件
func TestRuntimeStateMachine_GetAvailableEvents(t *testing.T) {
	sm := &RuntimeStateMachine{
		States:     []string{"IDLE", "RUNNING"},
		StateIndex: map[string]int{"IDLE": 0, "RUNNING": 1},
		TransitionTable: map[string]map[string]*RuntimeTransition{
			"IDLE": {
				"start": &RuntimeTransition{From: "IDLE", To: "RUNNING", Event: "start"},
				"reset": &RuntimeTransition{From: "IDLE", To: "IDLE", Event: "reset"},
			},
		},
		CurrentState: "IDLE",
	}

	events := sm.GetAvailableEvents()
	assert.Len(t, events, 2)
	assert.Contains(t, events, "start")
	assert.Contains(t, events, "reset")
}

// TestCompiler_GetOptimizationHints 测试获取优化提示
func TestCompiler_GetOptimizationHints(t *testing.T) {
	compiler := NewCompiler(testLog)

	tests := []struct {
		name           string
		consensusType  string
		expectPipeline bool
	}{
		{
			name:           "PoW consensus",
			consensusType:  "pow",
			expectPipeline: false,
		},
		{
			name:           "HotStuff consensus",
			consensusType:  "hotstuff",
			expectPipeline: true,
		},
		{
			name:           "PBFT consensus",
			consensusType:  "pbft",
			expectPipeline: true,
		},
		{
			name:           "Custom consensus",
			consensusType:  "custom",
			expectPipeline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			descriptor := &CDLDescriptor{
				Consensus: ConsensusSpec{
					Type: tt.consensusType,
				},
			}

			hints := compiler.GetOptimizationHints(descriptor)
			assert.NotNil(t, hints)
			assert.Equal(t, tt.expectPipeline, hints.EnablePipelining)
		})
	}
}

// mockCryptoConfig 模拟加密配置
type mockCryptoConfig struct{}

func (m *mockCryptoConfig) Hash(data []byte) []byte {
	// 简单的模拟哈希
	result := make([]byte, 32)
	for i, b := range data {
		if i >= 32 {
			break
		}
		result[i] = b
	}
	return result
}

func (m *mockCryptoConfig) Sign(data []byte) ([]byte, error) {
	return []byte("mock_signature"), nil
}

func (m *mockCryptoConfig) Verify(data, signature []byte) bool {
	return true
}

// mockP2PAdaptor 模拟 P2P 适配器
type mockP2PAdaptor struct{}

func (m *mockP2PAdaptor) Broadcast(data []byte) error {
	return nil
}

func (m *mockP2PAdaptor) Send(nodeID int64, data []byte) error {
	return nil
}

// TestCompiler_CompileComponents 测试编译组件
func TestCompiler_CompileComponents(t *testing.T) {
	compiler := NewCompiler(testLog)

	components := &Components{
		Crypto: CryptoComponent{
			Hash:      "SHA256",
			Signature: "ECDSA",
		},
		Network: NetworkComponent{
			Topology:  "gossip",
			Broadcast: "reliable",
		},
		Storage: StorageComponent{
			Blockchain: "merkle-chain",
			State:      "merkle-patricia",
		},
	}

	runtimeComp, err := compiler.compileComponents(components)

	require.NoError(t, err)
	assert.NotNil(t, runtimeComp)
	assert.Equal(t, "SHA256", runtimeComp.Crypto.HashAlgorithm)
	assert.Equal(t, "gossip", runtimeComp.Network.Topology)
	assert.Equal(t, "merkle-chain", runtimeComp.Storage.BlockchainType)
}

// TestCompiler_CompileStateMachine 测试编译状态机
func TestCompiler_CompileStateMachine(t *testing.T) {
	compiler := NewCompiler(testLog)

	sm := &StateMachine{
		States: []string{"IDLE", "RUNNING", "STOPPED"},
		Transitions: []Transition{
			{From: "IDLE", To: "RUNNING", Event: "start"},
			{From: "RUNNING", To: "STOPPED", Event: "stop"},
		},
	}

	runtimeSM, err := compiler.compileStateMachine(sm)

	require.NoError(t, err)
	assert.NotNil(t, runtimeSM)
	assert.Len(t, runtimeSM.States, 3)
	assert.Equal(t, "IDLE", runtimeSM.CurrentState)
	assert.NotNil(t, runtimeSM.TransitionTable["IDLE"]["start"])
}

// TestCompiler_CompileCryptoComponent_InvalidHash 测试编译无效哈希
func TestCompiler_CompileCryptoComponent_InvalidHash(t *testing.T) {
	compiler := NewCompiler(testLog)

	crypto := &CryptoComponent{
		Hash: "INVALID_HASH",
	}

	_, err := compiler.compileCryptoComponent(crypto)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported hash algorithm")
}
