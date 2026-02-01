'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Image from 'next/image'
import { useGame } from '@/lib/gameContext'
import { CardSetInfo } from '@/lib/types'
import { fetchCardSets } from '@/lib/api'
import GameRules from '@/components/GameRules'
import CardSetBrowser from '@/components/CardSetBrowser'

export default function Home() {
  const router = useRouter()
  const { state, connected, error, ws, clearError } = useGame()
  const [name, setName] = useState('')
  const [roomCode, setRoomCode] = useState('')
  const [step, setStep] = useState<'name' | 'choice'>('name')
  const [isJoining, setIsJoining] = useState(false)
  const [scoreToWin, setScoreToWin] = useState(30)
  const [handSize, setHandSize] = useState(6)
  const [showAdvanced, setShowAdvanced] = useState(false)
  const [deckSetId, setDeckSetId] = useState('default')
  const [cardSets, setCardSets] = useState<CardSetInfo[]>([])

  // Fetch available card sets on mount
  useEffect(() => {
    fetchCardSets()
      .then(setCardSets)
      .catch((err) => console.error('Failed to fetch card sets:', err))
  }, [])

  // Redirect to room when state includes room info
  useEffect(() => {
    if (state?.room?.code) {
      router.push(`/room/${state.room.code}`)
    }
  }, [state?.room?.code, router])

  const handleSetName = (e: React.FormEvent) => {
    e.preventDefault()
    if (name.trim() && ws) {
      ws.hello(name.trim())
      setStep('choice')
    }
  }

  const handleCreateRoom = () => {
    if (ws) {
      ws.createRoom({ scoreToWin, handSize, deckSetId })
    }
  }

  const handleJoinRoom = (e: React.FormEvent) => {
    e.preventDefault()
    if (roomCode.trim() && ws) {
      ws.joinRoom(roomCode.trim().toUpperCase())
    }
  }

  return (
    <div className="flex flex-col items-center min-h-screen p-8">
      <div className="flex flex-col items-center justify-center flex-1 w-full animate-fade-in">
        <Image
          src="/logo.svg"
          alt="Story Teller"
          width={280}
          height={48}
          className="mb-2"
          priority
        />
        <p className="text-gray-400 mb-8">A game of creative storytelling</p>

      {!connected && (
        <div className="bg-red-500/20 border border-red-500 rounded-lg p-4 mb-6">
          <p className="text-red-400">Connecting to server...</p>
        </div>
      )}

      {error && (
        <div className="bg-red-500/20 border border-red-500 rounded-lg p-4 mb-6 flex items-center gap-4">
          <p className="text-red-400">{error}</p>
          <button
            onClick={clearError}
            className="text-red-400 hover:text-red-300"
          >
            ✕
          </button>
        </div>
      )}

      {step === 'name' ? (
        <form onSubmit={handleSetName} className="w-full max-w-sm">
          <div className="bg-slate-800/50 rounded-xl p-6 shadow-xl">
            <label className="block text-gray-300 mb-2">Your Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Enter your name"
              maxLength={20}
              className="w-full px-4 py-3 rounded-lg bg-slate-700 border border-slate-600
                       text-white placeholder-gray-400 focus:outline-none focus:border-primary"
              autoFocus
            />
            <button
              type="submit"
              disabled={!name.trim() || !connected}
              className="w-full mt-4 px-4 py-3 rounded-lg bg-primary hover:bg-primary/80
                       text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed
                       transition-colors"
            >
              Continue
            </button>
          </div>
        </form>
      ) : (
        <div className="w-full max-w-sm space-y-4">
          <div className="bg-slate-800/50 rounded-xl p-6 shadow-xl">
            <p className="text-gray-400 mb-4">
              Welcome, <span className="text-white font-semibold">{name}</span>!
            </p>

            <button
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="text-sm text-gray-400 hover:text-white mb-3 flex items-center gap-1"
            >
              {showAdvanced ? '▼' : '▶'} Room Settings
            </button>

            {showAdvanced && (
              <div className="space-y-3 mb-4 p-3 bg-slate-700/50 rounded-lg">
                <div>
                  <label className="block text-sm text-gray-400 mb-1">
                    Score to Win ({scoreToWin} pts)
                  </label>
                  <input
                    type="range"
                    min="10"
                    max="100"
                    step="5"
                    value={scoreToWin}
                    onChange={(e) => setScoreToWin(Number(e.target.value))}
                    className="w-full accent-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1">
                    Hand Size ({handSize} cards)
                  </label>
                  <input
                    type="range"
                    min="4"
                    max="10"
                    value={handSize}
                    onChange={(e) => setHandSize(Number(e.target.value))}
                    className="w-full accent-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Card Set</label>
                  <select
                    value={deckSetId}
                    onChange={(e) => setDeckSetId(e.target.value)}
                    className="w-full px-3 py-2 rounded bg-slate-600 border border-slate-500
                             text-white focus:outline-none focus:border-primary"
                  >
                    {cardSets.map((set) => (
                      <option key={set.id} value={set.id}>
                        {set.name} ({set.cardCount} cards)
                      </option>
                    ))}
                    {cardSets.length === 0 && (
                      <option value="default">Default</option>
                    )}
                  </select>
                </div>
              </div>
            )}

            <button
              onClick={handleCreateRoom}
              disabled={!connected}
              className="w-full px-4 py-3 rounded-lg bg-primary hover:bg-primary/80
                       text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed
                       transition-colors"
            >
              Create New Room
            </button>
          </div>

          <div className="bg-slate-800/50 rounded-xl p-6 shadow-xl">
            <p className="text-gray-400 mb-4">Or join an existing room:</p>
            <form onSubmit={handleJoinRoom}>
              <input
                type="text"
                value={roomCode}
                onChange={(e) => setRoomCode(e.target.value.toUpperCase())}
                placeholder="Enter room code"
                maxLength={4}
                className="w-full px-4 py-3 rounded-lg bg-slate-700 border border-slate-600
                         text-white placeholder-gray-400 focus:outline-none focus:border-primary
                         uppercase tracking-widest text-center text-2xl"
              />
              <button
                type="submit"
                disabled={roomCode.length !== 4 || !connected}
                className="w-full mt-4 px-4 py-3 rounded-lg bg-secondary hover:bg-secondary/80
                         text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed
                         transition-colors"
              >
                Join Room
              </button>
            </form>
          </div>

          <button
            onClick={() => setStep('name')}
            className="w-full text-gray-400 hover:text-white transition-colors"
          >
            ← Change name
          </button>
        </div>
      )}
      </div>

      {/* Rules and Card Browser sections */}
      <div className="w-full max-w-4xl mt-12 space-y-6">
        <GameRules />
        <CardSetBrowser browseOnly />
      </div>
    </div>
  )
}
