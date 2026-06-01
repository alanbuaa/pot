package p2padaptor

// type MsgReceiver interface {
// 	// Get message receiving channel
// 	GetMsgByteEntrance() chan<- []byte
// }

// type P2PAdaptor interface {
// 	// ------ For broadcast ------ //

// 	// Subscribe to the topic you are interested in
// 	Subscribe(topic []byte) error

// 	// UnSubscribe to topic
// 	UnSubscribe(topic []byte) error

// 	// Send a message to a subscribed topic, and the message
// 	// will be broadcast within the topic
// 	Broadcast(msgByte []byte, consensusID int64, topic []byte) error

// 	// ------- For Unicast ------- //

// 	// Start the unicast service and receive unicast
// 	// messages from other peers
// 	StartUnicast() error

// 	// Set up a channel to receive unicast messages from other peers
// 	SetUnicastReceiver(receiver MsgReceiver)

// 	// Send a unicast message to a given node, attempting
// 	// to find it and establish a connection with it
// 	Unicast(address string, msgByte []byte, consensusID int64) error

// 	// ------- For Network ------- //

// 	// Get the PeerID of the local node
// 	GetPeerID() string

// 	// Stop network
// 	Stop()
// }
