'use client'

import { useState } from 'react'

interface GameRulesProps {
  defaultExpanded?: boolean
}

export default function GameRules({ defaultExpanded = false }: GameRulesProps) {
  const [expanded, setExpanded] = useState(defaultExpanded)

  return (
    <div className="bg-slate-800/50 rounded-xl overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full px-6 py-4 flex items-center justify-between text-left hover:bg-slate-700/30 transition-colors"
      >
        <h2 className="text-xl font-bold text-white">How to Play Dixit</h2>
        <span className="text-gray-400 text-2xl">{expanded ? '−' : '+'}</span>
      </button>

      {expanded && (
        <div className="px-6 pb-6 space-y-6">
          <section>
            <h3 className="text-lg font-semibold text-primary mb-2">What is Dixit?</h3>
            <p className="text-gray-300">
              Dixit is a creative storytelling game where players use beautifully illustrated cards
              to give clues and guess which card belongs to the storyteller. The key is to be just
              vague enough—not too obvious, not too obscure!
            </p>
          </section>

          <section>
            <h3 className="text-lg font-semibold text-primary mb-2">Game Setup</h3>
            <ul className="text-gray-300 space-y-1 list-disc list-inside">
              <li>4-6 players</li>
              <li>Each player starts with 6 cards (configurable)</li>
              <li>First to reach 30 points wins (configurable)</li>
            </ul>
          </section>

          <section>
            <h3 className="text-lg font-semibold text-primary mb-2">How a Round Works</h3>
            <ol className="text-gray-300 space-y-2 list-decimal list-inside">
              <li>
                <span className="font-semibold text-accent">Storyteller&apos;s Clue:</span> The
                storyteller picks a card from their hand and gives a clue (word, phrase, or
                sentence) that describes it.
              </li>
              <li>
                <span className="font-semibold text-accent">Card Submission:</span> Other players
                choose a card from their hand that best matches the clue.
              </li>
              <li>
                <span className="font-semibold text-accent">Voting:</span> All cards are shuffled
                and revealed. Players vote for the card they think is the storyteller&apos;s
                (you can&apos;t vote for your own card).
              </li>
              <li>
                <span className="font-semibold text-accent">Scoring:</span> Points are awarded
                based on votes (see scoring rules below).
              </li>
              <li>
                <span className="font-semibold text-accent">Next Round:</span> Everyone draws a new
                card, and the next player becomes the storyteller.
              </li>
            </ol>
          </section>

          <section>
            <h3 className="text-lg font-semibold text-primary mb-2">Scoring Rules</h3>
            <div className="bg-slate-900/50 rounded-lg p-4 space-y-3">
              <div>
                <p className="text-amber-400 font-semibold">
                  If ALL or NONE of the players guess correctly:
                </p>
                <ul className="text-gray-300 ml-4 list-disc list-inside">
                  <li>Storyteller: 0 points</li>
                  <li>Everyone else: 2 points</li>
                </ul>
                <p className="text-gray-500 text-sm mt-1">
                  (The clue was too easy or too hard!)
                </p>
              </div>

              <div>
                <p className="text-green-400 font-semibold">If SOME players guess correctly:</p>
                <ul className="text-gray-300 ml-4 list-disc list-inside">
                  <li>Storyteller: 3 points</li>
                  <li>Players who guessed correctly: 3 points each</li>
                </ul>
              </div>

              <div>
                <p className="text-blue-400 font-semibold">Bonus points:</p>
                <ul className="text-gray-300 ml-4 list-disc list-inside">
                  <li>+1 point for each vote your decoy card receives</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h3 className="text-lg font-semibold text-primary mb-2">Strategy Tips</h3>
            <ul className="text-gray-300 space-y-1 list-disc list-inside">
              <li>As storyteller, aim for a clue that 1-2 players will get, not everyone</li>
              <li>Pay attention to what cards others play—their style reveals their thinking</li>
              <li>Choose decoy cards that could plausibly match the clue</li>
              <li>Watch for patterns in how other players interpret clues</li>
            </ul>
          </section>
        </div>
      )}
    </div>
  )
}
