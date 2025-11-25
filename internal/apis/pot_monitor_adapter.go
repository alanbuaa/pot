package apis

import (
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
)

// PotMonitorAdapter adapts pot.PoTEngine to provide monitoring service
type PotMonitorAdapter struct {
	engine  *pot.PoTEngine
	monitor model.MonitorService
}

// NewPotMonitorAdapter creates a new adapter for POT monitoring
func NewPotMonitorAdapter(engine *pot.PoTEngine, log *logrus.Entry) model.MonitorService {
	monitor := NewPotMonitor(engine.Worker, engine, log)
	return &PotMonitorAdapter{
		engine:  engine,
		monitor: monitor,
	}
}

// GetSystemOverview returns system overview information
func (a *PotMonitorAdapter) GetSystemOverview() (*model.SystemOverview, error) {
	return a.monitor.GetSystemOverview()
}

// GetPOTStatus returns POT consensus status
func (a *PotMonitorAdapter) GetPOTStatus() (*model.POTStatus, error) {
	return a.monitor.GetPOTStatus()
}

// GetVDFStatus returns VDF computation status
func (a *PotMonitorAdapter) GetVDFStatus() (*model.VDFStatus, error) {
	return a.monitor.GetVDFStatus()
}

// GetCommitteeStatus returns committee consensus status
func (a *PotMonitorAdapter) GetCommitteeStatus() (*model.CommitteeStatus, error) {
	return a.monitor.GetCommitteeStatus()
}

// GetBCIStatus returns BCI incentive system status
func (a *PotMonitorAdapter) GetBCIStatus() (*model.BCIStatus, error) {
	return a.monitor.GetBCIStatus()
}

// GetMempoolStatus returns mempool status
func (a *PotMonitorAdapter) GetMempoolStatus() (*model.MempoolStatus, error) {
	return a.monitor.GetMempoolStatus()
}

// GetNetworkTopology returns network topology information
func (a *PotMonitorAdapter) GetNetworkTopology() (*model.NetworkTopology, error) {
	return a.monitor.GetNetworkTopology()
}
