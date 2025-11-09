import { ref } from 'vue'
import type { WSMessage } from '@/types/api'

export class WebSocketService {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectInterval = 3000
  private listeners: Map<string, Set<(data: any) => void>> = new Map()
  
  public connected = ref(false)
  public error = ref<string | null>(null)
  
  constructor(private url: string) {}
  
  connect() {
    try {
      this.ws = new WebSocket(this.url)
      
      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.connected.value = true
        this.error.value = null
        this.reconnectAttempts = 0
      }
      
      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data)
          this.notifyListeners(message.type, message.data)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }
      
      this.ws.onerror = (event) => {
        console.error('WebSocket error:', event)
        this.error.value = 'WebSocket connection error'
      }
      
      this.ws.onclose = () => {
        console.log('WebSocket disconnected')
        this.connected.value = false
        this.reconnect()
      }
    } catch (err) {
      console.error('Failed to create WebSocket:', err)
      this.error.value = 'Failed to create WebSocket connection'
    }
  }
  
  private reconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`)
      
      setTimeout(() => {
        this.connect()
      }, this.reconnectInterval)
    } else {
      console.error('Max reconnection attempts reached')
      this.error.value = 'Failed to reconnect to WebSocket'
    }
  }
  
  subscribe(types: string[]) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        action: 'subscribe',
        types
      }))
    }
  }
  
  on(type: string, callback: (data: any) => void) {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set())
    }
    this.listeners.get(type)!.add(callback)
  }
  
  off(type: string, callback: (data: any) => void) {
    const listeners = this.listeners.get(type)
    if (listeners) {
      listeners.delete(callback)
    }
  }
  
  private notifyListeners(type: string, data: any) {
    const listeners = this.listeners.get(type)
    if (listeners) {
      listeners.forEach(callback => callback(data))
    }
  }
  
  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }
}
