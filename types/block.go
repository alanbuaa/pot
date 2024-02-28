package types

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/reflect/protoreflect"
	"math/big"
	"time"
)

type ConsensusBlock interface {
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
	PoTProof   [][]byte
	Address    int64
	PeerId     string
	TxHash     []byte
	Hashes     []byte
	PublicKey  []byte
}

type Block struct {
	Header *Header
	Txs    []*Tx
}

func (b *Block) Hash() []byte {
	if b != nil {
		if b.GetHeader() != nil {
			return b.GetHeader().Hash()
		}
	}
	return nil
}

func (b *Block) GetHeader() *Header {
	if b.Header != nil {
		return b.Header
	}
	return nil
}

func (b *Block) GetTxs() []*Tx {
	if len(b.Txs) != 0 {
		return b.Txs
	} else {
		return nil
	}
}

func (b *Block) ToProto() *pb.Block {
	pbheader := b.GetHeader().ToProto()
	if b.Txs != nil {
		txs := make([]*pb.Tx, len(b.Txs))
		for i := 0; i < len(b.Txs); i++ {
			txs[i] = b.Txs[i].ToProto()
		}
		pbblock := &pb.Block{
			Header: pbheader,
			Txs:    txs,
		}
		return pbblock
	} else {
		pbblock := &pb.Block{
			Header: pbheader,
			Txs:    make([]*pb.Tx, 0),
		}
		return pbblock
	}
}

func ToBlock(block *pb.Block) *Block {
	pbheader := block.GetHeader()
	pbtxs := block.GetTxs()

	header := ToHeader(pbheader)
	txs := ToTxs(pbtxs)
	return &Block{
		Header: header,
		Txs:    txs,
	}
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
		address, peeridbyte, b.TxHash, b.PublicKey,
	}, []byte(""))
	hashes := crypto.Hash(hashinput)
	b.Hashes = hashes
	return hashes

}

func (b *Header) BasicVerify() (bool, error) {
	// TODO: compare with genesis block
	if b.Height == 0 {
		return true, nil
	}
	if b.ParentHash == nil {
		return false, fmt.Errorf("the block without parent hash")
	}
	if len(b.ParentHash) != crypto.Hashlen {
		return false, fmt.Errorf("the block parent hash length is not legal")
	}
	if b.Difficulty.Cmp(common.Big0) < 0 {
		return false, fmt.Errorf("the block difficulty is negative")
	}
	if b.Difficulty.Cmp(common.Big0) == 0 {
		return b.CheckNilBlock()
	}
	if !b.CheckMixdigest() {
		return false, fmt.Errorf("the mixdigest of block is wrong")
	}
	flag, err := b.CheckHash()
	if !flag {
		return false, err
	}
	return true, nil
}

func (b *Header) CheckMixdigest() bool {
	unclehash := make([]byte, 0)
	for i := 0; i < len(b.UncleHash); i++ {
		if len(b.UncleHash[i]) != crypto.Hashlen {
			return false
		}
		unclehash = append(unclehash, b.UncleHash[i]...)
	}
	// mixdigest verify
	tmp := new(big.Int)
	tmp.Set(b.Difficulty)
	difficultybyte := tmp.Bytes()
	idbyte := []byte(b.PeerId)
	tmp.SetInt64(int64(b.Height))
	epochbyte := tmp.Bytes()
	hashinput := bytes.Join([][]byte{epochbyte, b.ParentHash, unclehash, difficultybyte, idbyte}, []byte(""))
	res := crypto.Hash(hashinput)
	if !bytes.Equal(res[:], b.Mixdigest) {
		return false
	}
	return true
}

func (b *Header) CheckHash() (bool, error) {
	if b.Hashes == nil {
		return false, fmt.Errorf("the block without hash")
	}

	difficulty := b.Difficulty.Bytes()
	tmp := new(big.Int)
	height := tmp.SetUint64(b.Height).Bytes()
	nonce := tmp.SetInt64(b.Nonce).Bytes()
	address := tmp.SetInt64(b.Address).Bytes()

	timestamp, err := b.Timestamp.MarshalJSON()
	if err != nil {
		return false, fmt.Errorf("the timestamp marshal failed")
	}

	unclehash := make([]byte, 0)
	for i := 0; i < len(b.UncleHash); i++ {
		if len(b.UncleHash[i]) != crypto.Hashlen {
			return false, fmt.Errorf("the unclehash length is not legal")
		}
		unclehash = append(unclehash, b.UncleHash[i]...)
	}

	peeridbyte := []byte(b.PeerId)

	hashinput := bytes.Join([][]byte{
		height, b.ParentHash, unclehash,
		b.Mixdigest, difficulty, nonce,
		timestamp, b.PoTProof[0], b.PoTProof[1],
		address, peeridbyte, b.TxHash, b.PublicKey,
	}, []byte(""))

	hashes := crypto.Hash(hashinput)
	if !bytes.Equal(hashes, b.Hashes) {
		return false, fmt.Errorf("the block hash check failed")
	}

	return true, nil
}

func (b *Header) CheckNilBlock() (bool, error) {
	if len(b.ParentHash) != crypto.Hashlen {
		return false, fmt.Errorf("the parent hash length is not legal")
	}
	if b.Nonce != 0 {
		return false, fmt.Errorf("the nil block nonce should be zero")
	}
	if !bytes.Equal(crypto.NilTxsHash, b.TxHash) {
		return false, fmt.Errorf("the nil block txhash should be niltxhash")
	}
	return true, nil
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
		PublicKey:  header.GetPubkey(),
		TxHash:     header.GetTxhash(),
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
		Pubkey:     b.PublicKey,
		Txhash:     b.TxHash,
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
		PublicKey:  []byte(""),
		TxHash:     crypto.NilTxsHash,
	}
	h.Hash()
	return h
}

func DefaultGenesisBlock() *Block {
	return &Block{
		Header: DefaultGenesisHeader(),
		Txs:    TestTxs(),
	}
}

func RandByte() []byte {
	res := make([]byte, 32)
	_, _ = rand.Read(res)
	return res
}

func (b *Block) GetExcutedTx() []*ExecutedTxData {
	txs := b.GetTxs()
	excutedtx := make([]*ExecutedTxData, 0)
	if txs != nil {
		for i := 0; i < len(txs); i++ {
			if txs[i].GetTxType() == pb.TxDataType_ExcutedTx {
				excuteddata := txs[i].GetExcutedTxData()
				if excuteddata != nil {
					excutedtx = append(excutedtx, excuteddata)
				}
			}
		}
	}
	return excutedtx
}

func (b *Block) GetRawTx() []*RawTx {
	txs := b.GetTxs()
	rawtxs := make([]*RawTx, 0)
	if txs != nil {
		for i := 0; i < len(txs); i++ {
			if txs[i].GetTxType() == pb.TxDataType_RawTx {
				rawtx := txs[i].GetRawTxData()
				if rawtx != nil {
					rawtxs = append(rawtxs, rawtx)
				}
			}
		}
	}
	return rawtxs
}
