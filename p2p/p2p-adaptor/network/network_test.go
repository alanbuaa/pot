package network

import (
	"fmt"
	"testing"
	"time"
)

func TestNewNetwork(t *testing.T) {
	fmt.Println("start test...")
	// n, _ := NewNetwork()
	// fmt.Println(n.h)
}

func TestQueryAndFindCID(t *testing.T) {
	t.Log("test query and find cid...")

	network, _, err := NewNetwork("1234", "store/dht-store/", "store/host-key/", nil, false)
	if err != nil {
		t.Error("new network error")
	}

	id, _ := NsToCid("test" + time.Now().String())
	err = network.PublishContentWithCID(id)
	if err != nil {
		t.Error("do publish content with cid error")
	}

	peers, err := network.QueryPeersWithCID(id)
	if err != nil {
		t.Error("do query peers with cid error")
	}

	if len(peers) != 1 {
		t.Error("too many peers")
	}

	if peers[0].ID != network.host.ID() {
		t.Error("peer id must equal host id  error")
	}
}
