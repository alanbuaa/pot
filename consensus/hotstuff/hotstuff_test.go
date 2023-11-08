package hotstuff

import (
	"os"
	"testing"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"github.com/zzz136454872/upgradeable-consensus/utils"
)

func TestMain(m *testing.M) {
	err := os.Chdir("../../")
	utils.PanicOnError(err)
	os.Exit(m.Run())
}

func TestGenerateGenesisBlock(t *testing.T) {
	block := GenerateGenesisBlock()
	t.Log(types.String(block))
}

func TestHotStuffImpl_GetSelfInfo(t *testing.T) {
	cfg, err := config.NewConfig("config/config.yaml", 1)
	utils.PanicOnError(err)
	h := &HotStuffImpl{}
	h.Config = cfg.Consensus.Upgradeable.InitConsensus
	h.ID = 1
	t.Log(h.GetSelfInfo())
}

func TestHotStuffImpl_GetLeader(t *testing.T) {
	wd, err := os.Getwd()
	utils.PanicOnError(err)
	t.Log("pwd", wd)
	cfg, err := config.NewConfig("config/config.yaml", 1)
	utils.PanicOnError(err)
	h := &HotStuffImpl{}
	h.Config = cfg.Consensus.Upgradeable.InitConsensus
	h.Config.Nodes = cfg.Nodes
	h.View = &View{
		ViewNum: 6,
		Primary: 1,
	}
	h.ID = 1
	leader := h.GetLeader()
	t.Log(leader)
}
