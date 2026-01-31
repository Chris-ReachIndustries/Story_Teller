'use client'

import { PlayerInfo, RoundResult } from '@/lib/types'

interface ScoreboardProps {
  players: PlayerInfo[]
  roundResults?: RoundResult
  isGameEnd?: boolean
}

export default function Scoreboard({
  players,
  roundResults,
  isGameEnd = false,
}: ScoreboardProps) {
  // Sort players by score (descending)
  const sortedPlayers = [...players].sort((a, b) => b.score - a.score)

  return (
    <div className="bg-slate-800/50 rounded-xl p-6">
      <h3 className="text-xl font-bold mb-4 text-center text-white">
        {isGameEnd ? '🏆 Final Scores' : 'Scoreboard'}
      </h3>
      <div className="space-y-3">
        {sortedPlayers.map((player, index) => {
          const roundPoints = roundResults?.pointsThisRound?.[player.id] || 0

          return (
            <div
              key={player.id}
              className={`flex items-center justify-between p-3 rounded-lg
                ${index === 0 && isGameEnd ? 'bg-yellow-500/20 border border-yellow-500/30' : 'bg-slate-700/50'}
              `}
            >
              <div className="flex items-center gap-3">
                <span className="text-2xl font-bold text-gray-400 w-8">
                  {index === 0 && isGameEnd ? '🥇' : index === 1 && isGameEnd ? '🥈' : index === 2 && isGameEnd ? '🥉' : `#${index + 1}`}
                </span>
                <span className="text-white font-semibold">{player.name}</span>
              </div>
              <div className="flex items-center gap-4">
                {roundResults && roundPoints !== 0 && (
                  <span className={`text-sm font-bold ${roundPoints > 0 ? 'text-green-400' : 'text-red-400'}`}>
                    {roundPoints > 0 ? '+' : ''}{roundPoints}
                  </span>
                )}
                <span className="text-2xl font-bold text-accent">{player.score}</span>
              </div>
            </div>
          )
        })}
      </div>

      {roundResults && (
        <div className="mt-6 pt-4 border-t border-slate-700">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-400">
              Storyteller&apos;s card was <span className="text-white font-medium">#{roundResults.storytellerCard + 1}</span>
            </span>
            <span className="text-gray-500">
              {Object.values(roundResults.votes).filter(v => v === roundResults.storytellerCard).length} / {Object.keys(roundResults.votes).length} correct
            </span>
          </div>
        </div>
      )}
    </div>
  )
}
