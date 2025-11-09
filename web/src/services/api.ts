import axios, { AxiosInstance } from 'axios'
import { mockService } from './mock'

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
        // 可以添加 token 等
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
  
  private async mockOrFetch<T>(mockFn: () => T, apiFn: () => Promise<any>): Promise<T> {
    if (this.useMock) {
      console.log('[Mock Mode] Using mock data')
      return Promise.resolve(mockFn())
    }
    return apiFn()
  }

  // 系统 API
  getSystemOverview() {
    return this.mockOrFetch(
      () => mockService.getSystemOverview(),
      () => this.client.get('/system/overview')
    )
  }

  // POT API
  getPotStatus() {
    return this.mockOrFetch(
      () => mockService.getPotStatus(),
      () => this.client.get('/pot/status')
    )
  }

  getVDFStatus() {
    return this.mockOrFetch(
      () => mockService.getVDFStatus(),
      () => this.client.get('/pot/vdf')
    )
  }

  // 委员会 API
  getCommitteeStatus() {
    return this.mockOrFetch(
      () => mockService.getCommitteeStatus(),
      () => this.client.get('/committee/status')
    )
  }

  // BCI API
  getBCIStatus() {
    return this.mockOrFetch(
      () => mockService.getBCIStatus(),
      () => this.client.get('/bci/status')
    )
  }

  getBalance(address: string) {
    return this.client.get(`/bci/balance/${address}`)
  }

  // 交易池 API
  getMempoolStatus() {
    return this.mockOrFetch(
      () => mockService.getMempoolStatus(),
      () => this.client.get('/mempool/status')
    )
  }

  // 网络 API
  getNetworkTopology() {
    return this.mockOrFetch(
      () => mockService.getNetworkTopology(),
      () => this.client.get('/network/topology')
    )
  }

  // 存储 API
  getStorageStatus() {
    return this.mockOrFetch(
      () => mockService.getStorageStatus(),
      () => this.client.get('/storage/status')
    )
  }

  // 历史数据 API
  getMetricsHistory(metric: string, timeRange: string, interval: string) {
    return this.client.get('/metrics/history', {
      params: { metric, timeRange, interval }
    })
  }
}

export const api = new ApiClient()
