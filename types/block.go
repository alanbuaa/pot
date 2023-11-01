package types

import (
	"bytes"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/reflect/protoreflect"
	"math/big"
	"time"
)

type Block interface {
	GetTxs() [][]byte
	protoreflect.ProtoMessage
}
type Header struct {
	Height     uint64
	ParentHash []byte
	UncleHash  [][]byte

	Mixdigest  []byte
	Difficulty *big.Int
	Nonce      int64
	Timestamp  time.Time

	PoTProof [][]byte
	Address  int64
	Hashes   []byte
	PeerId   string
}

func (b *Header) Hash() []byte {
	if b.Hashes != nil {
		return b.Hashes
	}
	difficulty := b.Difficulty.Bytes()
	tmp := new(big.Int)
	height := tmp.SetUint64(b.Height).Bytes()
	nonce := tmp.SetInt64(b.Nonce).Bytes()
	address := tmp.SetInt64(b.Address).Bytes()
	timestamp, err := b.Timestamp.MarshalJSON()
	if err != nil {
		return nil
	}
	unclehash := make([]byte, 0)
	for i := 0; i < len(b.UncleHash); i++ {
		unclehash = append(unclehash, b.UncleHash[i]...)
	}
	peeridbyte := []byte(b.PeerId)
	hashinput := bytes.Join([][]byte{
		height, b.ParentHash, unclehash,
		b.Mixdigest, difficulty, nonce,
		timestamp, b.PoTProof[0], b.PoTProof[1],
		address, peeridbyte,
	}, []byte(""))
	hashes := crypto.Hash(hashinput)
	b.Hashes = hashes
	return hashes
}

func (b *Header) Verify() bool {
	// TODO: compare with genesis block
	if b.Height == 0 {
		return true
	}

	return true
}

func ToHeader(header *pb.Header) *Header {
	var timestamp time.Time
	err := timestamp.UnmarshalJSON(header.Timestamp)
	if err != nil {
		return nil
	}
	var bigint big.Int
	difficulty := bigint.SetBytes(header.Difficulty)
	//var Evidence Evidence
	//Evidence.FromByte(header.Evidence)
	h := &Header{
		Height:     header.GetHeight(),
		ParentHash: header.GetParentHash(),
		UncleHash:  header.GetUncleHash(),
		Mixdigest:  header.GetMixdigest(),
		Difficulty: difficulty,
		Nonce:      header.GetNonce(),
		Timestamp:  timestamp,
		PoTProof:   header.PoTProof,
		Address:    header.GetAddress(),
		Hashes:     header.GetHashes(),
		PeerId:     header.GetPeerId(),
	}
	return h
}

func (b *Header) ToProto() *pb.Header {
	ts, err := b.Timestamp.MarshalJSON()
	if err != nil {
		return nil
	}
	if b.Hashes == nil {
		b.Hash()
	}

	return &pb.Header{
		Height:     b.Height,
		ParentHash: b.ParentHash,
		UncleHash:  b.UncleHash,
		Mixdigest:  b.Mixdigest,
		Difficulty: b.Difficulty.Bytes(),
		Nonce:      b.Nonce,
		Timestamp:  ts,
		PoTProof:   b.PoTProof,
		Address:    b.Address,
		Hashes:     b.Hashes,
		PeerId:     b.PeerId,
	}
}

func DefaultGenesisHeader() *Header {
	h := &Header{
		Height:     0,
		ParentHash: nil,
		UncleHash:  nil,
		Mixdigest:  nil,
		Difficulty: big.NewInt(1),
		Nonce:      0,
		Timestamp:  time.Date(2023, 8, 14, 15, 35, 00, 0, time.Local),
		PoTProof:   [][]byte{crypto.Hash([]byte("aa")), []byte{}},
		Address:    0,
		Hashes:     nil,
		PeerId:     "0",
	}
	h.Hash()
	return h
}

type PoTBlock struct {
	Header Header
	//Data   Data
	Txs [][]byte
}
