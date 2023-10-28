package eventdriven

import (
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/hotstuff"
	"github.com/zzz136454872/upgradeable-consensus/pb"
)

type Pacemaker interface {
	UpdateHighQC(qcHigh *pb.QuorumCert)
	OnBeat()
	OnNextSyncView()
	OnReceiverNewView(qc *pb.QuorumCert)
	Run(closed chan []byte)
}

type pacemakerImpl struct {
	ehs    *EventDrivenHotStuffImpl
	notify chan Event
	log    *logrus.Entry
}

func NewPacemaker(e *EventDrivenHotStuffImpl, log *logrus.Entry) *pacemakerImpl {
	return &pacemakerImpl{
		ehs:    e,
		notify: e.GetEvents(),
		log:    log,
	}
}

func (p *pacemakerImpl) UpdateHighQC(qcHigh *pb.QuorumCert) {
	p.log.Debug("[EVENT-DRIVEN HOTSTUFF] UpdateHighQC.")
	block, _ := p.ehs.expectBlock(qcHigh.BlockHash)
	if block == nil {
		p.log.Warn("Could not find block of new QC.")
		return
	}
	oldQCHighBlock, _ := p.ehs.BlockStorage.BlockOf(p.ehs.qcHigh)
	if oldQCHighBlock == nil {
		p.log.Error("Block from the old qcHigh missing from storage.")
		return
	}

	if block.Height > oldQCHighBlock.Height {
		p.ehs.qcHigh = qcHigh
		p.ehs.bLeaf = block
	}
}

func (p *pacemakerImpl) OnBeat() {
	go p.ehs.OnPropose()
}

func (p *pacemakerImpl) OnNextSyncView() {
	p.log.Warn("[EVENT-DRIVEN HOTSTUFF] NewViewTimeout triggered.")
	// view change
	p.ehs.View.ViewNum++
	p.ehs.View.Primary = p.ehs.GetLeader()
	// create a dummyNode
	dummyBlock := p.ehs.CreateLeaf(p.ehs.GetLeaf().Hash, nil, nil)
	p.ehs.SetLeaf(dummyBlock)
	dummyBlock.Committed = true
	_ = p.ehs.BlockStorage.Put(dummyBlock)
	// create a new view msg
	newViewMsg := p.ehs.Msg(pb.MsgType_NEWVIEW, nil, p.ehs.GetHighQC())
	// send msg
	if p.ehs.ID != p.ehs.GetLeader() {
		_ = p.ehs.Unicast(p.ehs.GetNetworkInfo()[p.ehs.GetLeader()], newViewMsg)
	}
	// clean the current proposal
	p.ehs.CurExec = hotstuff.NewCurProposal()
	p.ehs.TimeChan.HardStartTimer()
}

func (p *pacemakerImpl) OnReceiverNewView(qc *pb.QuorumCert) {
	p.ehs.lock.Lock()
	defer p.ehs.lock.Unlock()
	p.log.Debug("[EVENT-DRIVEN HOTSTUFF] OnReceiveNewView.")
	p.ehs.emitEvent(ReceiveNewView)
	p.UpdateHighQC(qc)
}

func (p *pacemakerImpl) Run(closed chan []byte) {
	if p.ehs.ID == p.ehs.GetLeader() {
		go p.OnBeat()
	}
	go p.startNewViewTimeout(closed)
	defer p.ehs.TimeChan.Stop()
	defer p.ehs.BatchTimeChan.Stop()
	// get events
	n := <-p.notify

	for {
		switch n {
		case ReceiveProposal:
			p.ehs.TimeChan.HardStartTimer()
		case QCFinish:
			p.OnBeat()
		case ReceiveNewView:
			p.OnBeat()
		}

		var ok bool
		select {
		case n, ok = <-p.notify:
			if !ok {
				return
			}
		case <-closed:
			return
		}
	}
}

func (p *pacemakerImpl) startNewViewTimeout(closed chan []byte) {
	for {
		select {
		case <-p.ehs.TimeChan.Timeout():
			// To keep liveness, multiply the timeout duration by 2
			p.ehs.Config.HotStuff.Timeout *= 2
			// init timer
			p.ehs.TimeChan.Init()
			// send new view msg
			p.OnNextSyncView()
		case <-p.ehs.BatchTimeChan.Timeout():
			p.log.Debug("[EVENT-DRIVEN HOTSTUFF] BatchTimeout triggered")
			p.ehs.BatchTimeChan.Init()
			go p.ehs.OnPropose()
		case <-closed:
			p.ehs.TimeChan.Stop()
			p.ehs.BatchTimeChan.Stop()
			return
		}
	}
}
