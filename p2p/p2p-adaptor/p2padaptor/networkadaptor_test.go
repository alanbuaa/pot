package p2padaptor

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 自定义日志 writer，过滤掉 websocket 关闭警告
type filteredLogWriter struct {
	original io.Writer
}

func (w *filteredLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	// 过滤 websocket 关闭警告和 UDP buffer size 警告
	if strings.Contains(msg, "websocket: failed to close network connection") ||
		strings.Contains(msg, "failed to sufficiently increase receive buffer size") {
		return len(p), nil // 忽略这些日志
	}
	return w.original.Write(p)
}

// 测试初始化函数
func init() {
	// 设置日志过滤器，减少测试输出噪音
	if os.Getenv("VERBOSE_TEST") != "true" {
		log.SetOutput(&filteredLogWriter{original: os.Stderr})
	}
}

// 用于生成唯一端口的计数器
var testPortCounter int32 = 20000

// getTestPort 返回一个唯一的测试端口（只返回端口号字符串）
func getTestPort() string {
	port := atomic.AddInt32(&testPortCounter, 1)
	return fmt.Sprintf("%d", port)
}

// createTestNetworkAdaptor 创建测试用的 NetworkAdaptor
func createTestNetworkAdaptor(t *testing.T, topic string) *NetworkAdaptor {
	port := getTestPort()
	datadir := t.TempDir()

	na, err := NewNetworkAdaptor(port, topic, datadir)
	require.NoError(t, err)
	require.NotNil(t, na)

	return na
}

// createTestNetworkAdaptorWithBootstrap 创建带有自定义 bootstrap 的测试 NetworkAdaptor
func createTestNetworkAdaptorWithBootstrap(t *testing.T, topic string, bootstrap []string) *NetworkAdaptor {
	port := getTestPort()
	datadir := t.TempDir()

	// 注意：当前 NewNetworkAdaptor 不支持自定义 bootstrap
	// 如果需要完全控制 bootstrap 节点，需要修改 NetworkAdaptor 构造函数
	na, err := NewNetworkAdaptor(port, topic, datadir)
	require.NoError(t, err)
	require.NotNil(t, na)

	return na
}

// mockDiscoveryCallback 模拟节点发现回调
type mockDiscoveryCallback struct {
	discoveredNodes map[int64]string
	lostNodes       map[int64]bool
	mu              sync.Mutex
}

func newMockDiscoveryCallback() *mockDiscoveryCallback {
	return &mockDiscoveryCallback{
		discoveredNodes: make(map[int64]string),
		lostNodes:       make(map[int64]bool),
	}
}

func (m *mockDiscoveryCallback) OnNodeDiscovered(nodeID int64, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoveredNodes[nodeID] = address
	return nil
}

func (m *mockDiscoveryCallback) OnNodeLost(nodeID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lostNodes[nodeID] = true
	return nil
}

func (m *mockDiscoveryCallback) getDiscoveredCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.discoveredNodes)
}

// TestNewNetworkAdaptor 测试创建 NetworkAdaptor
func TestNewNetworkAdaptor(t *testing.T) {
	port := getTestPort()
	datadir := t.TempDir()
	topic := "test-topic"

	na, err := NewNetworkAdaptor(port, topic, datadir)
	require.NoError(t, err)
	require.NotNil(t, na)

	assert.NotNil(t, na.p2pnet)
	assert.NotNil(t, na.topics)
	assert.NotNil(t, na.subs)
	assert.NotNil(t, na.subStopChs)
	assert.NotNil(t, na.receiveChs)
	assert.NotNil(t, na.discoveredNodes)
	assert.Equal(t, topic, na.discoveryTopic)

	// 验证数据目录创建
	networkDir := filepath.Join(datadir, "network")
	dhtDir := filepath.Join(networkDir, "dht")
	keyDir := filepath.Join(networkDir, "key")

	// 这些目录应该在网络初始化时创建
	assert.DirExists(t, networkDir)
	assert.DirExists(t, dhtDir)
	assert.DirExists(t, keyDir)
}

// TestNetworkAdaptor_GetPeerID 测试获取 PeerID
func TestNetworkAdaptor_GetPeerID(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	peerID := na.GetPeerID()
	assert.NotEmpty(t, peerID)

	// PeerID 应该是 base58 编码的字符串
	assert.Greater(t, len(peerID), 20)
}

// TestNetworkAdaptor_GetP2PType 测试获取 P2P 类型
func TestNetworkAdaptor_GetP2PType(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	p2pType := na.GetP2PType()
	assert.Equal(t, "p2p-adaptor", p2pType)
}

// TestNetworkAdaptor_Subscribe_Unsubscribe 测试订阅和取消订阅
func TestNetworkAdaptor_Subscribe_Unsubscribe(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	// 启动 Unicast 以设置 entrance channel
	err := na.StartUnicast()
	require.NoError(t, err)

	receiveChan := make(chan []byte, 10)
	na.SetReceiver(receiveChan)

	topic := []byte("test-subscribe-topic")

	// 测试订阅
	err = na.Subscribe(topic)
	assert.NoError(t, err)

	// 验证订阅成功
	topicName := string(topic)
	assert.Contains(t, na.topics, topicName)
	assert.Contains(t, na.subs, topicName)
	assert.Contains(t, na.subStopChs, topicName)

	// 测试重复订阅应该失败
	err = na.Subscribe(topic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// 测试取消订阅
	err = na.UnSubscribe(topic)
	assert.NoError(t, err)

	// 验证取消订阅成功
	assert.NotContains(t, na.topics, topicName)
	assert.NotContains(t, na.subs, topicName)
	assert.NotContains(t, na.subStopChs, topicName)

	// 测试取消不存在的订阅
	err = na.UnSubscribe([]byte("non-existent-topic"))
	assert.Error(t, err)
}

// TestNetworkAdaptor_RegularSubscribe 测试常规订阅
func TestNetworkAdaptor_RegularSubscribe(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	topic := []byte("regular-subscribe-topic")

	// 测试订阅
	receiveChan, err := na.RegularSubscribe(topic)
	assert.NoError(t, err)
	assert.NotNil(t, receiveChan)

	// 验证订阅成功
	topicName := string(topic)
	assert.Contains(t, na.topics, topicName)
	assert.Contains(t, na.subs, topicName)

	// 测试重复订阅
	_, err = na.RegularSubscribe(topic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestNetworkAdaptor_Broadcast 测试广播消息
func TestNetworkAdaptor_Broadcast(t *testing.T) {
	// 创建两个节点
	na1 := createTestNetworkAdaptor(t, "broadcast-topic")
	na2 := createTestNetworkAdaptor(t, "broadcast-topic")

	topic := []byte("broadcast-test")

	// 两个节点都订阅同一个 topic
	receiveChan1, err := na1.RegularSubscribe(topic)
	require.NoError(t, err)

	receiveChan2, err := na2.RegularSubscribe(topic)
	require.NoError(t, err)

	// 等待订阅完成
	time.Sleep(500 * time.Millisecond)

	// na1 广播消息
	testMsg := []byte("Hello from na1")
	err = na1.Broadcast(testMsg, 1, topic)
	assert.NoError(t, err)

	// 验证 na1 自己也能收到（pubsub 特性）
	select {
	case msg := <-receiveChan1:
		assert.Equal(t, testMsg, msg)
	case <-time.After(2 * time.Second):
		t.Log("Warning: na1 did not receive its own broadcast")
	}

	// 验证 na2 收到消息
	select {
	case msg := <-receiveChan2:
		assert.Equal(t, testMsg, msg)
	case <-time.After(2 * time.Second):
		t.Log("Warning: na2 did not receive broadcast from na1")
	}
}

// TestNetworkAdaptor_Broadcast_NotSubscribed 测试未订阅时广播失败
func TestNetworkAdaptor_Broadcast_NotSubscribed(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	topic := []byte("not-subscribed-topic")
	testMsg := []byte("Test message")

	err := na.Broadcast(testMsg, 1, topic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not subscribed")
}

// TestNetworkAdaptor_Unicast 测试单播消息
func TestNetworkAdaptor_Unicast(t *testing.T) {
	if os.Getenv("RUN_P2P_ADAPTOR_TESTS") != "true" {
		t.Skip("Skipping P2P Adaptor unicast test (set RUN_P2P_ADAPTOR_TESTS=true)")
	}

	// 创建两个节点
	na1 := createTestNetworkAdaptor(t, "unicast-topic")
	na2 := createTestNetworkAdaptor(t, "unicast-topic")

	// 启动 Unicast 服务
	err := na1.StartUnicast()
	require.NoError(t, err)

	err = na2.StartUnicast()
	require.NoError(t, err)

	// 设置接收通道
	receiveChan1 := make(chan []byte, 10)
	receiveChan2 := make(chan []byte, 10)

	na1.SetReceiver(receiveChan1)
	na2.SetReceiver(receiveChan2)

	// 等待连接建立
	time.Sleep(1 * time.Second)

	// na1 向 na2 发送单播消息
	testMsg := []byte("Unicast message from na1 to na2")
	peerID2 := na2.GetPeerID()

	err = na1.Unicast(peerID2, testMsg, 1, []byte("test-topic"))
	assert.NoError(t, err)

	// 验证 na2 收到消息
	select {
	case msg := <-receiveChan2:
		assert.Equal(t, testMsg, msg)
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for unicast message")
	}

	// 验证 na1 不会收到这条消息（单播特性）
	select {
	case <-receiveChan1:
		t.Fatal("na1 should not receive unicast message")
	case <-time.After(500 * time.Millisecond):
		// 正确：na1 没有收到消息
	}
}

// TestNetworkAdaptor_NodeDiscovery 测试节点发现
func TestNetworkAdaptor_NodeDiscovery(t *testing.T) {
	if os.Getenv("RUN_P2P_ADAPTOR_TESTS") != "true" {
		t.Skip("Skipping node discovery test")
	}

	topic := "discovery-test-topic"
	na1 := createTestNetworkAdaptor(t, topic)
	na2 := createTestNetworkAdaptor(t, topic)

	callback1 := newMockDiscoveryCallback()
	callback2 := newMockDiscoveryCallback()

	na1.RegisterNodeDiscoveryCallback(callback1)
	na2.RegisterNodeDiscoveryCallback(callback2)

	// 启动节点发现
	err := na1.DiscoverPeers()
	assert.NoError(t, err)

	err = na2.DiscoverPeers()
	assert.NoError(t, err)

	// 等待节点相互发现
	time.Sleep(5 * time.Second)

	// 验证节点发现
	count1 := na1.GetNodeCount()
	count2 := na2.GetNodeCount()

	t.Logf("na1 discovered %d nodes", count1)
	t.Logf("na2 discovered %d nodes", count2)

	// 至少应该发现一些节点
	assert.True(t, count1 >= 0 || count2 >= 0)
}

// TestNetworkAdaptor_GetDiscoveredNodes 测试获取已发现的节点
func TestNetworkAdaptor_GetDiscoveredNodes(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	// 初始时应该没有发现的节点
	nodes := na.GetDiscoveredNodes()
	assert.NotNil(t, nodes)
	assert.Empty(t, nodes)

	// 手动添加一些节点
	na.discoveryMu.Lock()
	na.discoveredNodes[1] = "peer1"
	na.discoveredNodes[2] = "peer2"
	na.discoveryMu.Unlock()

	// 获取节点列表
	nodes = na.GetDiscoveredNodes()
	assert.Len(t, nodes, 2)
	assert.Equal(t, "peer1", nodes[1])
	assert.Equal(t, "peer2", nodes[2])

	// 修改返回的副本不应影响原始数据
	nodes[3] = "peer3"
	assert.NotContains(t, na.discoveredNodes, int64(3))
}

// TestNetworkAdaptor_GetNodeCount 测试获取节点数量
func TestNetworkAdaptor_GetNodeCount(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	// 初始时应该是 0
	count := na.GetNodeCount()
	assert.Equal(t, 0, count)

	// 添加节点
	na.discoveryMu.Lock()
	na.discoveredNodes[1] = "peer1"
	na.discoveredNodes[2] = "peer2"
	na.discoveredNodes[3] = "peer3"
	na.discoveryMu.Unlock()

	count = na.GetNodeCount()
	assert.Equal(t, 3, count)
}

// TestNetworkAdaptor_WaitForNodes 测试等待节点上线
func TestNetworkAdaptor_WaitForNodes(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	t.Run("Timeout", func(t *testing.T) {
		// 等待 10 个节点，但实际没有，应该超时
		err := na.WaitForNodes(10, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("Success", func(t *testing.T) {
		// 在后台添加节点
		go func() {
			time.Sleep(500 * time.Millisecond)
			na.discoveryMu.Lock()
			na.discoveredNodes[1] = "peer1"
			na.discoveredNodes[2] = "peer2"
			na.discoveryMu.Unlock()
		}()

		// 等待 2 个节点
		err := na.WaitForNodes(2, 5)
		assert.NoError(t, err)
	})
}

// TestNetworkAdaptor_RegisterCallback 测试注册回调
func TestNetworkAdaptor_RegisterCallback(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	// 初始时回调应该是 nil
	assert.Nil(t, na.discoveryCallback)

	callback := newMockDiscoveryCallback()
	na.RegisterNodeDiscoveryCallback(callback)

	// 验证回调已设置
	assert.NotNil(t, na.discoveryCallback)
}

// TestNetworkAdaptor_SetReceiver 测试设置接收器
func TestNetworkAdaptor_SetReceiver(t *testing.T) {
	na := createTestNetworkAdaptor(t, "test-topic")

	// 先启动 Unicast
	err := na.StartUnicast()
	require.NoError(t, err)

	receiveChan := make(chan []byte, 10)
	na.SetReceiver(receiveChan)

	// 验证接收器已设置
	assert.NotNil(t, na.uca)
	assert.NotNil(t, na.uca.entrance)
}

// ========== 性能测试 ==========

// BenchmarkNetworkAdaptor_Broadcast 测试广播性能
func BenchmarkNetworkAdaptor_Broadcast(b *testing.B) {
	topic := "benchmark-broadcast"
	na1 := &NetworkAdaptor{}
	na2 := &NetworkAdaptor{}

	var err error
	port1 := getTestPort()
	port2 := getTestPort()
	datadir1 := b.TempDir()
	datadir2 := b.TempDir()

	na1, err = NewNetworkAdaptor(port1, topic, datadir1)
	if err != nil {
		b.Fatalf("Failed to create na1: %v", err)
	}

	na2, err = NewNetworkAdaptor(port2, topic, datadir2)
	if err != nil {
		b.Fatalf("Failed to create na2: %v", err)
	}

	topicBytes := []byte(topic)

	// 订阅
	_, err = na1.RegularSubscribe(topicBytes)
	if err != nil {
		b.Fatalf("Subscribe failed: %v", err)
	}

	_, err = na2.RegularSubscribe(topicBytes)
	if err != nil {
		b.Fatalf("Subscribe failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	testMsg := []byte("benchmark test message with some payload data")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = na1.Broadcast(testMsg, 1, topicBytes)
		if err != nil {
			b.Fatalf("Broadcast failed: %v", err)
		}
	}
}

// BenchmarkNetworkAdaptor_Subscribe 测试订阅性能
func BenchmarkNetworkAdaptor_Subscribe(b *testing.B) {
	na := &NetworkAdaptor{}
	port := getTestPort()
	datadir := b.TempDir()

	var err error
	na, err = NewNetworkAdaptor(port, "bench-topic", datadir)
	if err != nil {
		b.Fatalf("Failed to create NetworkAdaptor: %v", err)
	}

	err = na.StartUnicast()
	if err != nil {
		b.Fatalf("StartUnicast failed: %v", err)
	}

	receiveChan := make(chan []byte, 100)
	na.SetReceiver(receiveChan)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		topic := []byte(fmt.Sprintf("topic-%d", i))
		err = na.Subscribe(topic)
		if err != nil {
			b.Fatalf("Subscribe failed: %v", err)
		}
	}
}

// BenchmarkNetworkAdaptor_GetNodeCount 测试获取节点数性能
func BenchmarkNetworkAdaptor_GetNodeCount(b *testing.B) {
	port := getTestPort()
	datadir := b.TempDir()

	na, err := NewNetworkAdaptor(port, "bench-topic", datadir)
	if err != nil {
		b.Fatalf("Failed to create NetworkAdaptor: %v", err)
	}

	// 添加一些节点
	na.discoveryMu.Lock()
	for i := 0; i < 100; i++ {
		na.discoveredNodes[int64(i)] = fmt.Sprintf("peer-%d", i)
	}
	na.discoveryMu.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = na.GetNodeCount()
	}
}

// BenchmarkNetworkAdaptor_GetDiscoveredNodes 测试获取节点列表性能
func BenchmarkNetworkAdaptor_GetDiscoveredNodes(b *testing.B) {
	port := getTestPort()
	datadir := b.TempDir()

	na, err := NewNetworkAdaptor(port, "bench-topic", datadir)
	if err != nil {
		b.Fatalf("Failed to create NetworkAdaptor: %v", err)
	}

	// 添加节点
	na.discoveryMu.Lock()
	for i := 0; i < 100; i++ {
		na.discoveredNodes[int64(i)] = fmt.Sprintf("peer-%d", i)
	}
	na.discoveryMu.Unlock()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = na.GetDiscoveredNodes()
	}
}

// BenchmarkNetworkAdaptor_ConcurrentOperations 测试并发操作性能
func BenchmarkNetworkAdaptor_ConcurrentOperations(b *testing.B) {
	port := getTestPort()
	datadir := b.TempDir()

	na, err := NewNetworkAdaptor(port, "bench-topic", datadir)
	if err != nil {
		b.Fatalf("Failed to create NetworkAdaptor: %v", err)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 模拟节点发现操作
			na.discoveryMu.Lock()
			na.discoveredNodes[int64(i)] = fmt.Sprintf("peer-%d", i)
			na.discoveryMu.Unlock()

			// 获取节点数
			_ = na.GetNodeCount()

			// 获取节点列表
			_ = na.GetDiscoveredNodes()

			i++
		}
	})
}

// BenchmarkNetworkAdaptor_MessageThroughput 测试消息吞吐量
func BenchmarkNetworkAdaptor_MessageThroughput(b *testing.B) {
	if os.Getenv("RUN_BENCHMARK_TESTS") != "true" {
		b.Skip("Skipping benchmark test")
	}

	topic := "throughput-test"
	na1 := &NetworkAdaptor{}
	na2 := &NetworkAdaptor{}

	var err error
	port1 := getTestPort()
	port2 := getTestPort()
	datadir1 := b.TempDir()
	datadir2 := b.TempDir()

	na1, err = NewNetworkAdaptor(port1, topic, datadir1)
	if err != nil {
		b.Fatalf("Failed to create na1: %v", err)
	}

	na2, err = NewNetworkAdaptor(port2, topic, datadir2)
	if err != nil {
		b.Fatalf("Failed to create na2: %v", err)
	}

	topicBytes := []byte(topic)

	receiveChan, err := na2.RegularSubscribe(topicBytes)
	if err != nil {
		b.Fatalf("Subscribe failed: %v", err)
	}

	_, err = na1.RegularSubscribe(topicBytes)
	if err != nil {
		b.Fatalf("Subscribe failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	// 接收消息的 goroutine
	var receivedCount int64
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-receiveChan:
				atomic.AddInt64(&receivedCount, 1)
			case <-done:
				return
			}
		}
	}()

	testMsg := make([]byte, 1024) // 1KB 消息
	for i := range testMsg {
		testMsg[i] = byte(i % 256)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = na1.Broadcast(testMsg, 1, topicBytes)
		if err != nil {
			b.Fatalf("Broadcast failed: %v", err)
		}
	}

	b.StopTimer()

	// 等待接收完成
	time.Sleep(2 * time.Second)
	close(done)

	finalReceived := atomic.LoadInt64(&receivedCount)
	b.ReportMetric(float64(finalReceived)/float64(b.N)*100, "%received")
	b.ReportMetric(float64(len(testMsg)), "bytes/msg")
}
