package p2padaptor

import (
	"context"
	"errors"
	"fmt"
	net "network"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

// NodeDiscoveryCallback 节点发现回调接口
type NodeDiscoveryCallback interface {
	// OnNodeDiscovered 当发现新节点时调用
	OnNodeDiscovered(nodeID int64, address string) error
	// OnNodeLost 当节点丢失时调用
	OnNodeLost(nodeID int64) error
}

type NetworkAdaptor struct {
	p2pnet     *net.Network
	topics     map[string]*pubsub.Topic
	subs       map[string]*pubsub.Subscription
	subStopChs map[string]chan struct{}
	receiveChs map[string]chan []byte
	// feedbackchan chan *net.ValidationFeedback
	uca *UnicastAdapter

	// 节点发现相关
	discoveredNodes   map[int64]string      // nodeID -> peerID
	discoveryCallback NodeDiscoveryCallback // 使用本地定义的接口
	discoveryMu       sync.RWMutex
	discoveryTopic    string // 用于节点发现的 topic
}

func NewNetworkAdaptor(netPort string, topic string, datadir string) (*NetworkAdaptor, error) {

	datadir = datadir + "/network/"
	dhtPath := datadir + "dht/"
	keyPath := datadir + "key/"

	network, _, err := net.NewNetwork(netPort, dhtPath, keyPath, nil, false)
	if err != nil {
		return &NetworkAdaptor{}, err
	}

	netadaptor := &NetworkAdaptor{
		p2pnet:          network,
		topics:          make(map[string]*pubsub.Topic),
		subs:            make(map[string]*pubsub.Subscription),
		subStopChs:      make(map[string]chan struct{}),
		receiveChs:      make(map[string]chan []byte),
		discoveredNodes: make(map[int64]string),
		discoveryTopic:  topic,
	}

	return netadaptor, nil
}

func (na *NetworkAdaptor) RegularSubscribe(topic []byte) (chan []byte, error) {

	topicName := string(topic)
	_, ok := na.topics[topicName]
	if ok {
		return nil, errors.New("topic already exists")
	}

	// Join topic
	mytopic, mysub, err := na.p2pnet.JoinTopic(topicName, false)
	if err != nil {
		return nil, errors.New("Subscribe topic error")
	}

	subStopCh := make(chan struct{})
	receiveCh := make(chan []byte)

	na.topics[topicName] = mytopic
	na.subs[topicName] = mysub
	na.subStopChs[topicName] = subStopCh

	// monitor sub.Next()
	go func(subStopCh chan struct{}) {
		ctx := context.Background()
		for {
			select {
			case <-subStopCh:
				return
			default:
				m, err := mysub.Next(ctx)
				if err != nil {
					// panic(err)
					continue
				}
				// fmt.Println(m.ReceivedFrom, ": ", string(m.Message.Data))

				receiveCh <- m.Data
			}
		}
	}(subStopCh)

	return receiveCh, nil

}

// Only for consensus
func (na *NetworkAdaptor) Subscribe(topic []byte) error {
	// if na.uca == nil || na.uca.entrance == nil {
	// 	return errors.New("please set the entrance channel first")
	// }

	// Check if topic exists
	topicName := string(topic)
	_, ok := na.topics[topicName]
	if ok {
		return errors.New("topic already exists")
	}

	// Join topic
	mytopic, mysub, err := na.p2pnet.JoinTopic(topicName, false)
	if err != nil {
		return errors.New("Subscribe topic error")
	}

	subStopCh := make(chan struct{})

	na.topics[topicName] = mytopic
	na.subs[topicName] = mysub
	na.subStopChs[topicName] = subStopCh

	// monitor sub.Next()
	go func(subStopCh chan struct{}) {
		ctx := context.Background()
		for {
			select {
			case <-subStopCh:
				return
			default:
				m, err := mysub.Next(ctx)
				if err != nil {
					// panic(err)
					continue
				}
				// fmt.Println(m.ReceivedFrom, ": ", string(m.Message.Data))

				// if entrance is nil
				if na.uca.entrance != nil {
					na.uca.entrance <- m.Data
				}
			}
		}
	}(subStopCh)

	return nil
}

func (na *NetworkAdaptor) UnSubscribe(topic []byte) error {
	topicName := string(topic)

	// Check if topic exists
	_, ok := na.topics[topicName]
	if !ok { // if no topic
		return errors.New("there is no this topic")
	}

	sub, ok1 := na.subs[topicName]
	if !ok1 { // if no sub
		return errors.New("there is no this sub")
	}

	// Notify the goroutine to exit
	na.subStopChs[topicName] <- struct{}{}

	// Cancel subscribe
	sub.Cancel()

	// Delete topic, sub and subStopCh
	delete(na.topics, topicName)
	delete(na.subs, topicName)
	delete(na.subStopChs, topicName)

	return nil
}

func (na *NetworkAdaptor) Broadcast(msgByte []byte, consensusID int64, topic []byte) error {

	// Check if topic exists
	mytopic, ok := na.topics[string(topic)]
	if !ok {
		return errors.New("not subscribed to this topic")
	}

	ctx := context.Background()

	// Publish message to this topic
	if err := mytopic.Publish(ctx, msgByte); err != nil {
		return errors.New("topic.Publish error")
	}

	return nil
}

// ------- For Unicast ------- //

func (na *NetworkAdaptor) StartUnicast() error {
	var err error
	na.uca, err = NewUniCastAdaptor(na.p2pnet)
	if err != nil {
		return errors.New("new unicast adaptor error")
	}

	na.uca.StartUnicastService()

	return nil
}

func (na *NetworkAdaptor) SetReceiver(ch chan<- []byte) {
	// na.uca.entrance = receiver.GetMsgByteEntrance()
	na.uca.entrance = ch
}

func (na *NetworkAdaptor) Unicast(address string, msgByte []byte, consensusID int64, topic []byte) error {
	return na.uca.SendUnicast(address, msgByte, consensusID)
}

// Close network
func (na *NetworkAdaptor) Stop() {
	// TODO
}

// Get peer id
func (na *NetworkAdaptor) GetPeerID() string {
	return na.p2pnet.GetPeerID()
}

// Get peer id
func (na *NetworkAdaptor) GetP2PType() string {
	return "p2p-adaptor"
}

// DiscoverPeers 启动节点发现进程 (使用 libp2p DHT)
func (na *NetworkAdaptor) DiscoverPeers() error {
	if na.p2pnet == nil {
		return errors.New("p2p network not initialized")
	}

	fmt.Printf("[P2P-Adaptor] Starting node discovery via DHT on topic: %s\n", na.discoveryTopic)

	// 使用 libp2p 的 DHT 进行节点发现
	go func() {
		ctx := context.Background()
		routingDiscovery := drouting.NewRoutingDiscovery(na.p2pnet.GetDHT())

		// 1. Advertise 自己对该 topic 感兴趣
		dutil.Advertise(ctx, routingDiscovery, na.discoveryTopic)
		fmt.Printf("[P2P-Adaptor] Advertised on discovery topic: %s\n", na.discoveryTopic)

		// 2. 持续发现其他节点
		for {
			peerChan, err := routingDiscovery.FindPeers(ctx, na.discoveryTopic)
			if err != nil {
				fmt.Printf("[P2P-Adaptor] FindPeers error: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for peerInfo := range peerChan {
				// 跳过自己
				if peerInfo.ID == na.p2pnet.GetHost().ID() {
					continue
				}

				// 尝试连接
				err := na.p2pnet.GetHost().Connect(ctx, peerInfo)
				if err != nil {
					// fmt.Printf("[P2P-Adaptor] Failed to connect to peer %s: %v\n", peerInfo.ID, err)
					continue
				}

				// 连接成功,记录发现的节点
				peerIDStr := peerInfo.ID.String()
				fmt.Printf("[P2P-Adaptor] Discovered and connected to peer: %s\n", peerIDStr)

				// TODO: 这里需要从 peerID 映射到 nodeID
				// 目前使用 peerID 的哈希作为临时 nodeID
				nodeID := int64(peerInfo.ID.Size()) // 临时方案

				na.discoveryMu.Lock()
				if _, exists := na.discoveredNodes[nodeID]; !exists {
					na.discoveredNodes[nodeID] = peerIDStr
					na.discoveryMu.Unlock()

					// 通知回调
					if na.discoveryCallback != nil {
						if err := na.discoveryCallback.OnNodeDiscovered(nodeID, peerIDStr); err != nil {
							fmt.Printf("[P2P-Adaptor] Discovery callback error: %v\n", err)
						}
					}
				} else {
					na.discoveryMu.Unlock()
				}
			}

			// 每隔一段时间重新搜索
			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

// RegisterNodeDiscoveryCallback 注册节点发现回调
func (na *NetworkAdaptor) RegisterNodeDiscoveryCallback(callback NodeDiscoveryCallback) {
	na.discoveryMu.Lock()
	defer na.discoveryMu.Unlock()
	na.discoveryCallback = callback
	fmt.Println("[P2P-Adaptor] Node discovery callback registered")
}

// GetDiscoveredNodes 获取已发现的节点列表
func (na *NetworkAdaptor) GetDiscoveredNodes() map[int64]string {
	na.discoveryMu.RLock()
	defer na.discoveryMu.RUnlock()

	// 返回副本以避免并发问题
	nodes := make(map[int64]string)
	for k, v := range na.discoveredNodes {
		nodes[k] = v
	}
	return nodes
}

// GetNodeCount 获取已发现的节点数量
func (na *NetworkAdaptor) GetNodeCount() int {
	na.discoveryMu.RLock()
	defer na.discoveryMu.RUnlock()
	return len(na.discoveredNodes)
}

// WaitForNodes 等待指定数量的节点上线
func (na *NetworkAdaptor) WaitForNodes(count int, timeoutSec int) error {
	fmt.Printf("[P2P-Adaptor] Waiting for %d nodes to be discovered (timeout: %ds)...\n", count, timeoutSec)

	timeout := time.After(time.Duration(timeoutSec) * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			currentCount := na.GetNodeCount()
			return fmt.Errorf("timeout waiting for nodes: expected %d, got %d", count, currentCount)
		case <-ticker.C:
			na.discoveryMu.RLock()
			currentCount := len(na.discoveredNodes)
			na.discoveryMu.RUnlock()

			if currentCount >= count {
				fmt.Printf("[P2P-Adaptor] Successfully discovered %d nodes\n", currentCount)
				return nil
			}
		}
	}
}
