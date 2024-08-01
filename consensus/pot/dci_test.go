package pot

import (
	"bytes"
	"encoding/binary"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"golang.org/x/exp/rand"
	"math/big"
	"sort"
	"testing"
)

func TestDci(t *testing.T) {
	vdf1res := []byte("abcdefg789456121323")
	rand.Seed(binary.BigEndian.Uint64(vdf1res[:8]))
	//test := big.NewInt(0)
	dcireward1 := &DciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  10,
		Proof:   DciProof{},
		ChainID: 1,
		weight:  0,
	}
	dcireward2 := &DciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  5,
		Proof:   DciProof{},
		ChainID: 1,
		weight:  0,
	}
	dcireward3 := &DciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  1,
		Proof:   DciProof{},
		ChainID: 1,
		weight:  0,
	}
	dcireward4 := &DciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  5,
		Proof:   DciProof{},
		ChainID: 2,
		weight:  0,
	}

	dcirewards := []*DciReward{dcireward1, dcireward2, dcireward3, dcireward4}
	groupsdata := groupByChainID(dcirewards)
	for _, rewards := range groupsdata {
		total := int64(0)
		for _, reward := range rewards {
			total += reward.Amount
		}
		for _, reward := range rewards {
			reward.weight = float64(reward.Amount) / float64(total)
			//t.Log(reward.weight)
		}
	}
	selectreward := make(map[int64][]*DciReward)
	vdf0res := []byte("abcdefg789456121323sssswererewrerwwerssssessssssss")
	for chainID, rewards := range groupsdata {
		sort.Slice(rewards, func(i, j int) bool {
			return bytes.Compare(rewards[i].Address, rewards[j].Address) < 0
		})
		t.Log(rewards)
		for _, reward := range rewards {
			t.Log(reward.weight)
		}
		rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
		for i := 0; i < Selectn; i++ {
			r := rand.Float64()
			t.Log(r)
			acnum := 0.0

			for _, reward := range rewards {
				//t.Log(reward.weight)
				acnum += reward.weight
				if r <= acnum {
					selectreward[chainID] = append(selectreward[chainID], reward)
					break
				}
			}
		}
	}
	for chainID, rewards := range selectreward {
		t.Log(chainID)
		t.Log(rewards)

	}

}

func TestRandom(t *testing.T) {
	vdf0res := []byte("abcdefg789456121323")
	for i := 0; i < 10; i++ {
		rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
		t.Log(rand.Float64())
	}
}
