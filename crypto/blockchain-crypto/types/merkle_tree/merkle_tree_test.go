package merkle_tree

import (
	"fmt"
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	// 示例切片
	blobs := [][]byte{
		[]byte("banana"),
		[]byte("apple"),
		[]byte("cherry"),
	}

	// 使用sort.Slice对切片进行排序
	sort.Slice(blobs, func(i, j int) bool {
		return string(blobs[i]) < string(blobs[j])
	})

	// 打印排序后的切片
	for _, blob := range blobs {
		fmt.Println(string(blob))
	}
}
