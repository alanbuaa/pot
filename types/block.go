package types

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ConsensusBlock interface {
	GetTxs() [][]byte
	GetShardingName() []byte
	GetIncentive() []byte
	protoreflect.ProtoMessage
}

type Header struct {
	Height         uint64
	ParentHash     []byte
	UncleHash      [][]byte
	Mixdigest      []byte
	Difficulty     *big.Int
	Nonce          int64
	Timestamp      time.Time
	PoTProof       [][]byte
	Address        int64
	PeerId         string
	TxHash         []byte
	ExeHash        []byte
	Hashes         []byte
	PublicKey      []byte
	CryptoElement  CryptoElement
	CommiteePubkey []byte
}

type Block struct {
	Header     *Header
	Txs        []*Tx
	ExeHeaders []*ExecuteHeader
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
	if b == nil {
		return nil
	}
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

func (b *Block) Copy() *Block {
	h := b.GetHeader()
	header := &Header{
		Height:         h.Height,
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Mixdigest:      h.Mixdigest,
		Difficulty:     big.NewInt(h.Difficulty.Int64()),
		Nonce:          h.Nonce,
		Timestamp:      h.Timestamp,
		PoTProof:       h.PoTProof,
		Address:        h.Address,
		PeerId:         h.PeerId,
		TxHash:         h.TxHash,
		ExeHash:        h.ExeHash,
		Hashes:         h.Hashes,
		PublicKey:      h.PublicKey,
		CryptoElement:  h.CryptoElement,
		CommiteePubkey: h.CommiteePubkey,
	}
	txs := make([]*Tx, len(b.Txs))
	copy(txs, b.Txs)
	exeheader := make([]*ExecuteHeader, len(b.ExeHeaders))
	copy(exeheader, b.ExeHeaders)
	return &Block{
		Header:     header,
		Txs:        txs,
		ExeHeaders: exeheader,
	}

}
func (b *Block) ToProto() *pb.Block {
	if b == nil {
		return nil
	}

	newb := b.Copy()

	pbheader := newb.GetHeader().ToProto()
	pbtxs := make([]*pb.Tx, 0)
	for _, tx := range newb.Txs {
		pbtxs = append(pbtxs, tx.ToProto())
	}
	pbexeheader := make([]*pb.ExecuteHeader, 0)
	for _, header := range newb.ExeHeaders {
		pbexeheader = append(pbexeheader, header.ToProto())
	}
	return &pb.Block{
		Header:         pbheader,
		Txs:            pbtxs,
		ExecuteHeaders: pbexeheader,
	}
}

func ToBlock(block *pb.Block) *Block {
	pbheader := block.GetHeader()
	pbtxs := block.GetTxs()
	pbexeheaders := block.GetExecuteHeaders()
	header := ToHeader(pbheader)
	txs := ToTxs(pbtxs)
	exeheader := ToExeHeaders(pbexeheaders)
	return &Block{
		Header:     header,
		Txs:        txs,
		ExeHeaders: exeheader,
	}
}

func ToExeHeader(exeheader *pb.ExecuteHeader) *ExecuteHeader {
	if exeheader == nil {
		return nil
	}
	return &ExecuteHeader{
		Height:        exeheader.GetHeight(),
		BlockHash:     exeheader.GetBlockHash(),
		ChainID:       exeheader.GetChainID(),
		TxsHash:       exeheader.GetTxsHash(),
		CommitedTxNum: exeheader.GetCommitedTxNum(),
		ExecutedTxNum: exeheader.GetExecutedTxNum(),
		GasIncentive:  exeheader.GetGasIncentive(),
		PoSLeader:     exeheader.GetPoSLeader(),
		PoSVoteInfo:   exeheader.GetPoSVoteInfo(),
		Checkpoints:   ToCrosschainCheckpoints(exeheader.GetCheckpoints()),
	}
}

func ToExeHeaders(exeheaders []*pb.ExecuteHeader) []*ExecuteHeader {
	exes := make([]*ExecuteHeader, len(exeheaders))
	for i := 0; i < len(exeheaders); i++ {
		exes[i] = ToExeHeader(exeheaders[i])
	}
	return exes
}

func (b *Header) Hash() []byte {
	if b == nil {
		return crypto.NilTxsHash
	}
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
	if b.PoTProof == nil {
		panic("pot proof is nil")
	}
	if b.Mixdigest == nil {
		panic("pot Mixdigest is nil")
	}
	if b.ParentHash == nil {
		panic("pot proof is nil")
	}
	if b.TxHash == nil {
		panic("txhash is nil")
	}
	if b.ExeHash == nil {
		panic("exehash is nil")
	}
	if b.PublicKey == nil {
		panic("PublicKey is nil")
	}
	//fmt.Println("height:", height)
	//fmt.Println("b.ParentHash:", b.ParentHash)
	//fmt.Println("b.unclelen", len(b.UncleHash))
	//fmt.Println("unclehash:", unclehash)
	//fmt.Println("b.Mixdigest:", b.Mixdigest)
	//fmt.Println("difficulty:", difficulty)
	//fmt.Println("nonce:", nonce)
	//fmt.Println("timestamp:", timestamp)
	//fmt.Println("b.PoTProof[0]:", b.PoTProof[0])
	//fmt.Println("b.PoTProof[1]:", b.PoTProof[1])
	//fmt.Println("address:", address)
	//fmt.Println("peeridbyte:", peeridbyte)
	//fmt.Println("b.TxHash:", b.TxHash)
	//fmt.Println("b.PublicKey:", b.PublicKey)

	hashinput := bytes.Join([][]byte{
		height, b.ParentHash, unclehash,
		b.Mixdigest, difficulty, nonce,
		timestamp, b.PoTProof[0], b.PoTProof[1],
		address, peeridbyte, b.TxHash, b.ExeHash, b.PublicKey,
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
	flag, err := b.CheckHash()
	if !flag {
		return false, err
	}
	return true, nil
}

// TODO: need to complete
func (h *Header) CheckMixdigest() bool {
	unclehash := make([]byte, 0)
	for i := 0; i < len(h.UncleHash); i++ {
		if len(h.UncleHash[i]) != crypto.Hashlen {
			return false
		}
		unclehash = append(unclehash, h.UncleHash[i]...)
	}
	// mixdigest verify
	tmp := new(big.Int)
	tmp.Set(h.Difficulty)
	difficultybyte := tmp.Bytes()
	idbyte := []byte(h.PeerId)
	tmp.SetInt64(int64(h.Height))
	epochbyte := tmp.Bytes()

	hashinput := bytes.Join([][]byte{epochbyte, h.ParentHash, unclehash, difficultybyte, idbyte}, []byte(""))
	res := crypto.Hash(hashinput)
	return bytes.Equal(res[:], h.Mixdigest)
}

func (b *Header) CheckHash() (bool, error) {
	if b.Hashes == nil {
		return false, fmt.Errorf("the block without hash")
	}
	if b.ExeHash == nil {
		return false, fmt.Errorf("the block without exehash")
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
		address, peeridbyte, b.TxHash, b.ExeHash, b.PublicKey,
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
	if b.ExeHash == nil {
		return false, fmt.Errorf("the nil block without exehash")
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
		Height:         header.GetHeight(),
		ParentHash:     header.GetParentHash(),
		UncleHash:      header.GetUncleHash(),
		Mixdigest:      header.GetMixdigest(),
		Difficulty:     difficulty,
		Nonce:          header.GetNonce(),
		Timestamp:      timestamp,
		PoTProof:       header.PoTProof,
		Address:        header.GetAddress(),
		Hashes:         header.GetHashes(),
		PeerId:         header.GetPeerId(),
		PublicKey:      header.GetPubkey(),
		TxHash:         header.GetTxhash(),
		ExeHash:        header.GetExeHash(),
		CommiteePubkey: header.GetCommiteePubkey(),
	}
	return h
}

func (b *Header) ToProto() *pb.Header {
	ts, err := b.Timestamp.MarshalJSON()
	if err != nil {
		return nil
	}
	if b.PoTProof == nil {
		panic("pot proof is nil")
	}
	if b.Mixdigest == nil {
		panic("pot Mixdigest is nil")
	}
	if b.ParentHash == nil {
		panic("pot proof is nil")
	}
	if b.TxHash == nil {
		panic("txhash is nil")
	}
	if b.ExeHash == nil {
		panic("exehash is nil")
	}
	if b.PublicKey == nil {
		panic("PublicKey is nil")
	}
	if b.PoTProof == nil {
		panic("potproof nil")
	}
	if b.PublicKey == nil {
		panic("PublicKey nil")
	}
	if b.TxHash == nil {
		panic("txhash is nil")
	}
	if b.Hashes == nil {
		b.Hash()
	}
	if b.PublicKey == nil {
		panic("PublicKey is nil")
	}

	return &pb.Header{
		Height:         b.Height,
		ParentHash:     b.ParentHash,
		UncleHash:      b.UncleHash,
		Mixdigest:      b.Mixdigest,
		Difficulty:     b.Difficulty.Bytes(),
		Nonce:          b.Nonce,
		Timestamp:      ts,
		PoTProof:       b.PoTProof,
		Address:        b.Address,
		Hashes:         b.Hashes,
		PeerId:         b.PeerId,
		Pubkey:         b.PublicKey,
		Txhash:         b.TxHash,
		CommiteePubkey: b.CommiteePubkey,
		ExeHash:        b.ExeHash,
	}
}

func DefaultGenesisHeader() *Header {
	pubkey, _ := hexutil.Decode("0xa47155b42648816998e576ffbfa045ab00000000000000007662a30d4b5875b73a8768fcf01bf52d2326a7660e24033a")
	h := &Header{
		Height:     0,
		ParentHash: crypto.NilTxsHash,
		UncleHash:  [][]byte{crypto.NilTxsHash},
		Mixdigest:  crypto.NilTxsHash,
		Difficulty: big.NewInt(1),
		Nonce:      0,
		Timestamp:  time.Date(2023, 8, 14, 15, 35, 00, 0, time.Local),
		PoTProof:   [][]byte{([]byte("aa")), {}},
		Address:    0,
		Hashes:     nil,
		PeerId:     "0",
		PublicKey:  pubkey,
		TxHash:     crypto.NilTxsHash,
		ExeHash:    crypto.NilTxsHash,
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

func (b *Block) GetExecutedHeaders() []*ExecuteHeader {
	return b.ExeHeaders
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
