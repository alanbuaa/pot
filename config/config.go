package config

import (
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/niclabs/tcrsa"
	"gopkg.in/yaml.v3"
)

type NetworkType = int64

const (
	NetworkSync  NetworkType = iota
	NetworkAsync NetworkType = iota
)

type KeySet struct {
	PublicKey  *tcrsa.KeyMeta
	PrivateKey *tcrsa.KeyShare
}

type PoWConfig struct {
	InitDifficulty *big.Int `yaml:"init_difficulty"`
	Hash           string   `yaml:"hash"`
}

type HotStuffConfig struct {
	Type         string `yaml:"type"`
	BatchSize    int    `yaml:"batch_size"`
	BatchTimeout int    `yaml:"batch_timeout"`
	Timeout      int    `yaml:"timeout"`
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
	BciRpcAddress   string `yaml:"bciRpcAddress"`
	VdfType         string `yaml:"vdfType"`
}

type ConsensusConfig struct {
	Type         string            `yaml:"type"`
	ConsensusID  int64             `yaml:"consensus_id"`
	HotStuff     *HotStuffConfig   `yaml:"hotstuff"`
	Pow          *PoWConfig        `yaml:"pow"`
	Upgradeable  *UpgradableConfig `yaml:"upgradeable"`
	Whirly       *WhirlyConfig     `yaml:"whirly"`
	PoT          *PoTConfig        `yaml:"pot"`
	BlockTime    int               `yaml:"block_time"`     // 区块时间（秒）
	MaxBlockSize int               `yaml:"max_block_size"` // 最大区块大小（字节）
	Nodes        []*ReplicaInfo    // no need to have the following fields in config.yaml
	Keys         *KeySet
	Topic        string
	F            int // initialize in NewConsensus if neeeded
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

type P2PConfig struct {
	Type string `yaml:"type"`
}

type Config struct {
	CfgPath             string           `yaml:"-"`
	RestServerAddress   string           `yaml:"http_server_address"`
	AddressStartPort    string           `yaml:"address_start_port"`
	RpcAddressStartPort string           `yaml:"rpc_address_start_port"`
	DataDir             string           `yaml:"data_dir"`
	AddressIp           string           `yaml:"address_ip"`
	Nodes               []*ReplicaInfo   `yaml:"nodes"`
	PublicKeyPath       string           `yaml:"public_key_path"`
	Log                 *LogConfig       `yaml:"log"`
	Executor            *ExecutorConfig  `yaml:"executor"`
	P2P                 *P2PConfig       `yaml:"p2p"`
	Consensus           *ConsensusConfig `yaml:"consensus"`
	Topic               string           `yaml:"topic"`
	Total               int              `yaml:"total"`
	Keys                *KeySet
}

type ReplicaInfo struct {
	ID             int64  `yaml:"id"`
	Address        string `yaml:"address"`
	Datadir        string `yaml:"datadir"`
	RpcAddress     string `yaml:"rpc_address"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

func NewConfig(path string, id int64) (*Config, error) {
	cfg := new(Config)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// setup nodes info
	cfg.Nodes = make([]*ReplicaInfo, cfg.Total)
	for i := 0; i < cfg.Total; i++ {
		node := &ReplicaInfo{
			ID:         int64(i),
			Address:    cfg.GetAddress(i, cfg.AddressStartPort, cfg.AddressIp),
			RpcAddress: cfg.GetAddress(i, cfg.RpcAddressStartPort, cfg.AddressIp),
			Datadir:    cfg.DataDir + fmt.Sprintf("/node-%d", i),
		}
		cfg.Nodes[i] = node
	}
	cfg.Consensus.Nodes = cfg.Nodes
	// cfg.Keys = keys
	// cfg.Consensus.Keys = keys
	// cfg.Consensus.F = (len(cfg.Nodes) - 1) / 3
	cfg.Consensus.F = (cfg.Total - 1) / 3
	// cfg.Consensus.F = 1
	cfg.Consensus.Topic = cfg.Topic
	cfg.CfgPath = path
	return cfg, nil
}

func (c *ConsensusConfig) GetNodeInfo(id int64) *ReplicaInfo {
	for _, info := range c.Nodes {
		if info.ID == id {
			return info
		}
	}
	panic(fmt.Sprintf("node %d does not exist", id))
}

func (c *Config) GetNodeInfo(id int64) *ReplicaInfo {
	for _, info := range c.Nodes {
		if info != nil {
			if info.ID == id {
				return info
			}
		}
	}
	panic(fmt.Sprintf("node %d does not exist", id))
}

func DefaultWhirlyConfig() *ConsensusConfig {
	return &ConsensusConfig{
		Type:        "whirly",
		ConsensusID: 1009,
		HotStuff:    nil,
		Pow:         nil,
		Upgradeable: nil,
		Whirly: &WhirlyConfig{
			Type:      "simple",
			BatchSize: 10,
			Timeout:   2000,
		},
		PoT:   nil,
		Nodes: nil,
		Keys:  nil,
		F:     0,
	}
}

func (c *Config) GetAddress(id int, initPortStr string, ip string) string {
	initPort, err := strconv.Atoi(initPortStr)
	if err != nil {
		panic(fmt.Sprintf("GetAddress error: %s", err))
	}
	address := ip + ":" + strconv.Itoa(initPort+id)
	return address
}
