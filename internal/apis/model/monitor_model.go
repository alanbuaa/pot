package model

// SystemOverview represents the system overview status
type SystemOverview struct {
	Uptime             int64    `json:"uptime"`
	TotalNodes         int      `json:"totalNodes"`
	OnlineNodes        int      `json:"onlineNodes"`
	ConsensusTypes     []string `json:"consensusTypes"`
	NetworkStatus      string   `json:"networkStatus"`
	NetworkUtilization float64  `json:"networkUtilization"`
	CurrentHeight      uint64   `json:"currentHeight"`
	AvgBlockTime       float64  `json:"avgBlockTime"`
	LastBlockTime      string   `json:"lastBlockTime"`
}

// POTStatus represents POT consensus status
type POTStatus struct {
	ConsensusType     string  `json:"consensusType"`
	Epoch             uint64  `json:"epoch"`
	Difficulty        string  `json:"difficulty"`
	CurrentHeight     uint64  `json:"currentHeight"`
	WorkFlag          bool    `json:"workFlag"`
	Timestamp         string  `json:"timestamp"`
	Nonce             uint64  `json:"nonce"`
	UncleCount        int     `json:"uncleCount"`
	AvgMiningTime     float64 `json:"avgMiningTime"`
	MiningSuccessRate float64 `json:"miningSuccessRate"`
}

// VDFWorker represents a single VDF worker status
type VDFWorker struct {
	Progress   float64 `json:"progress"`
	Iterations uint64  `json:"iterations,omitempty"`
	Status     string  `json:"status"`
}

// VDFThreadWorker represents VDF1 thread worker with ID
type VDFThreadWorker struct {
	WorkerID   int     `json:"workerId"`
	Progress   float64 `json:"progress"`
	Iterations uint64  `json:"iterations,omitempty"`
	Status     string  `json:"status"`
}

// VDFWorkerWithChannel represents VDF worker with channel buffer info
type VDFWorkerWithChannel struct {
	Progress      float64 `json:"progress"`
	Iterations    uint64  `json:"iterations"`
	Status        string  `json:"status"`
	ChannelBuffer int     `json:"channelBuffer"`
}

// VDFChecker represents VDF checker status
type VDFChecker struct {
	Status          string `json:"status"`
	VerifyFailCount int    `json:"verifyFailCount"`
}

// VDFStatus represents VDF computation status
type VDFStatus struct {
	VDF0            VDFWorkerWithChannel `json:"vdf0"`
	VDF1            []VDFThreadWorker    `json:"vdf1"`
	VDFHalf         VDFWorkerWithChannel `json:"vdfHalf"`
	VDFChecker      VDFChecker           `json:"vdfChecker"`
	AbortStatus     bool                 `json:"abortStatus"`
	AvgComputeTime  float64              `json:"avgComputeTime"`
	TotalIterations uint64               `json:"totalIterations"`
	CPUCounter      int                  `json:"cpuCounter"`
}

// CommitteeMember represents a committee member
type CommitteeMember struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
	IsLeader  bool   `json:"isLeader"`
}

// Shard represents a shard information
type Shard struct {
	Name             string `json:"name"`
	ID               int    `json:"id"`
	LeaderAddress    string `json:"leaderAddress"`
	CommitteeMembers int    `json:"committeeMembers"`
	Status           string `json:"status"`
}

// CommitteeStatus represents committee consensus status
type CommitteeStatus struct {
	ConsensusType      string            `json:"consensusType"`
	CommitteeSize      int               `json:"committeeSize"`
	CommitteeCount     int               `json:"committeeCount"`
	ConfirmDelay       int               `json:"confirmDelay"`
	WorkHeight         uint64            `json:"workHeight"`
	BatchSize          int               `json:"batchSize"`
	InCommittee        bool              `json:"inCommittee"`
	Role               string            `json:"role"`
	Committee          []CommitteeMember `json:"committee"`
	CommitteePublicKey string            `json:"committeePublicKey"`
	Shardings          []Shard           `json:"shardings"`
	WorkStage          string            `json:"workStage"`
	Timeout            int               `json:"timeout"`
	MessageQueueLength int               `json:"messageQueueLength"`
	ElectionHeight     uint64            `json:"electionHeight"`
}

// RewardRatio represents reward distribution ratios
type RewardRatio struct {
	Exchequer       float64 `json:"exchequer"`
	Miner           float64 `json:"miner"`
	UncleBlockMiner float64 `json:"uncleBlockMiner"`
	CommitteeLeader float64 `json:"committeeLeader"`
	CommitteeMember float64 `json:"committeeMember"`
}

// LockRates represents lock period interest rates
type LockRates struct {
	Saving     float64 `json:"saving"`
	HalfYear   float64 `json:"halfYear"`
	OneYear    float64 `json:"oneYear"`
	ThreeYears float64 `json:"threeYears"`
	TenYears   float64 `json:"tenYears"`
}

// BCIStatus represents BCI incentive system status
type BCIStatus struct {
	TotalReward      float64     `json:"totalReward"`
	LockedReward     float64     `json:"lockedReward"`
	TotalInterest    float64     `json:"totalInterest"`
	RewardRatio      RewardRatio `json:"rewardRatio"`
	LockRates        LockRates   `json:"lockRates"`
	CoinbaseLock     int         `json:"coinbaseLock"`
	ExecuteHeight    uint64      `json:"executeHeight"`
	IncentiveHeight  uint64      `json:"incentiveHeight"`
	PendingRewards   int         `json:"pendingRewards"`
	UTXOCount        int         `json:"utxoCount"`
	TotalDistributed float64     `json:"totalDistributed"`
}

// TxTypeStats represents transaction type statistics
type TxTypeStats struct {
	Normal    int `json:"normal"`
	BCI       int `json:"bci"`
	Devastate int `json:"devastate"`
}

// RecentTransaction represents a recent transaction
type RecentTransaction struct {
	Hash      string `json:"hash"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Status    string `json:"status"`
}

// MempoolStatus represents mempool status
type MempoolStatus struct {
	TotalSize         int                 `json:"totalSize"`
	MarkedTxs         int                 `json:"markedTxs"`
	UnmarkedTxs       int                 `json:"unmarkedTxs"`
	TxTypes           TxTypeStats         `json:"txTypes"`
	AvgConfirmTime    float64             `json:"avgConfirmTime"`
	VerifySuccessRate float64             `json:"verifySuccessRate"`
	MemoryUsage       int64               `json:"memoryUsage"`
	RecentTxs         []RecentTransaction `json:"recentTxs"`
}

// NetworkNode represents a network node
type NetworkNode struct {
	ID          int    `json:"id"`
	PeerID      string `json:"peerId"`
	Address     string `json:"address"`
	Status      string `json:"status"`
	Connections int    `json:"connections"`
	Latency     int    `json:"latency"`
	Type        string `json:"type"`
	IsLeader    bool   `json:"isLeader"`
}

// NetworkEdge represents a connection between nodes
type NetworkEdge struct {
	Source       int `json:"source"`
	Target       int `json:"target"`
	Latency      int `json:"latency"`
	MessageCount int `json:"messageCount"`
}

// NetworkTopology represents network topology
type NetworkTopology struct {
	Nodes              []NetworkNode `json:"nodes"`
	Edges              []NetworkEdge `json:"edges"`
	P2PAdaptorType     string        `json:"p2pAdaptorType"`
	SubscribedTopics   []string      `json:"subscribedTopics"`
	MessageQueueLength int           `json:"messageQueueLength"`
	NetworkBandwidth   int64         `json:"networkBandwidth"`
}

// BlockInfo represents basic block information
type BlockInfo struct {
	Height    uint64 `json:"height"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
	TxCount   int    `json:"txCount"`
	Size      int    `json:"size"`
	Miner     string `json:"miner"`
}

// BlockDetail represents detailed block information
type BlockDetail struct {
	Height          uint64   `json:"height"`
	Hash            string   `json:"hash"`
	ParentHash      string   `json:"parentHash"`
	Timestamp       int64    `json:"timestamp"`
	TxCount         int      `json:"txCount"`
	Size            int      `json:"size"`
	Miner           string   `json:"miner"`
	Difficulty      string   `json:"difficulty"`
	Nonce           int64    `json:"nonce"`
	Mixdigest       string   `json:"mixdigest"`
	UncleHashes     []string `json:"uncleHashes"`
	Transactions    []string `json:"transactions"`
	CommitteePubkey string   `json:"committeePubkey"`
}

// MonitorService defines the interface for monitoring service
type MonitorService interface {
	GetSystemOverview() (*SystemOverview, error)
	GetPOTStatus() (*POTStatus, error)
	GetVDFStatus() (*VDFStatus, error)
	GetCommitteeStatus() (*CommitteeStatus, error)
	GetBCIStatus() (*BCIStatus, error)
	GetMempoolStatus() (*MempoolStatus, error)
	GetNetworkTopology() (*NetworkTopology, error)
	GetRecentBlocks(count int) ([]BlockInfo, error)
	GetBlockByHeight(height uint64) (*BlockDetail, error)
}
