package utils

import (
	"fmt"
	"testing"
)

func TestNewSet(t *testing.T) {
	set := NewSet([]uint8{1, 3, 5})
	fmt.Println(set.Contains(4))
	fmt.Println(set.Contains(1))
}
