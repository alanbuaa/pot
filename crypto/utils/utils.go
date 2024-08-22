package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"math/bits"
)

func BigFromHex(hex string) *big.Int {
	if len(hex) > 1 && hex[:2] == "0x" {
		hex = hex[2:]
	}
	n, _ := new(big.Int).SetString(hex, 16)
	return n
}

func HashToBig(input []byte, modulus *big.Int) *big.Int {
	h := sha256.New()
	h.Write(input)
	output := h.Sum(nil)
	res := new(big.Int).SetBytes(output)
	if modulus == nil {
		return res
	}
	return res.Mod(res, modulus)
}

// NextPowerOfTwo returns the next power of 2 of n
func NextPowerOfTwo(n uint64) uint64 {
	c := bits.OnesCount64(n)
	if c == 0 {
		return 1
	}
	if c == 1 {
		return n
	}
	t := bits.LeadingZeros64(n)
	if t == 0 {
		panic("next power of 2 overflows uint64")
	}
	return uint64(1) << (64 - t)
}

// GenRandomPermutation 生成一个从1到n的随机排列
func GenRandomPermutation(n uint32) ([]uint32, error) {
	// 创建一个切片，用于存储排列
	permutation := make([]uint32, n)
	for i := uint32(0); i < n; i++ {
		permutation[i] = i + 1
	}

	// 使用Fisher-Yates洗牌算法生成随机排列
	for i := n - 1; i > 0; i-- {
		// 生成一个[0, i]范围内的随机数
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, err
		}

		// 因为rand.Int返回的是*big.Int类型，我们需要转换为int
		jInt := int(j.Int64())
		// 交换元素
		permutation[i], permutation[jInt] = permutation[jInt], permutation[i]
	}

	return permutation, nil
}

func CalcMinLimitedDegree(n uint32, m uint32) (uint32, uint32) {
	// g1: max(n, m^2+m+2)
	// g2: max(n-1, m^2+m+2)
	mLimit := (m+1)*(m+1) + 1
	if mLimit >= n {
		return mLimit, mLimit
	} else {
		return n, n - 1
	}
}

// Lattice represents a Z module spanned by V1, V2.
// det is the associated determinant.
type Lattice struct {
	V1, V2      [2]big.Int
	Det, b1, b2 big.Int
}

// FromHex returns the bytes represented by the hexadecimal string s.
// s may be prefixed with "0x".
func FromHex(s string) []byte {
	if has0xPrefix(s) {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hexToBytes(s)
}

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// hexToBytes returns the bytes represented by the hexadecimal string str.
func hexToBytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

func Uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
	return b
}
