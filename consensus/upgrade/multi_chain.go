package upgrade

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// MultiChainManager 多链管理器
type MultiChainManager struct {
	mainChain       *ChainState
	candidateChains map[string]*CandidateChainState // candidateID -> 候选链状态

	storage storage.MultiChainStorage
	log     *logrus.Entry
	mu      sync.RWMutex
}

// CandidateChainState 候选链状态
type CandidateChainState struct {
	*ChainState
	CandidateID string
	ForkPoint   uint64
	Active      bool
}

// NewMultiChainManager 创建多链管理器
func NewMultiChainManager(
	mainConsensus model.Consensus,
	multiChainStorage storage.MultiChainStorage,
	log *logrus.Entry,
) *MultiChainManager {
	return &MultiChainManager{
		mainChain: &ChainState{
			ConsensusID:   0, // 默认共识 ID
			CurrentHeight: 0,
			Consensus:     mainConsensus,
		},
		candidateChains: make(map[string]*CandidateChainState),
		storage:         multiChainStorage,
		log:             log,
	}
}

// StartCandidateChain 启动候选链
func (mcm *MultiChainManager) StartCandidateChain(
	candidateID string,
	forkHeight uint64,
	newConsensus model.Consensus,
) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	mcm.log.WithFields(logrus.Fields{
		"candidate_id": candidateID,
		"fork_height":  forkHeight,
	}).Info("Starting candidate chain")

	// 检查候选链是否已存在
	if _, exists := mcm.candidateChains[candidateID]; exists {
		mcm.log.WithField("candidate_id", candidateID).Error("Candidate chain already exists")
		return fmt.Errorf("candidate chain %s already exists", candidateID)
	}

	// 获取分叉点状态
	mcm.log.WithField("fork_height", forkHeight).Debug("Fetching fork block from main chain")
	forkBlock, err := mcm.storage.GetMainBlock(forkHeight)
	if err != nil {
		mcm.log.WithError(err).WithField("fork_height", forkHeight).Error("Failed to get fork block")
		return fmt.Errorf("failed to get fork block: %w", err)
	}

	var forkHash types.TxHash
	if forkBlock != nil && forkBlock.Hash() != nil {
		copy(forkHash[:], forkBlock.Hash())
	}

	// 创建候选链状态
	mcm.candidateChains[candidateID] = &CandidateChainState{
		ChainState: &ChainState{
			ConsensusID:     int64(len(mcm.candidateChains) + 1), // 动态分配 ID
			CurrentHeight:   forkHeight,
			LatestBlockHash: forkHash,
			Consensus:       newConsensus,
		},
		CandidateID: candidateID,
		ForkPoint:   forkHeight,
		Active:      true,
	}

	mcm.log.WithFields(logrus.Fields{
		"candidate_id": candidateID,
		"fork_height":  forkHeight,
	}).Info("Started candidate chain")

	return nil
}

// ProcessMainChainBlock 处理主链区块
func (mcm *MultiChainManager) ProcessMainChainBlock(block *types.Block) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	mcm.log.WithField("height", block.Header.Height).Trace("Processing main chain block")

	// 验证区块
	if err := mcm.validateBlock(mcm.mainChain, block); err != nil {
		mcm.log.WithError(err).WithField("height", block.Header.Height).Warn("Main chain block validation failed")
		return fmt.Errorf("invalid main chain block: %w", err)
	}

	// 存储区块
	if err := mcm.storage.StoreMainBlock(block); err != nil {
		mcm.log.WithError(err).WithField("height", block.Header.Height).Error("Failed to store main block")
		return fmt.Errorf("failed to store main block: %w", err)
	}

	// 更新状态
	mcm.mainChain.CurrentHeight = block.Header.Height
	var blockHash types.TxHash
	if block.Hash() != nil {
		copy(blockHash[:], block.Hash())
	}
	mcm.mainChain.LatestBlockHash = blockHash
	mcm.log.WithField("height", block.Header.Height).Debug("Main chain state updated")

	// 同步交易到所有活跃的候选链
	activeCount := 0
	for candidateID, candidate := range mcm.candidateChains {
		if candidate.Active {
			activeCount++
			if err := mcm.syncTransactionsToCandidate(candidateID, block); err != nil {
				mcm.log.WithError(err).WithField("candidate_id", candidateID).
					Warn("Failed to sync transactions to candidate chain")
			}
		}
	}
	if activeCount > 0 {
		mcm.log.WithFields(logrus.Fields{
			"height":            block.Header.Height,
			"active_candidates": activeCount,
		}).Debug("Synced transactions to active candidate chains")
	}

	return nil
}

// syncTransactionsToCandidate 同步交易到候选链
func (mcm *MultiChainManager) syncTransactionsToCandidate(candidateID string, mainBlock *types.Block) error {
	candidate, exists := mcm.candidateChains[candidateID]
	if !exists || !candidate.Active {
		return nil
	}

	// 过滤掉升级相关交易 (不在候选链上执行)
	txs := mcm.filterNormalTransactions(mainBlock.Txs)

	// 在候选链上处理这些交易
	// 注意: 这里需要调用新共识的出块逻辑
	// 具体实现取决于新共识的接口

	mcm.log.WithFields(logrus.Fields{
		"candidate_id": candidateID,
		"main_height":  mainBlock.Header.Height,
		"tx_count":     len(txs),
	}).Debug("Synced transactions to candidate chain")

	return nil
}

// ProcessCandidateBlock 处理候选链区块
func (mcm *MultiChainManager) ProcessCandidateBlock(candidateID string, block *types.Block) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	mcm.log.WithFields(logrus.Fields{
		"candidate_id": candidateID,
		"height":       block.Header.Height,
	}).Trace("Processing candidate chain block")

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		mcm.log.WithField("candidate_id", candidateID).Warn("Candidate chain not found")
		return fmt.Errorf("candidate chain %s not found", candidateID)
	}

	if !candidate.Active {
		mcm.log.WithFields(logrus.Fields{
			"candidate_id": candidateID,
			"height":       block.Header.Height,
		}).Warn("Candidate chain not active, skipping block")
		return fmt.Errorf("candidate chain %s not active", candidateID)
	}

	// 验证区块
	if err := mcm.validateBlock(candidate.ChainState, block); err != nil {
		mcm.log.WithError(err).WithFields(logrus.Fields{
			"candidate_id": candidateID,
			"height":       block.Header.Height,
		}).Warn("Candidate block validation failed")
		return fmt.Errorf("invalid candidate block: %w", err)
	}

	// 存储候选链区块
	if err := mcm.storage.StoreCandidateBlock(candidateID, block); err != nil {
		mcm.log.WithError(err).WithFields(logrus.Fields{
			"candidate_id": candidateID,
			"height":       block.Header.Height,
		}).Error("Failed to store candidate block")
		return fmt.Errorf("failed to store candidate block: %w", err)
	}

	// 更新状态
	candidate.CurrentHeight = block.Header.Height
	var blockHash types.TxHash
	if block.Hash() != nil {
		copy(blockHash[:], block.Hash())
	}
	candidate.LatestBlockHash = blockHash
	mcm.log.WithFields(logrus.Fields{
		"candidate_id": candidateID,
		"height":       block.Header.Height,
	}).Debug("Candidate chain state updated")

	return nil
}

// validateBlock 验证区块
func (mcm *MultiChainManager) validateBlock(chain *ChainState, block *types.Block) error {
	// 高度检查
	if block.Header.Height != chain.CurrentHeight+1 {
		return fmt.Errorf("invalid block height: expected %d, got %d",
			chain.CurrentHeight+1, block.Header.Height)
	}

	// 父哈希检查
	if chain.CurrentHeight > 0 && block.Header.ParentHash != nil {
		if !bytes.Equal(block.Header.ParentHash, chain.LatestBlockHash[:]) {
			return fmt.Errorf("invalid parent hash")
		}
	}

	// 使用对应共识验证区块
	// (这里简化处理,实际需要调用共识的验证接口)

	return nil
}

// MergeCandidateChain 合并候选链到主链
func (mcm *MultiChainManager) MergeCandidateChain(candidateID string, switchHeight uint64) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		return fmt.Errorf("candidate chain %s not found", candidateID)
	}

	if !candidate.Active {
		return fmt.Errorf("candidate chain %s not active", candidateID)
	}

	mcm.log.WithFields(logrus.Fields{
		"candidate_id":  candidateID,
		"fork_point":    candidate.ForkPoint,
		"switch_height": switchHeight,
	}).Info("Merging candidate chain to main chain")

	// 获取候选链从分叉点到切换点的所有区块
	candidateBlocks, err := mcm.storage.GetCandidateBlocks(candidateID, candidate.ForkPoint, switchHeight)
	if err != nil {
		return fmt.Errorf("failed to get candidate blocks: %w", err)
	}

	// 删除主链上从分叉点之后的区块 (它们将被候选链替换)
	if err := mcm.storage.DeleteMainBlocksFrom(candidate.ForkPoint + 1); err != nil {
		return fmt.Errorf("failed to delete old main blocks: %w", err)
	}

	// 将候选链区块标记为主链区块
	for _, block := range candidateBlocks {
		if err := mcm.storage.PromoteToMainChain(candidateID, block); err != nil {
			return fmt.Errorf("failed to promote block %d: %w", block.Header.Height, err)
		}
	}

	// 更新主链状态
	if len(candidateBlocks) > 0 {
		lastBlock := candidateBlocks[len(candidateBlocks)-1]
		mcm.mainChain.ConsensusID = candidate.ConsensusID
		mcm.mainChain.CurrentHeight = switchHeight
		var blockHash types.TxHash
		if lastBlock.Hash() != nil {
			copy(blockHash[:], lastBlock.Hash())
		}
		mcm.mainChain.LatestBlockHash = blockHash
		mcm.mainChain.Consensus = candidate.Consensus
	}

	// 清理候选链
	delete(mcm.candidateChains, candidateID)

	mcm.log.WithField("candidate_id", candidateID).Info("Candidate chain merged successfully")

	return nil
}

// RollbackCandidateChain 回退候选链
func (mcm *MultiChainManager) RollbackCandidateChain(candidateID string) error {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		return fmt.Errorf("candidate chain %s not found", candidateID)
	}

	mcm.log.WithField("candidate_id", candidateID).Info("Rolling back candidate chain")

	// 停止候选链共识
	if candidate.Consensus != nil {
		candidate.Consensus.Stop()
	}

	// 删除候选链数据
	if err := mcm.storage.DeleteCandidateChain(candidateID); err != nil {
		mcm.log.WithError(err).WithField("candidate_id", candidateID).
			Warn("Failed to delete candidate blocks")
	}

	// 清理状态
	delete(mcm.candidateChains, candidateID)

	mcm.log.WithField("candidate_id", candidateID).Info("Candidate chain rolled back successfully")

	return nil
}

// GetMainChainHeight 获取主链高度
func (mcm *MultiChainManager) GetMainChainHeight() uint64 {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()
	return mcm.mainChain.CurrentHeight
}

// GetCandidateChainHeight 获取候选链高度
func (mcm *MultiChainManager) GetCandidateChainHeight(candidateID string) uint64 {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		return 0
	}
	return candidate.CurrentHeight
}

// IsCandidateActive 候选链是否活跃
func (mcm *MultiChainManager) IsCandidateActive(candidateID string) bool {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		return false
	}
	return candidate.Active
}

// GetForkPoint 获取候选链分叉点高度
func (mcm *MultiChainManager) GetForkPoint(candidateID string) uint64 {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		return 0
	}
	return candidate.ForkPoint
}

// GetMainConsensus 获取主链共识
func (mcm *MultiChainManager) GetMainConsensus() model.Consensus {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()
	return mcm.mainChain.Consensus
}

// GetCandidateConsensus 获取候选链共识
func (mcm *MultiChainManager) GetCandidateConsensus(candidateID string) model.Consensus {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	candidate, exists := mcm.candidateChains[candidateID]
	if !exists {
		return nil
	}
	return candidate.Consensus
}

// ListCandidateChains 列出所有候选链ID
func (mcm *MultiChainManager) ListCandidateChains() []string {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	chains := make([]string, 0, len(mcm.candidateChains))
	for candidateID := range mcm.candidateChains {
		chains = append(chains, candidateID)
	}
	return chains
}

// GetCandidateState 获取候选链状态
func (mcm *MultiChainManager) GetCandidateState(candidateID string) *CandidateChainState {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	return mcm.candidateChains[candidateID]
}

// filterNormalTransactions 过滤普通交易 (排除升级交易)
func (mcm *MultiChainManager) filterNormalTransactions(txs []*types.Tx) []*types.Tx {
	filtered := make([]*types.Tx, 0, len(txs))
	for _, tx := range txs {
		// 简化实现：假设所有交易都是普通交易
		// 实际需要检查交易类型，排除 UPGRADE 和 LOCK 类型
		filtered = append(filtered, tx)
	}
	return filtered
}
