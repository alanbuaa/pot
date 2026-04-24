package types

import (
	"bytes"
	"sort"

	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/proto"
)

type CrosschainCheckpoint struct {
	Chain            string
	ChainID          int64
	TxHash           []byte
	BlockHeight      int64
	BlockHash        []byte
	CurrentHeight    int64
	CrosschainTxHash []byte
}

func ToCrosschainCheckpoint(checkpoint *pb.CrosschainCheckpoint) *CrosschainCheckpoint {
	if checkpoint == nil {
		return nil
	}

	return &CrosschainCheckpoint{
		Chain:            checkpoint.GetChain(),
		ChainID:          checkpoint.GetChainID(),
		TxHash:           checkpoint.GetTxHash(),
		BlockHeight:      checkpoint.GetBlockHeight(),
		BlockHash:        checkpoint.GetBlockHash(),
		CurrentHeight:    checkpoint.GetCurrentHeight(),
		CrosschainTxHash: checkpoint.GetCrosschainTxHash(),
	}
}

func ToCrosschainCheckpoints(checkpoints []*pb.CrosschainCheckpoint) []*CrosschainCheckpoint {
	res := make([]*CrosschainCheckpoint, 0, len(checkpoints))
	for _, checkpoint := range checkpoints {
		res = append(res, ToCrosschainCheckpoint(checkpoint))
	}
	return res
}

func (c *CrosschainCheckpoint) ToProto() *pb.CrosschainCheckpoint {
	if c == nil {
		return nil
	}

	return &pb.CrosschainCheckpoint{
		Chain:            c.Chain,
		ChainID:          c.ChainID,
		TxHash:           c.TxHash,
		BlockHeight:      c.BlockHeight,
		BlockHash:        c.BlockHash,
		CurrentHeight:    c.CurrentHeight,
		CrosschainTxHash: c.CrosschainTxHash,
	}
}

func CloneCrosschainCheckpoint(checkpoint *CrosschainCheckpoint) *CrosschainCheckpoint {
	if checkpoint == nil {
		return nil
	}

	return &CrosschainCheckpoint{
		Chain:            checkpoint.Chain,
		ChainID:          checkpoint.ChainID,
		TxHash:           append([]byte(nil), checkpoint.TxHash...),
		BlockHeight:      checkpoint.BlockHeight,
		BlockHash:        append([]byte(nil), checkpoint.BlockHash...),
		CurrentHeight:    checkpoint.CurrentHeight,
		CrosschainTxHash: append([]byte(nil), checkpoint.CrosschainTxHash...),
	}
}

func CloneCrosschainCheckpoints(checkpoints []*CrosschainCheckpoint) []*CrosschainCheckpoint {
	res := make([]*CrosschainCheckpoint, 0, len(checkpoints))
	for _, checkpoint := range checkpoints {
		res = append(res, CloneCrosschainCheckpoint(checkpoint))
	}
	return res
}

func SortCrosschainCheckpoints(checkpoints []*CrosschainCheckpoint) []*CrosschainCheckpoint {
	sorted := CloneCrosschainCheckpoints(checkpoints)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i] == nil || sorted[j] == nil {
			return sorted[i] != nil
		}
		if sorted[i].ChainID != sorted[j].ChainID {
			return sorted[i].ChainID < sorted[j].ChainID
		}
		if sorted[i].BlockHeight != sorted[j].BlockHeight {
			return sorted[i].BlockHeight < sorted[j].BlockHeight
		}
		if cmp := bytes.Compare(sorted[i].BlockHash, sorted[j].BlockHash); cmp != 0 {
			return cmp < 0
		}
		if cmp := bytes.Compare(sorted[i].TxHash, sorted[j].TxHash); cmp != 0 {
			return cmp < 0
		}
		if cmp := bytes.Compare(sorted[i].CrosschainTxHash, sorted[j].CrosschainTxHash); cmp != 0 {
			return cmp < 0
		}
		if sorted[i].CurrentHeight != sorted[j].CurrentHeight {
			return sorted[i].CurrentHeight < sorted[j].CurrentHeight
		}
		return sorted[i].Chain < sorted[j].Chain
	})
	return sorted
}

func MarshalExecuteHeaderDeterministic(header *ExecuteHeader) ([]byte, error) {
	if header == nil {
		return nil, nil
	}

	pbHeader := header.ToProto()
	pbHeader.Checkpoints = make([]*pb.CrosschainCheckpoint, 0)
	for _, checkpoint := range SortCrosschainCheckpoints(header.Checkpoints) {
		if checkpoint != nil {
			pbHeader.Checkpoints = append(pbHeader.Checkpoints, checkpoint.ToProto())
		}
	}

	return proto.MarshalOptions{Deterministic: true}.Marshal(pbHeader)
}

func ComputeExecuteHeadersRoot(headers []*ExecuteHeader) ([]byte, error) {
	if len(headers) == 0 {
		return crypto.NilTxsHash, nil
	}

	encoded := make([][]byte, 0, len(headers))
	for _, header := range headers {
		b, err := MarshalExecuteHeaderDeterministic(header)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, b)
	}

	root := crypto.ComputeMerkleRoot(encoded)
	if len(root) == 0 {
		return crypto.NilTxsHash, nil
	}
	return root, nil
}
