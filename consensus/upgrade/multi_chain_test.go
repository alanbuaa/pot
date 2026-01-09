package upgrade

import (
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// Helper function to create test block
func newTestBlock(height uint64) *types.Block {
	return &types.Block{
		Header: &types.Header{
			Height:     height,
			Timestamp:  time.Now(),
			Difficulty: big.NewInt(1),
			PoTProof:   [][]byte{{1}, {2}},
			Mixdigest:  []byte{0},
			ParentHash: []byte{0},
			TxHash:     []byte{0},
			PublicKey:  []byte{0},
		},
	}
}

// Helper function to create block with specific parent hash
func newTestBlockWithParent(height uint64, parentHash []byte) *types.Block {
	return &types.Block{
		Header: &types.Header{
			Height:     height,
			Timestamp:  time.Now(),
			Difficulty: big.NewInt(1),
			PoTProof:   [][]byte{{1}, {2}},
			Mixdigest:  []byte{0},
			ParentHash: parentHash,
			TxHash:     []byte{0},
			PublicKey:  []byte{0},
		},
	}
}

// Mock Consensus implementation
type mockConsensus struct {
	mu          sync.Mutex
	blocks      []*types.Block
	processErr  error
	shouldPanic bool
	consensusID int64
}

func newMockConsensus() *mockConsensus {
	return &mockConsensus{
		blocks:      make([]*types.Block, 0),
		consensusID: 0,
	}
}

func (m *mockConsensus) ProcessBlock(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldPanic {
		panic("test panic")
	}

	if m.processErr != nil {
		return m.processErr
	}

	m.blocks = append(m.blocks, block)
	return nil
}

func (m *mockConsensus) GetRequestEntrance() chan<- *pb.Request {
	return make(chan *pb.Request)
}

func (m *mockConsensus) GetMsgByteEntrance() chan<- []byte {
	return make(chan []byte)
}

func (m *mockConsensus) Stop() {}

func (m *mockConsensus) GetConsensusID() int64 {
	return m.consensusID
}

func (m *mockConsensus) GetConsensusType() string {
	return "mock"
}

func (m *mockConsensus) VerifyBlock(block []byte, proof []byte) bool {
	return true
}

func (m *mockConsensus) UpdateExternalStatus(status model.ExternalStatus) {}

func (m *mockConsensus) NewEpochConfirmation(epoch int64, proof []byte, committee []string) {}

func (m *mockConsensus) RequestLatestBlock(epoch int64, proof []byte, committee []string) {}

func (m *mockConsensus) GetWeight(nid int64) float64 {
	return 1.0
}

func (m *mockConsensus) GetMaxAdversaryWeight() float64 {
	return 0.33
}

func (m *mockConsensus) GetBlockCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.blocks)
}

func (m *mockConsensus) GetBlocks() []*types.Block {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*types.Block, len(m.blocks))
	copy(result, m.blocks)
	return result
}

func (m *mockConsensus) SetProcessError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processErr = err
}

// Mock MultiChainStorage implementation
type mockMultiChainStorage struct {
	mu              sync.RWMutex
	mainBlocks      map[uint64]*types.Block
	candidateBlocks map[string]map[uint64]*types.Block // candidateID -> height -> block
}

func newMockMultiChainStorage() *mockMultiChainStorage {
	return &mockMultiChainStorage{
		mainBlocks:      make(map[uint64]*types.Block),
		candidateBlocks: make(map[string]map[uint64]*types.Block),
	}
}

func (m *mockMultiChainStorage) StoreMainBlock(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	height := block.GetHeader().Height
	m.mainBlocks[height] = block
	return nil
}

func (m *mockMultiChainStorage) GetMainBlock(height uint64) (*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	block, ok := m.mainBlocks[height]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockMultiChainStorage) DeleteMainBlocksFrom(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for h := range m.mainBlocks {
		if h >= height {
			delete(m.mainBlocks, h)
		}
	}
	return nil
}

func (m *mockMultiChainStorage) StoreCandidateBlock(candidateID string, block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.candidateBlocks[candidateID]; !ok {
		m.candidateBlocks[candidateID] = make(map[uint64]*types.Block)
	}
	height := block.GetHeader().Height
	m.candidateBlocks[candidateID][height] = block
	return nil
}

func (m *mockMultiChainStorage) GetCandidateBlock(candidateID string, height uint64) (*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	candidateChain, ok := m.candidateBlocks[candidateID]
	if !ok {
		return nil, fmt.Errorf("candidate chain not found")
	}
	block, ok := candidateChain[height]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockMultiChainStorage) GetCandidateBlocks(candidateID string, from, to uint64) ([]*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	candidateChain, ok := m.candidateBlocks[candidateID]
	if !ok {
		return nil, fmt.Errorf("candidate chain not found")
	}
	blocks := make([]*types.Block, 0)
	for h := from; h <= to; h++ {
		if block, ok := candidateChain[h]; ok {
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func (m *mockMultiChainStorage) DeleteCandidateChain(candidateID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.candidateBlocks, candidateID)
	return nil
}

func (m *mockMultiChainStorage) DeleteCandidateBlocksFrom(candidateID string, forkPoint uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	candidateChain, ok := m.candidateBlocks[candidateID]
	if !ok {
		return nil
	}
	for h := range candidateChain {
		if h >= forkPoint {
			delete(candidateChain, h)
		}
	}
	return nil
}

func (m *mockMultiChainStorage) ListCandidateChains() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chains := make([]string, 0, len(m.candidateBlocks))
	for candidateID := range m.candidateBlocks {
		chains = append(chains, candidateID)
	}
	return chains, nil
}

func (m *mockMultiChainStorage) PromoteToMainChain(candidateID string, block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	height := block.GetHeader().Height
	m.mainBlocks[height] = block
	return nil
}

func (m *mockMultiChainStorage) Close() error {
	return nil
}

func TestNewMultiChainManager(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	if manager == nil {
		t.Fatal("NewMultiChainManager returned nil")
	}

	if len(manager.ListCandidateChains()) != 0 {
		t.Error("no candidate chains should be active by default")
	}
}

func TestMultiChainManager_StartCandidateChain(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	candidateConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 先保存一个主链区块作为分叉点（高度 1）
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动候选链
	candidateID := "pow-upgrade-1"
	err := manager.StartCandidateChain(candidateID, 1, candidateConsensus)
	if err != nil {
		t.Fatalf("StartCandidateChain failed: %v", err)
	}

	if !manager.IsCandidateActive(candidateID) {
		t.Error("candidate chain should be active after start")
	}

	// 尝试再次启动相同的候选链应该失败
	err = manager.StartCandidateChain(candidateID, 1, candidateConsensus)
	if err == nil {
		t.Error("starting same candidate chain twice should fail")
	}
}

func TestMultiChainManager_MultipleCandidateChains(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 保存分叉点区块
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动多个候选链
	candidates := []string{"pow-upgrade-1", "hotstuff-upgrade-1", "custom-upgrade-1"}
	for _, candidateID := range candidates {
		consensus := newMockConsensus()
		err := manager.StartCandidateChain(candidateID, 1, consensus)
		if err != nil {
			t.Fatalf("Failed to start candidate %s: %v", candidateID, err)
		}
	}

	// 验证所有候选链都在运行
	activeCandidates := manager.ListCandidateChains()
	if len(activeCandidates) != len(candidates) {
		t.Errorf("Expected %d active candidates, got %d", len(candidates), len(activeCandidates))
	}

	// 验证每个候选链都是活跃的
	for _, candidateID := range candidates {
		if !manager.IsCandidateActive(candidateID) {
			t.Errorf("Candidate %s should be active", candidateID)
		}
	}
}

func TestMultiChainManager_ProcessMainChainBlock(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 从高度 1 开始
	block := newTestBlock(1)

	err := manager.ProcessMainChainBlock(block)
	if err != nil {
		t.Fatalf("ProcessMainChainBlock failed: %v", err)
	}

	// 验证区块已存储
	storedBlock, err := storage.GetMainBlock(1)
	if err != nil {
		t.Fatalf("Failed to get stored block: %v", err)
	}

	if storedBlock.GetHeader().Height != 1 {
		t.Errorf("Expected height 1, got %d", storedBlock.GetHeader().Height)
	}
}

func TestMultiChainManager_ProcessCandidateBlock(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	candidateConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 保存分叉点区块
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动候选链
	candidateID := "test-candidate"
	err := manager.StartCandidateChain(candidateID, 1, candidateConsensus)
	if err != nil {
		t.Fatalf("StartCandidateChain failed: %v", err)
	}

	// 处理候选链区块 - 使用正确的 parent hash
	block := newTestBlockWithParent(2, forkBlock.Hash())
	err = manager.ProcessCandidateBlock(candidateID, block)
	if err != nil {
		t.Fatalf("ProcessCandidateBlock failed: %v", err)
	}

	// 验证候选链高度更新
	if manager.GetCandidateChainHeight(candidateID) != 2 {
		t.Errorf("Expected candidate height 2, got %d", manager.GetCandidateChainHeight(candidateID))
	}
}

func TestMultiChainManager_MergeCandidateChain(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	candidateConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 保存主链区块
	for i := uint64(1); i <= 5; i++ {
		block := newTestBlock(i)
		storage.StoreMainBlock(block)
	}

	// 启动候选链（从高度3分叉）
	candidateID := "merge-test"
	err := manager.StartCandidateChain(candidateID, 3, candidateConsensus)
	if err != nil {
		t.Fatalf("StartCandidateChain failed: %v", err)
	}

	// 在候选链上创建区块
	for i := uint64(3); i <= 7; i++ {
		block := newTestBlock(i)
		storage.StoreCandidateBlock(candidateID, block)
	}

	// 执行合并
	err = manager.MergeCandidateChain(candidateID, 7)
	if err != nil {
		t.Fatalf("MergeCandidateChain failed: %v", err)
	}

	// 验证候选链已被删除
	if manager.IsCandidateActive(candidateID) {
		t.Error("Candidate chain should not be active after merge")
	}

	// 验证主链高度已更新
	if manager.GetMainChainHeight() != 7 {
		t.Errorf("Expected main chain height 7, got %d", manager.GetMainChainHeight())
	}
}

func TestMultiChainManager_RollbackCandidateChain(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	candidateConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 保存分叉点区块
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动候选链
	candidateID := "rollback-test"
	err := manager.StartCandidateChain(candidateID, 1, candidateConsensus)
	if err != nil {
		t.Fatalf("StartCandidateChain failed: %v", err)
	}

	// 在候选链上创建一些区块
	for i := uint64(2); i <= 5; i++ {
		block := newTestBlock(i)
		storage.StoreCandidateBlock(candidateID, block)
	}

	// 执行回退
	err = manager.RollbackCandidateChain(candidateID)
	if err != nil {
		t.Fatalf("RollbackCandidateChain failed: %v", err)
	}

	// 验证候选链已被删除
	if manager.IsCandidateActive(candidateID) {
		t.Error("Candidate chain should not be active after rollback")
	}

	// 验证候选链列表中不存在该候选链
	chains := manager.ListCandidateChains()
	for _, chain := range chains {
		if chain == candidateID {
			t.Error("Candidate chain should not be in list after rollback")
		}
	}
}

func TestMultiChainManager_GetCandidateState(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	candidateConsensus := newMockConsensus()
	storage := newMockMultiChainStorage()

	manager := NewMultiChainManager(mainConsensus, storage, log)

	// 保存分叉点区块
	forkBlock := newTestBlock(10)
	storage.StoreMainBlock(forkBlock)

	// 启动候选链
	candidateID := "state-test"
	err := manager.StartCandidateChain(candidateID, 10, candidateConsensus)
	if err != nil {
		t.Fatalf("StartCandidateChain failed: %v", err)
	}

	// 获取候选链状态
	state := manager.GetCandidateState(candidateID)
	if state == nil {
		t.Fatal("GetCandidateState returned nil")
	}

	if state.CandidateID != candidateID {
		t.Errorf("Expected candidateID %s, got %s", candidateID, state.CandidateID)
	}

	if state.ForkPoint != 10 {
		t.Errorf("Expected fork point 10, got %d", state.ForkPoint)
	}

	if !state.Active {
		t.Error("Candidate chain should be active")
	}
}
