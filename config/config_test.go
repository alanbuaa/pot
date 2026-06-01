package config

import (
	"os"
	"testing"

	"github.com/zzz136454872/upgradeable-consensus/utils"
)

func TestMain(m *testing.M) {
	err := os.Chdir("../")
	utils.PanicOnError(err)
	os.Exit(m.Run())
}

func TestHotStuffConfig_ReadConfig(t *testing.T) {
	cfg, err := NewConfig("config/config.yaml", 1)
	utils.PanicOnError(err)
	utils.PanicOnError(err)
	t.Log(cfg.Nodes[0].Address)
}

func funGen(a int) func() int {
	return func() int {
		return a
	}
}

func TestFun(t *testing.T) {
	f := funGen(1)
	t.Log(f())
	f2 := funGen(2)
	t.Log(f2())
	t.Log(f())
}
