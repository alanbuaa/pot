package network

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	ldb "github.com/ipfs/go-ds-leveldb"

	cid "github.com/ipfs/go-cid"
	libp2p "github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
)

type Network struct {
	host host.Host
	dht  *kaddht.IpfsDHT
	ctx  context.Context
	ps   *pubsub.PubSub
}

type Message struct {
	*pubsub.Message
	IsAddTransferEvidence bool
	// TransferEvidence
}

type NetworkOption struct {
	// port string
}

type Option func(no *NetworkOption)

var BootstrapPeers []ma.Multiaddr

func NewNetwork(port string, dhtPath string, keyPath string, bootstrapStrings []string, isWithScore bool) (*Network, chan *ValidationFeedback, error) {
	var net *Network
	var feedbackchan chan *ValidationFeedback

	// Context
	ctx := context.Background()

	// Get or create a private key for host
	PathExistsOrCreates(keyPath)
	priv, err := GetHostPrivKey(keyPath)
	if err != nil {
		return net, feedbackchan, errors.New("GetHostPrivKey error")
	}

	// New Host
	listen, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", port))
	h, err := libp2p.New(
		// listen addresses
		libp2p.ListenAddrs(listen),
		// use private key to create host
		libp2p.Identity(priv),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.WithDialTimeout(2*time.Second),
	)
	if err != nil {
		return net, feedbackchan, errors.New(err.Error() + "New Host Error.")
	}

	// log.Println("Host: ", h.ID())

	// New DHT
	PathExistsOrCreates(dhtPath)
	mydb, err := ldb.NewDatastore(dhtPath, nil)
	if err != nil {
		return net, feedbackchan, errors.New(err.Error() + "New Datastore Error.")
	}

	// Build kaddht from datastore
	kademliaDHT, err := kaddht.New(ctx, h, kaddht.Datastore(mydb))
	if err != nil {
		return net, feedbackchan, errors.New(err.Error() + "New DHT Error.")
	}

	// Owner bootstrap peers
	if bootstrapStrings == nil {
		BootstrapPeers = kaddht.DefaultBootstrapPeers
	} else {
		BootstrapPeers, err = BuildBootstrapPeers(bootstrapStrings)
		if err != nil {
			return net, feedbackchan, errors.New(err.Error() + "Build bootstrap peers Error.")
		}
	}

	// Connect BootstrapPeers to build dht
	bootstrap(h, kademliaDHT, BootstrapPeers, ctx)

	var ps *pubsub.PubSub

	// With score
	if isWithScore {
		// New AppScore
		appScore := NewAppScore(ctx)
		// appScore.refreshScores(ctx)
		// Set feedback channel
		feedbackchan = appScore.feedbackchan

		// Setup scoreParams and scorethresholds
		scoreParams, scorethresholds := SetupScoreParamsAndthresholds()
		// New Gossipsub
		ps, err = pubsub.NewGossipSub(ctx, h,
			pubsub.WithPeerScore(scoreParams, scorethresholds),
			pubsub.WithMessageIdFn(genMessageID))
		if err != nil {
			return net, feedbackchan, errors.New(err.Error() + "New Gossipsub With PeerScore Error.")
		}
	} else {
		ps, err = pubsub.NewGossipSub(ctx, h, pubsub.WithMessageIdFn(genMessageID))
		if err != nil {
			return net, feedbackchan, errors.New(err.Error() + "New Gossipsub Error.")
		}
	}

	// New network
	net = &Network{
		host: h,
		dht:  kademliaDHT,
		ctx:  ctx,
		ps:   ps,
	}

	return net, feedbackchan, nil
}

// function to generate message id
func genMessageID(pmsg *pb.Message) string {
	// Generate pb.Message's unique ID(string)
	h := sha256.New()
	h.Write(pmsg.Data)
	sum := h.Sum(nil)

	// To hexadecimal string
	s := hex.EncodeToString(sum)
	return s
}

// Connect BootstrapPeers to build dht
func bootstrap(host host.Host, dht *kaddht.IpfsDHT, bootstrappeers []ma.Multiaddr, ctx context.Context) error {
	anybootstrappeer := false

	if err := dht.Bootstrap(ctx); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	for _, peerAddr := range bootstrappeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := host.Connect(ctx, *peerinfo)
			if err != nil {
				// log.Println("Bootstrap warning:", err)
			} else {
				anybootstrappeer = true
				// log.Println("Connected BootstrapPeer ID: ", peerinfo.ID)
			}
		}()
	}

	wg.Wait()

	if !anybootstrappeer {
		return errors.New("can not find or connect to a bootstrap peer")
	}

	return nil
}

// Discovery Peers who subscribe this topic
func (n *Network) findTopicPeers(topic string) {
	// Tell other peers I am interested in this topic
	routingDiscovery := drouting.NewRoutingDiscovery(n.dht)
	dutil.Advertise(n.ctx, routingDiscovery, topic)

	// Look for others who have announced and attempt to connect to them
	log.Printf("Searching for '%s' peers... \n", topic)

	// Find peers until connected to at least one peer
	anyConnected := false
	count := 0
	for !anyConnected {
		peerChan, err := routingDiscovery.FindPeers(n.ctx, topic)
		// peerChan, err := routingDiscovery.FindPeers(n.ctx, topic, discovery.Limit(10))
		if err != nil {
			panic(err)
		}

		for peer := range peerChan {
			if peer.ID == n.host.ID() {
				continue // No self connection
			}

			// fmt.Println("Addrs: ", peer.Addrs)
			err := n.host.Connect(n.ctx, peer)
			if err != nil {
				// fmt.Println("Failed connecting to ", peer.ID.Pretty(), ", error:", err)
			} else {
				log.Println("Connected to:", peer.ID)
				anyConnected = true
			}
		}

		count++
		if anyConnected || count > 2 {
			break
		}
	}

	log.Println("Peer discovery complete")
}

// Using sub.Next to receive message from topic
func (n *Network) JoinTopic(topicName string, isOnlyGossip bool) (*pubsub.Topic, *pubsub.Subscription, error) {
	var topic *pubsub.Topic
	var sub *pubsub.Subscription

	if isOnlyGossip {
		topicName += "-GossipOnlyTopic"
	}

	// Find Peers who subscribe this topic and connect them
	n.findTopicPeers(topicName)

	topic, err := n.ps.Join(topicName)
	if err != nil {
		return topic, sub, errors.New("join topic error")
	}

	// Set up topic score
	scoreParam := &pubsub.TopicScoreParams{
		TopicWeight:                     1,
		TimeInMeshWeight:                0.0002777, // P1
		TimeInMeshCap:                   3600,
		TimeInMeshQuantum:               time.Second,
		FirstMessageDeliveriesWeight:    1, // P2
		FirstMessageDeliveriesDecay:     0.9997,
		FirstMessageDeliveriesCap:       200,
		MeshMessageDeliveriesWeight:     -1, // P3
		MeshMessageDeliveriesActivation: time.Second,
		MeshMessageDeliveriesWindow:     10 * time.Millisecond,
		MeshMessageDeliveriesThreshold:  20,
		MeshMessageDeliveriesCap:        100,
		MeshMessageDeliveriesDecay:      0.9997,
		InvalidMessageDeliveriesWeight:  -1, // P4
		InvalidMessageDeliveriesDecay:   0.9999,
	}

	// If topic is OnlyGossip, close score about mesh
	if isOnlyGossip {
		scoreParam.TimeInMeshWeight = 0
		scoreParam.FirstMessageDeliveriesWeight = 0
		scoreParam.MeshMessageDeliveriesWeight = 0
	}

	topic.SetScoreParams(scoreParam)

	sub, err = topic.Subscribe()
	if err != nil {
		return topic, sub, errors.New("subscribe topic error")
	}

	return topic, sub, nil
}

// Content query with key (string), return Peer AddrInfo List.
func (n *Network) QueryPeersWithCID(id cid.Cid) ([]peer.AddrInfo, error) {
	var peers []peer.AddrInfo

	limit := 100 // that's just arbitrary, but FindProvidersAsync needs a count

	peerChan := n.dht.FindProvidersAsync(n.ctx, id, limit)
	// Append Peers
	for peerAddr := range peerChan {
		peers = append(peers, peerAddr)
	}

	return peers, nil
}

// Tell all nodes it has new content with key.
func (n *Network) PublishContentWithCID(id cid.Cid) error {
	// Publish the key of content
	//routingDiscovery := drouting.NewRoutingDiscovery(n.dht)
	//routingDiscovery.Provide(n.ctx, id, true)
	err := n.dht.Provide(n.ctx, id, true)
	if err != nil {
		return errors.New(err.Error() + "publish content with cid error")
	}

	return nil
}

func (n *Network) FindPeerWithID(peer peer.ID) (peer.AddrInfo, error) {

	ctx := context.Background()

	peerAddr, err := n.dht.FindPeer(ctx, peer)
	if err != nil {
		return peerAddr, errors.New("can not find peer: " + peer.String() + err.Error())
	}

	return peerAddr, nil
}

func (n *Network) GetHost() host.Host {
	return n.host
}

// Publish message to topic
func (n *Network) PublishMessage(topic *pubsub.Topic, data []byte) error {
	// Publish data to topic
	err := topic.Publish(n.ctx, data)
	if err != nil {
		return errors.New(err.Error() + "Publish data to topic error.")
	}

	return nil
}

func (n *Network) GetPeerID() string {
	return n.host.ID().String()
}

func PathExistsOrCreates(path string) {
	_, err := os.Stat(path)

	var exist = false
	if err == nil {
		exist = true
	}
	if os.IsNotExist(err) {
		exist = false
	}

	if exist {
		//fmt.Printf("has dir![%v]\n", path)
	} else {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Printf("mkdir failed![%v]\n", err)
		}
	}
}

func GetHostPrivKey(path string) (crypto.PrivKey, error) {
	var priv crypto.PrivKey
	// Read private key file
	tmpFile, err := os.ReadFile(path + "private.key")
	if err != nil {
		// log.Println("Read private key file error: ", err.Error())
	} else {
		priv, err = crypto.UnmarshalPrivateKey(tmpFile)
		if err != nil {
			log.Println("Unmarshal private key error: ", err.Error())
		} else {
			return priv, nil
		}
	}

	// If read old privKey faild, create a new random privKey
	priv, _, err = crypto.GenerateKeyPair(crypto.ECDSA, 256)
	if err != nil {
		log.Println("Generate private key error: ", err.Error())
		return priv, errors.New("generate private key error")
	}

	// Store privKey to kayPath file
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		log.Println("Marshal private key error: ", err.Error())
		return priv, errors.New("marshal private key error")
	}

	err = os.WriteFile(path+"private.key", privBytes, 0664)
	if err != nil {
		log.Println("Write private key error: ", err.Error())
		return priv, errors.New("write private key error")
	}

	return priv, err
}

func NsToCid(ns string) (cid.Cid, error) {
	h, err := mh.Sum([]byte(ns), mh.SHA2_256, -1)
	if err != nil {
		return cid.Undef, err
	}

	return cid.NewCidV1(cid.Raw, h), nil
}

func BuildBootstrapPeers(peers []string) ([]ma.Multiaddr, error) {
	var bootstrappeers []ma.Multiaddr
	for _, s := range peers {
		ma, err := ma.NewMultiaddr(s)
		if err != nil {
			continue
		}
		bootstrappeers = append(bootstrappeers, ma)
	}

	// if bootstrap peer less than 1
	if len(bootstrappeers) < 1 {
		return bootstrappeers, errors.New("there is no enough bootstrappeer")
	}

	return bootstrappeers, nil
}
