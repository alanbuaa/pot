package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/internal/storage"
	"github.com/zzz136454872/upgradeable-consensus/types"
)

// VotingManager 投票管理器
// 管理升级提案的投票过程
type VotingManager struct {
	proposals map[types.TxHash]*ProposalVote
	storage   storage.UpgradeStorage
	mu        sync.RWMutex
	log       *logrus.Entry
}

// ProposalVote 提案投票信息
type ProposalVote struct {
	Proposal      *UpgradeProposal
	Votes         map[int64]*Vote    // 节点ID -> 投票
	VoteCount     map[VoteOption]int // 投票选项 -> 数量
	Status        VoteStatus
	StartTime     time.Time
	EndTime       time.Time
	VotingPeriod  time.Duration
	QuorumReached bool
}

// Vote 单个投票
type Vote struct {
	NodeID    int64
	Option    VoteOption
	Timestamp time.Time
	Signature []byte
}

// VoteOption 投票选项
type VoteOption int

const (
	VoteAbstain VoteOption = iota // 弃权
	VoteYes                       // 赞成
	VoteNo                        // 反对
)

func (v VoteOption) String() string {
	switch v {
	case VoteYes:
		return "Yes"
	case VoteNo:
		return "No"
	case VoteAbstain:
		return "Abstain"
	default:
		return "Unknown"
	}
}

// VoteStatus 投票状态
type VoteStatus int

const (
	VoteStatusPending  VoteStatus = iota // 进行中
	VoteStatusPassed                     // 通过
	VoteStatusRejected                   // 拒绝
	VoteStatusExpired                    // 过期
)

func (v VoteStatus) String() string {
	switch v {
	case VoteStatusPending:
		return "Pending"
	case VoteStatusPassed:
		return "Passed"
	case VoteStatusRejected:
		return "Rejected"
	case VoteStatusExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// NewVotingManager 创建投票管理器
func NewVotingManager(log *logrus.Entry) *VotingManager {
	return NewVotingManagerWithStorage(nil, log)
}

// NewVotingManagerWithStorage 创建带存储的投票管理器
func NewVotingManagerWithStorage(upgradeStorage storage.UpgradeStorage, log *logrus.Entry) *VotingManager {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	vm := &VotingManager{
		proposals: make(map[types.TxHash]*ProposalVote),
		storage:   upgradeStorage,
		log:       log,
	}

	// 如果有存储，尝试恢复活跃投票
	if upgradeStorage != nil {
		// TODO: 实现投票恢复逻辑
		vm.log.Info("VotingManager initialized with persistent storage")
	}

	return vm
}

// StartVoting 启动投票
func (vm *VotingManager) StartVoting(proposal *UpgradeProposal, votingPeriod time.Duration) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.proposals[proposal.ProposalID]; exists {
		return fmt.Errorf("voting already started for this proposal")
	}

	proposalVote := &ProposalVote{
		Proposal:     proposal,
		Votes:        make(map[int64]*Vote),
		VoteCount:    make(map[VoteOption]int),
		Status:       VoteStatusPending,
		StartTime:    time.Now(),
		VotingPeriod: votingPeriod,
		EndTime:      time.Now().Add(votingPeriod),
	}

	vm.proposals[proposal.ProposalID] = proposalVote

	vm.log.WithFields(logrus.Fields{
		"proposal_id":   proposal.ProposalID.String(),
		"voting_period": votingPeriod,
		"end_time":      proposalVote.EndTime,
	}).Info("Started voting for proposal")

	// 启动自动检查goroutine
	go vm.monitorVoting(proposal.ProposalID)

	return nil
}

// CastVote 投票
func (vm *VotingManager) CastVote(proposalID types.TxHash, nodeID int64, option VoteOption, signature []byte) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	proposalVote, exists := vm.proposals[proposalID]
	if !exists {
		return fmt.Errorf("proposal not found")
	}

	if proposalVote.Status != VoteStatusPending {
		return fmt.Errorf("voting is not active (status: %s)", proposalVote.Status.String())
	}

	// 检查是否已投票
	if _, voted := proposalVote.Votes[nodeID]; voted {
		return fmt.Errorf("node %d has already voted", nodeID)
	}

	// 检查投票期限
	if time.Now().After(proposalVote.EndTime) {
		proposalVote.Status = VoteStatusExpired
		return fmt.Errorf("voting period has expired")
	}

	// 记录投票
	vote := &Vote{
		NodeID:    nodeID,
		Option:    option,
		Timestamp: time.Now(),
		Signature: signature,
	}

	proposalVote.Votes[nodeID] = vote
	proposalVote.VoteCount[option]++

	vm.log.WithFields(logrus.Fields{
		"proposal_id": proposalID.String(),
		"node_id":     nodeID,
		"option":      option.String(),
		"total_votes": len(proposalVote.Votes),
	}).Info("Vote cast")

	// 持久化投票记录
	if vm.storage != nil {
		voteRecord := &storage.VoteRecord{
			ProposalID: proposalID,
			NodeID:     nodeID,
			Option:     int(option),
			Timestamp:  vote.Timestamp,
			Signature:  signature,
		}
		if err := vm.storage.StoreVote(proposalID, voteRecord); err != nil {
			vm.log.WithError(err).Error("Failed to persist vote")
		}
	}

	// 检查是否达到法定人数
	vm.checkQuorum(proposalVote)

	return nil
}

// checkQuorum 检查是否达到法定人数
func (vm *VotingManager) checkQuorum(proposalVote *ProposalVote) {
	totalNodes := int64(proposalVote.Proposal.Threshold) // 使用提案的阈值作为总节点数
	if totalNodes == 0 {
		totalNodes = 7 // 默认7个节点
	}

	yesVotes := proposalVote.VoteCount[VoteYes]
	noVotes := proposalVote.VoteCount[VoteNo]
	totalVotes := len(proposalVote.Votes)

	// 检查是否所有节点都已投票
	if int64(totalVotes) >= totalNodes {
		proposalVote.QuorumReached = true
	}

	// 计算是否通过（需要超过2/3的赞成票）
	quorum := (totalNodes * 2) / 3
	if int64(yesVotes) > quorum {
		proposalVote.Status = VoteStatusPassed
		proposalVote.QuorumReached = true
		vm.log.WithField("proposal_id", proposalVote.Proposal.ProposalID.String()).Info("Proposal passed")
		vm.persistProposalStatus(proposalVote)
	} else if int64(noVotes) > totalNodes-quorum {
		// 反对票过多，提案被拒绝
		proposalVote.Status = VoteStatusRejected
		vm.log.WithField("proposal_id", proposalVote.Proposal.ProposalID.String()).Info("Proposal rejected")
		vm.persistProposalStatus(proposalVote)
	}
}

// monitorVoting 监控投票进度
func (vm *VotingManager) monitorVoting(proposalID types.TxHash) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			vm.mu.Lock()
			proposalVote, exists := vm.proposals[proposalID]
			if !exists {
				vm.mu.Unlock()
				return
			}

			// 检查投票是否已结束
			if proposalVote.Status != VoteStatusPending {
				vm.mu.Unlock()
				return
			}

			// 检查是否过期
			if time.Now().After(proposalVote.EndTime) {
				if !proposalVote.QuorumReached {
					proposalVote.Status = VoteStatusExpired
					vm.log.WithField("proposal_id", proposalID.String()).Warn("Voting period expired")
				}
				vm.mu.Unlock()
				return
			}

			vm.mu.Unlock()
		}
	}
}

// GetVotingResult 获取投票结果
func (vm *VotingManager) GetVotingResult(proposalID types.TxHash) (*VotingResult, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	proposalVote, exists := vm.proposals[proposalID]
	if !exists {
		return nil, fmt.Errorf("proposal not found")
	}

	return &VotingResult{
		ProposalID:    proposalID,
		Status:        proposalVote.Status,
		YesVotes:      proposalVote.VoteCount[VoteYes],
		NoVotes:       proposalVote.VoteCount[VoteNo],
		AbstainVotes:  proposalVote.VoteCount[VoteAbstain],
		TotalVotes:    len(proposalVote.Votes),
		QuorumReached: proposalVote.QuorumReached,
		StartTime:     proposalVote.StartTime,
		EndTime:       proposalVote.EndTime,
	}, nil
}

// VotingResult 投票结果
type VotingResult struct {
	ProposalID    types.TxHash
	Status        VoteStatus
	YesVotes      int
	NoVotes       int
	AbstainVotes  int
	TotalVotes    int
	QuorumReached bool
	StartTime     time.Time
	EndTime       time.Time
}

// ListActiveVotings 列出所有活跃的投票
func (vm *VotingManager) ListActiveVotings() []*VotingResult {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	results := make([]*VotingResult, 0)
	for proposalID, proposalVote := range vm.proposals {
		if proposalVote.Status == VoteStatusPending {
			result := &VotingResult{
				ProposalID:    proposalID,
				Status:        proposalVote.Status,
				YesVotes:      proposalVote.VoteCount[VoteYes],
				NoVotes:       proposalVote.VoteCount[VoteNo],
				AbstainVotes:  proposalVote.VoteCount[VoteAbstain],
				TotalVotes:    len(proposalVote.Votes),
				QuorumReached: proposalVote.QuorumReached,
				StartTime:     proposalVote.StartTime,
				EndTime:       proposalVote.EndTime,
			}
			results = append(results, result)
		}
	}

	return results
}

// HasVoted 检查节点是否已投票
func (vm *VotingManager) HasVoted(proposalID types.TxHash, nodeID int64) bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	proposalVote, exists := vm.proposals[proposalID]
	if !exists {
		return false
	}

	_, voted := proposalVote.Votes[nodeID]
	return voted
}

// GetVote 获取特定节点的投票
func (vm *VotingManager) GetVote(proposalID types.TxHash, nodeID int64) (*Vote, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	proposalVote, exists := vm.proposals[proposalID]
	if !exists {
		return nil, fmt.Errorf("proposal not found")
	}

	vote, voted := proposalVote.Votes[nodeID]
	if !voted {
		return nil, fmt.Errorf("node has not voted")
	}

	return vote, nil
}

// persistProposalStatus 持久化提案投票状态
func (vm *VotingManager) persistProposalStatus(proposalVote *ProposalVote) {
	if vm.storage == nil {
		return
	}

	status := &storage.ProposalVoteStatus{
		ProposalID:    proposalVote.Proposal.ProposalID,
		Status:        int(proposalVote.Status),
		StartTime:     proposalVote.StartTime,
		EndTime:       proposalVote.EndTime,
		VotingPeriod:  int64(proposalVote.VotingPeriod),
		YesCount:      proposalVote.VoteCount[VoteYes],
		NoCount:       proposalVote.VoteCount[VoteNo],
		AbstainCount:  proposalVote.VoteCount[VoteAbstain],
		QuorumReached: proposalVote.QuorumReached,
	}

	if err := vm.storage.StoreProposalStatus(proposalVote.Proposal.ProposalID, status); err != nil {
		vm.log.WithError(err).Error("Failed to persist proposal status")
	}
}

// LoadProposalStatus 从存储加载提案状态
func (vm *VotingManager) LoadProposalStatus(proposalID types.TxHash) (*ProposalVote, error) {
	if vm.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	status, err := vm.storage.GetProposalStatus(proposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to load proposal status: %w", err)
	}

	// 加载投票记录
	voteRecords, err := vm.storage.GetProposalVotes(proposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to load votes: %w", err)
	}

	votes := make(map[int64]*Vote)
	for _, record := range voteRecords {
		votes[record.NodeID] = &Vote{
			NodeID:    record.NodeID,
			Option:    VoteOption(record.Option),
			Timestamp: record.Timestamp,
			Signature: record.Signature,
		}
	}

	proposalVote := &ProposalVote{
		Proposal:      nil, // 需要单独加载提案数据
		Votes:         votes,
		VoteCount:     make(map[VoteOption]int),
		Status:        VoteStatus(status.Status),
		StartTime:     status.StartTime,
		EndTime:       status.EndTime,
		VotingPeriod:  time.Duration(status.VotingPeriod),
		QuorumReached: status.QuorumReached,
	}

	proposalVote.VoteCount[VoteYes] = status.YesCount
	proposalVote.VoteCount[VoteNo] = status.NoCount
	proposalVote.VoteCount[VoteAbstain] = status.AbstainCount

	return proposalVote, nil
}

// VerifyVoteSignature 验证投票签名
func (vm *VotingManager) VerifyVoteSignature(proposalID types.TxHash, nodeID int64, signature []byte) error {
	// TODO: 实现实际的签名验证逻辑
	// 当前版本仅检查签名是否存在
	if len(signature) == 0 {
		return fmt.Errorf("empty signature")
	}

	// 实际实现应该:
	// 1. 获取节点的公钥
	// 2. 构造待签名消息（proposalID + nodeID + option）
	// 3. 使用公钥验证签名
	// 示例:
	// pubKey := vm.getNodePublicKey(nodeID)
	// message := constructVoteMessage(proposalID, nodeID, option)
	// return crypto.Verify(pubKey, message, signature)

	vm.log.WithFields(logrus.Fields{
		"proposal_id": proposalID.String(),
		"node_id":     nodeID,
	}).Debug("Signature verification placeholder - accepting all signatures")

	return nil
}
