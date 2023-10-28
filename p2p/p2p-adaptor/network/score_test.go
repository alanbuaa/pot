package network

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestAppScoreForNetwork(t *testing.T) {

	// Setup scoreParams and scorethresholds
	appScore := &AppScore{
		feedbackchan:                  make(chan *ValidationFeedback),
		DecayInterval:                 time.Second,
		DecayMessageDeliveries:        0.997,
		DecayInvalidMessageDeliveries: 0.9997,
		DecayToZero:                   0.01,
		MessageDeliveriesCap:          200,
	}

	peerStates = make(map[peer.ID]*PeerAppState)
	ctx := context.Background()
	go appScore.refreshScores(ctx)

	peerA := peer.ID("A")
	for i := 0; i < 20; i++ {
		var feed *ValidationFeedback
		if i < 10 {
			feed = &ValidationFeedback{
				Peer:   peerA,
				Result: false,
			}
		} else {
			feed = &ValidationFeedback{
				Peer:   peerA,
				Result: true,
			}
		}
		appScore.feedbackchan <- feed
	}

	time.Sleep(time.Second * 5)
	expected := 20*0.1 - 10*1
	aScore := peerStates[peerA].messageDeliveries*0.1 - peerStates[peerA].invalidMessageDeliveries*1
	t.Logf("1: %f 2: %f", peerStates[peerA].messageDeliveries, peerStates[peerA].invalidMessageDeliveries)
	if aScore > expected {
		t.Fatalf("Score: %f > Expected: %f", aScore, expected)
	}

	for i := 0; i < 20; i++ {
		feed := &ValidationFeedback{
			Peer:   peerA,
			Result: true,
		}
		appScore.feedbackchan <- feed
	}
	time.Sleep(time.Second * 2)
	bScore := peerStates[peerA].messageDeliveries*0.1 - peerStates[peerA].invalidMessageDeliveries*1
	if bScore < aScore {
		t.Fatalf("bScore: %f < aScore: %f", bScore, aScore)
	}
}
