package apis

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// PotMonitor implements the MonitorService interface for POT consensus monitoring
type PotMonitor struct {
	worker    *pot.Worker
	engine    *pot.PoTEngine
	log       *logrus.Entry
	startTime time.Time
	mu        sync.RWMutex

	// Cached statistics
	avgMiningTime     float64
	miningSuccessRate float64
	avgComputeTime    float64
	avgBlockTime      float64
}

// NewPotMonitor creates a new POT monitor instance
func NewPotMonitor(worker *pot.Worker, engine *pot.PoTEngine, log *logrus.Entry) model.MonitorService {
	return &PotMonitor{
		worker:    worker,
		engine:    engine,
		log:       log,
		startTime: time.Now(),
	}
}

// GetSystemOverview returns system overview information
func (m *PotMonitor) GetSystemOverview() (*model.SystemOverview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := int64(time.Since(m.startTime).Seconds())
	currentHeight := m.worker.GetChainReader().GetCurrentHeight()

	// Get last block time
	lastBlock, _ := m.worker.GetChainReader().GetByHeight(currentHeight)
	var lastBlockTime string
	if lastBlock != nil && lastBlock.GetHeader() != nil {
		lastBlockTime = lastBlock.GetHeader().Timestamp.Format(time.RFC3339)
	} else {
		lastBlockTime = time.Now().Format(time.RFC3339)
	}

	// Calculate average block time (simplified)
	avgBlockTime := 5.0 // Default value, can be calculated from recent blocks
	if currentHeight > 10 {
		// Get last 10 blocks and calculate average time
		var totalTime int64
		for i := currentHeight; i > currentHeight-10 && i > 0; i-- {
			block, err := m.worker.GetChainReader().GetByHeight(i)
			if err == nil && block != nil && block.GetHeader() != nil && i > 1 {
				prevBlock, err2 := m.worker.GetChainReader().GetByHeight(i - 1)
				if err2 == nil && prevBlock != nil && prevBlock.GetHeader() != nil {
					totalTime += block.GetHeader().Timestamp.Unix() - prevBlock.GetHeader().Timestamp.Unix()
				}
			}
		}
		if totalTime > 0 {
			avgBlockTime = float64(totalTime) / 10.0
		}
	}

	return &model.SystemOverview{
		Uptime:             uptime,
		TotalNodes:         len(m.engine.GetConfig().Nodes),
		OnlineNodes:        len(m.engine.GetConfig().Nodes), // Simplified: assume all nodes online
		ConsensusTypes:     []string{"POT", "SimpleWhirly"},
		NetworkStatus:      "healthy",
		NetworkUtilization: 80, // Placeholder
		CurrentHeight:      currentHeight,
		AvgBlockTime:       avgBlockTime,
		LastBlockTime:      lastBlockTime,
	}, nil
}

// GetPOTStatus returns POT consensus status
func (m *PotMonitor) GetPOTStatus() (*model.POTStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	currentHeight := m.worker.GetChainReader().GetCurrentHeight()
	currentBlock, _ := m.worker.GetChainReader().GetByHeight(currentHeight)

	var difficulty string
	var nonce uint64
	var uncleCount int
	if currentBlock != nil && currentBlock.GetHeader() != nil {
		header := currentBlock.GetHeader()
		difficulty = hexutil.EncodeBig(header.Difficulty)
		nonce = uint64(header.Nonce)
		uncleCount = len(header.UncleHash)
	} else {
		difficulty = "0x0"
	}

	return &model.POTStatus{
		ConsensusType:     "POT",
		Epoch:             m.worker.GetEpoch(),
		Difficulty:        difficulty,
		CurrentHeight:     currentHeight,
		WorkFlag:          m.worker.GetWorkFlag(),
		Timestamp:         time.Now().Format(time.RFC3339),
		Nonce:             nonce,
		UncleCount:        uncleCount,
		AvgMiningTime:     3.5,  // Placeholder
		MiningSuccessRate: 85.5, // Placeholder
	}, nil
}

// GetVDFStatus returns VDF computation status
func (m *PotMonitor) GetVDFStatus() (*model.VDFStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get VDF0 status
	vdf0 := m.worker.GetVDF0()
	vdf0Status := model.VDFWorkerWithChannel{
		Progress:      calculateVDFProgress(vdf0),
		Iterations:    getVDFIterations(vdf0),
		Status:        getVDFStatus(vdf0),
		ChannelBuffer: len(m.worker.GetVDF0Chan()),
	}

	// Get VDF1 status (multiple workers)
	vdf1List := m.worker.GetVDF1()
	vdf1Status := make([]model.VDFThreadWorker, 0, len(vdf1List))
	for i, vdf := range vdf1List {
		vdf1Status = append(vdf1Status, model.VDFThreadWorker{
			WorkerID:   i,
			Progress:   calculateVDFProgress(vdf),
			Iterations: getVDFIterations(vdf),
			Status:     getVDFStatus(vdf),
		})
	}

	// Get VDFHalf status
	vdfHalf := m.worker.GetVDFHalf()
	vdfHalfStatus := model.VDFWorkerWithChannel{
		Progress:      calculateVDFProgress(vdfHalf),
		Iterations:    getVDFIterations(vdfHalf),
		Status:        getVDFStatus(vdfHalf),
		ChannelBuffer: len(m.worker.GetVDFHalfChan()),
	}

	// Get abort status
	abort := m.worker.GetAbort()
	abortStatus := false
	if abort != nil {
		abortStatus = abort.GetAbortFlag()
	}

	// Calculate total iterations
	totalIterations := uint64(0)
	if vdf0 != nil {
		totalIterations += getVDFIterations(vdf0)
	}
	for _, vdf := range vdf1List {
		if vdf != nil {
			totalIterations += getVDFIterations(vdf)
		}
	}
	if vdfHalf != nil {
		totalIterations += getVDFIterations(vdfHalf)
	}

	return &model.VDFStatus{
		VDF0:            vdf0Status,
		VDF1:            vdf1Status,
		VDFHalf:         vdfHalfStatus,
		VDFChecker:      model.VDFChecker{Status: "active", VerifyFailCount: 0},
		AbortStatus:     abortStatus,
		AvgComputeTime:  45.5, // Placeholder
		TotalIterations: totalIterations,
		CPUCounter:      len(vdf1List),
	}, nil
}

// GetCommitteeStatus returns committee consensus status
func (m *PotMonitor) GetCommitteeStatus() (*model.CommitteeStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	whirly := m.worker.GetWhirly()
	if whirly == nil {
		return &model.CommitteeStatus{
			ConsensusType:  "SimpleWhirly",
			CommitteeSize:  0,
			CommitteeCount: 0,
			InCommittee:    false,
			Role:           "observer",
		}, nil
	}

	// Get committee members
	committee := make([]model.CommitteeMember, 0)
	// This would need access to actual committee data structure
	// Placeholder implementation

	// Get shardings info
	shardings := make([]model.Shard, 0)
	// Placeholder implementation

	return &model.CommitteeStatus{
		ConsensusType:      "SimpleWhirly",
		CommitteeSize:      4,
		CommitteeCount:     4,
		ConfirmDelay:       6,
		WorkHeight:         m.worker.GetChainReader().GetCurrentHeight(),
		BatchSize:          100,
		InCommittee:        true,
		Role:               "member",
		Committee:          committee,
		CommitteePublicKey: "",
		Shardings:          shardings,
		WorkStage:          "consensus",
		Timeout:            5000,
		MessageQueueLength: 0,
		ElectionHeight:     0,
	}, nil
}

// GetBCIStatus returns BCI incentive system status
func (m *PotMonitor) GetBCIStatus() (*model.BCIStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// BCI data is embedded in the block and transaction system
	// We return the static configuration and calculated values
	return &model.BCIStatus{
		TotalReward:   65536,
		LockedReward:  0, // Would need to calculate from UTXOs
		TotalInterest: 0, // Would need to calculate from locked UTXOs
		RewardRatio: model.RewardRatio{
			Exchequer:       30,
			Miner:           50,
			UncleBlockMiner: 2,
			CommitteeLeader: 20,
			CommitteeMember: 10,
		},
		LockRates: model.LockRates{
			Saving:     0.1,
			HalfYear:   0.5,
			OneYear:    1.0,
			ThreeYears: 2.0,
			TenYears:   5.0,
		},
		CoinbaseLock:     6,
		ExecuteHeight:    m.worker.GetExecuteHeight(),
		IncentiveHeight:  m.worker.GetIncentiveHeight(),
		PendingRewards:   len(m.worker.GetMempool().GetAllBciRewards()),
		UTXOCount:        0, // Would need access to UTXO set
		TotalDistributed: 0, // Would need to calculate from chain history
	}, nil
}

// GetMempoolStatus returns mempool status
func (m *PotMonitor) GetMempoolStatus() (*model.MempoolStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mempool := m.worker.GetMempool()
	if mempool == nil {
		return &model.MempoolStatus{
			TotalSize:   0,
			MarkedTxs:   0,
			UnmarkedTxs: 0,
			TxTypes: model.TxTypeStats{
				Normal:    0,
				BCI:       0,
				Devastate: 0,
			},
		}, nil
	}

	totalSize, markedTxs := mempool.GetSize()
	unmarkedTxs := totalSize - markedTxs

	// Get recent transactions
	recentTxs := make([]model.RecentTransaction, 0)
	rawTxs := mempool.GetRecentTxs(10)
	for _, tx := range rawTxs {
		recentTxs = append(recentTxs, model.RecentTransaction{
			Hash:      hexutil.Encode(tx.Txid[:]),
			Type:      getTxType(tx),
			Timestamp: time.Now().Format(time.RFC3339),
			Status:    "pending",
		})
	}

	// Count transaction types
	txTypes := model.TxTypeStats{
		Normal:    0,
		BCI:       0,
		Devastate: 0,
	}
	allTxs := mempool.GetAllTxs()
	for _, tx := range allTxs {
		switch getTxType(tx) {
		case "normal":
			txTypes.Normal++
		case "bci":
			txTypes.BCI++
		case "devastate":
			txTypes.Devastate++
		}
	}

	return &model.MempoolStatus{
		TotalSize:         totalSize,
		MarkedTxs:         markedTxs,
		UnmarkedTxs:       unmarkedTxs,
		TxTypes:           txTypes,
		AvgConfirmTime:    8.5,
		VerifySuccessRate: 95.5,
		MemoryUsage:       int64(totalSize * 1024), // Rough estimate
		RecentTxs:         recentTxs,
	}, nil
}

// GetNetworkTopology returns network topology information
func (m *PotMonitor) GetNetworkTopology() (*model.NetworkTopology, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]model.NetworkNode, 0)
	edges := make([]model.NetworkEdge, 0)

	// Build nodes from config
	for i, node := range m.engine.GetConfig().Nodes {
		nodeStatus := model.NetworkNode{
			ID:          int(i),
			PeerID:      fmt.Sprintf("peer-%d", i),
			Address:     node.P2PAddress,
			Status:      "online",
			Connections: 0,
			Latency:     20,
			Type:        "pot",
			IsLeader:    false,
		}

		// Determine node type based on configuration
		if i < 4 {
			nodeStatus.Type = "committee"
			if i == 0 {
				nodeStatus.IsLeader = true
			}
		}

		nodes = append(nodes, nodeStatus)
	}

	// Build edges (simplified: create a mesh topology)
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			edges = append(edges, model.NetworkEdge{
				Source:       int(i),
				Target:       int(j),
				Latency:      15,
				MessageCount: 500,
			})
		}
	}

	p2pType := "libp2p"
	if m.engine.GetAdaptor() != nil {
		p2pType = m.engine.GetAdaptor().GetP2PType()
	}

	return &model.NetworkTopology{
		Nodes:              nodes,
		Edges:              edges,
		P2PAdaptorType:     p2pType,
		SubscribedTopics:   []string{"blocks", "transactions", "consensus"},
		MessageQueueLength: len(m.engine.GetMsgByteEntrance()),
		NetworkBandwidth:   1048576,
	}, nil
}

// Helper functions

func calculateVDFProgress(vdf *types.VDF) float64 {
	if vdf == nil {
		return 0.0
	}
	// Calculate progress based on whether VDF is finished
	if vdf.IsFinished() {
		return 100.0
	}
	// For running VDF, return a reasonable estimate
	return 50.0
}

func getVDFIterations(vdf *types.VDF) uint64 {
	if vdf == nil {
		return 0
	}
	return uint64(vdf.Iterations)
}

func getVDFStatus(vdf *types.VDF) string {
	if vdf == nil {
		return "idle"
	}
	if vdf.IsFinished() {
		return "done"
	}
	return "computing"
}

func getTxType(tx *types.RawTx) string {
	if tx == nil {
		return "normal"
	}
	// Determine transaction type based on TxOutput BciType
	for _, output := range tx.TxOutput {
		// Check if it's a BCI transaction
		if output.BciType > 0 {
			return "bci"
		}
		// Check if it's a devastate transaction (based on data field)
		if len(output.Data) > 0 && string(output.Data) == "devastate" {
			return "devastate"
		}
	}
	return "normal"
}

// GetRecentBlocks returns the most recent N blocks
func (m *PotMonitor) GetRecentBlocks(count int) ([]model.BlockInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	currentHeight := m.worker.GetChainReader().GetCurrentHeight()
	if count <= 0 {
		count = 10
	}

	blocks := make([]model.BlockInfo, 0, count)
	startHeight := currentHeight
	if currentHeight >= uint64(count) {
		startHeight = currentHeight - uint64(count) + 1
	} else {
		startHeight = 1 // Skip genesis block at height 0
	}

	for h := startHeight; h <= currentHeight; h++ {
		block, err := m.worker.GetChainReader().GetByHeight(h)
		if err != nil {
			m.log.Warnf("Failed to get block at height %d: %v", h, err)
			continue
		}

		if block == nil || block.GetHeader() == nil {
			continue
		}

		header := block.GetHeader()
		blockInfo := model.BlockInfo{
			Height:    header.Height,
			Hash:      hexutil.Encode(header.Hash()),
			Timestamp: header.Timestamp.Unix(),
			TxCount:   len(block.GetTxs()),
			Size:      estimateBlockSize(block),
			Miner:     header.PeerId,
		}
		blocks = append(blocks, blockInfo)
	}

	return blocks, nil
}

// GetBlockByHeight returns detailed information about a specific block
func (m *PotMonitor) GetBlockByHeight(height uint64) (*model.BlockDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	block, err := m.worker.GetChainReader().GetByHeight(height)
	if err != nil {
		return nil, fmt.Errorf("block not found at height %d: %w", height, err)
	}

	if block == nil || block.GetHeader() == nil {
		return nil, fmt.Errorf("block or header is nil at height %d", height)
	}

	header := block.GetHeader()
	txs := block.GetTxs()

	// Convert transactions to hash strings
	txHashes := make([]string, 0, len(txs))
	for _, tx := range txs {
		if tx != nil {
			// Get RawTx data from Tx
			rawTx := tx.GetRawTxData()
			if rawTx != nil {
				txHashes = append(txHashes, hexutil.Encode(rawTx.Txid[:]))
			}
		}
	}

	// Convert uncle hashes
	uncleHashes := make([]string, 0, len(header.UncleHash))
	for _, uncleHash := range header.UncleHash {
		uncleHashes = append(uncleHashes, hexutil.Encode(uncleHash))
	}

	detail := &model.BlockDetail{
		Height:          header.Height,
		Hash:            hexutil.Encode(header.Hash()),
		ParentHash:      hexutil.Encode(header.ParentHash),
		Timestamp:       header.Timestamp.Unix(),
		TxCount:         len(txs),
		Size:            estimateBlockSize(block),
		Miner:           header.PeerId,
		Difficulty:      hexutil.EncodeBig(header.Difficulty),
		Nonce:           header.Nonce,
		Mixdigest:       hexutil.Encode(header.Mixdigest),
		UncleHashes:     uncleHashes,
		Transactions:    txHashes,
		CommitteePubkey: hexutil.Encode(header.CommiteePubkey),
	}

	return detail, nil
}

// estimateBlockSize estimates the size of a block in bytes
func estimateBlockSize(block *types.Block) int {
	if block == nil {
		return 0
	}
	// Rough estimate: header size + transactions
	size := 500 // Base header size estimate
	for _, tx := range block.GetTxs() {
		if tx != nil {
			// Each transaction roughly 200 bytes base + data
			size += 200 + len(tx.Data)
		}
	}
	return size
}
