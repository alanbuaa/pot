package cdl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLog = logrus.NewEntry(logrus.New())

// TestParser_Parse 测试解析 YAML
func TestParser_Parse(t *testing.T) {
	yamlContent := `
consensus:
  name: "TestConsensus"
  version: "1.0.0"
  type: "test"
  components:
    crypto:
      hash: "SHA256"
      signature: "ECDSA"
    network:
      topology: "gossip"
      broadcast: "reliable"
    storage:
      blockchain: "merkle-chain"
      state: "merkle-patricia"
  parameters:
    block_time: "5s"
    max_block_size: 1048576
  phases:
    - name: "init"
      entry: "start"
      actions:
        - type: "function"
          name: "initialize"
      exit: "initialized"
  state_machine:
    states: ["IDLE", "RUNNING"]
    transitions:
      - from: "IDLE"
        to: "RUNNING"
        event: "start"
        action: "run()"
  safety_properties:
    - name: "agreement"
      formula: "test"
  performance_requirements:
    min_throughput: 100
    max_latency: 10
    fault_tolerance: 0.33
`

	parser := NewParser(testLog)
	descriptor, err := parser.Parse(yamlContent)

	require.NoError(t, err)
	assert.NotNil(t, descriptor)
	assert.Equal(t, "TestConsensus", descriptor.Consensus.Name)
	assert.Equal(t, "1.0.0", descriptor.Consensus.Version)
	assert.Equal(t, "test", descriptor.Consensus.Type)
	assert.Equal(t, "SHA256", descriptor.Consensus.Components.Crypto.Hash)
	assert.Equal(t, "gossip", descriptor.Consensus.Components.Network.Topology)
	assert.Equal(t, 2, len(descriptor.Consensus.StateMachine.States))
}

// TestParser_ParseFile 测试从文件解析
func TestParser_ParseFile(t *testing.T) {
	// 创建临时文件
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yaml")

	yamlContent := `
consensus:
  name: "FileTest"
  version: "1.0"
  type: "test"
  components:
    crypto:
      hash: "SHA256"
    network:
      topology: "gossip"
      broadcast: "reliable"
    storage:
      blockchain: "merkle-chain"
      state: "simple-map"
  parameters:
    block_time: "10s"
    max_block_size: 1000
  state_machine:
    states: ["IDLE"]
    transitions: []
  performance_requirements:
    min_throughput: 10
    max_latency: 100
    fault_tolerance: 0.25
`

	err := os.WriteFile(filePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	parser := NewParser(testLog)
	descriptor, err := parser.ParseFile(filePath)

	require.NoError(t, err)
	assert.NotNil(t, descriptor)
	assert.Equal(t, "FileTest", descriptor.Consensus.Name)
}

// TestParser_ParseAndValidate 测试解析并验证
func TestParser_ParseAndValidate(t *testing.T) {
	yamlContent := `
consensus:
  name: "ValidTest"
  version: "1.0.0"
  type: "test"
  components:
    crypto:
      hash: "SHA256"
    network:
      topology: "gossip"
      broadcast: "reliable"
    storage:
      blockchain: "merkle-chain"
      state: "merkle-patricia"
  parameters:
    block_time: "5s"
    max_block_size: 1000
  state_machine:
    states: ["IDLE", "RUNNING"]
    transitions:
      - from: "IDLE"
        to: "RUNNING"
        event: "start"
  performance_requirements:
    min_throughput: 50
    max_latency: 20
    fault_tolerance: 0.33
`

	parser := NewParser(testLog)
	descriptor, err := parser.ParseAndValidate(yamlContent)

	require.NoError(t, err)
	assert.NotNil(t, descriptor)
}

// TestParser_ParseAndValidate_Invalid 测试解析无效 CDL
func TestParser_ParseAndValidate_Invalid(t *testing.T) {
	yamlContent := `
consensus:
  name: ""
  version: "1.0"
  type: "test"
`

	parser := NewParser(testLog)
	_, err := parser.ParseAndValidate(yamlContent)

	assert.Error(t, err)
}

// TestParser_Serialize 测试序列化
func TestParser_Serialize(t *testing.T) {
	descriptor := createTestCDL()

	parser := NewParser(testLog)
	yamlContent, err := parser.Serialize(descriptor)

	require.NoError(t, err)
	assert.NotEmpty(t, yamlContent)
	assert.Contains(t, yamlContent, "TestConsensus")
}

// TestParser_SerializeToFile 测试序列化到文件
func TestParser_SerializeToFile(t *testing.T) {
	descriptor := createTestCDL()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "output.yaml")

	parser := NewParser(testLog)
	err := parser.SerializeToFile(descriptor, filePath)

	require.NoError(t, err)

	// 验证文件是否存在
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// 读取并解析文件
	descriptor2, err := parser.ParseFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, descriptor.Consensus.Name, descriptor2.Consensus.Name)
}

// TestParser_MalformedYAML 测试格式错误的 YAML
func TestParser_MalformedYAML(t *testing.T) {
	yamlContent := `
consensus:
  name: "Test"
  invalid yaml here: [
`

	parser := NewParser(testLog)
	_, err := parser.Parse(yamlContent)

	assert.Error(t, err)
}
