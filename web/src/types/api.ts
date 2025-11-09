// 系统概览
export interface SystemOverview {
  uptime: number;              // 系统运行时长（秒）
  totalNodes: number;          // 总节点数
  onlineNodes: number;         // 在线节点数
  consensusTypes: string[];    // 支持的共识类型
  networkStatus: string;       // 网络状态
  networkUtilization: number;  // 网络利用率 (0-100)
  executorStatus: string;      // 执行器状态
  storageUsage: {
    total: number;             // 总存储空间
    used: number;              // 已使用空间
    percentage: number;        // 使用百分比
  };
  mempoolSize: number;         // 交易池大小
  currentTPS: number;          // 当前TPS
  currentHeight: number;       // 当前区块高度
  avgBlockTime: number;        // 平均出块时间（秒）
  lastBlockTime: string;       // 最新区块时间
}

// POT共识状态
export interface POTStatus {
  consensusType: string;       // "POT"
  epoch: number;               // 当前epoch
  difficulty: string;          // 挖矿难度值
  currentHeight: number;       // 当前区块高度
  workFlag: boolean;           // 挖矿工作状态
  timestamp: string;           // 当前时间戳
  nonce: number;               // Nonce值
  uncleCount: number;          // 叔块数量
  avgMiningTime: number;       // 平均挖矿耗时（秒）
  miningSuccessRate: number;   // 挖矿成功率 (0-100)
}

// VDF状态
export interface VDFStatus {
  vdf0: {
    progress: number;          // 进度百分比 (0-100)
    iterations: number;        // 迭代次数
    status: string;            // "computing" | "done" | "idle"
    channelBuffer: number;     // 通道缓冲区大小
  };
  vdf1: Array<{
    workerId: number;          // 工作线程ID (0-3)
    progress: number;          // 进度百分比
    status: string;            // 状态
  }>;
  vdfHalf: {
    progress: number;          // VDF半计算进度
    status: string;            // 状态
    channelBuffer: number;     // 缓冲区大小
  };
  vdfChecker: {
    status: string;            // 验证器状态
    verifyFailCount: number;   // 验证失败次数
  };
  abortStatus: boolean;        // 中止状态
  avgComputeTime: number;      // 平均计算耗时（毫秒）
  totalIterations: number;     // 总迭代次数
  cpuCounter: number;          // 并行工作线程数
}

// 委员会状态
export interface CommitteeStatus {
  consensusType: string;       // "SimpleWhirly" | "CRWhirly" | "Whirly"
  committeeSize: number;       // 委员会大小
  committeeCount: number;      // 委员会数量
  confirmDelay: number;        // 确认延迟
  workHeight: number;          // 委员会工作高度
  batchSize: number;           // 批处理大小
  inCommittee: boolean;        // 是否在委员会中
  role: string;                // "leader" | "member" | "observer"
  committee: Array<{
    address: string;           // 地址
    publicKey: string;         // 公钥
    isLeader: boolean;         // 是否为Leader
  }>;
  committeePublicKey: string;  // 委员会聚合公钥
  shardings: Array<{
    name: string;              // 分片名称
    id: number;                // 分片ID
    leaderAddress: string;     // Leader地址
    committeeMembers: number;  // 委员会成员数
    status: string;            // "active" | "inactive"
  }>;
  workStage: string;           // 工作阶段
  timeout: number;             // 超时配置（毫秒）
  messageQueueLength: number;  // 消息队列长度
  electionHeight: number;      // 选举区块高度
}

// BCI激励状态
export interface BCIStatus {
  totalReward: number;         // 总奖励金额
  lockedReward: number;        // 锁定的激励金额
  totalInterest: number;       // 累计利息总数
  rewardRatio: {
    exchequer: number;         // 国库
    miner: number;             // 矿工
    uncleBlockMiner: number;   // 叔块矿工
    committeeLeader: number;   // 委员会Leader
    committeeMember: number;   // 委员会成员
  };
  lockRates: {
    saving: number;            // 活期
    halfYear: number;          // 半年
    oneYear: number;           // 一年
    threeYears: number;        // 三年
    tenYears: number;          // 十年
  };
  coinbaseLock: number;        // Coinbase锁定期
  executeHeight: number;       // 执行高度
  incentiveHeight: number;     // 激励高度
  pendingRewards: number;      // 待分配奖励数量
  utxoCount: number;           // UTXO总数
  totalDistributed: number;    // 历史累计分发总额
}

// 交易池状态
export interface MempoolStatus {
  totalSize: number;           // 交易池总大小
  markedTxs: number;           // 已提议交易数
  unmarkedTxs: number;         // 待提议交易数
  txTypes: {
    normal: number;            // 普通交易数
    bci: number;               // BCI交易数
    devastate: number;         // Devastate交易数
  };
  avgConfirmTime: number;      // 平均确认时间（秒）
  verifySuccessRate: number;   // 验证成功率 (0-100)
  memoryUsage: number;         // 内存占用（字节）
  recentTxs: Array<{
    hash: string;              // 交易哈希
    type: string;              // 交易类型
    timestamp: string;         // 时间戳
    status: string;            // "pending" | "confirmed"
  }>;
}

// 网络拓扑
export interface NetworkTopology {
  nodes: Array<{
    id: number;                // 节点ID
    peerId: string;            // PeerID
    address: string;           // 地址
    status: string;            // "online" | "offline"
    connections: number;       // 连接数
    latency: number;           // 延迟（毫秒）
    type?: string;             // 节点类型：committee | pot
    isLeader?: boolean;        // 是否为Leader
  }>;
  edges: Array<{
    source: number;            // 源节点ID
    target: number;            // 目标节点ID
    latency: number;           // 延迟（毫秒）
    messageCount: number;      // 消息数量
  }>;
  p2pAdaptorType: string;      // "p2p" | "libp2p"
  subscribedTopics: string[];  // 订阅主题列表
  messageQueueLength: number;  // 消息队列长度
  networkBandwidth: number;    // 网络带宽（字节/秒）
}

// 存储状态
export interface StorageStatus {
  totalSize: number;           // 总大小（字节）
  buckets: {
    blocks: {
      size: number;            // BlocksBucket大小
      count: number;           // 记录数
    };
    executed: {
      size: number;            // ExecutedBucket大小
      count: number;           // 记录数
    };
    utxo: {
      size: number;            // UTXOBucket大小
      count: number;           // 记录数
    };
    client: {
      size: number;            // ClientBucket大小
      count: number;           // 记录数
    };
  };
  compressionRatio: number;    // 压缩率 (0-1)
  vdfHeight: number;           // VDF高度记录
  blockCount: number;          // 区块总数
}

// 历史指标
export interface MetricsHistory {
  metric: string;              // 指标类型
  timeRange: string;           // 时间范围
  interval: string;            // 采样间隔
  data: Array<{
    timestamp: string;         // 时间戳
    value: number;             // 指标值
  }>;
  avg: number;                 // 平均值
  max: number;                 // 最大值
  min: number;                 // 最小值
}

// WebSocket 消息
export interface WSMessage {
  type: string;
  timestamp: string;
  data: any;
}
