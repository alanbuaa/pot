package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"

	"github.com/niclabs/tcrsa"
)

func CreateDocumentHash(msgBytes []byte, meta *tcrsa.KeyMeta) ([]byte, error) {
	bytes := sha256.Sum256(msgBytes)
	documentHash, err := tcrsa.PrepareDocumentHash(meta.PublicKey.Size(), crypto.SHA256, bytes[:])
	if err != nil {
		return nil, err
	}
	return documentHash, nil
}

// TSign create the partial signature of replica
func TSign(documentHash []byte, privateKey *tcrsa.KeyShare, publicKey *tcrsa.KeyMeta) (*tcrsa.SigShare, error) {
	// sign
	partSig, err := privateKey.Sign(documentHash, crypto.SHA256, publicKey)
	if err != nil {
		return nil, err
	}
	// ensure the partial signature is correct
	err = partSig.Verify(documentHash, publicKey)
	if err != nil {
		return nil, err
	}

	return partSig, nil
}

func VerifyPartSig(partSig *tcrsa.SigShare, documentHash []byte, publicKey *tcrsa.KeyMeta) error {
	err := partSig.Verify(documentHash, publicKey)
	if err != nil {
		return err
	}
	return nil
}

func CreateFullSignature(documentHash []byte, partSigs tcrsa.SigShareList, publicKey *tcrsa.KeyMeta) (tcrsa.Signature, error) {
	signature, err := partSigs.Join(documentHash, publicKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func TVerify(publicKey *tcrsa.KeyMeta, signature tcrsa.Signature, msgBytes []byte) (bool, error) {
	// get the msg hash
	hash := sha256.Sum256(msgBytes)
	err := rsa.VerifyPKCS1v15(publicKey.PublicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return false, err
	}
	return true, nil
}
