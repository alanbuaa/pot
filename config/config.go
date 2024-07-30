package config

import (
	"fmt"
	"math/big"
	"os"

	"github.com/niclabs/tcrsa"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
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
	Snum           int64  `yaml:"snum"`
	SysPara        string `yaml:"sysPara"`
	Vdf0Iteration  int    `yaml:"vdf0Iteration"`
	Vdf1Iteration  int    `yaml:"vdf1Iteration"`
	Batchsize      int    `yaml:"batchsize"`
	ExcutorAddress string `yaml:"excutorAddress"`
	Timeout        int    `yaml:"timeout"`
	CommiteeSize   int    `yaml:"commiteeSize"`
	ConfirmDelay   int    `yaml:"confirmDelay"`
	Slowrate       int    `yaml:"slowrate"`
}
type ConsensusConfig struct {
	Type        string            `yaml:"type"`
	ConsensusID int64             `yaml:"consensus_id"`
	HotStuff    *HotStuffConfig   `yaml:"hotstuff"`
	Pow         *PoWConfig        `yaml:"pow"`
	Upgradeable *UpgradableConfig `yaml:"upgradeable"`
	Whirly      *WhirlyConfig     `yaml:"whirly"`
	PoT         *PoTConfig        `yaml:"pot"`
	Nodes       []*ReplicaInfo    // no need to have the following fields in config.yaml
	Keys        *KeySet
	Topic       string
	F           int // initialize in NewConsensus if neeeded
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
	Nodes         []*ReplicaInfo   `yaml:"nodes"`
	PublicKeyPath string           `yaml:"public_key_path"`
	Log           *LogConfig       `yaml:"log"`
	Executor      *ExecutorConfig  `yaml:"executor"`
	P2P           *P2PConfig       `yaml:"p2p"`
	Consensus     *ConsensusConfig `yaml:"consensus"`
	Topic         string           `yaml:"topic"`
	Total         int              `yaml:"total"`
	Keys          *KeySet
}

type ReplicaInfo struct {
	ID             int64  `yaml:"id"`
	Address        string `yaml:"address"`
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
	keys := new(KeySet)
	keys.PublicKey, err = crypto.ReadThresholdPublicKeyFromFile(cfg.PublicKeyPath)
	if err != nil {
		return nil, err
	}
	keys.PrivateKey, err = crypto.ReadThresholdPrivateKeyFromFile(cfg.GetNodeInfo(id).PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	cfg.Keys = keys
	cfg.Consensus.Nodes = cfg.Nodes
	cfg.Consensus.Keys = keys
	// cfg.Consensus.F = (len(cfg.Nodes) - 1) / 3
	cfg.Consensus.F = (cfg.Total - 1) / 3
	// cfg.Consensus.F = 1
	cfg.Consensus.Topic = cfg.Topic
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
		if info.ID == id {
			return info
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
