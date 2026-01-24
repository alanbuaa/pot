package p2p

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzz136454872/upgradeable-consensus/config"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"google.golang.org/protobuf/proto"
)

// 用于生成唯一端口的全局计数器
var portCounter int32 = 10000

// mockNodeDiscoveryCallback 模拟节点发现回调
type mockNodeDiscoveryCallback struct {
	discoveredNodes map[int64]string
	lostNodes       map[int64]bool
}

func newMockCallback() *mockNodeDiscoveryCallback {
	return &mockNodeDiscoveryCallback{
		discoveredNodes: make(map[int64]string),
		lostNodes:       make(map[int64]bool),
	}
}

func (m *mockNodeDiscoveryCallback) OnNodeDiscovered(nodeID int64, address string) error {
	m.discoveredNodes[nodeID] = address
	return nil
}

func (m *mockNodeDiscoveryCallback) OnNodeLost(nodeID int64) error {
	m.lostNodes[nodeID] = true
	return nil
}

// createTestConfig 创建测试用配置
func createTestConfig(t *testing.T, nodeCount int, p2pType string) *config.Config {
	// 生成唯一的起始端口
	basePort := int(atomic.AddInt32(&portCounter, 100))

	cfg := &config.Config{
		Network: &config.NetworkConfig{
			P2PType: p2pType,
			Topic:   "test-consensus",
		},
		Node: &config.NodeInfo{
			ID:         0,
			P2PAddress: fmt.Sprintf("127.0.0.1:%d", basePort),
			DataDir:    t.TempDir(),
		},
		Nodes: make(map[int64]*config.NodeInfo),
	}

	// 创建节点列表
	for i := 0; i < nodeCount; i++ {
		cfg.Nodes[int64(i)] = &config.NodeInfo{
			ID:         int64(i),
			P2PAddress: fmt.Sprintf("127.0.0.1:%d", basePort+i),
			DataDir:    filepath.Join(t.TempDir(), fmt.Sprintf("node-%d", i)),
		}
	}

	return cfg
}

// TestBaseP2P_BasicOperations 测试 BaseP2P 基础操作
func TestBaseP2P_BasicOperations(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.DebugLevel)

	t.Run("NewBaseP2P", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, peerID, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		require.NotNil(t, p2p)
		assert.NotEmpty(t, peerID)
		assert.Equal(t, "p2p", p2p.GetP2PType())
		defer p2p.Stop()
	})

	t.Run("SetReceiver", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		receiveChan := make(chan []byte, 10)
		p2p.SetReceiver(receiveChan)
		assert.NotNil(t, p2p.output)
	})

	t.Run("GetPeerID", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		peerID := p2p.GetPeerID()
		assert.Equal(t, cfg.Node.P2PAddress, peerID)
	})

	t.Run("Subscribe_UnSubscribe", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		// BaseP2P 的 Subscribe/UnSubscribe 是空实现
		topic := []byte("test-topic")
		err = p2p.Subscribe(topic)
		assert.NoError(t, err)

		err = p2p.UnSubscribe(topic)
		assert.NoError(t, err)
	})
}

// TestBaseP2P_NodeDiscovery 测试 BaseP2P 节点发现
func TestBaseP2P_NodeDiscovery(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.DebugLevel)

	t.Run("GetNodeCount", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		// 初始时应该只有自己
		count := p2p.GetNodeCount()
		assert.Equal(t, 1, count)
	})

	t.Run("GetDiscoveredNodes", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		nodes := p2p.GetDiscoveredNodes()
		assert.NotEmpty(t, nodes)
		assert.Contains(t, nodes, int64(0)) // 应该包含自己
	})

	t.Run("RegisterNodeDiscoveryCallback", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		callback := newMockCallback()
		p2p.RegisterNodeDiscoveryCallback(callback)
		assert.NotNil(t, p2p.discoveryCallback)
	})

	t.Run("DiscoverPeers", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		callback := newMockCallback()
		p2p.RegisterNodeDiscoveryCallback(callback)

		err = p2p.DiscoverPeers()
		assert.NoError(t, err)

		// 等待探测完成
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("WaitForNodes_Timeout", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		// 要求 10 个节点,但只有 3 个,应该超时
		err = p2p.WaitForNodes(10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("WaitForNodes_Success", func(t *testing.T) {
		cfg := createTestConfig(t, 3, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		// 要求 1 个节点(自己),应该立即成功
		err = p2p.WaitForNodes(1, 5)
		assert.NoError(t, err)
	})
}

// TestBaseP2P_Messaging 测试 BaseP2P 消息传递
func TestBaseP2P_Messaging(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.DebugLevel)

	t.Run("Unicast_Invalid", func(t *testing.T) {
		cfg := createTestConfig(t, 2, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		testPacket := &pb.Packet{
			Type:        pb.PacketType_P2PPACKET,
			ConsensusID: 1,
			Epoch:       1,
			Msg:         []byte("test"),
		}
		msgBytes, err := proto.Marshal(testPacket)
		require.NoError(t, err)

		// 发送到不存在的地址
		err = p2p.Unicast("127.0.0.1:99999", msgBytes, 1, []byte("test"))
		assert.Error(t, err)
	})

	t.Run("Broadcast", func(t *testing.T) {
		cfg := createTestConfig(t, 2, "p2p-base")
		p2p, _, err := NewBaseP2P(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		testPacket := &pb.Packet{
			Type:        pb.PacketType_P2PPACKET,
			ConsensusID: 1,
			Epoch:       1,
			Msg:         []byte("broadcast test"),
		}
		msgBytes, err := proto.Marshal(testPacket)
		require.NoError(t, err)

		// 广播会尝试发送给所有节点,由于节点未启动会失败
		err = p2p.Broadcast(msgBytes, 1, []byte("test"))
		assert.Error(t, err) // 预期失败,因为其他节点未运行
	})
}

// TestP2PAdaptor_BasicOperations 测试 P2PAdaptor 基础操作
func TestP2PAdaptor_BasicOperations(t *testing.T) {
	// 跳过需要完整 libp2p 环境的测试
	if os.Getenv("RUN_P2P_ADAPTOR_TESTS") != "true" {
		t.Skip("Skipping P2PAdaptor tests (set RUN_P2P_ADAPTOR_TESTS=true to run)")
	}

	cfg := createTestConfig(t, 2, "p2p-adaptor")
	cfg.Nodes[0].P2PAddress = "127.0.0.1:20000"
	cfg.Nodes[1].P2PAddress = "127.0.0.1:20001"

	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.DebugLevel)

	t.Run("NewP2PAdaptor", func(t *testing.T) {
		p2p, peerID, err := NewP2PAdaptor(cfg, log, 0)
		require.NoError(t, err)
		require.NotNil(t, p2p)
		assert.NotEmpty(t, peerID)
		assert.Equal(t, "p2p-adaptor", p2p.GetP2PType())
		defer p2p.Stop()
	})

	t.Run("GetPeerID", func(t *testing.T) {
		p2p, expectedPeerID, err := NewP2PAdaptor(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		assert.Equal(t, expectedPeerID, p2p.GetPeerID())
	})
}

// TestP2PAdaptor_NodeDiscovery 测试 P2PAdaptor 节点发现
func TestP2PAdaptor_NodeDiscovery(t *testing.T) {
	if os.Getenv("RUN_P2P_ADAPTOR_TESTS") != "true" {
		t.Skip("Skipping P2PAdaptor tests")
	}

	cfg := createTestConfig(t, 2, "p2p-adaptor")
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.DebugLevel)

	t.Run("RegisterCallback_And_DiscoverPeers", func(t *testing.T) {
		p2p, _, err := NewP2PAdaptor(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		callback := newMockCallback()
		p2p.RegisterNodeDiscoveryCallback(callback)

		err = p2p.DiscoverPeers()
		assert.NoError(t, err)

		// 等待 DHT 初始化
		time.Sleep(2 * time.Second)
	})

	t.Run("GetDiscoveredNodes", func(t *testing.T) {
		p2p, _, err := NewP2PAdaptor(cfg, log, 0)
		require.NoError(t, err)
		defer p2p.Stop()

		nodes := p2p.GetDiscoveredNodes()
		assert.NotNil(t, nodes)
	})
}

// TestBuildP2P 测试 BuildP2P 工厂方法
func TestBuildP2P(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	t.Run("BuildP2P_BaseP2P", func(t *testing.T) {
		cfg := createTestConfig(t, 2, "p2p-base")
		p2p, peerID, err := BuildP2P(cfg, log, 0)
		require.NoError(t, err)
		assert.NotNil(t, p2p)
		assert.NotEmpty(t, peerID)
		assert.Equal(t, "p2p", p2p.GetP2PType())
		defer p2p.Stop()
	})

	t.Run("BuildP2P_Invalid_Type", func(t *testing.T) {
		cfg := createTestConfig(t, 2, "invalid-type")
		p2p, _, err := BuildP2P(cfg, log, 0)
		// 根据实现,返回 nil 或错误
		if p2p != nil {
			defer p2p.Stop()
		}
		assert.True(t, p2p == nil || err != nil)
	})
}

// TestP2PAdaptor_Integration 集成测试(需要完整环境)
func TestP2PAdaptor_Integration(t *testing.T) {
	if os.Getenv("RUN_P2P_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests")
	}

	cfg := createTestConfig(t, 2, "p2p-adaptor")
	log := logrus.NewEntry(logrus.New())
	log.Logger.SetLevel(logrus.InfoLevel)

	// 启动两个节点
	p2p1, _, err := NewP2PAdaptor(cfg, log, 0)
	require.NoError(t, err)
	defer p2p1.Stop()

	cfg.Node.ID = 1
	cfg.Node.P2PAddress = "127.0.0.1:20001"
	p2p2, _, err := NewP2PAdaptor(cfg, log, 1)
	require.NoError(t, err)
	defer p2p2.Stop()

	// 等待节点相互发现
	time.Sleep(5 * time.Second)

	// 验证节点发现
	nodes1 := p2p1.GetDiscoveredNodes()
	nodes2 := p2p2.GetDiscoveredNodes()

	t.Logf("Node1 discovered %d nodes", len(nodes1))
	t.Logf("Node2 discovered %d nodes", len(nodes2))
}
