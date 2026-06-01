package main

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func main() {

	// pot.StartAPI()
	// TODO:

	str := "0x123456"

	fmt.Println([]byte(str))
	b, _ := hex.DecodeString(str[2:])
	fmt.Println(b)
	fmt.Println(hex.EncodeToString(b))
	fmt.Println(bytes.Equal([]byte(str), b))
	b2, _ := hexutil.Decode(str)
	fmt.Println(b2)
	fmt.Println(hexutil.Encode(b2))
	fmt.Println(bytes.Equal(b, b2))
}
