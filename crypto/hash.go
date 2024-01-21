package crypto

import "crypto/sha256"

var Hashlen = 32

func Hash(input []byte) []byte {

	sum := sha256.Sum256(input)
	return sum[:]
}
