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
- executor中发送交易的CommitBlock问题

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

## 脚本启动

请根据这些配置文件，帮我在项目根目录下写一个run.sh脚本，具备以下功能：

1. 初始化。输入一个命名空间、节点数量、客户端数量来生成对应的节点和客户端配置文件，通过新建命名空间来保存到对应的目录下，生成的配置文件以及节点运行产生的数据也保存到这个文件夹下，并使用make run_keyegn来生成对应数量的节点密钥，每个节点的密钥保存在自己的目录下，且公钥也保存到每个节点下。
2. 生成配置文件时，根据输入的参数和模版template.yaml对涉及的端口和日志路径等进行替换，同时在脚本中设置节点组件的默认端口后，支持创建多个节点和多个节点的配置文件的生成，默认填充递增的端口后，且避免端口重复。
3. 操作节点。包括运行、停止。根据输入的命名空间，默认将其对应的所有节点启动、停止，如果有指定节点，则只操作那个节点。启动后可选择将节点的日志输出到终端。
4. 清理。清理指定命名空间下的所有数据。
5. 默认有一个名为test的命名空间，是单节点单客户端的测试。
