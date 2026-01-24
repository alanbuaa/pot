package config

import (
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/niclabs/tcrsa"
	hsf "github.com/wjbbig/go-hotstuff"
	"gopkg.in/yaml.v3"
)

type NetworkType = int64

const (
	NetworkSync  NetworkType = iota
	NetworkAsync NetworkType = iota
)

type CryptoConfig struct {
	PublicKey  *tcrsa.KeyMeta
	PrivateKey *tcrsa.KeyShare
}

type PoWConfig struct {
	InitDifficulty *big.Int `yaml:"init_difficulty"`
	Hash           string   `yaml:"hash"`
	Dynamic        bool     `yaml:"dynamic"`
}

type HotStuffConfig struct {
	Type         string `yaml:"type"`
	BatchSize    int    `yaml:"batch_size"`
	BatchTimeout int    `yaml:"batch_timeout"`
	Timeout      int    `yaml:"timeout"`
	Dynamic      bool   `yaml:"dynamic"`
}

type UpgradableConfig struct {
	InitConsensus *ConsensusConfig `yaml:"init_consensus"`
	CommitTime    int              `yaml:"commit_time"`
	NetworkType   NetworkType      `yaml:"network_type"` // 0=>synchronous 1=> asynchronous or partial synchronous
}

type WhirlyConfig struct {
	Type      string `yaml:"type"`
	BatchSize int    `yaml:"batch_size"`
	Timeout   int    `yaml:"timeout"`
	Dynamic   bool   `yaml:"dynamic"`
}

type PoTConfig struct {
	Snum            int64  `yaml:"snum"`
	SysPara         string `yaml:"sysPara"`
	Vdf0Iteration   int    `yaml:"vdf0Iteration"`
	Vdf1Iteration   int    `yaml:"vdf1Iteration"`
	Batchsize       int    `yaml:"batchsize"`
	ExecutorAddress string `yaml:"executorAddress"`
	Timeout         int    `yaml:"timeout"`
	CommiteeSize    int    `yaml:"commiteeSize"`
	ConfirmDelay    int    `yaml:"confirmDelay"`
	Slowrate        int    `yaml:"slowrate"`
	VdfType         string `yaml:"vdfType"`
	Dynamic         bool   `yaml:"dynamic"`
	BciRpcAddress   string `yaml:"-"`
}

type ConsensusConfig struct {
	Type        string            `yaml:"type"`
	FaultModel  string            `yaml:"fault_model"` // 1/3 or 1/2
	ConsensusID int64             `yaml:"consensus_id"`
	HotStuff    *HotStuffConfig   `yaml:"hotstuff"`
	Pow         *PoWConfig        `yaml:"pow"`
	Upgradeable *UpgradableConfig `yaml:"upgradeable"`
	Whirly      *WhirlyConfig     `yaml:"whirly"`
	PoT         *PoTConfig        `yaml:"pot"`

	BlockTime    int `yaml:"block_time"`
	MaxBlockSize int `yaml:"max_block_size"`

	NodeId int64               `yaml:"-"`
	Nodes  map[int64]*NodeInfo `yaml:"-"` // 所有节点配置映射 key=nodeID, value=NodeInfo
	Keys   *CryptoConfig
	Topic  string
	Total  int // initialize in NewConsensus if neeeded
	Fault  int // initialize in NewConsensus if neeeded
}

type LogConfig struct {
	Level    string `yaml:"level"`
	ToFile   bool   `yaml:"to_file"`
	Filename string `yaml:"filename"`
}

type ExecutorConfig struct {
	Type    string `yaml:"type"`
	Address string `yaml:"address"`
}

type Config struct {
	CfgPath string `yaml:"-"`
	Crypto  *CryptoConfig

	Node      *NodeInfo           `yaml:"node"`  // 本节点配置
	Nodes     map[int64]*NodeInfo `yaml:"-"`     // 节点映射 key=nodeID, value=NodeInfo
	Total     int                 `yaml:"total"` // used if exit
	Executor  *ExecutorConfig     `yaml:"executor"`
	Log       *LogConfig          `yaml:"log"`
	Network   *NetworkConfig      `yaml:"network"`
	Consensus *ConsensusConfig    `yaml:"consensus"`
}

type NodeInfo struct {
	ID               int64  `yaml:"id"`
	P2PAddress       string `yaml:"p2p_address"`
	RpcAddress       string `yaml:"rpc_address"`
	BciRpcAddress    string `yaml:"bci_rpc_server"`
	PrivateKeyPath   string `yaml:"private_key_path"`
	PublicKeyPath    string `yaml:"public_key_path"`
	ApiServerAddress string `yaml:"api_server_address"`
	DataDir          string `yaml:"data_dir"`
}

type NetworkConfig struct {
	P2PType   string `yaml:"p2p-type"`
	Topic     string `yaml:"topic"`
	ModelType string `yaml:"model-type"`
}

type ClientConfig struct {
	RpcAddress string     `yaml:"rpc_address"`
	Log        *LogConfig `yaml:"log"`
}

func NewConfig(path string) (*Config, error) {
	cfg := new(Config)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Node == nil {
		return nil, fmt.Errorf("node configuration is missing")
	}

	if cfg.Consensus == nil {
		return nil, fmt.Errorf("consensus configuration is missing")
	}

	cryptoKeys, err := cfg.LoadCryptoKeys()
	if err != nil {
		return nil, fmt.Errorf("load crypto keys error: %s", err)
	}
	cfg.Consensus.Keys = cryptoKeys

	// 初始化 Nodes
	cfg.Nodes = make(map[int64]*NodeInfo)
	// 将本节点配置加入 Nodes
	if cfg.Node != nil {
		cfg.Nodes[cfg.Node.ID] = cfg.Node
	}
	cfg.Consensus.Nodes = cfg.Nodes

	if cfg.Consensus.FaultModel == "1/3" {
		cfg.Consensus.Fault = (cfg.Total - 1) / 3
	} else if cfg.Consensus.FaultModel == "1/2" {
		cfg.Consensus.Fault = (cfg.Total - 1) / 2
	} else {
		return nil, fmt.Errorf("unknown consensus faulty type: %s", cfg.Consensus.FaultModel)
	}
	cfg.Consensus.Topic = cfg.Network.Topic
	cfg.Consensus.NodeId = cfg.Node.ID

	cfg.CfgPath = path
	return cfg, nil
}

func (c *Config) LoadCryptoKeys() (*CryptoConfig, error) {
	privateKey, err := hsf.ReadThresholdPrivateKeyFromFile(c.Node.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Load private key error: %s", err)
	}
	publicKey, err := hsf.ReadThresholdPublicKeyFromFile(c.Node.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Load public key error: %s", err)
	}
	keys := &CryptoConfig{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}
	c.Crypto = keys
	return keys, nil
}

func (c *Config) GetAddress(id int, initPortStr string, ip string) string {
	initPort, err := strconv.Atoi(initPortStr)
	if err != nil {
		panic(fmt.Sprintf("GetAddress error: %s", err))
	}
	address := ip + ":" + strconv.Itoa(initPort+id)
	return address
}

func NewClientConfig(path string) (*ClientConfig, error) {
	cfg := new(ClientConfig)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Log == nil {
		return nil, fmt.Errorf("log configuration is missing")
	}

	return cfg, nil
}

// ========== Nodes Management Methods ==========

// AddNode 向配置中添加节点
func (c *Config) AddNode(node *NodeInfo) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}
	if c.Nodes == nil {
		c.Nodes = make(map[int64]*NodeInfo)
		// 确保 Consensus.Nodes 引用同一个 map
		if c.Consensus != nil {
			c.Consensus.Nodes = c.Nodes
		}
	}
	c.Nodes[node.ID] = node
	return nil
}

// RemoveNode 从配置中移除节点
func (c *Config) RemoveNode(nodeID int64) error {
	if c.Nodes == nil {
		return fmt.Errorf("Nodes is not initialized")
	}
	if _, exists := c.Nodes[nodeID]; !exists {
		return fmt.Errorf("node %d does not exist", nodeID)
	}
	delete(c.Nodes, nodeID)
	return nil
}

// GetNodeFromSet 从 Nodes 中获取节点信息
func (c *Config) GetNodeFromSet(nodeID int64) (*NodeInfo, error) {
	if c.Nodes == nil {
		return nil, fmt.Errorf("Nodes is not initialized")
	}
	node, exists := c.Nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %d not found in Nodes", nodeID)
	}
	return node, nil
}

// GetAllNodes 获取所有节点信息
func (c *Config) GetAllNodes() map[int64]*NodeInfo {
	if c.Nodes == nil {
		return make(map[int64]*NodeInfo)
	}
	return c.Nodes
}

// GetNodeCount 获取节点总数
func (c *Config) GetNodeCount() int {
	if c.Nodes == nil {
		return 0
	}
	return len(c.Nodes)
}

// ========== ConsensusConfig Nodes Management Methods ==========

// AddNode 向共识配置中添加节点
func (cc *ConsensusConfig) AddNode(node *NodeInfo) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}
	if cc.Nodes == nil {
		cc.Nodes = make(map[int64]*NodeInfo)
	}
	cc.Nodes[node.ID] = node
	return nil
}

// RemoveNode 从共识配置中移除节点
func (cc *ConsensusConfig) RemoveNode(nodeID int64) error {
	if cc.Nodes == nil {
		return fmt.Errorf("Nodes is not initialized")
	}
	if _, exists := cc.Nodes[nodeID]; !exists {
		return fmt.Errorf("node %d does not exist", nodeID)
	}
	delete(cc.Nodes, nodeID)
	return nil
}

// GetNodeFromSet 从共识配置的 Nodes 中获取节点信息
func (cc *ConsensusConfig) GetNodeFromSet(nodeID int64) (*NodeInfo, error) {
	if cc.Nodes == nil {
		return nil, fmt.Errorf("Nodes is not initialized")
	}
	node, exists := cc.Nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %d not found in Nodes", nodeID)
	}
	return node, nil
}

// GetAllNodes 获取共识配置中所有节点信息
func (cc *ConsensusConfig) GetAllNodes() map[int64]*NodeInfo {
	if cc.Nodes == nil {
		return make(map[int64]*NodeInfo)
	}
	return cc.Nodes
}

// GetNodeCount 获取共识配置中的节点总数
func (cc *ConsensusConfig) GetNodeCount() int {
	if cc.Nodes == nil {
		return 0
	}
	return len(cc.Nodes)
}

// GetNodesSlice 将 Nodes 转换为切片（用于需要遍历的场景）
func (cc *ConsensusConfig) GetNodesSlice() []*NodeInfo {
	if cc.Nodes == nil {
		return []*NodeInfo{}
	}
	nodes := make([]*NodeInfo, 0, len(cc.Nodes))
	for _, node := range cc.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// isDynamicConsensus 判断是否为动态共识
// 根据配置文件中各共识配置的 dynamic 字段判断
func IsDynamicConsensus(cfg *Config) bool {
	if cfg == nil || cfg.Consensus == nil {
		return false
	}

	consensusType := cfg.Consensus.Type
	switch consensusType {
	case "pot":
		return cfg.Consensus.PoT != nil && cfg.Consensus.PoT.Dynamic
	case "pow":
		return cfg.Consensus.Pow != nil && cfg.Consensus.Pow.Dynamic
	case "hotstuff":
		return cfg.Consensus.HotStuff != nil && cfg.Consensus.HotStuff.Dynamic
	case "whirly":
		return cfg.Consensus.Whirly != nil && cfg.Consensus.Whirly.Dynamic
	default:
		return false
	}
}
