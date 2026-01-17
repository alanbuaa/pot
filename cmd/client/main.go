package main

import (
	"os"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
)

// main 是客户端程序的入口函数
//  1. 无参数：正常运行模式，持续发送交易
//  2. 带参数 "upgrade <consensus_type>"：发起共识升级
func main() {
	// 初始化日志系统
	logging.Setup("config/config.yaml")
	logger := logging.GetLogger().WithField("module", "CLIENT")
	logger.Info("Starting client application")

	// 加载配置文件
	cfg, err := config.NewConfig("config/config.yaml", 0)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config.yaml")
	}
	logger.Info("Configuration loaded successfully")

	// 创建客户端实例
	client := NewClient(cfg, logger)

	// 根据命令行参数选择运行模式
	if len(os.Args) == 1 {
		// 正常运行模式
		logger.Info("Running in normal mode (continuous transaction sending)")
		client.normalRun()
	} else if len(os.Args) == 3 {
		// 共识升级模式
		if os.Args[1] != "upgrade" {
			logger.WithField("command", os.Args[1]).Error("Unsupported command, use 'upgrade'")
			os.Exit(1)
		}
		logger.WithField("target", os.Args[2]).Info("Running in upgrade mode")
		client.upgradeConsensus(os.Args[2])
		time.Sleep(2 * time.Second) // 等待交易发送完成
	} else {
		logger.Error("Invalid arguments. Usage: client OR client upgrade <consensus_type>")
		os.Exit(1)
	}
	// 停止客户端
	logger.Info("Shutting down client")
	client.Stop()
	logger.Info("Client stopped successfully")
}
