'use client'

import { PlayerInfo } from '@/lib/types'

interface PlayerListProps {
  players: PlayerInfo[]
  hostId?: string
  storytellerId?: string
  showStatus?: boolean
  showScores?: boolean
}

export default function PlayerList({
  players,
  hostId,
  storytellerId,
  showStatus = false,
  showScores = false,
}: PlayerListProps) {
  return (
    <div className="bg-slate-800/50 rounded-xl p-4">
      <h3 className="text-lg font-semibold mb-3 text-gray-300">Players</h3>
      <ul className="space-y-2">
        {players.map((player) => (
          <li
            key={player.id}
            className={`flex items-center justify-between p-2 rounded-lg
              ${!player.connected ? 'opacity-50' : ''}
              ${player.id === storytellerId ? 'bg-primary/20 border border-primary/30' : 'bg-slate-700/50'}
            `}
          >
            <div className="flex items-center gap-2">
              <span
                className={`w-2 h-2 rounded-full ${
                  player.connected ? 'bg-green-400' : 'bg-red-400'
                }`}
              />
              <span className="text-white">{player.name}</span>
              {player.playerType === 'bot' && (
                <span className="text-xs bg-purple-500/20 text-purple-400 px-2 py-0.5 rounded">Bot</span>
              )}
              {player.id === hostId && (
                <span className="text-xs bg-accent/20 text-accent px-2 py-0.5 rounded">Host</span>
              )}
              {player.id === storytellerId && (
                <span className="text-xs bg-primary/20 text-primary px-2 py-0.5 rounded">Storyteller</span>
              )}
            </div>
            <div className="flex items-center gap-3">
              {showStatus && (
                <div className="flex items-center gap-2">
                  {player.hasSubmitted && (
                    <span className="text-green-400 text-sm">✓ Submitted</span>
                  )}
                  {player.hasVoted && (
                    <span className="text-blue-400 text-sm">✓ Voted</span>
                  )}
                </div>
              )}
              {showScores && (
                <span className="text-lg font-bold text-accent">{player.score}</span>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
