
# 可更新共识的可视化规划文档

根据以下参数指标，制作可视化大屏页

## 可视化的指标

### 整体架构板块
1. 共识节点数量
2. 支持的共识类型
3. 共识池
   - 共识池状态：占用中/空闲中
4. 网络服务状态和利用率：正常/百分比
5. 执行器状态：正常/异常
6. 存储空间消耗
7. 交易池大小
8. 交易吞吐率
9. 系统启动时间和运行时长
10. P2P节点连接数和节点拓扑
11. 节点地址和PeerID信息
12. 消息队列长度（MsgByteEntrance）
13. 区块链总高度和最新区块时间
14. 平均出块时间
15. 区块确认延迟

### POT共识状态
- 当前共识类型：POT
- 当前epoch数量
- 挖矿难度值
- 当前区块高度
- VDF0迭代进度
- VDF半计算进度（vdfhalf）
- 并行VDF1工作线程数（cpuCounter）
- VDF验证失败次数
- 挖矿工作状态（workFlag）
- 当前时间戳
- Nonce值
- 叔块数量
- 挖矿成功率统计
- 平均挖矿耗时

### 委员会共识
- 共识类型 (Simple Whirly / CR Whirly / Whirly)
- 委员会大小 (当前: 4, 参数: Commiteelen)
- 委员会数量 (SmallN: 4)
- 确认延迟 (ConfirmDelay: 6)
- 委员会工作高度
- 批处理大小 (BatchSize: 2-10)
- 委员会激活状态 (inCommittee: true/false)
- 委员会权重 (Weight)
- 委员工作阶段：初始化阶段/置换阶段/抽选阶段/分布式份额分发/委员会共识工作
- 当前分片（Sharding）状态
- 分片名称和ID
- Leader地址
- 委员会超时配置（Timeout）
- 委员会共识消息队列状态
- 委员会选举的区块高度
- 自身在委员会的身份（Leader/Member/Observer）

### BCI激励
1. 总奖励金额（TotalReward: 65536）
2. 锁定的激励金额
3. 利息总数
4. 奖励分配比例
5. 锁定利率参数
6. 各类型奖励占比：
   - 国库（exchequer: 30%）
   - 矿工（Miner: 50%）
   - 叔块矿工（UncleBlockMiner: 2%）
   - 委员会Leader（CommitteeLeader: 20%）
   - 委员会成员（CommitteeMember: 10%）
8. 定期存款利率：
   - 半年利率（HalfYearRate: 0.5%）
   - 一年利率（OneYearRate: 1%）
   - 三年利率（ThreeYearRate: 2%）
   - 十年利率（TenYearRate: 5%）
   - 活期利率（savingRate: 0.1%）
9. 执行高度（executeheight）
10. 激励高度（incentiveheight）
11. 待分配奖励队列
12. UTXO总数和状态
15. 历史激励总额统计
16. 地址余额查询

### 交易池和执行状态
1. 交易池大小（marked/unmarked）
2. 待提议交易数量
3. 交易哈希映射状态
4. 区块执行队列长度
5. 已执行区块高度
7. 交易验证成功率
8. 交易平均确认时间
9. 交易类型分布统计
10. Mempool内存占用

### 网络和P2P
2. 订阅主题（Topic）
6. 节点间延迟统计
8. 网络带宽使用率