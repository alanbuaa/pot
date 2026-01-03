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

// DualChainManager 双链管理器
type DualChainManager struct {
	mainChain    *ChainState
	preexecChain *ChainState

	forkPoint uint64
	active    bool

	storage storage.DualChainStorage
	log     *logrus.Entry
	mu      sync.RWMutex
}

// NewDualChainManager 创建双链管理器
func NewDualChainManager(
	mainConsensus model.Consensus,
	dualChainStorage storage.DualChainStorage,
	log *logrus.Entry,
) *DualChainManager {
	return &DualChainManager{
		mainChain: &ChainState{
			ConsensusID:   0, // 默认共识 ID
			CurrentHeight: 0,
			Consensus:     mainConsensus,
		},
		storage: dualChainStorage,
		log:     log,
		active:  false,
	}
}

// StartPreexecution 启动预执行链
func (dcm *DualChainManager) StartPreexecution(
	forkHeight uint64,
	newConsensus model.Consensus,
) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if dcm.active {
		return fmt.Errorf("preexecution already active")
	}

	// 获取分叉点状态
	forkBlock, err := dcm.storage.GetMainBlock(forkHeight)
	if err != nil {
		return fmt.Errorf("failed to get fork block: %w", err)
	}

	var forkHash types.TxHash
	if forkBlock != nil && forkBlock.Hash() != nil {
		copy(forkHash[:], forkBlock.Hash())
	}

	// 创建预执行链状态
	dcm.preexecChain = &ChainState{
		ConsensusID:     1, // 新共识 ID
		CurrentHeight:   forkHeight,
		LatestBlockHash: forkHash,
		Consensus:       newConsensus,
	}

	dcm.forkPoint = forkHeight
	dcm.active = true

	dcm.log.WithFields(logrus.Fields{
		"fork_height": forkHeight,
	}).Info("Started preexecution chain")

	return nil
}

// ProcessMainChainBlock 处理主链区块
func (dcm *DualChainManager) ProcessMainChainBlock(block *types.Block) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	// 验证区块
	if err := dcm.validateBlock(dcm.mainChain, block); err != nil {
		return fmt.Errorf("invalid main chain block: %w", err)
	}

	// 存储区块
	if err := dcm.storage.StoreMainBlock(block); err != nil {
		return fmt.Errorf("failed to store main block: %w", err)
	}

	// 更新状态
	dcm.mainChain.CurrentHeight = block.Header.Height
	var blockHash types.TxHash
	if block.Hash() != nil {
		copy(blockHash[:], block.Hash())
	}
	dcm.mainChain.LatestBlockHash = blockHash

	// 如果预执行链活跃,同步交易
	if dcm.active {
		if err := dcm.syncTransactionsToPreexec(block); err != nil {
			dcm.log.WithError(err).Warn("Failed to sync transactions to preexec chain")
		}
	}

	return nil
}

// syncTransactionsToPreexec 同步交易到预执行链
func (dcm *DualChainManager) syncTransactionsToPreexec(mainBlock *types.Block) error {
	if !dcm.active || dcm.preexecChain == nil {
		return nil
	}

	// 过滤掉升级相关交易 (不在预执行链上执行)
	txs := dcm.filterNormalTransactions(mainBlock.Txs)

	// 在预执行链上处理这些交易
	// 注意: 这里需要调用新共识的出块逻辑
	// 具体实现取决于新共识的接口

	dcm.log.WithFields(logrus.Fields{
		"main_height": mainBlock.Header.Height,
		"tx_count":    len(txs),
	}).Debug("Synced transactions to preexec chain")

	return nil
}

// ProcessPreexecBlock 处理预执行链区块
func (dcm *DualChainManager) ProcessPreexecBlock(block *types.Block) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if !dcm.active {
		return fmt.Errorf("preexecution not active")
	}

	// 验证区块
	if err := dcm.validateBlock(dcm.preexecChain, block); err != nil {
		return fmt.Errorf("invalid preexec block: %w", err)
	}

	// 存储预执行区块
	if err := dcm.storage.StorePreexecBlock(block); err != nil {
		return fmt.Errorf("failed to store preexec block: %w", err)
	}

	// 更新状态
	dcm.preexecChain.CurrentHeight = block.Header.Height
	var blockHash types.TxHash
	if block.Hash() != nil {
		copy(blockHash[:], block.Hash())
	}
	dcm.preexecChain.LatestBlockHash = blockHash

	return nil
}

// validateBlock 验证区块
func (dcm *DualChainManager) validateBlock(chain *ChainState, block *types.Block) error {
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

// MergePreexecChain 合并预执行链到主链
func (dcm *DualChainManager) MergePreexecChain(switchHeight uint64) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if !dcm.active {
		return fmt.Errorf("no active preexecution")
	}

	dcm.log.WithFields(logrus.Fields{
		"fork_point":    dcm.forkPoint,
		"switch_height": switchHeight,
	}).Info("Merging preexec chain to main chain")

	// 获取预执行链从分叉点到切换点的所有区块
	preexecBlocks, err := dcm.storage.GetPreexecBlocks(dcm.forkPoint, switchHeight)
	if err != nil {
		return fmt.Errorf("failed to get preexec blocks: %w", err)
	}

	// 删除主链上从分叉点之后的区块 (它们将被预执行链替换)
	if err := dcm.storage.DeleteMainBlocksFrom(dcm.forkPoint + 1); err != nil {
		return fmt.Errorf("failed to delete old main blocks: %w", err)
	}

	// 将预执行链区块标记为主链区块
	for _, block := range preexecBlocks {
		if err := dcm.storage.PromoteToMainChain(block); err != nil {
			return fmt.Errorf("failed to promote block %d: %w", block.Header.Height, err)
		}
	}

	// 更新主链状态
	if len(preexecBlocks) > 0 {
		lastBlock := preexecBlocks[len(preexecBlocks)-1]
		dcm.mainChain.ConsensusID = dcm.preexecChain.ConsensusID
		dcm.mainChain.CurrentHeight = switchHeight
		var blockHash types.TxHash
		if lastBlock.Hash() != nil {
			copy(blockHash[:], lastBlock.Hash())
		}
		dcm.mainChain.LatestBlockHash = blockHash
		dcm.mainChain.Consensus = dcm.preexecChain.Consensus
	}

	// 清理预执行链
	dcm.preexecChain = nil
	dcm.active = false

	dcm.log.Info("Preexec chain merged successfully")

	return nil
}

// RollbackPreexecution 回退预执行
func (dcm *DualChainManager) RollbackPreexecution() error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if !dcm.active {
		return fmt.Errorf("no active preexecution to rollback")
	}

	dcm.log.Info("Rolling back preexecution")

	// 停止预执行链共识
	if dcm.preexecChain != nil && dcm.preexecChain.Consensus != nil {
		dcm.preexecChain.Consensus.Stop()
	}

	// 删除预执行链数据
	if err := dcm.storage.DeletePreexecBlocks(dcm.forkPoint); err != nil {
		dcm.log.WithError(err).Warn("Failed to delete preexec blocks")
	}

	// 清理状态
	dcm.preexecChain = nil
	dcm.active = false
	dcm.forkPoint = 0

	dcm.log.Info("Preexecution rolled back successfully")

	return nil
}

// GetMainChainHeight 获取主链高度
func (dcm *DualChainManager) GetMainChainHeight() uint64 {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()
	return dcm.mainChain.CurrentHeight
}

// GetPreexecChainHeight 获取预执行链高度
func (dcm *DualChainManager) GetPreexecChainHeight() uint64 {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	if dcm.preexecChain == nil {
		return 0
	}
	return dcm.preexecChain.CurrentHeight
}

// IsPreexecActive 预执行是否活跃
func (dcm *DualChainManager) IsPreexecActive() bool {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()
	return dcm.active
}

// GetForkPoint 获取分叉点高度
func (dcm *DualChainManager) GetForkPoint() uint64 {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()
	return dcm.forkPoint
}

// GetMainConsensus 获取主链共识
func (dcm *DualChainManager) GetMainConsensus() model.Consensus {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()
	return dcm.mainChain.Consensus
}

// GetPreexecConsensus 获取预执行链共识
func (dcm *DualChainManager) GetPreexecConsensus() model.Consensus {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()

	if dcm.preexecChain == nil {
		return nil
	}
	return dcm.preexecChain.Consensus
}

// filterNormalTransactions 过滤普通交易 (排除升级交易)
func (dcm *DualChainManager) filterNormalTransactions(txs []*types.Tx) []*types.Tx {
	filtered := make([]*types.Tx, 0, len(txs))
	for _, tx := range txs {
		// 简化实现：假设所有交易都是普通交易
		// 实际需要检查交易类型，排除 UPGRADE 和 LOCK 类型
		filtered = append(filtered, tx)
	}
	return filtered
}
