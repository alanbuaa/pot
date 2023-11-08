package whirly

import (
	"context"
	"encoding/hex"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/utils"
	"google.golang.org/protobuf/proto"
)

type Pacemaker interface {
	UpdateLockQC(qc *pb.QuorumCert)
	OnBeat()
	OnNextSyncView()
	OnReceiverNewView(qc *pb.QuorumCert)
	AdvanceView(viewNum uint64)
	Run(ctx context.Context)
}

type pacemakerImpl struct {
	whi    *WhirlyImpl
	notify chan Event
	log    *logrus.Entry
}

func NewPacemaker(w *WhirlyImpl, log *logrus.Entry) *pacemakerImpl {
	return &pacemakerImpl{
		whi:    w,
		notify: w.GetEvents(),
		log:    log,
	}
}

func (p *pacemakerImpl) UpdateLockQC(qc *pb.QuorumCert) {
	block, _ := p.whi.expectBlock(qc.BlockHash)
	if block == nil {
		p.log.Warn("Could not find block of new QC.")
		return
	}
	oldQCHighBlock, _ := p.whi.BlockStorage.BlockOf(p.whi.lockQC)
	if oldQCHighBlock == nil {
		p.log.Error("WhirlyBlock from the old qcHigh missing from storage.")
		return
	}

	if block.Height > oldQCHighBlock.Height {
		p.log.WithFields(logrus.Fields{
			"old":              p.whi.lockQC.ViewNum,
			"new":              qc.ViewNum,
			"lockQc.blockHash": hex.EncodeToString(qc.BlockHash),
			"bLock.Hash":       hex.EncodeToString(block.Hash),
		}).Trace("[replica_" + strconv.Itoa(int(p.whi.ID)) + "] [view_" + strconv.Itoa(int(p.whi.View.ViewNum)) + "] UpdateLockQC.")
		// p.log.Trace("[WHIRLY] UpdateLockQC.")
		p.whi.lockQC = qc
		p.whi.bLock = block
	}
}

func (p *pacemakerImpl) OnBeat() {
	go p.whi.OnPropose()
}

func (p *pacemakerImpl) OnNextSyncView() {
	// =====================================
	// == Testing the Byzantine situation ==
	// =====================================
	// if p.whi.ID == 3 {
	// 	return
	// }

	// view change
	p.whi.View.ViewNum++
	p.whi.View.Primary = p.whi.GetLeader(int64(p.whi.View.ViewNum))

	p.log.Warn("[replica_" + strconv.Itoa(int(p.whi.ID)) + "] [view_" + strconv.Itoa(int(p.whi.View.ViewNum)) + "] [WHIRLY] NewViewTimeout triggered.")
	// create a dummyNode
	dummyBlock := p.whi.CreateLeaf(p.whi.GetLock().Hash, p.whi.View.ViewNum, nil, nil)
	p.whi.SetLock(dummyBlock)
	dummyBlock.Committed = true
	_ = p.whi.BlockStorage.Put(dummyBlock)
	// create a new view msg
	newViewMsg := p.whi.NewViewMsg(p.whi.GetLockQC(), p.whi.View.ViewNum)
	// send msg
	if p.whi.ID != p.whi.GetLeader(int64(p.whi.View.ViewNum)) {
		// p.log.Warn("[replica_" + strconv.Itoa(int(p.whi.ID)) + "] [view_" + strconv.Itoa(int(p.whi.View.ViewNum)) + "] [WHIRLY] unicast in whi")
		_ = p.whi.Unicast(p.whi.GetNetworkInfo()[p.whi.GetLeader(int64(p.whi.View.ViewNum))], newViewMsg)
	} else {
		//send to self
		msgByte, err := proto.Marshal(newViewMsg)
		utils.PanicOnError(err)
		p.whi.MsgByteEntrance <- msgByte
	}
	// clean the current proposal
	// p.whi.CurExec = NewCurProposal()
	p.whi.TimeChan.HardStartTimer()
}

func (p *pacemakerImpl) OnReceiverNewView(qc *pb.QuorumCert) {
	p.whi.lock.Lock()
	defer p.whi.lock.Unlock()
	p.log.Trace("[replica_" + strconv.Itoa(int(p.whi.ID)) + "] [view_" + strconv.Itoa(int(p.whi.View.ViewNum)) + "] OnReceiveNewView.")
	p.whi.emitEvent(ReceiveNewView)
	p.UpdateLockQC(qc)
}

func (p *pacemakerImpl) AdvanceView(viewNum uint64) {
	if viewNum >= p.whi.View.ViewNum {
		p.whi.View.ViewNum = viewNum
		p.whi.View.ViewNum++
		p.whi.View.Primary = p.whi.GetLeader(int64(p.whi.View.ViewNum))
		p.log.Trace("[replica_" + strconv.Itoa(int(p.whi.ID)) + "] [view_" + strconv.Itoa(int(p.whi.View.ViewNum)) + "] advanceView success!")
	}
}

func (p *pacemakerImpl) Run(ctx context.Context) {
	if p.whi.ID == p.whi.GetLeader(int64(p.whi.View.ViewNum)) {
		go p.OnBeat()
	}
	go p.startNewViewTimeout(ctx)
	defer p.whi.TimeChan.Stop()
	// get events
	n := <-p.notify

	for {
		switch n {
		case ReceiveProposal:
			// p.log.Trace("[replica_" + strconv.Itoa(int(p.whi.ID)) + "] [view_" + strconv.Itoa(int(p.whi.View.ViewNum)) + "] HardStartTimer.")
			p.whi.TimeChan.HardStartTimer()
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
		case <-ctx.Done():
			return
		}
	}
}

func (p *pacemakerImpl) startNewViewTimeout(ctx context.Context) {
	for {
		select {
		case <-p.whi.TimeChan.Timeout():
			// To keep liveness, multiply the timeout duration by 2
			p.whi.Config.Whirly.Timeout *= 2
			// p.whi.TimeChan.SetDuration(p.whi.Config.Whirly.Timeout)
			// init timer
			p.whi.TimeChan.Init()
			// send new view msg
			p.OnNextSyncView()
		// case <-p.whi.BatchTimeChan.Timeout():
		// 	p.log.Debug("[EVENT-DRIVEN HOTSTUFF] BatchTimeout triggered")
		// 	p.whi.BatchTimeChan.Init()
		// 	go p.whi.OnPropose()
		case <-ctx.Done():
			p.whi.TimeChan.Stop()
			// p.whi.BatchTimeChan.Stop()
			return
		}
	}
}
