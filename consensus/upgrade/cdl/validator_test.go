package cdl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidator_Validate 测试完整验证
func TestValidator_Validate(t *testing.T) {
	descriptor := createTestCDL()

	validator := NewValidator(testLog)
	err := validator.Validate(descriptor)

	assert.NoError(t, err)
}

// TestValidator_ValidateBasicInfo 测试基本信息验证
func TestValidator_ValidateBasicInfo(t *testing.T) {
	tests := []struct {
		name        string
		spec        *ConsensusSpec
		expectError bool
	}{
		{
			name: "valid basic info",
			spec: &ConsensusSpec{
				Name:    "ValidName",
				Version: "1.0.0",
				Type:    "test",
			},
			expectError: false,
		},
		{
			name: "empty name",
			spec: &ConsensusSpec{
				Name:    "",
				Version: "1.0",
				Type:    "test",
			},
			expectError: true,
		},
		{
			name: "invalid name format",
			spec: &ConsensusSpec{
				Name:    "Invalid Name!",
				Version: "1.0",
				Type:    "test",
			},
			expectError: true,
		},
		{
			name: "invalid version format",
			spec: &ConsensusSpec{
				Name:    "Test",
				Version: "invalid",
				Type:    "test",
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateBasicInfo(tt.spec)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidateComponents 测试组件验证
func TestValidator_ValidateComponents(t *testing.T) {
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
				Network: NetworkComponent{
					Topology:  "gossip",
					Broadcast: "reliable",
				},
				Storage: StorageComponent{
					Blockchain: "merkle-chain",
					State:      "merkle-patricia",
				},
			},
			expectError: false,
		},
		{
			name: "unsupported hash",
			components: &Components{
				Crypto: CryptoComponent{
					Hash: "MD5",
				},
			},
			expectError: true,
		},
		{
			name: "unsupported topology",
			components: &Components{
				Crypto: CryptoComponent{
					Hash: "SHA256",
				},
				Network: NetworkComponent{
					Topology:  "invalid",
					Broadcast: "reliable",
				},
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateComponents(tt.components)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidateParameters 测试参数验证
func TestValidator_ValidateParameters(t *testing.T) {
	tests := []struct {
		name        string
		params      *Parameters
		expectError bool
	}{
		{
			name: "valid parameters",
			params: &Parameters{
				BlockTime:    "5s",
				MaxBlockSize: 1048576,
			},
			expectError: false,
		},
		{
			name: "invalid block time",
			params: &Parameters{
				BlockTime:    "invalid",
				MaxBlockSize: 1000,
			},
			expectError: true,
		},
		{
			name: "negative block size",
			params: &Parameters{
				BlockTime:    "5s",
				MaxBlockSize: -1,
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateParameters(tt.params)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidatePhases 测试阶段验证
func TestValidator_ValidatePhases(t *testing.T) {
	tests := []struct {
		name        string
		phases      []Phase
		expectError bool
	}{
		{
			name: "valid phases",
			phases: []Phase{
				{
					Name:  "init",
					Entry: "start",
					Exit:  "initialized",
					Actions: []Action{
						{Type: "function", Name: "init"},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "empty phases",
			phases:      []Phase{},
			expectError: true,
		},
		{
			name: "duplicate phase names",
			phases: []Phase{
				{Name: "init", Entry: "start", Exit: "end"},
				{Name: "init", Entry: "start", Exit: "end"},
			},
			expectError: true,
		},
		{
			name: "invalid action type",
			phases: []Phase{
				{
					Name:  "init",
					Entry: "start",
					Exit:  "end",
					Actions: []Action{
						{Type: "invalid_type"},
					},
				},
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePhases(tt.phases)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidateStateMachine 测试状态机验证
func TestValidator_ValidateStateMachine(t *testing.T) {
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
			name: "duplicate states",
			stateMachine: &StateMachine{
				States: []string{"IDLE", "IDLE"},
			},
			expectError: true,
		},
		{
			name: "unknown source state",
			stateMachine: &StateMachine{
				States: []string{"IDLE", "RUNNING"},
				Transitions: []Transition{
					{From: "UNKNOWN", To: "RUNNING", Event: "start"},
				},
			},
			expectError: true,
		},
		{
			name: "missing event",
			stateMachine: &StateMachine{
				States: []string{"IDLE", "RUNNING"},
				Transitions: []Transition{
					{From: "IDLE", To: "RUNNING", Event: ""},
				},
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateStateMachine(tt.stateMachine)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidateSafetyProperties 测试安全属性验证
func TestValidator_ValidateSafetyProperties(t *testing.T) {
	tests := []struct {
		name        string
		properties  []SafetyProperty
		expectError bool
	}{
		{
			name: "valid properties",
			properties: []SafetyProperty{
				{Name: "agreement", Formula: "test"},
				{Name: "validity", Formula: "test"},
			},
			expectError: false,
		},
		{
			name:        "empty properties",
			properties:  []SafetyProperty{},
			expectError: false, // 允许为空
		},
		{
			name: "duplicate property names",
			properties: []SafetyProperty{
				{Name: "agreement", Formula: "test"},
				{Name: "agreement", Formula: "test"},
			},
			expectError: true,
		},
		{
			name: "empty formula",
			properties: []SafetyProperty{
				{Name: "agreement", Formula: ""},
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSafetyProperties(tt.properties)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidatePerformanceRequirements 测试性能要求验证
func TestValidator_ValidatePerformanceRequirements(t *testing.T) {
	tests := []struct {
		name        string
		reqs        *PerformanceRequirements
		expectError bool
	}{
		{
			name: "valid requirements",
			reqs: &PerformanceRequirements{
				MinThroughput:  100,
				MaxLatency:     10,
				FaultTolerance: 0.33,
			},
			expectError: false,
		},
		{
			name: "negative throughput",
			reqs: &PerformanceRequirements{
				MinThroughput:  -1,
				MaxLatency:     10,
				FaultTolerance: 0.33,
			},
			expectError: true,
		},
		{
			name: "invalid fault tolerance",
			reqs: &PerformanceRequirements{
				MinThroughput:  100,
				MaxLatency:     10,
				FaultTolerance: 1.5,
			},
			expectError: true,
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validatePerformanceRequirements(tt.reqs)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ValidateSemantics 测试语义验证
func TestValidator_ValidateSemantics(t *testing.T) {
	descriptor := createTestCDL()

	validator := NewValidator(testLog)
	err := validator.ValidateSemantics(descriptor)

	assert.NoError(t, err)
}

// TestValidator_ValidateStateMachineReachability 测试状态可达性
func TestValidator_ValidateStateMachineReachability(t *testing.T) {
	tests := []struct {
		name         string
		stateMachine *StateMachine
		expectError  bool
	}{
		{
			name: "all states reachable",
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
			name: "unreachable state (warning only)",
			stateMachine: &StateMachine{
				States: []string{"IDLE", "RUNNING", "ISOLATED"},
				Transitions: []Transition{
					{From: "IDLE", To: "RUNNING", Event: "start"},
				},
			},
			expectError: false, // 只警告，不报错
		},
	}

	validator := NewValidator(testLog)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateStateMachineReachability(tt.stateMachine)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidator_ComplexCDL 测试复杂 CDL 验证
func TestValidator_ComplexCDL(t *testing.T) {
	yamlContent := `
consensus:
  name: "ComplexConsensus"
  version: "2.0.0"
  type: "custom"
  components:
    crypto:
      hash: "SHA3"
      signature: "BLS"
      vrf: "VRF-ED25519"
      vdf: "Wesolowski"
      threshold_sig: "BLS-TSig"
    network:
      topology: "mesh"
      broadcast: "best-effort"
    storage:
      blockchain: "dag"
      state: "simple-map"
  parameters:
    block_time: "2s"
    max_block_size: 2097152
    consensus_timeout: "30s"
    committee_size: 100
  phases:
    - name: "bootstrap"
      entry: "genesis"
      actions:
        - type: "function"
          name: "initializeGenesis"
      exit: "ready"
    - name: "consensus"
      entry: "ready"
      actions:
        - type: "event"
          name: "processProposal"
        - type: "event"
          name: "collectVotes"
      exit: "decided"
  state_machine:
    states: ["INIT", "READY", "PROPOSING", "VOTING", "DECIDED", "FINALIZED"]
    transitions:
      - from: "INIT"
        to: "READY"
        event: "bootstrap_complete"
        action: "enterReady()"
      - from: "READY"
        to: "PROPOSING"
        event: "new_round"
        action: "createProposal()"
      - from: "PROPOSING"
        to: "VOTING"
        event: "proposal_ready"
        action: "broadcastProposal()"
      - from: "VOTING"
        to: "DECIDED"
        event: "quorum_reached"
        condition: "votes >= threshold"
        action: "decide()"
      - from: "DECIDED"
        to: "FINALIZED"
        event: "commit"
        action: "commitBlock()"
      - from: "FINALIZED"
        to: "READY"
        event: "next_round"
        action: "reset()"
  safety_properties:
    - name: "agreement"
      formula: "forall nodes: decided(n1, v) AND decided(n2, v') => v = v'"
    - name: "validity"
      formula: "decided(v) => proposed(v)"
    - name: "termination"
      formula: "eventually decided(v)"
  performance_requirements:
    min_throughput: 1000
    max_latency: 5
    fault_tolerance: 0.33
`

	parser := NewParser(testLog)
	descriptor, err := parser.Parse(yamlContent)
	require.NoError(t, err)

	validator := NewValidator(testLog)
	err = validator.Validate(descriptor)
	assert.NoError(t, err)

	// 语义验证
	err = validator.ValidateSemantics(descriptor)
	assert.NoError(t, err)
}
