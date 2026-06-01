package blockchain_api

import (
	"blockchain-crypto/encrypt/ibve"
	"blockchain-crypto/types/curve/bls12381"
	"crypto/rand"
	"crypto/sha256"
	encoding_json "encoding/json"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	g1 = bls12381.NewG1()
	g2 = bls12381.NewG2()
	gt = bls12381.NewGT()
)

type TxCR struct {
	C           []byte // 轮ID
	CipherTxKey []byte // 交易密钥密文，交易密钥被门限加密
	CipherTx    []byte // 交易密文，交易被交易密钥对称加密
	Tx          []byte // 交易明文
}

func EncodeTxCRToBytes(tx TxCR) ([]byte, error) {
	txBytes, err := encoding_json.Marshal(tx)
	if err != nil {
		return nil, err
	}
	return txBytes, nil
}

func DecodeBytesToTxCR(txBytes []byte) (*TxCR, error) {
	tx := new(TxCR)
	err := encoding_json.Unmarshal(txBytes, tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func EncryptIBVE(yBytes []byte, msgBytes []byte) (c1 []byte, cipherTextBytes []byte, err error) {
	y, err := g1.FromCompressed(yBytes)
	if err != nil {
		return nil, nil, err
	}
	msg, err := gt.FromBytes(msgBytes)
	if err != nil {
		return nil, nil, err
	}
	cipherText := ibve.Encrypt(y, msg)
	return g1.ToCompressed(cipherText.C1), cipherText.ToBytes(), nil
}

func DecryptIBVE(sigmaBytes []byte, cipherTextBytes []byte) ([]byte, error) {
	sigma, err := g1.FromCompressed(sigmaBytes)
	if err != nil {
		return nil, err
	}
	cipherText, err := new(ibve.CipherText).FromBytes(cipherTextBytes)
	if err != nil {
		return nil, err
	}
	msg := ibve.Decrypt(sigma, cipherText)
	return gt.ToBytes(msg), nil
}

func VerifyIBVE(sigmaBytes []byte, yBytes []byte, cipherTextBytes []byte) bool {
	sigma, err := g1.FromCompressed(sigmaBytes)
	if err != nil {
		return false
	}
	y, err := g1.FromCompressed(yBytes)
	if err != nil {
		return false
	}
	cipherText, err := new(ibve.CipherText).FromBytes(cipherTextBytes)
	if err != nil {
		return false
	}
	return ibve.Verify(sigma, y, cipherText.C2)
}

func GenTxKey() (txKeyBytes []byte, err error) {
	txKey, err := bls12381.NewE().Rand(rand.Reader)
	if err != nil {
		return nil, err
	}
	txKeyBytes = bls12381.NewGT().ToBytes(txKey)
	return txKeyBytes, nil
}

func EncryptTx(txKey []byte, txBytes []byte) (cipherText []byte, err error) {
	nonce := make([]byte, chacha20poly1305.NonceSize)
	key := sha256.Sum256(txKey)
	aead, _ := chacha20poly1305.New(key[:])
	return aead.Seal(nil, nonce, txBytes, nil), nil
}

func DecryptTx(txKey []byte, cipherText []byte) (txBytes []byte, err error) {
	nonce := make([]byte, chacha20poly1305.NonceSize)
	key := sha256.Sum256(txKey)
	aead, _ := chacha20poly1305.New(key[:])
	return aead.Open(nil, nonce, cipherText, nil)
}
