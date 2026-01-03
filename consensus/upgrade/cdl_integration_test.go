package upgrade

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/upgrade/cdl"
)

// TestCDLIntegration 测试 CDL 集成
func TestCDLIntegration(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	// 创建一个示例 CDL 描述符
	cdlDesc := &CDLDescriptor{
		Name:    "custom-raft",
		Version: "1.0.0",
		Type:    "custom",
		Components: map[string]interface{}{
			"crypto": map[string]interface{}{
				"hash":      "SHA256",
				"signature": "ECDSA",
			},
			"network": map[string]interface{}{
				"topology":  "gossip",
				"broadcast": "reliable",
			},
			"storage": map[string]interface{}{
				"blockchain": "merkle-chain",
				"state":      "simple-map",
			},
		},
		Parameters: map[string]interface{}{
			"block_time":     "2s",
			"max_block_size": 2097152,
		},
		Phases: []interface{}{
			map[string]interface{}{
				"name":  "propose",
				"entry": "start",
				"exit":  "proposed",
				"actions": []interface{}{
					map[string]interface{}{
						"type": "function",
						"name": "createProposal",
					},
				},
			},
			map[string]interface{}{
				"name":  "vote",
				"entry": "proposed",
				"exit":  "committed",
				"actions": []interface{}{
					map[string]interface{}{
						"type": "function",
						"name": "collectVotes",
					},
				},
			},
		},
		StateMachine: map[string]interface{}{
			"states": []interface{}{"idle", "proposing", "voting", "committed"},
			"transitions": []interface{}{
				map[string]interface{}{
					"from":  "idle",
					"to":    "proposing",
					"event": "start_proposal",
				},
				map[string]interface{}{
					"from":  "proposing",
					"to":    "voting",
					"event": "proposal_created",
				},
				map[string]interface{}{
					"from":  "voting",
					"to":    "committed",
					"event": "votes_collected",
				},
				map[string]interface{}{
					"from":  "committed",
					"to":    "idle",
					"event": "reset",
				},
			},
		},
		SafetyProperties: []interface{}{
			map[string]interface{}{
				"name":    "agreement",
				"formula": "always(committed(x) -> committed(x))",
			},
		},
		PerformanceRequirements: map[string]interface{}{
			"min_throughput":  100,
			"max_latency":     5,
			"fault_tolerance": 0.33,
		},
	}

	// 创建共识工厂
	factory := NewConsensusFactory(1, nil, nil, log)

	// 测试 CDL 验证
	t.Run("ValidateCDL", func(t *testing.T) {
		proposal := &UpgradeProposal{
			TargetConsensus: "custom",
			CDLDescriptor:   cdlDesc,
		}

		err := factory.ValidateProposal(proposal)
		assert.NoError(t, err, "CDL validation should pass")
	})

	// 测试构建配置
	t.Run("BuildConfigFromCDL", func(t *testing.T) {
		baseConfig := &config.ConsensusConfig{
			ConsensusID: 100,
			Type:        "custom",
		}

		cfg, err := factory.buildConfigFromCDL(cdlDesc, baseConfig)
		require.NoError(t, err, "Building config should succeed")
		assert.Equal(t, "custom", cfg.Type)
		assert.Equal(t, 2, cfg.BlockTime)
		assert.Equal(t, 2097152, cfg.MaxBlockSize)
	})

	// 测试 CDL 描述符转换
	t.Run("ConvertCDLDescriptor", func(t *testing.T) {
		converted := factory.convertToCDLDescriptor(cdlDesc)
		assert.Equal(t, "custom-raft", converted.Consensus.Name)
		assert.Equal(t, "1.0.0", converted.Consensus.Version)
		assert.Equal(t, "custom", converted.Consensus.Type)
		assert.Equal(t, "SHA256", converted.Consensus.Components.Crypto.Hash)
		assert.Equal(t, "gossip", converted.Consensus.Components.Network.Topology)
		assert.Equal(t, "2s", converted.Consensus.Parameters.BlockTime)
		assert.Equal(t, 2097152, converted.Consensus.Parameters.MaxBlockSize)
		assert.Equal(t, 2, len(converted.Consensus.Phases))
		assert.Equal(t, 4, len(converted.Consensus.StateMachine.States))
		assert.Equal(t, 4, len(converted.Consensus.StateMachine.Transitions))
	})

	// 测试创建共识（模拟，因为需要 P2P）
	t.Run("CreateConsensusFromCDL", func(t *testing.T) {
		// 这个测试需要真实的 P2P 适配器，这里只验证错误处理
		baseConfig := &config.ConsensusConfig{
			ConsensusID: 100,
			Type:        "custom",
		}

		_, err := factory.CreateConsensusFromCDL(cdlDesc, baseConfig)
		// 预期会因为缺少 P2P 适配器而失败，但不应该是"未实现"错误
		if err != nil {
			assert.NotContains(t, err.Error(), "not yet implemented",
				"Should not return 'not yet implemented' error")
		}
	})
}

// TestCDLParser 测试 CDL 解析器
func TestCDLParser(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	parser := cdl.NewParser(log)

	yamlContent := `
consensus:
  name: test-consensus
  version: 1.0.0
  type: custom
  components:
    crypto:
      hash: SHA256
      signature: ECDSA
    network:
      topology: gossip
      broadcast: reliable
    storage:
      blockchain: merkle-chain
      state: simple-map
  parameters:
    block_time: 1s
    max_block_size: 1048576
  phases:
    - name: propose
      entry: start
      exit: proposed
      actions:
        - type: function
          name: createProposal
  state_machine:
    states:
      - idle
      - active
    transitions:
      - from: idle
        to: active
        event: start
  safety_properties:
    - name: agreement
      formula: always(agreed)
  performance_requirements:
    min_throughput: 100
    max_latency: 10
    fault_tolerance: 0.33
`

	t.Run("ParseYAML", func(t *testing.T) {
		descriptor, err := parser.Parse(yamlContent)
		require.NoError(t, err)
		assert.Equal(t, "test-consensus", descriptor.Consensus.Name)
		assert.Equal(t, "1.0.0", descriptor.Consensus.Version)
		assert.Equal(t, "custom", descriptor.Consensus.Type)
	})

	t.Run("ParseAndValidate", func(t *testing.T) {
		descriptor, err := parser.ParseAndValidate(yamlContent)
		require.NoError(t, err)
		assert.NotNil(t, descriptor)
	})

	t.Run("Serialize", func(t *testing.T) {
		descriptor, err := parser.Parse(yamlContent)
		require.NoError(t, err)

		serialized, err := parser.Serialize(descriptor)
		require.NoError(t, err)
		assert.Contains(t, serialized, "test-consensus")
	})
}

// TestCDLValidator 测试 CDL 验证器
func TestCDLValidator(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	validator := cdl.NewValidator(log)

	validDescriptor := &cdl.CDLDescriptor{
		Consensus: cdl.ConsensusSpec{
			Name:    "valid-consensus",
			Version: "1.0.0",
			Type:    "custom",
			Components: cdl.Components{
				Crypto: cdl.CryptoComponent{
					Hash:      "SHA256",
					Signature: "ECDSA",
				},
				Network: cdl.NetworkComponent{
					Topology:  "gossip",
					Broadcast: "reliable",
				},
				Storage: cdl.StorageComponent{
					Blockchain: "merkle-chain",
					State:      "simple-map",
				},
			},
			Parameters: cdl.Parameters{
				BlockTime:    "1s",
				MaxBlockSize: 1048576,
			},
			Phases: []cdl.Phase{
				{
					Name:  "propose",
					Entry: "start",
					Exit:  "end",
				},
			},
			StateMachine: cdl.StateMachine{
				States: []string{"idle", "active"},
				Transitions: []cdl.Transition{
					{From: "idle", To: "active", Event: "start"},
				},
			},
			SafetyProperties: []cdl.SafetyProperty{
				{Name: "safety", Formula: "always(safe)"},
			},
			PerformanceRequirements: cdl.PerformanceRequirements{
				MinThroughput:  100,
				MaxLatency:     10,
				FaultTolerance: 0.33,
			},
		},
	}

	t.Run("ValidDescriptor", func(t *testing.T) {
		err := validator.Validate(validDescriptor)
		assert.NoError(t, err)
	})

	t.Run("MissingName", func(t *testing.T) {
		invalidDesc := *validDescriptor
		invalidDesc.Consensus.Name = ""
		err := validator.Validate(&invalidDesc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("InvalidVersion", func(t *testing.T) {
		invalidDesc := *validDescriptor
		invalidDesc.Consensus.Version = "invalid"
		err := validator.Validate(&invalidDesc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version")
	})

	t.Run("InvalidHash", func(t *testing.T) {
		invalidDesc := *validDescriptor
		invalidDesc.Consensus.Components.Crypto.Hash = "INVALID"
		err := validator.Validate(&invalidDesc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hash")
	})

	t.Run("SemanticValidation", func(t *testing.T) {
		err := validator.ValidateSemantics(validDescriptor)
		assert.NoError(t, err)
	})
}

// TestCDLCompiler 测试 CDL 编译器
func TestCDLCompiler(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	compiler := cdl.NewCompiler(log)

	descriptor := &cdl.CDLDescriptor{
		Consensus: cdl.ConsensusSpec{
			Name:    "test-consensus",
			Version: "1.0.0",
			Type:    "custom",
			Components: cdl.Components{
				Crypto: cdl.CryptoComponent{
					Hash:      "SHA256",
					Signature: "ECDSA",
				},
				Network: cdl.NetworkComponent{
					Topology:  "gossip",
					Broadcast: "reliable",
				},
				Storage: cdl.StorageComponent{
					Blockchain: "merkle-chain",
					State:      "simple-map",
				},
			},
			Parameters: cdl.Parameters{
				BlockTime:    "1s",
				MaxBlockSize: 1048576,
			},
			Phases: []cdl.Phase{
				{Name: "propose", Entry: "start", Exit: "end"},
			},
			StateMachine: cdl.StateMachine{
				States:      []string{"idle", "active"},
				Transitions: []cdl.Transition{{From: "idle", To: "active", Event: "start"}},
			},
			PerformanceRequirements: cdl.PerformanceRequirements{
				MinThroughput:  100,
				MaxLatency:     10,
				FaultTolerance: 0.33,
			},
		},
	}

	cfg := &config.ConsensusConfig{
		ConsensusID: 1,
		Type:        "custom",
	}

	t.Run("CompileCDL", func(t *testing.T) {
		runtime, err := compiler.Compile(descriptor, 1, cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, runtime)
		assert.Equal(t, int64(1), runtime.GetConsensusID())
		assert.Equal(t, "custom", runtime.GetConsensusType())
	})

	t.Run("CompileComponents", func(t *testing.T) {
		runtime, err := compiler.Compile(descriptor, 1, cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, runtime.GetComponents())
		assert.Equal(t, "SHA256", runtime.GetComponents().Crypto.HashAlgorithm)
		assert.Equal(t, "gossip", runtime.GetComponents().Network.Topology)
	})

	t.Run("CompileStateMachine", func(t *testing.T) {
		runtime, err := compiler.Compile(descriptor, 1, cfg, nil)
		require.NoError(t, err)
		assert.NotNil(t, runtime.GetStateMachine())
		assert.Equal(t, "idle", runtime.GetStateMachine().GetCurrentState())

		// 测试状态转换
		err = runtime.GetStateMachine().Transition("start")
		assert.NoError(t, err)
		assert.Equal(t, "active", runtime.GetStateMachine().GetCurrentState())
	})
}

// TestInvalidCDL 测试无效的 CDL
func TestInvalidCDL(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	factory := NewConsensusFactory(1, nil, nil, log)

	t.Run("NilCDL", func(t *testing.T) {
		proposal := &UpgradeProposal{
			TargetConsensus: "custom",
			CDLDescriptor:   nil,
		}
		err := factory.ValidateProposal(proposal)
		assert.Error(t, err)
	})

	t.Run("EmptyName", func(t *testing.T) {
		invalidCDL := &CDLDescriptor{
			Name:    "",
			Version: "1.0.0",
			Type:    "custom",
		}
		proposal := &UpgradeProposal{
			TargetConsensus: "custom",
			CDLDescriptor:   invalidCDL,
		}
		err := factory.ValidateProposal(proposal)
		assert.Error(t, err)
	})

	t.Run("EmptyVersion", func(t *testing.T) {
		invalidCDL := &CDLDescriptor{
			Name:    "test",
			Version: "",
			Type:    "custom",
		}
		proposal := &UpgradeProposal{
			TargetConsensus: "custom",
			CDLDescriptor:   invalidCDL,
		}
		err := factory.ValidateProposal(proposal)
		assert.Error(t, err)
	})
}
