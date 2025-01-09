package main

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {

	println("Hello, World!")
	client, err := ethclient.Dial("http://111.119.239.159:36054")
	if err != nil {
		fmt.Println("Error:")
		fmt.Println(err)
	}
	key, err := crypto.GenerateKey()

	if err != nil {
		fmt.Println(err)
	}
	address := crypto.PubkeyToAddress(key.PublicKey)
	fill, err := os.OpenFile("key", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err)
	}

	fill.Write(crypto.FromECDSA(key))
	fill.WriteString("\n")

	fill.WriteString(fmt.Sprintf("[address]%s\n", address.Hex()))
	fill.Close()

	readfill, _ := os.OpenFile("key", os.O_RDWR, 0644)
	keydata := make([]byte, key.Params().BitSize/8)
	_, err = readfill.Read(keydata)

	if err != nil {
		fmt.Println(err)
		return
	}

	key3, err := crypto.ToECDSA(keydata)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(bytes.Equal(crypto.CompressPubkey(&key.PublicKey), crypto.CompressPubkey(&key3.PublicKey)))

	key2, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	fromaddress := address
	nonce, err := client.PendingNonceAt(context.Background(), fromaddress)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("nonce: ", nonce)

	value := big.NewInt(10)
	gaslimite := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("suggestGasPrice: ", gasPrice.Uint64())
	toaddress := crypto.PubkeyToAddress(key2.PublicKey)
	var data []byte

	//chainID, err := client.ChainID(context.Background())
	chainID, err := client.ChainID(context.Background())
	fmt.Println("chainID: ", chainID)
	if err != nil {
		fmt.Println(err)
		return

	}
	tx := types.NewTransaction(nonce, toaddress, value, gaslimite, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		fmt.Println(err)
		return
	}
	signedTx.Hash()
	fmt.Println(signedTx.Hash())

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println(err)
		return
	}
	signedTx.MarshalBinary()

	n, _ := client.BlockNumber(context.Background())

	fmt.Println(n)
}
