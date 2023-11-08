package crypto

import (
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
