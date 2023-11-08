package upgradeable_consensus

import (
	"testing"
)

func TestPoT_MsgReceive(t *testing.T) {
	var n name
	d := []int{1, 2, 3, 4, 5}
	n.data = d
	a, b := test(n.data)
	n.data = make([]int, 0)
	t.Log(a, b)
	t.Log(n.data)
}

func test(data []int) (int, []int) {
	a := data[0]
	b := append(data[:1], data[2:]...)
	data = nil
	return a, b
}

type name struct {
	data []int
}
