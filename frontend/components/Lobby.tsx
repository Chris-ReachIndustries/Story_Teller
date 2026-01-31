'use client'

import { useState, useEffect } from 'react'
import { useGame } from '@/lib/gameContext'
import { CardSetInfo } from '@/lib/types'
import { fetchCardSets } from '@/lib/api'
import PlayerList from './PlayerList'

export default function Lobby() {
  const { state, ws } = useGame()
  const [editingConfig, setEditingConfig] = useState(false)
  const [localScoreToWin, setLocalScoreToWin] = useState(30)
  const [localHandSize, setLocalHandSize] = useState(6)
  const [localStartingStoryteller, setLocalStartingStoryteller] = useState('host')
  const [localDeckSetId, setLocalDeckSetId] = useState('default')
  const [cardSets, setCardSets] = useState<CardSetInfo[]>([])

  // Fetch available card sets
  useEffect(() => {
    fetchCardSets()
      .then(setCardSets)
      .catch((err) => console.error('Failed to fetch card sets:', err))
  }, [])

  // Sync local config with server config
  useEffect(() => {
    if (state?.room?.config) {
      setLocalScoreToWin(state.room.config.scoreToWin)
      setLocalHandSize(state.room.config.handSize)
      setLocalStartingStoryteller(state.room.config.startingStoryteller || 'host')
      setLocalDeckSetId(state.room.config.deckSetId || 'default')
    }
  }, [state?.room?.config])

  if (!state || !state.room) return null

  const config = state.room.config
  const connectedCount = state.players.filter((p) => p.connected).length
  const minPlayers = parseInt(process.env.NEXT_PUBLIC_MIN_PLAYERS || '4', 10)
  const maxPlayers = 6
  const canStart = connectedCount >= minPlayers && connectedCount <= maxPlayers

  const handleStartGame = () => {
    if (ws && canStart) {
      ws.startGame()
    }
  }

  const handleCopyCode = () => {
    if (state.room) {
      navigator.clipboard.writeText(state.room.code)
    }
  }

  const handleSaveConfig = () => {
    if (ws) {
      ws.updateConfig({
        scoreToWin: localScoreToWin,
        handSize: localHandSize,
        startingStoryteller: localStartingStoryteller,
        deckSetId: localDeckSetId,
      })
      setEditingConfig(false)
    }
  }

  const handleCancelConfig = () => {
    if (config) {
      setLocalScoreToWin(config.scoreToWin)
      setLocalHandSize(config.handSize)
      setLocalStartingStoryteller(config.startingStoryteller || 'host')
      setLocalDeckSetId(config.deckSetId || 'default')
    }
    setEditingConfig(false)
  }

  // Get display name for starting storyteller
  const getStorytellerDisplayName = () => {
    const setting = config.startingStoryteller || 'host'
    if (setting === 'host') return 'Host'
    if (setting === 'random') return 'Random'
    const player = state.players.find((p) => p.id === setting)
    return player ? player.name : 'Host'
  }

  // Get display name for deck set
  const getDeckDisplayName = () => {
    const deckId = config.deckSetId || 'default'
    const set = cardSets.find((s) => s.id === deckId)
    return set ? set.name : deckId.charAt(0).toUpperCase() + deckId.slice(1)
  }

  return (
    <div className="max-w-xl mx-auto">
      {/* Room Code Display */}
      <div className="bg-slate-800/50 rounded-xl p-6 mb-6 text-center">
        <p className="text-gray-400 mb-2">Share this code with friends:</p>
        <div className="flex items-center justify-center gap-4">
          <span className="text-5xl font-mono font-bold tracking-widest text-white">
            {state.room.code}
          </span>
          <button
            onClick={handleCopyCode}
            className="px-3 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-gray-300 transition-colors"
          >
            Copy
          </button>
        </div>
      </div>

      {/* Room Config Display */}
      <div className="bg-slate-800/50 rounded-xl p-4 mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4 text-sm flex-wrap">
            <span className="text-gray-400">
              Win: <span className="text-white font-semibold">{config.scoreToWin} pts</span>
            </span>
            <span className="text-gray-600">|</span>
            <span className="text-gray-400">
              Hand: <span className="text-white font-semibold">{config.handSize} cards</span>
            </span>
            <span className="text-gray-600">|</span>
            <span className="text-gray-400">
              First: <span className="text-white font-semibold">{getStorytellerDisplayName()}</span>
            </span>
            <span className="text-gray-600">|</span>
            <span className="text-gray-400">
              Deck: <span className="text-white font-semibold">{getDeckDisplayName()}</span>
            </span>
          </div>
          {state.you?.isHost && !editingConfig && (
            <button
              onClick={() => setEditingConfig(true)}
              className="text-sm text-gray-400 hover:text-white transition-colors"
            >
              Edit
            </button>
          )}
        </div>

        {editingConfig && state.you?.isHost && (
          <div className="mt-4 pt-4 border-t border-slate-700 space-y-3">
            <div>
              <label className="block text-sm text-gray-400 mb-1">
                Score to Win ({localScoreToWin} pts)
              </label>
              <input
                type="range"
                min="10"
                max="100"
                step="5"
                value={localScoreToWin}
                onChange={(e) => setLocalScoreToWin(Number(e.target.value))}
                className="w-full accent-primary"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">
                Hand Size ({localHandSize} cards)
              </label>
              <input
                type="range"
                min="4"
                max="10"
                value={localHandSize}
                onChange={(e) => setLocalHandSize(Number(e.target.value))}
                className="w-full accent-primary"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">
                First Storyteller
              </label>
              <select
                value={localStartingStoryteller}
                onChange={(e) => setLocalStartingStoryteller(e.target.value)}
                className="w-full bg-slate-700 text-white rounded-lg px-3 py-2 text-sm"
              >
                <option value="host">Host</option>
                <option value="random">Random</option>
                {state.players.map((player) => (
                  <option key={player.id} value={player.id}>
                    {player.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Card Set</label>
              <select
                value={localDeckSetId}
                onChange={(e) => setLocalDeckSetId(e.target.value)}
                className="w-full bg-slate-700 text-white rounded-lg px-3 py-2 text-sm"
              >
                {cardSets.map((set) => (
                  <option key={set.id} value={set.id}>
                    {set.name} ({set.cardCount} cards)
                  </option>
                ))}
                {cardSets.length === 0 && <option value="default">Default</option>}
              </select>
            </div>
            <div className="flex gap-2">
              <button
                onClick={handleSaveConfig}
                className="flex-1 px-3 py-2 bg-primary hover:bg-primary/80 rounded-lg text-white text-sm transition-colors"
              >
                Save
              </button>
              <button
                onClick={handleCancelConfig}
                className="flex-1 px-3 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-gray-300 text-sm transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Player List */}
      <PlayerList
        players={state.players}
        hostId={state.room.hostId}
      />

      {/* Add Bot Button (Host only, when room not full) */}
      {state.you?.isHost && state.players.length < maxPlayers && (
        <button
          onClick={() => ws?.addBot()}
          className="w-full mt-4 px-4 py-3 rounded-xl bg-purple-600 hover:bg-purple-500
                   text-white font-semibold transition-colors flex items-center justify-center gap-2"
        >
          <span>Add AI Bot</span>
        </button>
      )}

      {/* Start Game Section */}
      <div className="mt-6">
        {state.you?.isHost ? (
          <>
            <button
              onClick={handleStartGame}
              disabled={!canStart}
              className="w-full px-4 py-4 rounded-xl bg-primary hover:bg-primary/80
                       text-white font-bold text-lg disabled:opacity-50 disabled:cursor-not-allowed
                       transition-colors"
            >
              Start Game
            </button>
            {!canStart && (
              <p className="text-center text-gray-400 mt-2">
                Need {minPlayers}-{maxPlayers} players to start
                {connectedCount < minPlayers && ` (${minPlayers - connectedCount} more needed)`}
              </p>
            )}
          </>
        ) : (
          <div className="text-center p-4 bg-slate-800/30 rounded-xl">
            <p className="text-gray-400">Waiting for host to start the game...</p>
          </div>
        )}
      </div>

      {/* Game Rules Summary */}
      <div className="mt-8 bg-slate-800/30 rounded-xl p-6">
        <h3 className="font-semibold text-gray-300 mb-3">How to Play</h3>
        <ol className="list-decimal list-inside text-gray-400 space-y-2 text-sm">
          <li>The Storyteller gives a clue and secretly picks a card</li>
          <li>Other players choose cards that match the clue</li>
          <li>All cards are shuffled and revealed</li>
          <li>Players vote for which card they think is the Storyteller&apos;s</li>
          <li>Score points for fooling others or guessing correctly!</li>
        </ol>
      </div>
    </div>
  )
}
