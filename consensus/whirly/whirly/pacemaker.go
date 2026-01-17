package whirly

import (
	"context"
	"encoding/hex"

	"github.com/sirupsen/logrus"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
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
	p.log.Trace("Updating lock QC")
	block, _ := p.whi.expectBlock(qc.BlockHash)
	if block == nil {
		p.log.WithField("block_hash", hex.EncodeToString(qc.BlockHash)).Warn("Block of new QC not found in storage")
		return
	}
	oldQCHighBlock, _ := p.whi.BlockStorage.BlockOf(p.whi.lockQC)
	if oldQCHighBlock == nil {
		p.log.Error("Previous lock QC block missing from storage")
		return
	}

	if block.Height > oldQCHighBlock.Height {
		p.log.WithFields(logrus.Fields{
			"replica_id":       p.whi.ID,
			"view":             p.whi.View.ViewNum,
			"old_qc_view":      p.whi.lockQC.ViewNum,
			"new_qc_view":      qc.ViewNum,
			"new_qc_hash":      hex.EncodeToString(qc.BlockHash),
			"new_block_hash":   hex.EncodeToString(block.Hash),
			"new_block_height": block.Height,
		}).Debug("Lock QC updated to higher block")
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

	p.log.WithFields(logrus.Fields{
		"replica_id": p.whi.ID,
		"old_view":   p.whi.View.ViewNum - 1,
		"new_view":   p.whi.View.ViewNum,
		"new_leader": p.whi.View.Primary,
	}).Info("View timeout triggered, advancing to new view")
	// create a dummyNode
	dummyBlock := p.whi.CreateLeaf(p.whi.GetLock().Hash, p.whi.View.ViewNum, nil, nil, nil)
	p.whi.SetLock(dummyBlock)
	dummyBlock.Committed = true
	_ = p.whi.BlockStorage.Put(dummyBlock)
	// create a new view msg
	newViewMsg := p.whi.NewViewMsg(p.whi.GetLockQC(), p.whi.View.ViewNum)
	// send msg
	if p.whi.ID != p.whi.GetLeader(int64(p.whi.View.ViewNum)) {
		// p.log.WithFields(logrus.Fields{"replica": p.whi.ID, "view": p.whi.View.ViewNum}).Warn("WHIRLY unicast in whi")
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
	p.log.WithFields(logrus.Fields{
		"replica_id": p.whi.ID,
		"view":       p.whi.View.ViewNum,
		"qc_view":    qc.ViewNum,
	}).Trace("Received new view message")
	p.whi.emitEvent(ReceiveNewView)
	p.UpdateLockQC(qc)
}

func (p *pacemakerImpl) AdvanceView(viewNum uint64) {
	if viewNum >= p.whi.View.ViewNum {
		oldView := p.whi.View.ViewNum
		p.whi.View.ViewNum = viewNum
		p.whi.View.ViewNum++
		p.whi.View.Primary = p.whi.GetLeader(int64(p.whi.View.ViewNum))
		p.log.WithFields(logrus.Fields{
			"replica_id": p.whi.ID,
			"old_view":   oldView,
			"new_view":   p.whi.View.ViewNum,
			"new_leader": p.whi.View.Primary,
		}).Debug("Advanced to new view")
	}
}

func (p *pacemakerImpl) Run(ctx context.Context) {
	p.log.WithFields(logrus.Fields{
		"replica_id": p.whi.ID,
		"view":       p.whi.View.ViewNum,
		"is_leader":  p.whi.ID == p.whi.GetLeader(int64(p.whi.View.ViewNum)),
	}).Debug("Pacemaker starting")
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
			p.log.WithFields(logrus.Fields{
				"replica_id": p.whi.ID,
				"view":       p.whi.View.ViewNum,
				"event":      "ReceiveProposal",
			}).Trace("Restarting view timer")
			p.whi.TimeChan.HardStartTimer()
		case QCFinish:
			p.log.WithFields(logrus.Fields{
				"replica_id": p.whi.ID,
				"view":       p.whi.View.ViewNum,
				"event":      "QCFinish",
			}).Trace("QC formed, triggering proposal")
			p.OnBeat()
		case ReceiveNewView:
			p.log.WithFields(logrus.Fields{
				"replica_id": p.whi.ID,
				"view":       p.whi.View.ViewNum,
				"event":      "ReceiveNewView",
			}).Trace("New view confirmed, triggering proposal")
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
			oldTimeout := p.whi.Config.Whirly.Timeout
			p.whi.Config.Whirly.Timeout *= 2
			p.log.WithFields(logrus.Fields{
				"replica_id":  p.whi.ID,
				"view":        p.whi.View.ViewNum,
				"old_timeout": oldTimeout,
				"new_timeout": p.whi.Config.Whirly.Timeout,
			}).Debug("View timeout occurred, increasing timeout duration")
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
