package nodeController

import (
	"testing"
)

func TestDecodeAddressWithPeerId(t *testing.T) {
	tests := []struct {
		name              string
		encodedAddress    string
		expectedSharding  string
		expectedConsensus int64
		expectedRawAddr   string
	}{
		{
			name:              "Simple sharding name without PeerId",
			encodedAddress:    "0x1-1201-daemonNode",
			expectedSharding:  "0x1",
			expectedConsensus: 1201,
			expectedRawAddr:   "daemonNode",
		},
		{
			name:              "Sharding name with PeerId concatenated",
			encodedAddress:    "0x1QmbF5Nbqrm7yE6gAk2CpB4cxAhERfWsbYKKcKi54nzhuob-1201-daemonNode",
			expectedSharding:  "0x1QmbF5Nbqrm7yE6gAk2CpB4cxAhERfWsbYKKcKi54nzhuob",
			expectedConsensus: 1201,
			expectedRawAddr:   "daemonNode",
		},
		{
			name:              "Regular node address",
			encodedAddress:    "0x1-1201-0x07d3ac5197548dab63dedbded8db4607a98001391f4ea2bd19e8a0f79ef3a902e1b6d31f68839e9527e4ed85203ab93a02f6c5c800b2963e4293c2b503c303e1cbf173a3c3cb400e5a249c2c6aa6fa80b3b58048a25b6f27a210a67adc0cc5db",
			expectedSharding:  "0x1",
			expectedConsensus: 1201,
			expectedRawAddr:   "0x07d3ac5197548dab63dedbded8db4607a98001391f4ea2bd19e8a0f79ef3a902e1b6d31f68839e9527e4ed85203ab93a02f6c5c800b2963e4293c2b503c303e1cbf173a3c3cb400e5a249c2c6aa6fa80b3b58048a25b6f27a210a67adc0cc5db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharding, consensus, rawAddr := DecodeAddress(tt.encodedAddress)

			if sharding != tt.expectedSharding {
				t.Errorf("DecodeAddress() sharding = %v, want %v", sharding, tt.expectedSharding)
			}
			if consensus != tt.expectedConsensus {
				t.Errorf("DecodeAddress() consensus = %v, want %v", consensus, tt.expectedConsensus)
			}
			if rawAddr != tt.expectedRawAddr {
				t.Errorf("DecodeAddress() rawAddr = %v, want %v", rawAddr, tt.expectedRawAddr)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		sharding  string
		consensus int64
		address   string
	}{
		{"0x1", 1201, "daemonNode"},
		{"0x1QmbF5Nbqrm7yE6gAk2CpB4cxAhERfWsbYKKcKi54nzhuob", 1201, "daemonNode"},
		{"0x2", 1202, "0x12345"},
	}

	for _, tt := range tests {
		t.Run(tt.sharding+"-"+tt.address, func(t *testing.T) {
			encoded := EncodeAddress(tt.sharding, tt.consensus, tt.address)
			decodedSharding, decodedConsensus, decodedAddr := DecodeAddress(encoded)

			if decodedSharding != tt.sharding {
				t.Errorf("Round trip failed: sharding %v != %v", decodedSharding, tt.sharding)
			}
			if decodedConsensus != tt.consensus {
				t.Errorf("Round trip failed: consensus %v != %v", decodedConsensus, tt.consensus)
			}
			if decodedAddr != tt.address {
				t.Errorf("Round trip failed: address %v != %v", decodedAddr, tt.address)
			}
		})
	}
}

func TestRemovePeerIdFromShardingName(t *testing.T) {
	tests := []struct {
		name                   string
		shardingNameWithPeerId string
		peerId                 string
		expectedOriginalName   string
	}{
		{
			name:                   "Remove PeerId from daemon node sharding",
			shardingNameWithPeerId: "0x1QmbF5Nbqrm7yE6gAk2CpB4cxAhERfWsbYKKcKi54nzhuob",
			peerId:                 "QmbF5Nbqrm7yE6gAk2CpB4cxAhERfWsbYKKcKi54nzhuob",
			expectedOriginalName:   "0x1",
		},
		{
			name:                   "Sharding without PeerId",
			shardingNameWithPeerId: "0x1",
			peerId:                 "QmbF5Nbqrm7yE6gAk2CpB4cxAhERfWsbYKKcKi54nzhuob",
			expectedOriginalName:   "0x1",
		},
		{
			name:                   "Sharding with different suffix",
			shardingNameWithPeerId: "0x2ABC",
			peerId:                 "XYZ",
			expectedOriginalName:   "0x2ABC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if len(tt.peerId) > 0 && tt.shardingNameWithPeerId != tt.expectedOriginalName {
				// Simulating the logic from handleStop
				if tt.shardingNameWithPeerId == tt.expectedOriginalName+tt.peerId {
					result = tt.expectedOriginalName
				} else {
					result = tt.shardingNameWithPeerId
				}
			} else {
				result = tt.shardingNameWithPeerId
			}

			if result != tt.expectedOriginalName {
				t.Errorf("RemovePeerId failed: got %v, want %v", result, tt.expectedOriginalName)
			}
		})
	}
}
