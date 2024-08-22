package verifiable_draw

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	. "github.com/zzz136454872/upgradeable-consensus/crypto/types/curve/bls12381"
	"github.com/zzz136454872/upgradeable-consensus/crypto/types/srs"
	"github.com/zzz136454872/upgradeable-consensus/crypto/utils"
)

func TestDraw(t *testing.T) {
	candidatesNum := uint32(256)
	quota := uint32(32)

	g1Degree, g2Degree := utils.CalcMinLimitedDegree(candidatesNum, quota)
	fmt.Println("start gen srs")
	start := time.Now()
	s, _ := srs.NewSRS(g1Degree, g2Degree)
	fmt.Println("gen srs time:", time.Since(start))

	generator := s.G1Power(0)

	fmt.Println("start create key list")
	start = time.Now()
	secretKeys := make([]*Fr, candidatesNum)
	publicKeys := make([]*PointG1, candidatesNum)
	for i := uint32(0); i < candidatesNum; i++ {
		secretKeys[i], _ = NewFr().Rand(rand.Reader)
		// secretKeys[i] = FromUInt32(i + 1)
		publicKeys[i] = group1.Affine(group1.MulScalar(group1.New(), generator, secretKeys[i]))
	}
	fmt.Println("create key list time:", time.Since(start))

	for i := uint32(0); i < quota; i++ {
		fmt.Printf("sk[%d]: %v\n", i+1, secretKeys[i])
		fmt.Printf("pk[%d]: %v\n", i+1, publicKeys[i])
		fmt.Printf("pk * r: %v\n", group1.Affine(group1.MulScalar(group1.New(), publicKeys[i], FrFromInt(123))))
	}

	indexList := []uint32{
		222, 172, 20, 119, 75, 252, 77, 128, 157, 247, 78, 255, 71, 44, 94, 126,
		101, 103, 110, 246, 120, 58, 162, 123, 12, 62, 177, 22, 53, 45, 173, 118,
		91, 249, 147, 200, 186, 116, 49, 92, 149, 76, 175, 202, 1, 23, 105, 10,
		85, 145, 143, 163, 189, 52, 73, 146, 144, 40, 217, 82, 184, 84, 106, 98,
		35, 140, 48, 37, 182, 95, 228, 121, 238, 135, 152, 160, 138, 231, 88, 191,
		30, 254, 195, 2, 159, 151, 176, 9, 124, 46, 42, 179, 170, 251, 127, 235,
		72, 59, 236, 54, 183, 125, 204, 80, 32, 242, 34, 214, 43, 90, 196, 213,
		190, 212, 19, 153, 230, 11, 221, 201, 148, 61, 169, 164, 227, 199, 239, 197,
		60, 206, 133, 156, 102, 193, 150, 134, 113, 248, 224, 203, 131, 210, 69, 67,
		13, 192, 130, 194, 87, 232, 28, 225, 233, 211, 66, 89, 226, 244, 36, 117,
		83, 25, 107, 136, 112, 31, 74, 21, 243, 187, 220, 17, 47, 16, 4, 68,
		240, 154, 93, 137, 168, 208, 174, 205, 181, 155, 97, 100, 114, 165, 8, 7,
		15, 180, 108, 250, 198, 215, 185, 14, 99, 50, 51, 129, 86, 218, 166, 219,
		241, 70, 132, 142, 188, 237, 256, 115, 167, 33, 5, 216, 104, 229, 109, 209,
		6, 38, 29, 223, 141, 64, 207, 122, 26, 253, 56, 79, 65, 178, 41, 55,
		63, 161, 96, 111, 81, 234, 171, 24, 27, 39, 158, 18, 57, 245, 3, 139,
	}

	indices := indexList[:quota]
	// indices = []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

	fmt.Println("start create draw proof")
	start = time.Now()
	drawProof, err := Draw(s, candidatesNum, publicKeys, quota, indices, group1.One())
	if err != nil {
		fmt.Println("gen draw proof error:", err)
		return
	}
	fmt.Println("create draw proof time:", time.Since(start))

	fmt.Println("start verify draw proof")
	start = time.Now()
	res := Verify(s, candidatesNum, publicKeys, quota, drawProof)
	fmt.Println("verify draw proof time:", time.Since(start))
	fmt.Println("verify draw proof:", res)

	for j := uint32(0); j < quota; j++ {
		fmt.Printf("y_%d : %v\n", j+1, drawProof.SelectedPubKeys[j])
	}

	for i := uint32(1); i < candidatesNum; i++ {
		selected := false
		for j := uint32(0); j < quota; j++ {
			if i == indices[j]-1 {
				selected = true
				break
			}
		}
		isSelected, _ := IsSelected(secretKeys[i], drawProof.RCommit, drawProof.SelectedPubKeys)
		if selected {

			if !isSelected {
				fmt.Println("selected but not in proof, at index:", i)
			}
		} else if isSelected {
			fmt.Println("not selected but in proof, at index:", i)
		}
	}

}
