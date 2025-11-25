/**
 * POT共识可视化系统 - API客户端
 * 
 * 提供与后端API交互的所有方法
 * 支持Mock模式和真实API模式切换
 */

import axios, { AxiosInstance } from 'axios'
import { mockService } from './mock'
import type {
  SystemOverview,
  POTStatus,
  VDFStatus,
  CommitteeStatus,
  BCIStatus,
  MempoolStatus,
  NetworkTopology
} from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'
const USE_MOCK = import.meta.env.VITE_USE_MOCK === 'true'

class ApiClient {
  private client: AxiosInstance
  private useMock: boolean

  constructor() {
    this.useMock = USE_MOCK
    
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      }
    })

    // 请求拦截器
    this.client.interceptors.request.use(
      config => {
        // 可以添加认证token等
        return config
      },
      error => Promise.reject(error)
    )

    // 响应拦截器
    this.client.interceptors.response.use(
      response => response.data,
      error => {
        console.error('API Error:', error)
        return Promise.reject(error)
      }
    )
  }
  
  /**
   * 通用的Mock或真实API调用方法
   */
  private async mockOrFetch<T>(mockFn: () => T, apiFn: () => Promise<any>): Promise<T> {
    if (this.useMock) {
      console.log('[Mock Mode] Using mock data')
      return Promise.resolve(mockFn())
    }
    return apiFn()
  }

  // ==================== 系统API ====================
  
  /**
   * 获取系统概览信息
   * @returns Promise<SystemOverview>
   */
  getSystemOverview(): Promise<SystemOverview> {
    return this.mockOrFetch(
      () => mockService.getSystemOverview(),
      () => this.client.get('/system/overview')
    )
  }

  // ==================== POT共识API ====================
  
  /**
   * 获取POT共识状态
   * @returns Promise<POTStatus>
   */
  getPotStatus(): Promise<POTStatus> {
    return this.mockOrFetch(
      () => mockService.getPotStatus(),
      () => this.client.get('/pot/status')
    )
  }

  /**
   * 获取VDF计算状态
   * @returns Promise<VDFStatus>
   */
  getVDFStatus(): Promise<VDFStatus> {
    return this.mockOrFetch(
      () => mockService.getVDFStatus(),
      () => this.client.get('/pot/vdf')
    )
  }

  // ==================== 委员会API ====================
  
  /**
   * 获取委员会状态
   * @returns Promise<CommitteeStatus>
   */
  getCommitteeStatus(): Promise<CommitteeStatus> {
    return this.mockOrFetch(
      () => mockService.getCommitteeStatus(),
      () => this.client.get('/committee/status')
    )
  }

  // ==================== BCI激励API ====================
  
  /**
   * 获取BCI激励状态
   * @returns Promise<BCIStatus>
   */
  getBCIStatus(): Promise<BCIStatus> {
    return this.mockOrFetch(
      () => mockService.getBCIStatus(),
      () => this.client.get('/bci/status')
    )
  }

  // ==================== 交易池API ====================
  
  /**
   * 获取交易池状态
   * @returns Promise<MempoolStatus>
   */
  getMempoolStatus(): Promise<MempoolStatus> {
    return this.mockOrFetch(
      () => mockService.getMempoolStatus(),
      () => this.client.get('/mempool/status')
    )
  }

  // ==================== 网络API ====================
  
  /**
   * 获取网络拓扑信息
   * @returns Promise<NetworkTopology>
   */
  getNetworkTopology(): Promise<NetworkTopology> {
    return this.mockOrFetch(
      () => mockService.getNetworkTopology(),
      () => this.client.get('/network/topology')
    )
  }
}

export const api = new ApiClient()

