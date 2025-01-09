package crypto

import (
	"blockchain-crypto/types/curve/bls12381"
	"fmt"
	"testing"

	"github.com/niclabs/tcrsa"
)

func TestThresholdSignature(t *testing.T) {
	cryptoStrBytes := []byte("hello, world")
	t.Log("generating keys")
	keys, meta, err := GenerateThresholdKeys(3, 4)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("creating hash")
	docHash, err := CreateDocumentHash(cryptoStrBytes, meta)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("creating signature shares")
	sigList := make(tcrsa.SigShareList, len(keys))
	for i := 0; i < len(keys); i++ {
		sigList[i], err = TSign(docHash, keys[i], meta)
		if err != nil {
			t.Fatal(err)
		}
		err := VerifyPartSig(sigList[i], docHash, meta)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Log("creating signature")
	signature, err := CreateFullSignature(docHash, sigList, meta)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("verifiing signature")
	_, err = TVerify(meta, signature, cryptoStrBytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok")
}

func TestGenkey(t *testing.T) {

}

func TestVerifySignature(t *testing.T) {
	alpha := []byte("hello alpha")
	pk := []byte("hello PK")
	x := bls12381.HashToFr(append(alpha, pk...))
	group1 := bls12381.NewG1()
	y := group1.MulScalar(group1.New(), group1.One(), x)
	gDot := group1.MulScalar(group1.New(), group1.One(), bls12381.FrFromInt(12345))
	yDot := group1.MulScalar(group1.New(), y, bls12381.FrFromInt(12345))
	fmt.Println(VerifyCommitteePKAPI(alpha, pk, group1.ToBytes(group1.One()), group1.ToBytes(y), group1.ToBytes(gDot), group1.ToBytes(yDot)))
}
