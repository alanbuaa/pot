package whirlyUtilities

import (
	"encoding/json"
)

type Incentive struct {
	Leader         string
	NodeIncentives []NodeIncentive
}

type NodeIncentive struct {
	NodePublicAddress string
	Type              string
	Number            int64
}

func NewIncentive(leader string) *Incentive {
	incentive := &Incentive{
		Leader:         leader,
		NodeIncentives: make([]NodeIncentive, 0),
	}
	return incentive
}

func (in *Incentive) InsertYesVote(address string) {
	nodeIncentive := NodeIncentive{
		NodePublicAddress: address,
		Type:              "YES",
		Number:            1,
	}
	in.NodeIncentives = append(in.NodeIncentives, nodeIncentive)
}

func (in *Incentive) InsertNoVote(address string) {
	nodeIncentive := NodeIncentive{
		NodePublicAddress: address,
		Type:              "No",
		Number:            1,
	}
	in.NodeIncentives = append(in.NodeIncentives, nodeIncentive)
}

func EncodeIncentive(incentive Incentive) ([]byte, error) {
	incentiveBytes, err := json.Marshal(incentive)
	if err != nil {
		return nil, err
	}
	return incentiveBytes, nil
}

func DecodeIncentive(incentiveBytes []byte) (*Incentive, error) {
	incentive := &Incentive{}
	err := json.Unmarshal(incentiveBytes, incentive)
	if err != nil {
		return nil, err
	}
	return incentive, nil
}
