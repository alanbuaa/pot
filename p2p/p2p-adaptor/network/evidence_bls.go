package network

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"

	bls "github.com/chuwt/chia-bls-go"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
)

const EvidenceMaxSize = 10

type Authorize struct {
	// BLS public key
	BLSPublicKey []byte
	// Signature for BLS public key
	SignatureForBLS []byte
	// ECDSA Public key
	ECDSAPublicKey []byte
}

type BLSEvidence struct {
	OwnerAuthorize Authorize
	PublicKeyList  [][]byte // BLS PublicKeys
	ShareList      [][]byte // 0~1
	Signature      []byte   // BLS signature
	Count          int      // number of Relayer
}

type BLSManagement struct {
	PublicKey  bls.PublicKey
	PrivateKey bls.PrivateKey
}

// Implement the interface of VerifyEvidence in TransmissionEvidence
func (be *BLSEvidence) VerifyEvidence() (bool, error) {
	// If > EvidenceMaxSize
	if len(be.PublicKeyList) > EvidenceMaxSize {
		return false, errors.New("over max evidence size")
	}

	// If ShareLeft < 0
	left, err := be.getShareLeft()
	if err != nil {
		return false, errors.New("get share left error")
	}
	if left < 0 {
		return false, errors.New("share is over 1.0")
	}

	// Verify authorize sturct
	//   Verify ECDSA signature
	dataHash := sha256.Sum256(be.OwnerAuthorize.BLSPublicKey)
	isRight := ecrypto.VerifySignature(be.OwnerAuthorize.ECDSAPublicKey, dataHash[:], be.OwnerAuthorize.SignatureForBLS)

	if !isRight {
		return false, errors.New("ecdsa signature verify failed")
	}

	// Verify BLS Aggregate Signature
	asm := bls.AugSchemeMPL{}
	isRight = asm.AggregateVerify(be.PublicKeyList, be.ShareList, be.Signature)

	if !isRight {
		return false, errors.New("aggregate verify failed")
	}

	return true, nil
}

// Implement the interface of GetShareList in TransmissionEvidence
func (be *BLSEvidence) GetShareList() ([][]byte, error) {
	return be.ShareList, nil
}

func (be *BLSEvidence) getShareLeft() (float64, error) {
	left := 1.0
	for _, share := range be.ShareList {
		shareFloat64, err := ByteToFloat64(share)
		if err != nil {
			return left, errors.New("share to float64 error")
		}
		left -= shareFloat64
	}
	return left, nil
}

func NewEvidenceManagement(privateKeyBytes []byte) *BLSManagement {
	privateKey := bls.KeyFromBytes(privateKeyBytes)
	return &BLSManagement{
		PublicKey:  privateKey.GetPublicKey(),
		PrivateKey: privateKey,
	}
}

// Implement the interface of AddEvidence in EvidenceManagement
func (bm *BLSManagement) AddEvidence(be *BLSEvidence) error {
	// Get share list
	shareList, _ := be.GetShareList()

	// Get best share for this evidence
	newShare, _ := bm.GetBestTakeShare(shareList)

	// Sign share with own privite key
	asm := bls.AugSchemeMPL{}
	sign := asm.Sign(bm.PrivateKey, newShare)

	// Aggregate old sign and own sign
	newSign, err := asm.Aggregate(be.Signature, sign)
	if err != nil {
		return err
	}

	// Update evidence
	be.Signature = newSign
	be.PublicKeyList = append(be.PublicKeyList, bm.PublicKey.Bytes())
	be.ShareList = append(be.ShareList, newShare)
	be.Count = be.Count + 1

	return nil
}

// Implement the interface of GetBestTakeShare in EvidenceManagement
func (bm *BLSManagement) GetBestTakeShare(shareList [][]byte) ([]byte, error) {
	left := 1.0
	for _, share := range shareList {
		shareFloat64, err := ByteToFloat64(share)
		if err != nil {
			return Float64ToByte(left), errors.New("share to float64 error")
		}
		left -= shareFloat64
	}

	myShare := 0.0
	if left > 0.1 {
		myShare = 0.1
	}

	return Float64ToByte(myShare), nil
}

// Float64ToByte Float64转byte
func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

// ByteToFloat64 byte转Float64
func ByteToFloat64(bytes []byte) (float64, error) {
	var number float64
	if len(bytes) <= 7 {
		return number, errors.New("ByteToFloat64 error")
	}
	bits := binary.LittleEndian.Uint64(bytes)
	number = math.Float64frombits(bits)
	return number, nil
}
