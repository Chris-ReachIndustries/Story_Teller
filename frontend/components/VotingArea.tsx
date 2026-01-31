'use client'

import { ShuffledSubmission } from '@/lib/types'
import Card from './Card'

interface VotingAreaProps {
  submissions: ShuffledSubmission[]
  selectedIndex?: number | null
  onSelectSubmission?: (index: number) => void
  disabled?: boolean
  ownSubmissionIndex?: number // Index of current player's submission (to disable voting for own card)
  revealedStorytellerIndex?: number // Index of storyteller's card (for reveal phase)
}

export default function VotingArea({
  submissions,
  selectedIndex,
  onSelectSubmission,
  disabled = false,
  ownSubmissionIndex,
  revealedStorytellerIndex,
}: VotingAreaProps) {
  return (
    <div className="bg-slate-800/50 rounded-xl p-4">
      <h3 className="text-lg font-semibold mb-3 text-gray-300">
        {revealedStorytellerIndex !== undefined ? 'Results' : 'Submissions'}
      </h3>
      <div className="flex flex-wrap gap-4 justify-center">
        {submissions.map((submission) => {
          const isOwnCard = submission.index === ownSubmissionIndex
          const isStorytellerCard = submission.index === revealedStorytellerIndex

          return (
            <div key={submission.index} className="flex flex-col items-center gap-2">
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
              <span className="text-gray-400 text-sm">Card #{submission.index + 1}</span>
            </div>
          )
        })}
      </div>
      {submissions.length === 0 && (
        <p className="text-gray-500 text-center py-8">No submissions yet</p>
      )}
    </div>
  )
}
