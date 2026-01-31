'use client'

import { useState, useEffect } from 'react'
import { useGame } from '@/lib/gameContext'
import Hand from './Hand'
import PlayerList from './PlayerList'
import VotingArea from './VotingArea'
import Scoreboard from './Scoreboard'
import Card from './Card'

export default function GameBoard() {
  const { state, ws } = useGame()
  const [selectedCard, setSelectedCard] = useState<string | null>(null)
  const [clue, setClue] = useState('')
  const [selectedVote, setSelectedVote] = useState<number | null>(null)

  // Reset selections when phase changes
  useEffect(() => {
    setSelectedCard(null)
    setClue('')
    setSelectedVote(null)
  }, [state?.phase])

  if (!state || !state.room || !state.you) return null

  const { phase, you, players, hand, submissions, clue: currentClue, storytellerId, round, roundResults, yourSubmissionIndex, gameEndReason, winnerName } = state

  const handleStorySubmit = () => {
    if (ws && selectedCard && clue.trim()) {
      ws.submitStory(clue.trim(), selectedCard)
    }
  }

  const handleCardSubmit = () => {
    if (ws && selectedCard) {
      ws.submitCard(selectedCard)
    }
  }

  const handleVote = () => {
    if (ws && selectedVote !== null) {
      ws.vote(selectedVote)
    }
  }

  const handleNextRound = () => {
    if (ws) {
      ws.nextRound()
    }
  }

  const handleReturnToLobby = () => {
    if (ws) {
      ws.returnToLobby()
    }
  }

  // Get phase-specific content
  const renderPhaseContent = () => {
    switch (phase) {
      case 'STORYTELLER_CLUE':
        return (
          <div className="space-y-6">
            <div className="bg-slate-800/50 rounded-xl p-6 text-center">
              {you.isStoryteller ? (
                <>
                  <h2 className="text-2xl font-bold text-primary mb-4">You are the Storyteller!</h2>
                  <p className="text-gray-400 mb-6">Choose a card and give a clue</p>
                  <input
                    type="text"
                    value={clue}
                    onChange={(e) => setClue(e.target.value)}
                    placeholder="Enter your clue..."
                    className="w-full max-w-md px-4 py-3 rounded-lg bg-slate-700 border border-slate-600
                             text-white placeholder-gray-400 focus:outline-none focus:border-primary mb-4"
                  />
                  <button
                    onClick={handleStorySubmit}
                    disabled={!selectedCard || !clue.trim()}
                    className="px-6 py-3 rounded-lg bg-primary hover:bg-primary/80
                             text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Submit Clue & Card
                  </button>
                </>
              ) : (
                <>
                  <h2 className="text-2xl font-bold text-gray-300 mb-4">Waiting for Storyteller</h2>
                  <p className="text-gray-400">
                    {players.find((p) => p.id === storytellerId)?.name} is choosing a card and clue...
                  </p>
                </>
              )}
            </div>
            {you.isStoryteller && hand && (
              <Hand
                cards={hand}
                selectedCardId={selectedCard}
                onSelectCard={setSelectedCard}
              />
            )}
          </div>
        )

      case 'SUBMISSIONS':
        return (
          <div className="space-y-6">
            <div className="bg-slate-800/50 rounded-xl p-6 text-center">
              <h2 className="text-2xl font-bold text-primary mb-2">The Clue</h2>
              <p className="text-3xl font-serif italic text-white">&ldquo;{currentClue}&rdquo;</p>
              <p className="text-gray-400 mt-4">
                {you.isStoryteller
                  ? 'Waiting for other players to choose cards...'
                  : state.hasSubmitted
                  ? 'You have submitted your card. Waiting for others...'
                  : 'Choose a card that matches the clue'}
              </p>
              {!you.isStoryteller && !state.hasSubmitted && (
                <button
                  onClick={handleCardSubmit}
                  disabled={!selectedCard}
                  className="mt-4 px-6 py-3 rounded-lg bg-primary hover:bg-primary/80
                           text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Submit Card
                </button>
              )}
            </div>
            {!you.isStoryteller && !state.hasSubmitted && hand && (
              <Hand
                cards={hand}
                selectedCardId={selectedCard}
                onSelectCard={setSelectedCard}
              />
            )}
          </div>
        )

      case 'VOTING':
        return (
          <div className="space-y-6">
            <div className="bg-slate-800/50 rounded-xl p-6 text-center">
              <h2 className="text-2xl font-bold text-primary mb-2">The Clue</h2>
              <p className="text-3xl font-serif italic text-white">&ldquo;{currentClue}&rdquo;</p>
              <p className="text-gray-400 mt-4">
                {you.isStoryteller
                  ? 'You cannot vote. Waiting for others...'
                  : state.hasVoted
                  ? 'You have voted. Waiting for others...'
                  : 'Vote for the card you think is the Storyteller\'s'}
              </p>
              {!you.isStoryteller && !state.hasVoted && (
                <button
                  onClick={handleVote}
                  disabled={selectedVote === null}
                  className="mt-4 px-6 py-3 rounded-lg bg-primary hover:bg-primary/80
                           text-white font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Submit Vote
                </button>
              )}
            </div>
            {submissions && (
              <VotingArea
                submissions={submissions}
                selectedIndex={selectedVote}
                onSelectSubmission={!you.isStoryteller && !state.hasVoted ? setSelectedVote : undefined}
                disabled={you.isStoryteller || state.hasVoted}
                ownSubmissionIndex={yourSubmissionIndex}
              />
            )}
          </div>
        )

      case 'SCORING':
      case 'ROUND_END':
        return (
          <div className="space-y-6">
            <div className="bg-slate-800/50 rounded-xl p-6 text-center">
              <h2 className="text-2xl font-bold text-primary mb-2">Round {round} Complete!</h2>
              <p className="text-3xl font-serif italic text-white mb-4">&ldquo;{currentClue}&rdquo;</p>
            </div>
            {submissions && roundResults && (
              <VotingArea
                submissions={submissions}
                disabled={true}
                revealedStorytellerIndex={roundResults.storytellerCard}
                roundResults={roundResults}
                players={players}
              />
            )}
            <Scoreboard players={players} roundResults={roundResults} />
            {you.isHost && (
              <button
                onClick={handleNextRound}
                className="w-full px-6 py-4 rounded-xl bg-primary hover:bg-primary/80
                         text-white font-bold text-lg transition-colors"
              >
                Next Round
              </button>
            )}
          </div>
        )

      case 'GAME_END':
        return (
          <div className="space-y-6">
            <div className="bg-slate-800/50 rounded-xl p-6 text-center">
              <h2 className="text-3xl font-bold text-accent mb-2">Game Over!</h2>
              {winnerName && (
                <p className="text-2xl text-white font-semibold mb-2">
                  {winnerName} wins!
                </p>
              )}
              <p className="text-gray-400">
                {gameEndReason === 'score_reached'
                  ? `Reached ${state.room?.config.scoreToWin} points!`
                  : gameEndReason === 'out_of_cards'
                  ? 'No more cards in the deck!'
                  : gameEndReason === 'player_disconnected'
                  ? 'A player disconnected from the game.'
                  : 'Thanks for playing!'}
              </p>
            </div>
            <Scoreboard players={players} isGameEnd={true} />
            {you.isHost && (
              <button
                onClick={handleReturnToLobby}
                className="w-full px-6 py-4 rounded-xl bg-primary hover:bg-primary/80
                         text-white font-bold text-lg transition-colors"
              >
                Return to Lobby
              </button>
            )}
          </div>
        )

      default:
        return null
    }
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
      {/* Main game area */}
      <div className="lg:col-span-3">
        {/* Phase indicator */}
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-gray-400">Round</span>
            <span className="text-2xl font-bold text-white">{round}</span>
          </div>
          <div className="px-4 py-2 bg-primary/20 rounded-lg">
            <span className="text-primary font-semibold">
              {phase.replace(/_/g, ' ')}
            </span>
          </div>
        </div>

        {renderPhaseContent()}
      </div>

      {/* Sidebar */}
      <div className="space-y-6">
        <PlayerList
          players={players}
          hostId={state.room.hostId}
          storytellerId={storytellerId}
          showStatus={phase === 'SUBMISSIONS' || phase === 'VOTING'}
          showScores={true}
        />

        {/* Hand (always visible when not storyteller phase and has hand) */}
        {hand && hand.length > 0 && phase !== 'STORYTELLER_CLUE' && phase !== 'SUBMISSIONS' && (
          <div className="bg-slate-800/50 rounded-xl p-4">
            <h3 className="text-sm font-semibold mb-2 text-gray-400">Your Hand</h3>
            <div className="flex flex-wrap gap-2">
              {hand.map((card) => (
                <Card key={card.id} card={card} size="small" />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
