'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useGame } from '@/lib/gameContext'
import Header from '@/components/Header'
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
      <Header
        roomCode={state.room.code}
        playerName={state.you?.name}
        isHost={state.you?.isHost}
        connected={connected}
      />

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
