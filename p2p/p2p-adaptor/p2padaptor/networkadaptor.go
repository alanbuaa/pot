package p2padaptor

import (
	"context"
	"errors"
	net "network"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var (
	dhtPath = "store/dht-store/"
	keyPath = "store/host-key/"
)

type NetworkAdaptor struct {
	p2pnet     *net.Network
	topics     map[string]*pubsub.Topic
	subs       map[string]*pubsub.Subscription
	subStopChs map[string]chan struct{}
	receiveChs map[string]chan []byte
	// feedbackchan chan *net.ValidationFeedback
	uca *UnicastAdapter
}

func NewNetworkAdaptor(netPort string) (*NetworkAdaptor, error) {

	network, _, err := net.NewNetwork(netPort, dhtPath+netPort+"/", keyPath+netPort+"/", nil, false)
	if err != nil {
		return &NetworkAdaptor{}, err
	}

	netadaptor := &NetworkAdaptor{
		p2pnet:     network,
		topics:     make(map[string]*pubsub.Topic),
		subs:       make(map[string]*pubsub.Subscription),
		subStopChs: make(map[string]chan struct{}),
		receiveChs: make(map[string]chan []byte),
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
