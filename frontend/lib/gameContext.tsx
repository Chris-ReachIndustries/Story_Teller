'use client'

import React, { createContext, useContext, useEffect, useState, useCallback, ReactNode } from 'react'
import { GameState, Card } from './types'
import { getWebSocket, GameWebSocket } from './websocket'

interface GameContextType {
  state: GameState | null
  connected: boolean
  error: string | null
  ws: GameWebSocket | null
  clearError: () => void
}

const GameContext = createContext<GameContextType | undefined>(undefined)

const initialState: GameState = {
  phase: 'LOBBY',
  players: [],
  round: 0,
}

export function GameProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<GameState | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [ws, setWs] = useState<GameWebSocket | null>(null)

  const clearError = useCallback(() => {
    setError(null)
  }, [])

  useEffect(() => {
    const websocket = getWebSocket()
    setWs(websocket)

    const unsubState = websocket.onState((newState) => {
      setState(newState)
      setError(null) // Clear error on successful state update
    })

    const unsubError = websocket.onError((errorMsg) => {
      setError(errorMsg)
    })

    const unsubConnection = websocket.onConnection((isConnected) => {
      setConnected(isConnected)
      if (!isConnected) {
        setError('Disconnected from server')
      }
    })

    websocket.connect()

    return () => {
      unsubState()
      unsubError()
      unsubConnection()
    }
  }, [])

  return (
    <GameContext.Provider value={{ state, connected, error, ws, clearError }}>
      {children}
    </GameContext.Provider>
  )
}

export function useGame() {
  const context = useContext(GameContext)
  if (context === undefined) {
    throw new Error('useGame must be used within a GameProvider')
  }
  return context
}
