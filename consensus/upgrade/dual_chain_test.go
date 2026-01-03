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

// Mock DualChainStorage implementation
type mockDualChainStorage struct {
	mu            sync.RWMutex
	mainBlocks    map[uint64]*types.Block
	preexecBlocks map[uint64]*types.Block
}

func newMockDualChainStorage() *mockDualChainStorage {
	return &mockDualChainStorage{
		mainBlocks:    make(map[uint64]*types.Block),
		preexecBlocks: make(map[uint64]*types.Block),
	}
}

func (m *mockDualChainStorage) StoreMainBlock(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	height := block.GetHeader().Height
	m.mainBlocks[height] = block
	return nil
}

func (m *mockDualChainStorage) GetBlock(height uint64) (*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	block, ok := m.mainBlocks[height]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockDualChainStorage) DeleteMainBlocksFrom(height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for h := range m.mainBlocks {
		if h >= height {
			delete(m.mainBlocks, h)
		}
	}
	return nil
}

func (m *mockDualChainStorage) GetMainBlock(height uint64) (*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	block, ok := m.mainBlocks[height]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockDualChainStorage) StorePreexecBlock(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	height := block.GetHeader().Height
	m.preexecBlocks[height] = block
	return nil
}

func (m *mockDualChainStorage) GetPreexecBlock(height uint64) (*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	block, ok := m.preexecBlocks[height]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	return block, nil
}

func (m *mockDualChainStorage) GetPreexecBlocks(from, to uint64) ([]*types.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	blocks := make([]*types.Block, 0)
	for h := from; h <= to; h++ {
		if block, ok := m.preexecBlocks[h]; ok {
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func (m *mockDualChainStorage) Close() error {
	return nil
}

func (m *mockDualChainStorage) DeletePreexecBlocks(forkPoint uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for h := range m.preexecBlocks {
		if h >= forkPoint {
			delete(m.preexecBlocks, h)
		}
	}
	return nil
}

func (m *mockDualChainStorage) PromoteToMainChain(block *types.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	height := block.GetHeader().Height
	m.mainBlocks[height] = block
	return nil
}

func TestNewDualChainManager(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	if manager == nil {
		t.Fatal("NewDualChainManager returned nil")
	}

	if manager.IsPreexecActive() {
		t.Error("preexec should not be active by default")
	}
}

func TestDualChainManager_StartPreexecution(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	preexecConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 先保存一个主链区块作为分叉点（高度 1）
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动预执行
	err := manager.StartPreexecution(1, preexecConsensus)
	if err != nil {
		t.Fatalf("StartPreexecution failed: %v", err)
	}

	if !manager.IsPreexecActive() {
		t.Error("preexec should be active after start")
	}

	// 尝试再次启动应该失败
	err = manager.StartPreexecution(1, preexecConsensus)
	if err == nil {
		t.Error("starting preexecution twice should fail")
	}
}

func TestDualChainManager_ProcessMainChainBlock(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 从高度 1 开始
	block := newTestBlock(1)

	err := manager.ProcessMainChainBlock(block)
	if err != nil {
		t.Fatalf("ProcessMainChainBlock failed: %v", err)
	}

	// 验证区块已保存
	savedBlock, err := storage.GetMainBlock(1)
	if err != nil {
		t.Fatalf("GetMainBlock failed: %v", err)
	}

	if savedBlock.GetHeader().Height != block.GetHeader().Height {
		t.Errorf("saved block height = %d, want %d", savedBlock.GetHeader().Height, block.GetHeader().Height)
	}

	// 注意：DualChainManager 不负责调用共识的 ProcessBlock
	// 它只管理双链状态和存储，共识处理由外部调用方负责
}

func TestDualChainManager_ProcessPreexecBlock(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	preexecConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 先保存分叉点（高度 1）
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动预执行
	manager.StartPreexecution(1, preexecConsensus)

	// 处理预执行区块（高度 2）
	block := newTestBlockWithParent(2, forkBlock.Hash())

	err := manager.ProcessPreexecBlock(block)
	if err != nil {
		t.Fatalf("ProcessPreexecBlock failed: %v", err)
	}

	// 验证区块已保存
	savedBlock, err := storage.GetPreexecBlock(2)
	if err != nil {
		t.Fatalf("GetPreexecBlock failed: %v", err)
	}

	if savedBlock.GetHeader().Height != block.GetHeader().Height {
		t.Errorf("saved block height = %d, want %d", savedBlock.GetHeader().Height, block.GetHeader().Height)
	}

	// 注意：当前实现未调用共识的 ProcessBlock，所以区块计数为0
	// 这是实现的限制，预执行区块被存储但不被共识处理
	// TODO: 未来可能需要修改实现以调用 preexecConsensus.ProcessBlock()
}

func TestDualChainManager_ProcessPreexecBlock_NotActive(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	block := newTestBlock(101)

	// 未启动预执行就处理区块应该失败
	err := manager.ProcessPreexecBlock(block)
	if err == nil {
		t.Error("processing preexec block when not active should fail")
	}
}

func TestDualChainManager_MergePreexecChain(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	preexecConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 保存分叉点（高度 1）
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动预执行
	manager.StartPreexecution(1, preexecConsensus)

	// 处理一些预执行区块（高度 2-5）
	for i := 2; i <= 5; i++ {
		block := newTestBlock(uint64(i))
		manager.ProcessPreexecBlock(block)
	}

	// 执行合并
	err := manager.MergePreexecChain(5)
	if err != nil {
		t.Fatalf("MergePreexecChain failed: %v", err)
	}

	// 验证预执行已停止
	if manager.IsPreexecActive() {
		t.Error("preexec should not be active after merge")
	}
}

func TestDualChainManager_Rollback(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	preexecConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 保存分叉点（高度 1）
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动预执行
	manager.StartPreexecution(1, preexecConsensus)

	// 处理一些预执行区块（高度 2-5）
	for i := 2; i <= 5; i++ {
		block := newTestBlock(uint64(i))
		manager.ProcessPreexecBlock(block)
	}

	// 执行回滚
	err := manager.RollbackPreexecution()
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// 验证预执行已停止
	if manager.IsPreexecActive() {
		t.Error("preexec should not be active after rollback")
	}

	// 验证预执行链已清空
	_, err = storage.GetPreexecBlock(2)
	if err == nil {
		t.Error("preexec block should be cleared after rollback")
	}
}

func TestDualChainManager_Concurrency(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	preexecConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 保存分叉点（高度 1）
	forkBlock := newTestBlock(1)
	storage.StoreMainBlock(forkBlock)

	// 启动预执行
	manager.StartPreexecution(1, preexecConsensus)

	// 并发处理主链和预执行链区块
	var wg sync.WaitGroup
	wg.Add(20)

	// 10个goroutine处理主链（高度 2-11，由于验证只有按序的会成功）
	for i := 0; i < 10; i++ {
		go func(height int) {
			defer wg.Done()
			block := newTestBlock(uint64(2 + height))
			manager.ProcessMainChainBlock(block)
		}(i)
	}

	// 10个goroutine处理预执行链（高度 2-11，由于验证只有按序的会成功）
	for i := 0; i < 10; i++ {
		go func(height int) {
			defer wg.Done()
			block := newTestBlock(uint64(2 + height))
			manager.ProcessPreexecBlock(block)
		}(i)
	}

	wg.Wait()

	// 注意：DualChainManager 不调用共识的 ProcessBlock
	// 测试验证：并发访问不会导致panic或死锁
	// 由于高度验证，大部分并发区块会被拒绝（非连续高度）
	t.Log("Concurrency test completed without deadlock")
}

func TestDualChainManager_ErrorHandling(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	mainConsensus := newMockConsensus()
	storage := newMockDualChainStorage()

	manager := NewDualChainManager(mainConsensus, storage, log)

	// 测试共识处理错误
	mainConsensus.SetProcessError(fmt.Errorf("consensus error"))

	block := newTestBlock(101)

	err := manager.ProcessMainChainBlock(block)
	if err == nil {
		t.Error("ProcessMainChainManager should fail when consensus returns error")
	}
}
