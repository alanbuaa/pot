package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
)

// main 是升级客户端程序的入口函数
// 运行模式：
//  1. 无参数或不指定-upgrade：正常运行模式，持续发送交易
//  2. 指定-upgrade参数：发起共识升级提案
//  3. 指定-confirm参数：发起升级确认投票
func main() {
	// 定义命令行参数
	cfgPath := flag.String("c", "cmd/upgrade/client/config.yaml", "path to config yaml file")
	upgradeTarget := flag.String("upgrade", "", "target consensus type for upgrade (e.g., hotstuff, pot, pow)")
	confirmProposal := flag.String("confirm", "", "proposal ID to confirm (hex string)")
	txDuration := flag.Int("duration", 30, "duration in seconds to send transactions (0 for infinite)")
	flag.Parse()

	if cfgPath == nil || len(*cfgPath) == 0 {
		fmt.Printf("config path is empty. Use -c to specify the config file\n")
		os.Exit(2)
	}

	// 初始化日志系统
	logging.Setup(*cfgPath)
	logger := logging.GetLogger().WithField("module", "UPGRADE-CLIENT")

	logger.Info("=================================================")
	logger.Info("  Upgradeable Consensus Client Starting...")
	logger.Info("=================================================")

	// 加载配置文件
	cfg, err := config.NewClientConfig(*cfgPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config")
	}
	logger.Info("Configuration loaded successfully")

	// 创建客户端实例
	client := NewUpgradeClient(cfg, logger)

	// 根据命令行参数选择运行模式
	if confirmProposal != nil && len(*confirmProposal) > 0 {
		// 确认提案模式
		logger.WithField("proposal_id", *confirmProposal).Info("Running in confirm mode")
		client.confirmUpgrade(*confirmProposal)
		time.Sleep(2 * time.Second)
	} else if upgradeTarget != nil && len(*upgradeTarget) > 0 {
		// 共识升级模式
		logger.WithField("target", *upgradeTarget).Info("Running in upgrade mode")
		client.proposeUpgrade(*upgradeTarget)
		time.Sleep(2 * time.Second)
	} else {
		// 正常运行模式
		duration := time.Duration(*txDuration) * time.Second
		if *txDuration <= 0 {
			logger.Info("Running in normal mode (continuous transaction sending)")
		} else {
			logger.WithField("duration_seconds", *txDuration).Info("Running in normal mode (limited duration)")
		}
		client.normalRun(duration)
	}

	// 停止客户端
	logger.Info("Shutting down client")
	client.Stop()
	logger.Info("Client stopped successfully")
}
