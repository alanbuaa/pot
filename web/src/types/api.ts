/**
 * POT共识可视化系统 - API类型定义
 * 
 * 该文件定义了可视化系统所需的所有数据接口类型
 */

// ==================== 系统概览 ====================
export interface SystemOverview {
  uptime: number;              // 系统运行时长（秒）
  totalNodes: number;          // 总节点数
  onlineNodes: number;         // 在线节点数
  consensusTypes: string[];    // 支持的共识类型
  networkStatus: 'healthy' | 'warning' | 'error'; // 网络状态
  networkUtilization: number;  // 网络利用率 (0-100)
  currentHeight: number;       // 当前区块高度
  avgBlockTime: number;        // 平均出块时间（秒）
  lastBlockTime: string;       // 最新区块时间
  currentTPS?: number;         // 当前TPS（可选）
  executorStatus?: string;     // 执行器状态（可选）
}

// ==================== POT共识状态 ====================
export interface POTStatus {
  consensusType: 'POT';        // 共识类型
  epoch: number;               // 当前epoch
  difficulty: string;          // 挖矿难度值（16进制字符串）
  currentHeight: number;       // 当前区块高度
  workFlag: boolean;           // 挖矿工作状态
  timestamp: string;           // 当前时间戳（ISO 8601格式）
  nonce: number;               // Nonce值
  uncleCount: number;          // 叔块数量
  avgMiningTime: number;       // 平均挖矿耗时（秒）
  miningSuccessRate: number;   // 挖矿成功率 (0-100)
}

// ==================== VDF状态 ====================
export interface VDFStatus {
  vdf0: {
    progress: number;          // 进度百分比 (0-100)
    iterations: number;        // 迭代次数
    status: 'computing' | 'done' | 'idle'; // 计算状态
    channelBuffer: number;     // 通道缓冲区大小
  };
  vdf1: Array<{
    workerId: number;          // 工作线程ID (0-3)
    progress: number;          // 进度百分比 (0-100)
    iterations?: number;       // 迭代次数（可选）
    status: 'computing' | 'done' | 'idle'; // 计算状态
  }>;
  vdfHalf: {
    progress: number;          // VDF半计算进度 (0-100)
    status: 'computing' | 'done' | 'idle'; // 计算状态
    channelBuffer: number;     // 缓冲区大小
  };
  vdfChecker: {
    status: 'active' | 'inactive'; // 验证器状态
    verifyFailCount: number;   // 验证失败次数
  };
  abortStatus: boolean;        // 中止状态
  avgComputeTime: number;      // 平均计算耗时（毫秒）
  totalIterations: number;     // 总迭代次数
  cpuCounter: number;          // 并行工作线程数
}

// ==================== 委员会状态 ====================
export interface CommitteeStatus {
  consensusType: 'SimpleWhirly' | 'CRWhirly' | 'Whirly'; // 共识类型
  committeeSize: number;       // 委员会大小
  committeeCount: number;      // 委员会数量
  confirmDelay: number;        // 确认延迟（区块数）
  workHeight: number;          // 委员会工作高度
  batchSize: number;           // 批处理大小
  inCommittee: boolean;        // 是否在委员会中
  role: 'leader' | 'member' | 'observer'; // 节点角色
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
    status: 'active' | 'inactive'; // 分片状态
  }>;
  workStage: 'init' | 'shuffle' | 'draw' | 'share' | 'consensus'; // 工作阶段
  timeout: number;             // 超时配置（毫秒）
  messageQueueLength: number;  // 消息队列长度
  electionHeight: number;      // 选举区块高度
}

// ==================== BCI激励状态 ====================
export interface BCIStatus {
  totalReward: number;         // 总奖励金额
  lockedReward: number;        // 锁定的激励金额
  totalInterest: number;       // 累计利息总数
  rewardRatio: {
    exchequer: number;         // 国库占比 (0-100)
    miner: number;             // 矿工占比 (0-100)
    uncleBlockMiner: number;   // 叔块矿工占比 (0-100)
    committeeLeader: number;   // 委员会Leader占比 (0-100)
    committeeMember: number;   // 委员会成员占比 (0-100)
  };
  lockRates: {
    saving: number;            // 活期利率
    halfYear: number;          // 半年期利率
    oneYear: number;           // 一年期利率
    threeYears: number;        // 三年期利率
    tenYears: number;          // 十年期利率
  };
  coinbaseLock: number;        // Coinbase锁定期（区块数）
  executeHeight: number;       // 执行高度
  incentiveHeight: number;     // 激励高度
  pendingRewards: number;      // 待分配奖励数量
  utxoCount: number;           // UTXO总数
  totalDistributed: number;    // 历史累计分发总额
}

// ==================== 交易池状态 ====================
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
    hash: string;              // 交易哈希（64位16进制字符串）
    type: 'normal' | 'bci' | 'devastate'; // 交易类型
    timestamp: string;         // 时间戳（ISO 8601格式）
    status: 'pending' | 'confirmed'; // 交易状态
  }>;
}

// ==================== 网络拓扑 ====================
export interface NetworkTopology {
  nodes: Array<{
    id: number;                // 节点ID
    peerId: string;            // PeerID
    address: string;           // 节点地址
    status: 'online' | 'offline'; // 节点状态
    connections: number;       // 连接数
    latency: number;           // 延迟（毫秒）
    type?: 'committee' | 'pot'; // 节点类型
    isLeader?: boolean;        // 是否为Leader（仅committee类型）
  }>;
  edges: Array<{
    source: number;            // 源节点ID
    target: number;            // 目标节点ID
    latency: number;           // 延迟（毫秒）
    messageCount: number;      // 消息数量
  }>;
  p2pAdaptorType: 'p2p' | 'libp2p'; // P2P适配器类型
  subscribedTopics: string[];  // 订阅主题列表
  messageQueueLength: number;  // 消息队列长度
  networkBandwidth: number;    // 网络带宽（字节/秒）
}

// ==================== 存储状态 ====================
export interface StorageStatus {
  totalSize: number;           // 总存储大小（字节）
  usedSize: number;            // 已使用大小（字节）
  availableSize: number;       // 可用大小（字节）
  utilizationRate: number;     // 使用率 (0-100)
  blockCount?: number;         // 区块数量（可选）
  vdfHeight?: number;          // VDF高度（可选）
  compressionRatio?: number;   // 压缩率 (0-1)（可选）
  buckets?: {                  // 存储桶（可选）
    blocks?: { size: number };
    utxo?: { size: number };
    executed?: { size: number };
    client?: { size: number };
  };
}

// ==================== WebSocket消息 ====================
export interface WSMessage {
  type: 'pot' | 'vdf' | 'committee' | 'mempool' | 'network' | 'system' | 'bci'; // 消息类型
  timestamp: string;           // 时间戳（ISO 8601格式）
  data: POTStatus | VDFStatus | CommitteeStatus | MempoolStatus | NetworkTopology | SystemOverview | BCIStatus; // 消息数据
}

// ==================== 区块信息 ====================
export interface BlockInfo {
  height: number;              // 区块高度
  hash: string;                // 区块哈希（16进制字符串）
  timestamp: number;           // 时间戳（Unix时间戳，秒）
  txCount: number;             // 交易数量
  size: number;                // 区块大小（字节）
  miner: string;               // 矿工地址/PeerID
}

export interface BlockDetail {
  height: number;              // 区块高度
  hash: string;                // 区块哈希（16进制字符串）
  parentHash: string;          // 父区块哈希
  timestamp: number;           // 时间戳（Unix时间戳，秒）
  txCount: number;             // 交易数量
  size: number;                // 区块大小（字节）
  miner: string;               // 矿工地址/PeerID
  difficulty: string;          // 难度值（16进制字符串）
  nonce: number;               // Nonce值
  mixdigest: string;           // Mixdigest（16进制字符串）
  uncleHashes: string[];       // 叔块哈希列表
  transactions: string[];      // 交易哈希列表
  committeePubkey: string;     // 委员会公钥
}
