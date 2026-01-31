'use client'

import { useState, useRef, useEffect } from 'react'
import { ShuffledSubmission, PlayerInfo, RoundResult } from '@/lib/types'
import Card from './Card'

const HOVER_DELAY_MS = 400 // Delay before showing zoom preview

interface VotingAreaProps {
  submissions: ShuffledSubmission[]
  selectedIndex?: number | null
  onSelectSubmission?: (index: number) => void
  disabled?: boolean
  ownSubmissionIndex?: number // Index of current player's submission (to disable voting for own card)
  revealedStorytellerIndex?: number // Index of storyteller's card (for reveal phase)
  roundResults?: RoundResult // Voting results to show who voted for what
  players?: PlayerInfo[] // Player info to get names
}

export default function VotingArea({
  submissions,
  selectedIndex,
  onSelectSubmission,
  disabled = false,
  ownSubmissionIndex,
  revealedStorytellerIndex,
  roundResults,
  players = [],
}: VotingAreaProps) {
  const [hoveredCard, setHoveredCard] = useState<ShuffledSubmission | null>(null)
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  // Clean up timeout on unmount
  useEffect(() => {
    return () => {
      if (hoverTimeoutRef.current) {
        clearTimeout(hoverTimeoutRef.current)
      }
    }
  }, [])

  const handleMouseEnter = (submission: ShuffledSubmission) => {
    // Clear any existing timeout
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
    }
    // Set new timeout for delayed zoom
    hoverTimeoutRef.current = setTimeout(() => {
      setHoveredCard(submission)
    }, HOVER_DELAY_MS)
  }

  const handleMouseLeave = () => {
    // Clear the timeout if still pending
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
      hoverTimeoutRef.current = null
    }
    // Hide the zoomed card immediately
    setHoveredCard(null)
  }

  // Get voters for a specific card index
  const getVotersForCard = (cardIndex: number): string[] => {
    if (!roundResults?.votes) return []
    const voters: string[] = []
    Object.entries(roundResults.votes).forEach(([playerId, votedIndex]) => {
      if (votedIndex === cardIndex) {
        const player = players.find((p) => p.id === playerId)
        if (player) voters.push(player.name)
      }
    })
    return voters
  }

  // Get initials from a name (first letter of first and last word)
  const getInitials = (name: string): string => {
    const words = name.trim().split(/\s+/)
    if (words.length === 1) {
      return words[0].substring(0, 2).toUpperCase()
    }
    return (words[0][0] + words[words.length - 1][0]).toUpperCase()
  }

  const isShowingResults = revealedStorytellerIndex !== undefined

  return (
    <div className="bg-slate-800/50 rounded-xl p-4">
      <h3 className="text-lg font-semibold mb-3 text-gray-300">
        {isShowingResults ? 'Results' : 'Submissions'}
      </h3>
      <div className="flex flex-wrap gap-4 justify-center">
        {submissions.map((submission) => {
          const isOwnCard = submission.index === ownSubmissionIndex
          const isStorytellerCard = submission.index === revealedStorytellerIndex
          const voters = isShowingResults ? getVotersForCard(submission.index) : []

          return (
            <div key={submission.index} className="flex flex-col items-center gap-2">
              <div
                className="relative"
                onMouseEnter={() => handleMouseEnter(submission)}
                onMouseLeave={handleMouseLeave}
              >
                <Card
                  card={submission}
                  selected={selectedIndex === submission.index}
                  onClick={
                    onSelectSubmission && !isOwnCard
                      ? () => onSelectSubmission(submission.index)
                      : undefined
                  }
                  disabled={disabled || isOwnCard}
                  showOverlay={
                    isStorytellerCard
                      ? '★ Storyteller'
                      : isOwnCard && !disabled
                      ? 'Your Card'
                      : undefined
                  }
                  size="large"
                />
                {/* Vote indicators shown during results - overlapping avatar circles */}
                {isShowingResults && voters.length > 0 && (
                  <div
                    className="absolute -bottom-3 left-1/2 -translate-x-1/2 flex items-center"
                    title={voters.join(', ')}
                  >
                    <div className="flex -space-x-2">
                      {voters.slice(0, 4).map((name, i) => (
                        <div
                          key={i}
                          className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold border-2 border-slate-800 shadow-md ${
                            submission.index === roundResults?.storytellerCard
                              ? 'bg-green-500 text-white'
                              : 'bg-slate-500 text-gray-100'
                          }`}
                          style={{ zIndex: voters.length - i }}
                        >
                          {getInitials(name)}
                        </div>
                      ))}
                      {voters.length > 4 && (
                        <div
                          className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold border-2 border-slate-800 bg-slate-600 text-gray-200 shadow-md"
                          style={{ zIndex: 0 }}
                        >
                          +{voters.length - 4}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
              <span className={`text-gray-400 text-sm ${isShowingResults && voters.length > 0 ? 'mt-4' : 'mt-2'}`}>
                Card #{submission.index + 1}
              </span>
            </div>
          )
        })}
      </div>
      {submissions.length === 0 && (
        <p className="text-gray-500 text-center py-8">No submissions yet</p>
      )}

      {/* Hover zoom preview */}
      {hoveredCard && (
        <div className="fixed inset-0 pointer-events-none z-50 flex items-center justify-center">
          <div className="pointer-events-none bg-slate-900/80 rounded-xl p-2 shadow-2xl transform transition-all duration-200">
            <img
              src={hoveredCard.image}
              alt="Card preview"
              className="max-h-[70vh] max-w-[90vw] rounded-lg object-contain"
            />
          </div>
        </div>
      )}
    </div>
  )
}
