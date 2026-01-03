package cdl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCDLDescriptor_Validate 测试 CDL 描述符验证
func TestCDLDescriptor_Validate(t *testing.T) {
	tests := []struct {
		name        string
		descriptor  *CDLDescriptor
		expectError bool
	}{
		{
			name: "valid descriptor",
			descriptor: &CDLDescriptor{
				Consensus: ConsensusSpec{
					Name:    "TestConsensus",
					Version: "1.0.0",
					Type:    "test",
					Components: Components{
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
					},
					Parameters: Parameters{
						BlockTime:    "10s",
						MaxBlockSize: 1048576,
					},
					StateMachine: StateMachine{
						States: []string{"IDLE", "RUNNING"},
						Transitions: []Transition{
							{From: "IDLE", To: "RUNNING", Event: "start"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			descriptor: &CDLDescriptor{
				Consensus: ConsensusSpec{
					Version: "1.0.0",
					Type:    "test",
				},
			},
			expectError: true,
		},
		{
			name: "invalid hash algorithm",
			descriptor: &CDLDescriptor{
				Consensus: ConsensusSpec{
					Name:    "Test",
					Version: "1.0",
					Type:    "test",
					Components: Components{
						Crypto: CryptoComponent{
							Hash: "INVALID",
						},
					},
					Parameters: Parameters{
						BlockTime:    "10s",
						MaxBlockSize: 1000,
					},
					StateMachine: StateMachine{
						States: []string{"IDLE"},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.descriptor.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParameters_GetBlockTime 测试获取区块时间
func TestParameters_GetBlockTime(t *testing.T) {
	params := &Parameters{
		BlockTime:    "5s",
		MaxBlockSize: 1000,
	}

	duration := params.GetBlockTime()
	assert.Equal(t, 5*time.Second, duration)
}

// TestComponents_Validate 测试组件验证
func TestComponents_Validate(t *testing.T) {
	tests := []struct {
		name        string
		components  *Components
		expectError bool
	}{
		{
			name: "valid components",
			components: &Components{
				Crypto: CryptoComponent{
					Hash:      "SHA256",
					Signature: "ECDSA",
				},
			},
			expectError: false,
		},
		{
			name: "invalid hash",
			components: &Components{
				Crypto: CryptoComponent{
					Hash: "MD5",
				},
			},
			expectError: true,
		},
		{
			name: "invalid signature",
			components: &Components{
				Crypto: CryptoComponent{
					Hash:      "SHA256",
					Signature: "INVALID",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.components.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStateMachine_Validate 测试状态机验证
func TestStateMachine_Validate(t *testing.T) {
	tests := []struct {
		name         string
		stateMachine *StateMachine
		expectError  bool
	}{
		{
			name: "valid state machine",
			stateMachine: &StateMachine{
				States: []string{"IDLE", "RUNNING", "STOPPED"},
				Transitions: []Transition{
					{From: "IDLE", To: "RUNNING", Event: "start"},
					{From: "RUNNING", To: "STOPPED", Event: "stop"},
				},
			},
			expectError: false,
		},
		{
			name: "empty states",
			stateMachine: &StateMachine{
				States: []string{},
			},
			expectError: true,
		},
		{
			name: "invalid transition",
			stateMachine: &StateMachine{
				States: []string{"IDLE", "RUNNING"},
				Transitions: []Transition{
					{From: "IDLE", To: "UNKNOWN", Event: "start"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.stateMachine.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCDLDescriptor_Hash 测试 CDL 哈希
func TestCDLDescriptor_Hash(t *testing.T) {
	descriptor := &CDLDescriptor{
		Consensus: ConsensusSpec{
			Name:    "Test",
			Version: "1.0.0",
			Type:    "test",
		},
	}

	hash := descriptor.Hash()
	assert.NotNil(t, hash)
	assert.NotEmpty(t, hash)

	// 相同的描述符应该产生相同的哈希
	hash2 := descriptor.Hash()
	assert.Equal(t, hash, hash2)
}

// TestCDLDescriptor_Serialize 测试序列化
func TestCDLDescriptor_Serialize(t *testing.T) {
	descriptor := &CDLDescriptor{
		Consensus: ConsensusSpec{
			Name:    "Test",
			Version: "1.0.0",
			Type:    "test",
		},
	}

	serialized := descriptor.Serialize()
	assert.NotEmpty(t, serialized)
	assert.Contains(t, serialized, "Test")
	assert.Contains(t, serialized, "1.0.0")
}

// createTestCDL 创建测试用 CDL 描述符
func createTestCDL() *CDLDescriptor {
	return &CDLDescriptor{
		Consensus: ConsensusSpec{
			Name:    "TestConsensus",
			Version: "1.0.0",
			Type:    "test",
			Components: Components{
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
			},
			Parameters: Parameters{
				BlockTime:    "5s",
				MaxBlockSize: 1048576,
			},
			Phases: []Phase{
				{
					Name:  "init",
					Entry: "start",
					Exit:  "initialized",
					Actions: []Action{
						{Type: "function", Name: "initialize"},
					},
				},
			},
			StateMachine: StateMachine{
				States: []string{"IDLE", "RUNNING", "STOPPED"},
				Transitions: []Transition{
					{From: "IDLE", To: "RUNNING", Event: "start", Action: "run()"},
					{From: "RUNNING", To: "STOPPED", Event: "stop", Action: "cleanup()"},
				},
			},
			SafetyProperties: []SafetyProperty{
				{Name: "agreement", Formula: "test"},
			},
			PerformanceRequirements: PerformanceRequirements{
				MinThroughput:  100,
				MaxLatency:     10,
				FaultTolerance: 0.33,
			},
		},
	}
}
