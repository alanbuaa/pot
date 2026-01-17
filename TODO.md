### 待完成列表
- 单机多节点
- 多机多节点
- 端口公网访问
- 容器化编排
- 数据/配置目录规范
- 项目代码结构

## 问题
- leveldb、boltdb分别做啥？
- http的结构
- bci文件的作用
- tests中各个测试的作用

## 规划
- 由client来发起升级操作，当然也包含发起普通交易


## 提示词
请帮我分析代码逻辑，根据软件开发规范在关键位置补充日志打印并确定输出日志的级别，现有日志打印请完善代码中的日志文本和输出等级，使用本项目日志库"github.com/zzz136454872/upgradeable-consensus/pkg/logging" ，在主模块中使用
// 初始化日志系统
	logging.Setup("config/config.yaml")
	logger := logging.GetLogger().WithField("module", "EXECUTOR")
来初始化模块名称。注意，使用了logging.GetLogger().WithField("module", "EXECUTOR")之后，后续的日志打印就不再需要写模块名"EXECUTOR"。

## 交易追踪

[TRACE-1] P2P layer received packet (p2p.go)
  ↓
[TRACE-1.1] Forwarding packet to consensus engine
  ↓
[TRACE-2] PoTEngine received packet (handle_msg.go)
  ↓
[TRACE-2.3] Received CLIENTPACKET
  ↓
[TRACE-2.4] Calling handleRequest
  ↓
[TRACE-3] handleRequest called
  ↓
[TRACE-3.2] Transaction verification passed
  ↓
[TRACE-3.4] Processing transaction
  ↓
[TRACE-3.5] Forwarding NORMAL transaction to UpperConsensus
  ↓
[TRACE-3.6] Transaction forwarded to Whirly
  ↓
[TRACE-4] Sharding received request (sharding.go)
  ↓
[TRACE-4.1] Forwarding request to node
  ↓
[TRACE-5] Sharding.CommitBlock called
  ↓
[TRACE-5.1] Committing block to executor
  ↓
[TRACE-6] LocalExecutor.CommitBlock called ← **应该到达这里**
  ↓
[TRACE-7] ExecutorServiceImpl.CommitBlock CALLED ← **最终目标**