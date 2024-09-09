package network

// import (
// 	"crypto/sha256"
// 	"fmt"
// 	"testing"

// 	bls "github.com/chuwt/chia-bls-go"
// 	ecrypto "github.com/ethereum/go-ethereum/crypto"
// )

// type TransactionWithEvidence struct {
// 	TX Transaction
// 	TE *TransmissionEvidence
// }

// type Transaction struct {
// 	tx []byte
// }

// func TestEvidence(t *testing.T) {

// 	// Generate an ECDSA public-private key pair
// 	priv, err := ecrypto.GenerateKey()
// 	if err != nil {
// 		fmt.Println("GenerateKey error: ", err.Error())
// 	}

// 	pub := priv.PublicKey
// 	pubBytes := ecrypto.CompressPubkey(&pub)

// 	// Generate a BLS public-private key pair
// 	blspriv := bls.KeyGen([]byte("34567823403456091230947650934091759345451345234620348502"))
// 	blspub := blspriv.GetPublicKey()

// 	// Generate ECDSA signatures for BLS public keys
// 	//dataHash := sha256.Sum256([]byte("ethereum"))
// 	dataHash := sha256.Sum256(blspub.Bytes())
// 	signature, _ := ecrypto.Sign(dataHash[:], priv)

// 	// BLS Signature
// 	share := 0.1
// 	asm := bls.AugSchemeMPL{}
// 	blssign := asm.Sign(blspriv, Float64ToByte(share))

// 	// Structure transmission evidence
// 	var publicKeyList [][]byte
// 	publicKeyList = append(publicKeyList, blspub.Bytes())
// 	var shareList [][]byte
// 	shareList = append(shareList, Float64ToByte(share))

// 	oa := Authorize{
// 		BLSPublicKey:    blspub.Bytes(),
// 		SignatureForBLS: signature[:64],
// 		ECDSAPublicKey:  pubBytes,
// 	}

// 	te := &TransmissionEvidence{
// 		OwnerAuthorize: oa,
// 		PublicKeyList:  publicKeyList,
// 		ShareList:      shareList,
// 		Signature:      blssign,
// 		Count:          1,
// 	}

// 	tx := Transaction{
// 		tx: []byte("12345678"),
// 	}

// 	twe := &TransactionWithEvidence{
// 		TX: tx,
// 		TE: te,
// 	}

// 	// Create EvidenceManagement
// 	blspriv1 := bls.KeyGen([]byte("34567823403456091230947650934091759"))
// 	evidenceManagement := NewEvidenceManagement(blspriv1.Bytes())

// 	isok, err := evidenceManagement.VerifyEvidence(twe.TE)
// 	if err != nil {
// 		fmt.Println("error: ", err.Error())
// 	}

// 	if !isok {
// 		t.Fatal("VerifyEvidence error")
// 	}

// 	err = evidenceManagement.AddEvidence(twe.TE)
// 	if err != nil {
// 		t.Fatal("AddEvidence error")
// 	}
// }
//
