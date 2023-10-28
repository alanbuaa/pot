package network

// Transmission Evidence Structure
type TransmissionEvidence interface {
	// Verify the validity of the transmission evidence
	VerifyEvidence() (bool, error)
	// Get a list of shares taken by each transmitter
	// in the current evidence
	GetShareList() ([][]byte, error)
}

type EvidenceManagement interface {
	// Add new transmission evidence
	AddEvidence(te *TransmissionEvidence) error
	// The strategy of taking shares is to calculate the optimal
	// share taking based on the current share list, remaining
	// shares and own network conditions
	GetBestTakeShare(shareList []byte) ([]byte, error)
}
