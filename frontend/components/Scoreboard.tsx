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
          <h4 className="text-lg font-semibold mb-3 text-gray-300">Round Details</h4>
          <div className="text-sm text-gray-400">
            <p>Storyteller&apos;s card was #{roundResults.storytellerCard + 1}</p>
            <div className="mt-2">
              <p className="font-semibold text-gray-300">Votes:</p>
              {Object.entries(roundResults.votes).map(([playerId, votedIndex]) => {
                const player = players.find((p) => p.id === playerId)
                return (
                  <p key={playerId} className="ml-2">
                    {player?.name || 'Unknown'} voted for Card #{(votedIndex as number) + 1}
                    {votedIndex === roundResults.storytellerCard && (
                      <span className="text-green-400 ml-2">✓ Correct!</span>
                    )}
                  </p>
                )
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
