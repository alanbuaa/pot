package pot

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

// Data represents a record with address, amount, and partition number
type Data struct {
	Address   string
	Amount    float64
	Partition int
	Weight    float64 // Normalized weight
}

// Normalize amounts within each partition
func normalizePartitions(data []Data) map[int][]Data {
	partitions := make(map[int][]Data)
	for _, d := range data {
		partitions[d.Partition] = append(partitions[d.Partition], d)
	}

	for partition, records := range partitions {
		totalAmount := 0.0
		for _, record := range records {
			totalAmount += record.Amount
		}

		for i := range records {
			records[i].Weight = records[i].Amount / totalAmount
		}
		partitions[partition] = records
	}

	return partitions
}

// Sort and randomly select Commitees records from each partition
func selectRecords(partitions map[int][]Data, n int) map[int][]Data {
	selectedRecords := make(map[int][]Data)

	for partition, records := range partitions {
		// Sort by address in ascending order
		sort.Slice(records, func(i, j int) bool {
			return records[i].Address < records[j].Address
		})

		// Shuffle the sorted records to randomly select Commitees records

		rand.Shuffle(len(records), func(i, j int) {
			records[i], records[j] = records[j], records[i]
		})

		if n < len(records) {
			selectedRecords[partition] = records[:n]
		} else {
			selectedRecords[partition] = records
		}
	}

	return selectedRecords
}

func TestPotShuffle(t *testing.T) {
	data := []Data{
		{"address1", 10, 1, 0},
		{"address2", 20, 1, 0},
		{"address3", 30, 2, 0},
		{"address4", 40, 2, 0},
		{"address5", 50, 3, 0},
		{"address6", 60, 3, 0},
	}

	// Normalize amounts within each partition
	partitions := normalizePartitions(data)

	// Select Commitees records from each partition
	n := 1
	selectedRecords := selectRecords(partitions, n)

	// Print selected records
	for partition, records := range selectedRecords {
		fmt.Printf("Partition %d:\n", partition)
		for _, record := range records {
			fmt.Printf("Address: %s, Amount: %.2f, Weight: %.4f\n", record.Address, record.Amount, record.Weight)
		}
	}
}
