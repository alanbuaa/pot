package node

import (
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/consensus/model"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis"
)

func RegisterApiServices(apiServer *apis.ApiServer, c model.Consensus, log *logrus.Entry) {
	consensusType := c.GetConsensusType()
	log.WithField("consensus_type", consensusType).Info("Registering API services")

	switch consensusType {
	case "pot":
		if potEngine, ok := c.(*pot.PoTEngine); ok {
			// Register transaction API service
			apiServer.RegisterPotService(apis.NewPotWorkerAdapter(potEngine.Worker))
			log.Info("Registered PoT transaction API service")

			// Register monitoring API service
			apiServer.RegisterMonitorService(apis.NewPotMonitorAdapter(potEngine, log))
			log.Info("Registered PoT monitoring API service")
		} else {
			log.Error("Failed to cast consensus to PoTEngine for API registration")
		}
	// case "pow":
	// 	apiServer.RegisterPowService(apis.NewPowWorkerAdapter(cons.worker))
	// 	log.Info("Registered PoW API service")
	default:
		log.WithField("consensus_type", consensusType).Info("No specific API service to register for this consensus type")
	}
}
