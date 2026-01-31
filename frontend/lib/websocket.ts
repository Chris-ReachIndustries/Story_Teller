import {
  ClientMessage,
  ServerMessage,
  GameState,
  ErrorPayload,
  RoomConfig,
} from './types'

type StateHandler = (state: GameState) => void
type ErrorHandler = (error: string) => void
type ConnectionHandler = (connected: boolean) => void
type ReconnectHandler = (success: boolean) => void

const STORAGE_KEY_PREFIX = 'dixit:'

export class GameWebSocket {
  private ws: WebSocket | null = null
  private url: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private pingInterval: NodeJS.Timeout | null = null

  private stateHandlers: StateHandler[] = []
  private errorHandlers: ErrorHandler[] = []
  private connectionHandlers: ConnectionHandler[] = []
  private reconnectHandlers: ReconnectHandler[] = []

  // Reconnection data
  private roomCode: string | null = null
  private playerId: string | null = null
  private pendingReconnect: {
    roomCode: string
    token: string
    name: string
  } | null = null

  constructor(url?: string) {
    // Use relative WebSocket URL in production (same host)
    if (typeof window !== 'undefined') {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      this.url = url || `${protocol}//${window.location.host}/ws`
    } else {
      this.url = url || 'ws://localhost:8080/ws'
    }
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.reconnectAttempts = 0
        this.notifyConnectionHandlers(true)
        this.startPingInterval()

        // Attempt token-based reconnection if pending
        if (this.pendingReconnect) {
          const { roomCode, token, name } = this.pendingReconnect
          this.send({
            type: 'hello',
            payload: { name, reconnectToken: token, roomCode },
          })
          this.pendingReconnect = null
        }
        // Legacy reconnection via player ID
        else if (this.roomCode && this.playerId) {
          this.send({
            type: 'reconnect',
            payload: { roomCode: this.roomCode, playerId: this.playerId },
          })
        }
      }

      this.ws.onmessage = (event) => {
        try {
          const message: ServerMessage = JSON.parse(event.data)
          this.handleMessage(message)
        } catch (err) {
          console.error('Failed to parse message:', err)
        }
      }

      this.ws.onclose = () => {
        console.log('WebSocket disconnected')
        this.stopPingInterval()
        this.notifyConnectionHandlers(false)
        this.attemptWsReconnect()
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    } catch (err) {
      console.error('Failed to connect:', err)
      this.attemptWsReconnect()
    }
  }

  disconnect(): void {
    this.stopPingInterval()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  private handleMessage(message: ServerMessage): void {
    switch (message.type) {
      case 'state':
        const state = message.payload as GameState
        // Store reconnection data
        if (state.room) {
          this.roomCode = state.room.code
        }
        if (state.you) {
          this.playerId = state.you.id
          // Save reconnect token to localStorage if provided
          if (state.you.reconnectToken && state.room) {
            this.saveReconnectData(
              state.room.code,
              state.you.reconnectToken,
              state.you.name
            )
            // Notify reconnect handlers of successful reconnection
            this.notifyReconnectHandlers(true)
          }
        }
        this.notifyStateHandlers(state)
        break
      case 'error':
        const errorPayload = message.payload as ErrorPayload
        this.notifyErrorHandlers(errorPayload.message)
        break
      case 'pong':
        // Pong received, connection is alive
        break
    }
  }

  private attemptWsReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('Max reconnection attempts reached')
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
    console.log(`Attempting reconnection in ${delay}ms (attempt ${this.reconnectAttempts})`)

    setTimeout(() => {
      this.connect()
    }, delay)
  }

  private startPingInterval(): void {
    this.pingInterval = setInterval(() => {
      this.send({ type: 'ping' })
    }, 30000) // Ping every 30 seconds
  }

  private stopPingInterval(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }

  send(message: ClientMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket not connected, cannot send message')
    }
  }

  // Handler management
  onState(handler: StateHandler): () => void {
    this.stateHandlers.push(handler)
    return () => {
      this.stateHandlers = this.stateHandlers.filter((h) => h !== handler)
    }
  }

  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.push(handler)
    return () => {
      this.errorHandlers = this.errorHandlers.filter((h) => h !== handler)
    }
  }

  onConnection(handler: ConnectionHandler): () => void {
    this.connectionHandlers.push(handler)
    return () => {
      this.connectionHandlers = this.connectionHandlers.filter((h) => h !== handler)
    }
  }

  private notifyStateHandlers(state: GameState): void {
    this.stateHandlers.forEach((h) => h(state))
  }

  private notifyErrorHandlers(error: string): void {
    this.errorHandlers.forEach((h) => h(error))
  }

  private notifyConnectionHandlers(connected: boolean): void {
    this.connectionHandlers.forEach((h) => h(connected))
  }

  private notifyReconnectHandlers(success: boolean): void {
    this.reconnectHandlers.forEach((h) => h(success))
  }

  public onReconnect(handler: ReconnectHandler): () => void {
    this.reconnectHandlers.push(handler)
    return () => {
      this.reconnectHandlers = this.reconnectHandlers.filter((h) => h !== handler)
    }
  }

  // LocalStorage helpers for reconnect data
  private saveReconnectData(
    roomCode: string,
    token: string,
    name: string
  ): void {
    if (typeof window === 'undefined') return
    try {
      localStorage.setItem(`${STORAGE_KEY_PREFIX}${roomCode}:token`, token)
      localStorage.setItem(`${STORAGE_KEY_PREFIX}${roomCode}:name`, name)
    } catch {
      // localStorage might be unavailable
    }
  }

  public getReconnectData(
    roomCode: string
  ): { token: string; name: string } | null {
    if (typeof window === 'undefined') return null
    try {
      const token = localStorage.getItem(`${STORAGE_KEY_PREFIX}${roomCode}:token`)
      const name = localStorage.getItem(`${STORAGE_KEY_PREFIX}${roomCode}:name`)
      if (token && name) {
        return { token, name }
      }
    } catch {
      // localStorage might be unavailable
    }
    return null
  }

  public clearReconnectData(roomCode: string): void {
    if (typeof window === 'undefined') return
    try {
      localStorage.removeItem(`${STORAGE_KEY_PREFIX}${roomCode}:token`)
      localStorage.removeItem(`${STORAGE_KEY_PREFIX}${roomCode}:name`)
    } catch {
      // localStorage might be unavailable
    }
  }

  // Game actions
  hello(
    name: string,
    reconnectToken?: string,
    roomCode?: string
  ): void {
    const payload: {
      name: string
      reconnectToken?: string
      roomCode?: string
    } = { name }
    if (reconnectToken) payload.reconnectToken = reconnectToken
    if (roomCode) payload.roomCode = roomCode
    this.send({ type: 'hello', payload })
  }

  // Attempt reconnection using stored data
  public attemptReconnect(roomCode: string): boolean {
    const data = this.getReconnectData(roomCode)
    if (data) {
      this.pendingReconnect = { roomCode, token: data.token, name: data.name }
      // If already connected, send immediately
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.send({
          type: 'hello',
          payload: {
            name: data.name,
            reconnectToken: data.token,
            roomCode,
          },
        })
        this.pendingReconnect = null
      }
      return true
    }
    return false
  }

  createRoom(config?: Partial<RoomConfig>): void {
    const payload: { config?: Partial<RoomConfig> } = {}
    if (config) {
      payload.config = config
    }
    this.send({ type: 'create_room', payload })
  }

  joinRoom(code: string): void {
    this.send({ type: 'join_room', payload: { code } })
  }

  public updateConfig(config: RoomConfig): void {
    this.send({ type: 'update_config', payload: { config } })
  }

  startGame(): void {
    this.send({ type: 'start_game', payload: {} })
  }

  submitStory(clue: string, cardId: string): void {
    this.send({ type: 'story_submit', payload: { clue, cardId } })
  }

  submitCard(cardId: string): void {
    this.send({ type: 'submit_card', payload: { cardId } })
  }

  vote(submissionIndex: number): void {
    this.send({ type: 'vote', payload: { submissionIndex } })
  }

  nextRound(): void {
    this.send({ type: 'next_round', payload: {} })
  }

  returnToLobby(): void {
    this.send({ type: 'return_to_lobby', payload: {} })
  }

  addBot(): void {
    this.send({ type: 'add_bot', payload: {} })
  }
}

// Singleton instance
let wsInstance: GameWebSocket | null = null

export function getWebSocket(): GameWebSocket {
  if (!wsInstance) {
    wsInstance = new GameWebSocket()
  }
  return wsInstance
}
