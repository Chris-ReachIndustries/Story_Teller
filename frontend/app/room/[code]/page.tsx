'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useGame } from '@/lib/gameContext'
import Lobby from '@/components/Lobby'
import GameBoard from '@/components/GameBoard'

export default function RoomPage() {
  const params = useParams()
  const router = useRouter()
  const { state, connected, error, ws, clearError } = useGame()
  const roomCode = params.code as string
  const [reconnectAttempted, setReconnectAttempted] = useState(false)
  const [showReconnectToast, setShowReconnectToast] = useState(false)

  // Attempt auto-reconnect on mount using stored token
  useEffect(() => {
    if (!ws || !connected || reconnectAttempted) return

    // Only attempt if not already in a room
    if (state?.room) return

    const attempted = ws.attemptReconnect(roomCode)
    setReconnectAttempted(true)

    if (attempted) {
      // Listen for reconnection success
      const unsubscribe = ws.onReconnect((success) => {
        if (success) {
          setShowReconnectToast(true)
          setTimeout(() => setShowReconnectToast(false), 3000)
        }
        unsubscribe()
      })
    }
  }, [ws, connected, roomCode, reconnectAttempted, state?.room])

  // If not in a room after reconnect attempt, redirect to home
  useEffect(() => {
    if (reconnectAttempted && state && !state.room) {
      router.push('/')
    }
  }, [state, router, reconnectAttempted])

  if (!state || !state.room) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-gray-400">Loading room...</p>
        </div>
      </div>
    )
  }

  const isLobby = state.phase === 'LOBBY'

  return (
    <div className="min-h-screen p-4 md:p-8">
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-primary">Dixit</h1>
          <p className="text-gray-400">
            Room: <span className="font-mono text-white">{state.room.code}</span>
          </p>
        </div>
        <div className="text-right">
          {state.you && (
            <p className="text-gray-400">
              Playing as <span className="text-white font-semibold">{state.you.name}</span>
              {state.you.isHost && <span className="ml-2 text-accent">(Host)</span>}
            </p>
          )}
          <div className={`inline-flex items-center gap-2 mt-1 ${connected ? 'text-green-400' : 'text-red-400'}`}>
            <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-400' : 'bg-red-400'}`}></span>
            {connected ? 'Connected' : 'Disconnected'}
          </div>
        </div>
      </div>

      {/* Reconnect toast */}
      {showReconnectToast && (
        <div className="fixed top-4 right-4 bg-green-500/20 border border-green-500 rounded-lg p-4 animate-fade-in">
          <p className="text-green-400">Reconnected successfully!</p>
        </div>
      )}

      {/* Error display */}
      {error && (
        <div className="bg-red-500/20 border border-red-500 rounded-lg p-4 mb-6 flex items-center justify-between">
          <p className="text-red-400">{error}</p>
          <button onClick={clearError} className="text-red-400 hover:text-red-300">✕</button>
        </div>
      )}

      {/* Main content */}
      {isLobby ? <Lobby /> : <GameBoard />}
    </div>
  )
}
