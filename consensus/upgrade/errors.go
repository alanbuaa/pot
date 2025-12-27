package upgrade

import "errors"

var (
	// ErrInvalidProposal 无效提案
	ErrInvalidProposal = errors.New("invalid upgrade proposal")

	// ErrInvalidSignature 无效签名
	ErrInvalidSignature = errors.New("invalid signature")

	// ErrInsufficientSignatures 签名不足
	ErrInsufficientSignatures = errors.New("insufficient signatures")

	// ErrInvalidHeight 无效高度
	ErrInvalidHeight = errors.New("invalid height")

	// ErrInvalidCDL 无效 CDL
	ErrInvalidCDL = errors.New("invalid CDL descriptor")

	// ErrPreexecNotActive 预执行未激活
	ErrPreexecNotActive = errors.New("preexecution not active")

	// ErrAlreadySwitched 已经切换
	ErrAlreadySwitched = errors.New("already switched")

	// ErrNotReady 未准备好
	ErrNotReady = errors.New("not ready")

	// ErrStorageFailure 存储失败
	ErrStorageFailure = errors.New("storage failure")

	// ErrInvalidBlock 无效区块
	ErrInvalidBlock = errors.New("invalid block")

	// ErrMetricsFailure 指标收集失败
	ErrMetricsFailure = errors.New("metrics collection failure")
)
