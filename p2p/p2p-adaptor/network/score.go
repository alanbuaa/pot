package network

import (
	"context"
	"sync"
	"time"

	pubsub "github.com/DXPlus/go-libp2p-pubsub-abci"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerAppState struct {
	messageDeliveries        float64
	invalidMessageDeliveries float64
}

var peerStates = make(map[peer.ID]*PeerAppState)

type ValidationFeedback struct {
	Peer   peer.ID
	Result bool
}

type AppScore struct {
	sync.Mutex
	feedbackchan                  chan *ValidationFeedback
	DecayInterval                 time.Duration
	DecayToZero                   float64
	DecayMessageDeliveries        float64
	DecayInvalidMessageDeliveries float64
	MessageDeliveriesCap          float64
}

func NewAppScore(ctx context.Context) *AppScore {
	appScore := &AppScore{
		feedbackchan:                  make(chan *ValidationFeedback),
		DecayInterval:                 time.Second,
		DecayMessageDeliveries:        0.997,
		DecayInvalidMessageDeliveries: 0.9997,
		DecayToZero:                   0.01,
		MessageDeliveriesCap:          200,
	}

	go appScore.refreshScores(ctx)

	return appScore
}

func (as *AppScore) refreshScores(ctx context.Context) {
	decayFresh := time.NewTicker(as.DecayInterval)
	defer decayFresh.Stop()

	// monitor feedbackchan
	for {
		select {
		case feed := <-as.feedbackchan:
			as.handleFeedBack(feed)
		case <-decayFresh.C:
			as.doDecay()
		case <-ctx.Done():
			logging.Logger("score").Info("score refreshScores shutting down")
			return
		}
	}
}

func (as *AppScore) doDecay() {
	// Lock to avoid modifying scores at the same time
	as.Lock()
	defer as.Unlock()

	// Decay App Score
	for peer := range peerStates {
		peerStates[peer].invalidMessageDeliveries *= as.DecayInvalidMessageDeliveries
		peerStates[peer].messageDeliveries *= as.DecayMessageDeliveries
		if peerStates[peer].invalidMessageDeliveries < as.DecayToZero {
			peerStates[peer].invalidMessageDeliveries = 0
		}
		if peerStates[peer].messageDeliveries < as.DecayToZero {
			peerStates[peer].messageDeliveries = 0
		}
	}
}

func (as *AppScore) handleFeedBack(vfb *ValidationFeedback) {
	// Lock to avoid modifying scores at the same time
	as.Lock()
	defer as.Unlock()

	// Update score
	_, ok := peerStates[vfb.Peer]
	if !ok {
		peerStates[vfb.Peer] = &PeerAppState{
			messageDeliveries:        0,
			invalidMessageDeliveries: 0,
		}
	}

	if !vfb.Result {
		peerStates[vfb.Peer].invalidMessageDeliveries++
	}

	peerStates[vfb.Peer].messageDeliveries++
	if peerStates[vfb.Peer].messageDeliveries > as.MessageDeliveriesCap {
		peerStates[vfb.Peer].messageDeliveries = as.MessageDeliveriesCap
	}
}

func GetAppSpecificScore(p peer.ID) float64 {
	// Calculate scores
	var score float64

	state, ok := peerStates[p]
	if !ok {
		score = 0
	} else {
		score = state.messageDeliveries*0.1 - state.invalidMessageDeliveries*1
	}

	return score
}

func SetupScoreParamsAndthresholds() (*pubsub.PeerScoreParams, *pubsub.PeerScoreThresholds) {
	// New PeerScoreParams and PeerScoreThresholds
	scoreParams := &pubsub.PeerScoreParams{
		AppSpecificScore:  GetAppSpecificScore,
		AppSpecificWeight: 1,
		DecayInterval:     time.Second,
		DecayToZero:       0.01,
	}

	scorethresholds := &pubsub.PeerScoreThresholds{
		GossipThreshold:   -10,
		PublishThreshold:  -100,
		GraylistThreshold: -10000,
	}

	return scoreParams, scorethresholds
}
