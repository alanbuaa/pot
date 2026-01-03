package upgrade

// UpgradeServiceAdapter adapts UpgradeManager to implement the UpgradeService interface
type UpgradeServiceAdapter struct {
	manager *UpgradeManager
}

// NewUpgradeServiceAdapter creates a new adapter
func NewUpgradeServiceAdapter(manager *UpgradeManager) *UpgradeServiceAdapter {
	return &UpgradeServiceAdapter{
		manager: manager,
	}
}

// GetUpgradeManager returns the underlying upgrade manager
func (a *UpgradeServiceAdapter) GetUpgradeManager() *UpgradeManager {
	return a.manager
}
