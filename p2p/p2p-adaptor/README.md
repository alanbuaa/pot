

## Package p2padaptor

### 1. 接口描述

```go
type P2PAdaptor interface {
	// ------ For broadcast ------ //

	// Subscribe to the topic you are interested in
	Subscribe(topic []byte) error

	// UnSubscribe to topic
	UnSubscribe(topic []byte) error

	// Send a message to a subscribed topic, and the message
	// will be broadcast within the topic
	Broadcast(msgByte []byte, consensusID int64, topic []byte) error

	// ------- For Unicast ------- //

	// Start the unicast service and receive unicast
	// messages from other peers
	StartUnicast() error

	// Set up a channel to receive unicast messages from other peers
	SetUnicastReceiver(receiver MsgReceiver)

	// Send a unicast message to a given node, attempting
	// to find it and establish a connection with it
	Unicast(address string, msgByte []byte, consensusID int64) error

	// ------- For Network ------- //

	// Get the PeerID of the local node
	GetPeerID() string

	// Stop network
	Stop()
}
```

### 2. 使用示例

#### 2.1 分别编译example中的recevier和sender代码

**注意：**引用包时需要在go.mod中添加包的引用地址，可根据目录修改相对路径，例如下面的格式：（示例中无需修改）

```go
replace p2padaptor => ../../../p2p-adaptor/p2padaptor
replace network => ../../../p2p-adaptor/network
```

#### 2.2 启动recevier

```
./recevier
```

```
PeerID:  QmacF2qBrVXVwTgATmb9FJLQcsnRvTLZdP8BdWxNttXBgo
Unicast service started
2023/07/13 20:19:56 Searching for 'this-is-consensus-topic' peers... 
2023/07/13 20:20:08 Peer discovery complete
Joined to topic
Receiving broadcast and unicast messages...
```

启动时会开启单播服务，连接节点并加入主题；

当出现Receiving broadcast and unicast messages...后复制上面的PeerID；

#### 2.3 启动sender

在peerid参数中输入刚才复制的PeerID

```
./sender -peerid QmacF2qBrVXVwTgATmb9FJLQcsnRvTLZdP8BdWxNttXBgo
```