package cdl

import (
	"encoding/json"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
)

// CDLDescriptor CDL 共识描述符完整定义
type CDLDescriptor struct {
	Consensus ConsensusSpec `yaml:"consensus" json:"consensus"`
}

// ConsensusSpec 共识规范
type ConsensusSpec struct {
	Name                    string                  `yaml:"name" json:"name"`
	Version                 string                  `yaml:"version" json:"version"`
	Type                    string                  `yaml:"type" json:"type"`
	Components              Components              `yaml:"components" json:"components"`
	Parameters              Parameters              `yaml:"parameters" json:"parameters"`
	Phases                  []Phase                 `yaml:"phases" json:"phases"`
	StateMachine            StateMachine            `yaml:"state_machine" json:"state_machine"`
	SafetyProperties        []SafetyProperty        `yaml:"safety_properties" json:"safety_properties"`
	PerformanceRequirements PerformanceRequirements `yaml:"performance_requirements" json:"performance_requirements"`
}

// Components 共识组件定义
type Components struct {
	Crypto  CryptoComponent  `yaml:"crypto" json:"crypto"`
	Network NetworkComponent `yaml:"network" json:"network"`
	Storage StorageComponent `yaml:"storage" json:"storage"`
}

// CryptoComponent 加密组件
type CryptoComponent struct {
	Hash         string `yaml:"hash" json:"hash"`                   // SHA256, SHA3, BLAKE2b
	Signature    string `yaml:"signature" json:"signature"`         // ECDSA, EdDSA, BLS
	VRF          string `yaml:"vrf" json:"vrf"`                     // VRF 算法
	VDF          string `yaml:"vdf" json:"vdf"`                     // VDF 算法
	ThresholdSig string `yaml:"threshold_sig" json:"threshold_sig"` // 门限签名
}

// NetworkComponent 网络组件
type NetworkComponent struct {
	Topology  string `yaml:"topology" json:"topology"`   // gossip, mesh, star
	Broadcast string `yaml:"broadcast" json:"broadcast"` // reliable, best-effort
}

// StorageComponent 存储组件
type StorageComponent struct {
	Blockchain string `yaml:"blockchain" json:"blockchain"` // merkle-chain, dag
	State      string `yaml:"state" json:"state"`           // merkle-patricia, simple-map
}

// Parameters 共识参数
type Parameters struct {
	BlockTime    string                 `yaml:"block_time" json:"block_time"`         // 出块时间
	MaxBlockSize int                    `yaml:"max_block_size" json:"max_block_size"` // 最大区块大小
	Custom       map[string]interface{} `yaml:",inline" json:"custom,omitempty"`      // 自定义参数
}

// Phase 共识阶段定义
type Phase struct {
	Name    string   `yaml:"name" json:"name"`
	Entry   string   `yaml:"entry" json:"entry"`     // 入口点
	Actions []Action `yaml:"actions" json:"actions"` // 阶段动作
	Exit    string   `yaml:"exit" json:"exit"`       // 退出点
}

// Action 阶段动作
type Action struct {
	Type       string                 `yaml:"type" json:"type"`             // function, event, condition
	Name       string                 `yaml:"name" json:"name"`             // 动作名称
	Parameters map[string]interface{} `yaml:"parameters" json:"parameters"` // 参数
	Code       string                 `yaml:"code" json:"code"`             // 代码片段（可选）
}

// StateMachine 状态机定义
type StateMachine struct {
	States      []string     `yaml:"states" json:"states"`           // 状态列表
	Transitions []Transition `yaml:"transitions" json:"transitions"` // 状态转换
}

// Transition 状态转换
type Transition struct {
	From      string `yaml:"from" json:"from"`           // 源状态
	To        string `yaml:"to" json:"to"`               // 目标状态
	Event     string `yaml:"event" json:"event"`         // 触发事件
	Condition string `yaml:"condition" json:"condition"` // 转换条件
	Action    string `yaml:"action" json:"action"`       // 转换动作
}

// SafetyProperty 安全属性
type SafetyProperty struct {
	Name    string `yaml:"name" json:"name"`       // 属性名称
	Formula string `yaml:"formula" json:"formula"` // 形式化公式
}

// PerformanceRequirements 性能要求
type PerformanceRequirements struct {
	MinThroughput  int     `yaml:"min_throughput" json:"min_throughput"`   // 最小吞吐量(TPS)
	MaxLatency     int     `yaml:"max_latency" json:"max_latency"`         // 最大延迟(秒)
	FaultTolerance float64 `yaml:"fault_tolerance" json:"fault_tolerance"` // 容错率
}

// Validate 验证 CDL 描述符
func (cdl *CDLDescriptor) Validate() error {
	if cdl.Consensus.Name == "" {
		return ErrInvalidName
	}
	if cdl.Consensus.Version == "" {
		return ErrInvalidVersion
	}
	if cdl.Consensus.Type == "" {
		return ErrInvalidType
	}

	// 验证组件
	if err := cdl.Consensus.Components.Validate(); err != nil {
		return err
	}

	// 验证参数
	if err := cdl.Consensus.Parameters.Validate(); err != nil {
		return err
	}

	// 验证状态机
	if err := cdl.Consensus.StateMachine.Validate(); err != nil {
		return err
	}

	return nil
}

// Validate 验证组件配置
func (c *Components) Validate() error {
	// 验证加密组件
	if c.Crypto.Hash == "" {
		return ErrInvalidCrypto
	}
	validHashes := map[string]bool{
		"SHA256": true, "SHA3": true, "BLAKE2b": true,
	}
	if !validHashes[c.Crypto.Hash] {
		return ErrInvalidHashAlgorithm
	}

	// 验证签名算法
	if c.Crypto.Signature != "" {
		validSigs := map[string]bool{
			"ECDSA": true, "EdDSA": true, "BLS": true,
		}
		if !validSigs[c.Crypto.Signature] {
			return ErrInvalidSignatureAlgorithm
		}
	}

	return nil
}

// Validate 验证参数
func (p *Parameters) Validate() error {
	if p.BlockTime == "" {
		return ErrInvalidBlockTime
	}

	// 解析区块时间
	_, err := time.ParseDuration(p.BlockTime)
	if err != nil {
		return ErrInvalidBlockTime
	}

	if p.MaxBlockSize <= 0 {
		return ErrInvalidBlockSize
	}

	return nil
}

// Validate 验证状态机
func (sm *StateMachine) Validate() error {
	if len(sm.States) == 0 {
		return ErrInvalidStateMachine
	}

	// 验证状态转换
	stateMap := make(map[string]bool)
	for _, state := range sm.States {
		stateMap[state] = true
	}

	for _, trans := range sm.Transitions {
		if !stateMap[trans.From] {
			return ErrInvalidState
		}
		if !stateMap[trans.To] {
			return ErrInvalidState
		}
	}

	return nil
}

// Hash 计算 CDL 哈希
func (cdl *CDLDescriptor) Hash() []byte {
	data, _ := json.Marshal(cdl)
	return crypto.Hash(data)
}

// Serialize 序列化为字符串
func (cdl *CDLDescriptor) Serialize() string {
	data, _ := json.Marshal(cdl)
	return string(data)
}

// GetBlockTime 获取区块时间
func (p *Parameters) GetBlockTime() time.Duration {
	duration, _ := time.ParseDuration(p.BlockTime)
	return duration
}

// CDL 验证错误
var (
	ErrInvalidName               = &CDLError{Code: "INVALID_NAME", Message: "invalid consensus name"}
	ErrInvalidVersion            = &CDLError{Code: "INVALID_VERSION", Message: "invalid version"}
	ErrInvalidType               = &CDLError{Code: "INVALID_TYPE", Message: "invalid consensus type"}
	ErrInvalidCrypto             = &CDLError{Code: "INVALID_CRYPTO", Message: "invalid crypto component"}
	ErrInvalidHashAlgorithm      = &CDLError{Code: "INVALID_HASH", Message: "invalid hash algorithm"}
	ErrInvalidSignatureAlgorithm = &CDLError{Code: "INVALID_SIGNATURE", Message: "invalid signature algorithm"}
	ErrInvalidBlockTime          = &CDLError{Code: "INVALID_BLOCK_TIME", Message: "invalid block time"}
	ErrInvalidBlockSize          = &CDLError{Code: "INVALID_BLOCK_SIZE", Message: "invalid block size"}
	ErrInvalidStateMachine       = &CDLError{Code: "INVALID_STATE_MACHINE", Message: "invalid state machine"}
	ErrInvalidState              = &CDLError{Code: "INVALID_STATE", Message: "invalid state in transition"}
)

// CDLError CDL 错误类型
type CDLError struct {
	Code    string
	Message string
}

func (e *CDLError) Error() string {
	return e.Message
}
