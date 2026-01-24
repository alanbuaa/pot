package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewConfig_Success 测试正常读取配置文件
func TestNewConfig_Success(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err, "读取配置文件应该成功")
	require.NotNil(t, cfg, "配置对象不应为空")

	// 验证节点配置
	assert.NotNil(t, cfg.Node, "节点配置不应为空")
	assert.Equal(t, int64(0), cfg.Node.ID, "节点ID应该为0")
	assert.NotEmpty(t, cfg.Node.P2PAddress, "节点地址不应为空")
	assert.NotEmpty(t, cfg.Node.RpcAddress, "RPC地址不应为空")

	// 验证共识配置
	assert.NotNil(t, cfg.Consensus, "共识配置不应为空")
	assert.NotEmpty(t, cfg.Consensus.Type, "共识类型不应为空")
	assert.NotEmpty(t, cfg.Consensus.FaultModel, "容错类型不应为空")

	// 验证密钥配置
	assert.NotNil(t, cfg.Crypto, "密钥配置不应为空")
	assert.NotNil(t, cfg.Crypto.PublicKey, "公钥不应为空")
	assert.NotNil(t, cfg.Crypto.PrivateKey, "私钥不应为空")

	// 验证Topic设置
	assert.Equal(t, cfg.Network.Topic, cfg.Consensus.Topic, "共识Topic应该与网络Topic一致")

	// 验证F值计算
	if cfg.Consensus.FaultModel == "1/3" {
		expectedF := (cfg.Total - 1) / 3
		assert.Equal(t, expectedF, cfg.Consensus.Fault, "F值计算应该正确")
	}

	t.Logf("成功读取配置: 节点地址=%s, 共识类型=%s", cfg.Node.P2PAddress, cfg.Consensus.Type)
}

// TestNewConfig_FileNotExist 测试配置文件不存在的情况
func TestNewConfig_FileNotExist(t *testing.T) {
	_, err := NewConfig("config/non_existent.yaml")
	assert.Error(t, err, "读取不存在的配置文件应该返回错误")
	t.Logf("预期错误: %v", err)
}

// TestNewConfig_InvalidYAML 测试无效的YAML格式
func TestNewConfig_InvalidYAML(t *testing.T) {
	// 创建临时的无效YAML文件
	tmpFile := filepath.Join(t.TempDir(), "invalid.yaml")
	invalidYAML := `
node:
  id: 0
  address: "invalid yaml content
    missing quote
`
	err := os.WriteFile(tmpFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	_, err = NewConfig(tmpFile)
	assert.Error(t, err, "解析无效YAML应该返回错误")
	t.Logf("预期错误: %v", err)
}

// TestNewConfig_MissingNodeConfig 测试缺少节点配置
func TestNewConfig_MissingNodeConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "missing_node.yaml")
	yamlContent := `
total: 1
consensus:
  type: "pot"
  faultTolerance: "1/3"
  consensus_id: 10001
network:
  topic: "test-topic"
`
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	_, err = NewConfig(tmpFile)
	assert.Error(t, err, "缺少节点配置应该返回错误")
	assert.Contains(t, err.Error(), "node configuration is missing", "错误信息应该包含节点配置缺失")
	t.Logf("预期错误: %v", err)
}

// TestNewConfig_MissingConsensusConfig 测试缺少共识配置
func TestNewConfig_MissingConsensusConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "missing_consensus.yaml")
	yamlContent := `
node:
  id: 0
  address: "127.0.0.1:8081"
  private_key_path: "data/keys/node-0.key"
  public_key_path: "data/keys/node-pub.key"
total: 1
network:
  topic: "test-topic"
`
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	_, err = NewConfig(tmpFile)
	assert.Error(t, err, "缺少共识配置应该返回错误")
	assert.Contains(t, err.Error(), "consensus configuration is missing", "错误信息应该包含共识配置缺失")
	t.Logf("预期错误: %v", err)
}

// TestNewConfig_FaultToleranceCalculation 测试容错值计算
func TestNewConfig_FaultToleranceCalculation(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		faultType   string
		expectedF   int
		shouldError bool
	}{
		{
			name:        "1/3容错-4节点",
			total:       4,
			faultType:   "1/3",
			expectedF:   1,
			shouldError: false,
		},
		{
			name:        "1/3容错-7节点",
			total:       7,
			faultType:   "1/3",
			expectedF:   2,
			shouldError: false,
		},
		{
			name:        "1/2容错-4节点",
			total:       4,
			faultType:   "1/2",
			expectedF:   1,
			shouldError: false,
		},
		{
			name:        "1/2容错-5节点",
			total:       5,
			faultType:   "1/2",
			expectedF:   2,
			shouldError: false,
		},
		{
			name:        "未知容错类型",
			total:       4,
			faultType:   "2/3",
			expectedF:   0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test_config.yaml")
			yamlContent := `
node:
  id: 0
  address: "127.0.0.1:8081"
  rpc_address: "127.0.0.1:9081"
  private_key_path: "data/keys/node-0.key"
  public_key_path: "data/keys/node-pub.key"
total: ` + string(rune(tt.total+'0')) + `
network:
  topic: "test-topic"
consensus:
  type: "pot"
  faultTolerance: "` + tt.faultType + `"
  consensus_id: 10001
`
			err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
			require.NoError(t, err)

			cfg, err := NewConfig(tmpFile)
			if tt.shouldError {
				assert.Error(t, err, "应该返回错误")
				assert.Contains(t, err.Error(), "unknown consensus faulty type", "错误信息应该包含未知容错类型")
			} else {
				require.NoError(t, err, "不应该返回错误")
				assert.Equal(t, tt.expectedF, cfg.Consensus.Fault, "F值计算错误")
				t.Logf("Total=%d, FaultType=%s, F=%d", tt.total, tt.faultType, cfg.Consensus.Fault)
			}
		})
	}
}

// TestLoadCryptoKeys_Success 测试成功加载密钥
func TestLoadCryptoKeys_Success(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	keys, err := cfg.LoadCryptoKeys()
	require.NoError(t, err, "加载密钥应该成功")
	assert.NotNil(t, keys, "密钥对象不应为空")
	assert.NotNil(t, keys.PublicKey, "公钥不应为空")
	assert.NotNil(t, keys.PrivateKey, "私钥不应为空")
	assert.Equal(t, cfg.Crypto, keys, "返回的密钥应该与配置中的一致")
	t.Logf("成功加载密钥: PublicKey存在=%v, PrivateKey存在=%v",
		keys.PublicKey != nil, keys.PrivateKey != nil)
}

// TestGetNodeFromSet 测试获取节点信息
func TestGetNodeFromSet(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	t.Run("获取存在的节点", func(t *testing.T) {
		nodeInfo, _ := cfg.GetNodeFromSet(0)
		assert.NotNil(t, nodeInfo, "节点信息不应为空")
		assert.Equal(t, int64(0), nodeInfo.ID, "节点ID应该为0")
		assert.NotEmpty(t, nodeInfo.P2PAddress, "节点地址不应为空")
		t.Logf("节点信息: ID=%d, Address=%s", nodeInfo.ID, nodeInfo.P2PAddress)
	})

	t.Run("获取不存在的节点应panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("预期panic: %v", r)
				assert.Contains(t, r, "does not exist", "panic信息应该包含节点不存在")
			} else {
				t.Error("应该触发panic")
			}
		}()
		_, _ = cfg.GetNodeFromSet(999)
	})
}

// TestGetAddress 测试地址生成
func TestGetAddress(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	tests := []struct {
		name     string
		id       int
		initPort string
		ip       string
		expected string
	}{
		{
			name:     "节点0",
			id:       0,
			initPort: "8000",
			ip:       "127.0.0.1",
			expected: "127.0.0.1:8000",
		},
		{
			name:     "节点1",
			id:       1,
			initPort: "8000",
			ip:       "127.0.0.1",
			expected: "127.0.0.1:8001",
		},
		{
			name:     "节点5",
			id:       5,
			initPort: "9000",
			ip:       "192.168.1.1",
			expected: "192.168.1.1:9005",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := cfg.GetAddress(tt.id, tt.initPort, tt.ip)
			assert.Equal(t, tt.expected, address, "生成的地址不正确")
			t.Logf("节点%d地址: %s", tt.id, address)
		})
	}

	t.Run("无效端口应panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("预期panic: %v", r)
				assert.Contains(t, r, "GetAddress error", "panic信息应该包含GetAddress错误")
			} else {
				t.Error("应该触发panic")
			}
		}()
		cfg.GetAddress(0, "invalid_port", "127.0.0.1")
	})
}

// TestConsensusTypeConfigs 测试不同共识类型的配置
func TestConsensusTypeConfigs(t *testing.T) {
	t.Run("POT配置", func(t *testing.T) {
		cfg, err := NewConfig("config/config.yaml")
		require.NoError(t, err)

		if cfg.Consensus.Type == "pot" && cfg.Consensus.PoT != nil {
			assert.Greater(t, cfg.Consensus.PoT.Snum, int64(0), "Snum应该大于0")
			assert.Greater(t, cfg.Consensus.PoT.Vdf0Iteration, 0, "VDF0迭代次数应该大于0")
			assert.Greater(t, cfg.Consensus.PoT.Vdf1Iteration, 0, "VDF1迭代次数应该大于0")
			assert.Greater(t, cfg.Consensus.PoT.Batchsize, 0, "批处理大小应该大于0")
			assert.NotEmpty(t, cfg.Consensus.PoT.VdfType, "VDF类型不应为空")
			t.Logf("POT配置: Snum=%d, VdfType=%s, BatchSize=%d",
				cfg.Consensus.PoT.Snum, cfg.Consensus.PoT.VdfType, cfg.Consensus.PoT.Batchsize)
		}
	})
}

// TestPoWConfig 测试POW配置解析
func TestPoWConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "pow_config.yaml")
	yamlContent := `
node:
  id: 0
  address: "127.0.0.1:8081"
  rpc_address: "127.0.0.1:9081"
  private_key_path: "data/keys/node-0.key"
  public_key_path: "data/keys/node-pub.key"
total: 1
network:
  topic: "test-topic"
consensus:
  type: "pow"
  faultTolerance: "1/2"
  consensus_id: 1008
  pow:
    init_difficulty: 0x0010000000000000000000000000000000000000000000000000000000000000
    hash: "sha256"
`
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := NewConfig(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "pow", cfg.Consensus.Type)
	if cfg.Consensus.Pow != nil {
		assert.NotNil(t, cfg.Consensus.Pow.InitDifficulty, "初始难度不应为空")
		assert.NotEmpty(t, cfg.Consensus.Pow.Hash, "哈希算法不应为空")
		t.Logf("POW配置: Hash=%s, Difficulty存在=%v",
			cfg.Consensus.Pow.Hash, cfg.Consensus.Pow.InitDifficulty != nil)
	}
}

// TestHotStuffConfig 测试HotStuff配置解析
func TestHotStuffConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "hotstuff_config.yaml")
	yamlContent := `
node:
  id: 0
  address: "127.0.0.1:8081"
  rpc_address: "127.0.0.1:9081"
  private_key_path: "data/keys/node-0.key"
  public_key_path: "data/keys/node-pub.key"
total: 4
network:
  topic: "test-topic"
consensus:
  type: "hotstuff"
  faultTolerance: "1/3"
  consensus_id: 1000
  hotstuff:
    type: "event-driven"
    batch_size: 100
    batch_timeout: 2
    timeout: 5
`
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := NewConfig(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "hotstuff", cfg.Consensus.Type)
	if cfg.Consensus.HotStuff != nil {
		assert.NotEmpty(t, cfg.Consensus.HotStuff.Type, "HotStuff类型不应为空")
		assert.Greater(t, cfg.Consensus.HotStuff.BatchSize, 0, "批处理大小应该大于0")
		assert.Greater(t, cfg.Consensus.HotStuff.Timeout, 0, "超时时间应该大于0")
		t.Logf("HotStuff配置: Type=%s, BatchSize=%d, Timeout=%d",
			cfg.Consensus.HotStuff.Type, cfg.Consensus.HotStuff.BatchSize, cfg.Consensus.HotStuff.Timeout)
	}
}

// TestWhirlyConfig 测试Whirly配置解析
func TestWhirlyConfig(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "whirly_config.yaml")
	yamlContent := `
node:
  id: 0
  address: "127.0.0.1:8081"
  rpc_address: "127.0.0.1:9081"
  private_key_path: "data/keys/node-0.key"
  public_key_path: "data/keys/node-pub.key"
total: 4
network:
  topic: "test-topic"
consensus:
  type: "whirly"
  faultTolerance: "1/3"
  consensus_id: 1009
  whirly:
    type: "simple"
    batch_size: 50
    timeout: 3
`
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := NewConfig(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "whirly", cfg.Consensus.Type)
	if cfg.Consensus.Whirly != nil {
		assert.NotEmpty(t, cfg.Consensus.Whirly.Type, "Whirly类型不应为空")
		assert.Greater(t, cfg.Consensus.Whirly.BatchSize, 0, "批处理大小应该大于0")
		assert.Greater(t, cfg.Consensus.Whirly.Timeout, 0, "超时时间应该大于0")
		t.Logf("Whirly配置: Type=%s, BatchSize=%d, Timeout=%d",
			cfg.Consensus.Whirly.Type, cfg.Consensus.Whirly.BatchSize, cfg.Consensus.Whirly.Timeout)
	}
}

// TestExecutorConfig 测试执行器配置
func TestExecutorConfig(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	assert.NotNil(t, cfg.Executor, "执行器配置不应为空")
	assert.NotEmpty(t, cfg.Executor.Type, "执行器类型不应为空")
	assert.NotEmpty(t, cfg.Executor.Address, "执行器地址不应为空")
	t.Logf("执行器配置: Type=%s, Address=%s", cfg.Executor.Type, cfg.Executor.Address)
}

// TestLogConfig 测试日志配置
func TestLogConfig(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	assert.NotNil(t, cfg.Log, "日志配置不应为空")
	assert.NotEmpty(t, cfg.Log.Level, "日志级别不应为空")
	if cfg.Log.ToFile {
		assert.NotEmpty(t, cfg.Log.Filename, "日志文件名不应为空")
	}
	t.Logf("日志配置: Level=%s, ToFile=%v, Filename=%s",
		cfg.Log.Level, cfg.Log.ToFile, cfg.Log.Filename)
}

// TestNetworkConfig 测试网络配置
func TestNetworkConfig(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	assert.NotNil(t, cfg.Network, "网络配置不应为空")
	assert.NotEmpty(t, cfg.Network.P2PType, "P2P类型不应为空")
	assert.NotEmpty(t, cfg.Network.Topic, "Topic不应为空")
	assert.NotEmpty(t, cfg.Network.ModelType, "网络模型类型不应为空")
	t.Logf("网络配置: P2PType=%s, Topic=%s, ModelType=%s",
		cfg.Network.P2PType, cfg.Network.Topic, cfg.Network.ModelType)
}

// TestConfigPathPreservation 测试配置路径保存
func TestConfigPathPreservation(t *testing.T) {
	configPath := "config/config.yaml"
	cfg, err := NewConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, configPath, cfg.CfgPath, "配置路径应该被保存")
	t.Logf("配置路径: %s", cfg.CfgPath)
}

// TestConsensusKeysAssignment 测试共识密钥分配
func TestConsensusKeysAssignment(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	assert.NotNil(t, cfg.Consensus.Keys, "共识密钥不应为空")
	assert.Equal(t, cfg.Crypto, cfg.Consensus.Keys, "共识密钥应该指向主密钥配置")
	assert.NotNil(t, cfg.Consensus.Keys.PublicKey, "共识公钥不应为空")
	assert.NotNil(t, cfg.Consensus.Keys.PrivateKey, "共识私钥不应为空")
	t.Log("共识密钥分配验证通过")
}

// ========== Nodes Management Tests ==========

// TestNodesInitialization 测试Nodes初始化
func TestNodesInitialization(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	assert.NotNil(t, cfg.Nodes, "Nodes应该被初始化")
	assert.NotNil(t, cfg.Consensus.Nodes, "共识配置的Nodes应该被初始化")
	assert.Equal(t, 1, cfg.GetNodeCount(), "单节点配置应该有1个节点")

	// 验证当前节点在Nodes中
	node, err := cfg.GetNodeFromSet(cfg.Node.ID)
	require.NoError(t, err)
	assert.Equal(t, cfg.Node, node, "Nodes中应该包含当前节点")

	// 验证内存共享：两个 Nodes 应该指向同一个 map
	assert.Same(t, cfg.Nodes, cfg.Consensus.Nodes, "Config.Nodes 和 Consensus.Nodes 应该共享同一块内存")
}

// TestAddNode 测试添加节点
func TestAddNode(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	newNode := &NodeInfo{
		ID:         1,
		P2PAddress: "127.0.0.1:8082",
		RpcAddress: "127.0.0.1:9082",
		DataDir:    "data/node1",
	}

	err = cfg.AddNode(newNode)
	require.NoError(t, err)

	assert.Equal(t, 2, cfg.GetNodeCount(), "添加后应该有2个节点")

	retrievedNode, err := cfg.GetNodeFromSet(1)
	require.NoError(t, err)
	assert.Equal(t, newNode, retrievedNode, "应该能获取到新添加的节点")

	// 验证内存共享：通过 Config 添加的节点自动出现在 Consensus.Nodes
	consensusNode, err := cfg.Consensus.GetNodeFromSet(1)
	require.NoError(t, err)
	assert.Equal(t, newNode, consensusNode, "节点应该自动同步到共识配置（内存共享）")
	assert.Same(t, retrievedNode, consensusNode, "两个引用应该指向同一个对象")
}

// TestRemoveNode 测试移除节点
func TestRemoveNode(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	// 先添加一个节点
	newNode := &NodeInfo{
		ID:         1,
		P2PAddress: "127.0.0.1:8082",
		RpcAddress: "127.0.0.1:9082",
		DataDir:    "data/node1",
	}
	err = cfg.AddNode(newNode)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.GetNodeCount())

	// 移除节点
	err = cfg.RemoveNode(1)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.GetNodeCount(), "移除后应该只有1个节点")

	// 验证节点已被移除
	_, err = cfg.GetNodeFromSet(1)
	assert.Error(t, err, "获取已移除的节点应该返回错误")

	// 验证内存共享：从 Config 移除的节点自动从 Consensus.Nodes 移除
	_, err = cfg.Consensus.GetNodeFromSet(1)
	assert.Error(t, err, "节点应该自动从共识配置中移除（内存共享）")
	assert.Equal(t, 1, cfg.Consensus.GetNodeCount(), "共识配置节点数应该一致")
}

// TestGetAllNodes 测试获取所有节点
func TestGetAllNodes(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	// 添加多个节点
	for i := int64(1); i <= 3; i++ {
		node := &NodeInfo{
			ID:         i,
			P2PAddress: fmt.Sprintf("127.0.0.1:808%d", i),
			RpcAddress: fmt.Sprintf("127.0.0.1:908%d", i),
			DataDir:    fmt.Sprintf("data/node%d", i),
		}
		err = cfg.AddNode(node)
		require.NoError(t, err)
	}

	allNodes := cfg.GetAllNodes()
	assert.Equal(t, 4, len(allNodes), "应该有4个节点（包括原始节点）")

	// 验证每个节点都存在
	for i := int64(0); i <= 3; i++ {
		_, exists := allNodes[i]
		assert.True(t, exists, fmt.Sprintf("节点%d应该存在", i))
	}
}

// TestGetNodesSlice 测试获取节点切片
func TestGetNodesSlice(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	// 添加节点
	for i := int64(1); i <= 2; i++ {
		node := &NodeInfo{
			ID:         i,
			P2PAddress: fmt.Sprintf("127.0.0.1:808%d", i),
			RpcAddress: fmt.Sprintf("127.0.0.1:908%d", i),
			DataDir:    fmt.Sprintf("data/node%d", i),
		}
		err = cfg.Consensus.AddNode(node)
		require.NoError(t, err)
	}

	nodesSlice := cfg.Consensus.GetNodesSlice()
	assert.Equal(t, 3, len(nodesSlice), "切片应该包含3个节点")

	// 验证切片中包含所有节点
	idMap := make(map[int64]bool)
	for _, node := range nodesSlice {
		idMap[node.ID] = true
	}
	assert.True(t, idMap[0], "应该包含节点0")
	assert.True(t, idMap[1], "应该包含节点1")
	assert.True(t, idMap[2], "应该包含节点2")
}

// TestNodesErrors 测试Nodes错误情况
func TestNodesErrors(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	// 测试添加nil节点
	err = cfg.AddNode(nil)
	assert.Error(t, err, "添加nil节点应该返回错误")

	// 测试移除不存在的节点
	err = cfg.RemoveNode(999)
	assert.Error(t, err, "移除不存在的节点应该返回错误")

	// 测试获取不存在的节点
	_, err = cfg.GetNodeFromSet(999)
	assert.Error(t, err, "获取不存在的节点应该返回错误")
}

// TestNodesMemorySharing 测试Nodes内存共享
func TestNodesMemorySharing(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml")
	require.NoError(t, err)

	// 验证初始状态：两个 Nodes 指向同一块内存
	assert.Same(t, cfg.Nodes, cfg.Consensus.Nodes, "应该共享同一块内存")

	// 通过 Config.Nodes 直接添加节点
	node1 := &NodeInfo{ID: 10, P2PAddress: "127.0.0.1:8090"}
	cfg.Nodes[10] = node1

	// 验证在 Consensus.Nodes 中可见
	retrieved, err := cfg.Consensus.GetNodeFromSet(10)
	require.NoError(t, err)
	assert.Same(t, node1, retrieved, "通过 Nodes 直接添加的节点应该在 Consensus.Nodes 中可见")

	// 通过 Consensus.Nodes 直接删除节点
	delete(cfg.Consensus.Nodes, 10)

	// 验证在 Config.Nodes 中已删除
	_, exists := cfg.Nodes[10]
	assert.False(t, exists, "通过 Consensus.Nodes 删除的节点应该在 Config.Nodes 中消失")

	// 通过 Consensus.AddNode 添加节点
	node2 := &NodeInfo{ID: 20, P2PAddress: "127.0.0.1:8091"}
	err = cfg.Consensus.AddNode(node2)
	require.NoError(t, err)

	// 验证在 Config.Nodes 中可见
	retrieved2, err := cfg.GetNodeFromSet(20)
	require.NoError(t, err)
	assert.Same(t, node2, retrieved2, "通过 Consensus.AddNode 添加的节点应该在 Config.Nodes 中可见")
}
