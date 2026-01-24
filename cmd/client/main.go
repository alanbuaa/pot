package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/pkg/logging"
)

// main 是客户端程序的入口函数
//  1. 无参数或不指定-upgrade：正常运行模式，持续发送交易
//  2. 指定-upgrade参数：发起共识升级到指定类型
func main() {

	// 定义命令行参数
	cfgPath := flag.String("c", "config/config.yaml", "path to config yaml file")
	upgradeTarget := flag.String("upgrade", "", "target consensus type for upgrade (e.g., hotstuff, pot, pow)")
	flag.Parse()

	if cfgPath == nil || len(*cfgPath) == 0 {
		fmt.Printf("config path is empty. Use -c to specify the config file")
		os.Exit(2)
	}

	// 初始化日志系统
	logging.Setup(*cfgPath)
	logger := logging.GetLogger().WithField("module", "CLIENT")
	logger.Info("Starting client application")

	// 加载配置文件
	cfg, err := config.NewClientConfig(*cfgPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config-client.yaml")
	}
	logger.Info("Configuration loaded successfully")

	// 创建客户端实例
	client := NewClient(cfg, logger)

	// 根据命令行参数选择运行模式
	if upgradeTarget != nil && len(*upgradeTarget) > 0 {
		// 共识升级模式
		logger.WithField("target", *upgradeTarget).Info("Running in upgrade mode")
		client.upgradeConsensus(*upgradeTarget)
		time.Sleep(2 * time.Second) // 等待交易发送完成
	} else {
		// 正常运行模式
		logger.Info("Running in normal mode (continuous transaction sending)")
		client.normalRun()
	}

	// 停止客户端
	logger.Info("Shutting down client")
	client.Stop()
	logger.Info("Client stopped successfully")
}
