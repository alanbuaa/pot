/**
 * POT共识可视化系统 - Mock数据服务
 * 
 * 提供模拟数据生成，用于开发和演示
 */

import type {
  SystemOverview,
  POTStatus,
  VDFStatus,
  CommitteeStatus,
  BCIStatus,
  MempoolStatus,
  NetworkTopology,
  BlockInfo,
  BlockDetail
} from '@/types/api'

/**
 * Mock数据生成器
 */
export class MockDataService {
  private startTime = Date.now()
  private blockHeight = 12345
  private epoch = 89

  /**
   * 获取系统概览Mock数据
   */
  getSystemOverview(): SystemOverview {
    const uptime = Math.floor((Date.now() - this.startTime) / 1000)
    return {
      uptime,
      totalNodes: 12,
      onlineNodes: 10,
      consensusTypes: ['POT', 'SimpleWhirly'],
      networkStatus: 'healthy',
      networkUtilization: Math.random() * 30 + 40, // 40-70
      currentHeight: this.blockHeight,
      avgBlockTime: Math.random() * 2 + 4, // 4-6s
      lastBlockTime: new Date().toISOString()
    }
  }

  /**
   * 获取POT共识状态Mock数据
   */
  getPotStatus(): POTStatus {
    return {
      consensusType: 'POT',
      epoch: this.epoch,
      difficulty: '0x' + Math.floor(Math.random() * 0xffffff).toString(16).padStart(6, '0'),
      currentHeight: this.blockHeight,
      workFlag: Math.random() > 0.3,
      timestamp: new Date().toISOString(),
      nonce: Math.floor(Math.random() * 1000000),
      uncleCount: Math.floor(Math.random() * 5),
      avgMiningTime: Math.random() * 3 + 2, // 2-5s
      miningSuccessRate: Math.random() * 20 + 75 // 75-95%
    }
  }

  /**
   * 获取VDF计算状态Mock数据
   */
  getVDFStatus(): VDFStatus {
    return {
      vdf0: {
        progress: Math.random() * 100,
        iterations: Math.floor(Math.random() * 100000),
        status: ['computing', 'done', 'idle'][Math.floor(Math.random() * 3)] as 'computing' | 'done' | 'idle',
        channelBuffer: Math.floor(Math.random() * 10)
      },
      vdf1: Array.from({ length: 4 }, (_, i) => ({
        workerId: i,
        progress: Math.random() * 100,
        iterations: Math.floor(Math.random() * 100000),
        status: ['computing', 'done', 'idle'][Math.floor(Math.random() * 3)] as 'computing' | 'done' | 'idle'
      })),
      vdfHalf: {
        progress: Math.random() * 100,
        status: ['computing', 'done', 'idle'][Math.floor(Math.random() * 3)] as 'computing' | 'done' | 'idle',
        channelBuffer: Math.floor(Math.random() * 10)
      },
      vdfChecker: {
        status: 'active',
        verifyFailCount: Math.floor(Math.random() * 3)
      },
      abortStatus: false,
      avgComputeTime: Math.random() * 50 + 20, // 20-70ms
      totalIterations: Math.floor(Math.random() * 1000000),
      cpuCounter: 4
    }
  }

  /**
   * 获取委员会状态Mock数据
   */
  getCommitteeStatus(): CommitteeStatus {
    return {
      consensusType: 'SimpleWhirly',
      committeeSize: 4,
      committeeCount: 4,
      confirmDelay: 6,
      workHeight: this.blockHeight - 6,
      batchSize: Math.floor(Math.random() * 8) + 2, // 2-10
      inCommittee: true,
      role: 'member',
      committee: Array.from({ length: 4 }, (_, i) => ({
        address: '0x' + Math.random().toString(16).substr(2, 40),
        publicKey: '0x' + Math.random().toString(16).substr(2, 64),
        isLeader: i === 0
      })),
      committeePublicKey: '0x' + Math.random().toString(16).substr(2, 64),
      shardings: [
        {
          name: 'Shard-0',
          id: 0,
          leaderAddress: '0x' + Math.random().toString(16).substr(2, 40),
          committeeMembers: 4,
          status: 'active'
        }
      ],
      workStage: ['init', 'shuffle', 'draw', 'share', 'consensus'][Math.floor(Math.random() * 5)] as CommitteeStatus['workStage'],
      timeout: 5000,
      messageQueueLength: Math.floor(Math.random() * 20),
      electionHeight: this.blockHeight - 100
    }
  }

  /**
   * 获取BCI激励状态Mock数据
   */
  getBCIStatus(): BCIStatus {
    return {
      totalReward: 65536,
      lockedReward: Math.floor(Math.random() * 10000),
      totalInterest: Math.floor(Math.random() * 5000),
      rewardRatio: {
        exchequer: 30,
        miner: 50,
        uncleBlockMiner: 2,
        committeeLeader: 20,
        committeeMember: 10
      },
      lockRates: {
        saving: 0.1,
        halfYear: 0.5,
        oneYear: 1,
        threeYears: 2,
        tenYears: 5
      },
      coinbaseLock: 6,
      executeHeight: this.blockHeight,
      incentiveHeight: this.blockHeight - 1,
      pendingRewards: Math.floor(Math.random() * 100),
      utxoCount: Math.floor(Math.random() * 1000) + 500,
      totalDistributed: Math.floor(Math.random() * 100000)
    }
  }

  /**
   * 获取交易池状态Mock数据
   */
  getMempoolStatus(): MempoolStatus {
    const totalSize = Math.floor(Math.random() * 100) + 50
    const markedTxs = Math.floor(totalSize * 0.6)
    
    return {
      totalSize,
      markedTxs,
      unmarkedTxs: totalSize - markedTxs,
      txTypes: {
        normal: Math.floor(totalSize * 0.7),
        bci: Math.floor(totalSize * 0.2),
        devastate: Math.floor(totalSize * 0.1)
      },
      avgConfirmTime: Math.random() * 10 + 5, // 5-15s
      verifySuccessRate: Math.random() * 10 + 90, // 90-100%
      memoryUsage: Math.floor(Math.random() * 10485760) + 5242880, // 5-15MB
      recentTxs: Array.from({ length: 5 }, () => ({
        hash: '0x' + Math.random().toString(16).substr(2, 64),
        type: ['normal', 'bci', 'devastate'][Math.floor(Math.random() * 3)] as 'normal' | 'bci' | 'devastate',
        timestamp: new Date(Date.now() - Math.random() * 60000).toISOString(),
        status: (Math.random() > 0.3 ? 'confirmed' : 'pending') as 'confirmed' | 'pending'
      }))
    }
  }

  /**
   * 获取网络拓扑Mock数据
   */
  getNetworkTopology(): NetworkTopology {
    // 生成委员会节点（4个）
    const committeeNodes = Array.from({ length: 4 }, (_, i) => ({
      id: i,
      peerId: 'peer-committee-' + i,
      address: '0x' + Math.random().toString(16).substr(2, 40),
      status: 'online' as const,
      connections: Math.floor(Math.random() * 8) + 5,
      latency: Math.floor(Math.random() * 50) + 10,
      type: 'committee' as const,
      isLeader: i === 0
    }))

    // 生成POT节点（8个）
    const potNodes = Array.from({ length: 8 }, (_, i) => ({
      id: i + 4,
      peerId: 'peer-pot-' + i,
      address: '0x' + Math.random().toString(16).substr(2, 40),
      status: (Math.random() > 0.1 ? 'online' : 'offline') as 'online' | 'offline',
      connections: Math.floor(Math.random() * 5) + 2,
      latency: Math.floor(Math.random() * 100) + 20,
      type: 'pot' as const,
      isLeader: false
    }))

    const allNodes = [...committeeNodes, ...potNodes]

    // 生成边
    const edges = []
    
    // 委员会节点之间全连接
    for (let i = 0; i < 4; i++) {
      for (let j = i + 1; j < 4; j++) {
        edges.push({
          source: i,
          target: j,
          latency: Math.floor(Math.random() * 30) + 10,
          messageCount: Math.floor(Math.random() * 1000)
        })
      }
    }

    // POT节点与委员会的连接
    for (let i = 4; i < 12; i++) {
      const committeeId = Math.floor(Math.random() * 4)
      edges.push({
        source: i,
        target: committeeId,
        latency: Math.floor(Math.random() * 80) + 20,
        messageCount: Math.floor(Math.random() * 500)
      })
    }

    // POT节点之间的部分连接
    for (let i = 4; i < 12; i++) {
      for (let j = i + 1; j < 12; j++) {
        if (Math.random() > 0.6) {
          edges.push({
            source: i,
            target: j,
            latency: Math.floor(Math.random() * 100) + 30,
            messageCount: Math.floor(Math.random() * 200)
          })
        }
      }
    }

    return {
      nodes: allNodes,
      edges,
      p2pAdaptorType: 'libp2p',
      subscribedTopics: ['blocks', 'transactions', 'consensus'],
      messageQueueLength: Math.floor(Math.random() * 50),
      networkBandwidth: Math.floor(Math.random() * 1048576) + 524288 // 0.5-1.5MB/s
    }
  }

  /**
   * 模拟区块增长（用于测试）
   */
  incrementBlock() {
    this.blockHeight++
    if (this.blockHeight % 100 === 0) {
      this.epoch++
    }
  }

  /**
   * 获取最近的N个区块Mock数据
   */
  getRecentBlocks(count: number = 10): BlockInfo[] {
    const blocks: BlockInfo[] = []
    const startHeight = Math.max(1, this.blockHeight - count + 1)
    
    for (let i = startHeight; i <= this.blockHeight; i++) {
      blocks.push({
        height: i,
        hash: '0x' + Array.from({ length: 64 }, () => 
          Math.floor(Math.random() * 16).toString(16)
        ).join(''),
        timestamp: Date.now() - (this.blockHeight - i) * 5000, // 每个块相隔5秒
        txCount: Math.floor(Math.random() * 200) + 10,
        size: Math.floor(Math.random() * 500000) + 50000,
        miner: 'peer-' + Math.floor(Math.random() * 10)
      })
    }
    
    return blocks
  }

  /**
   * 根据高度获取区块详情Mock数据
   */
  getBlockByHeight(height: number): BlockDetail {
    const txCount = Math.floor(Math.random() * 200) + 10
    const transactions = Array.from({ length: txCount }, () =>
      '0x' + Array.from({ length: 64 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join('')
    )

    const uncleCount = Math.floor(Math.random() * 3)
    const uncleHashes = Array.from({ length: uncleCount }, () =>
      '0x' + Array.from({ length: 64 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join('')
    )

    return {
      height,
      hash: '0x' + Array.from({ length: 64 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join(''),
      parentHash: '0x' + Array.from({ length: 64 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join(''),
      timestamp: Date.now() - (this.blockHeight - height) * 5000,
      txCount,
      size: Math.floor(Math.random() * 500000) + 50000,
      miner: 'peer-' + Math.floor(Math.random() * 10),
      difficulty: '0x' + Math.floor(Math.random() * 0xffffff).toString(16).padStart(6, '0'),
      nonce: Math.floor(Math.random() * 1000000),
      mixdigest: '0x' + Array.from({ length: 64 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join(''),
      uncleHashes,
      transactions,
      committeePubkey: '0x' + Array.from({ length: 128 }, () =>
        Math.floor(Math.random() * 16).toString(16)
      ).join('')
    }
  }
}

export const mockService = new MockDataService()

