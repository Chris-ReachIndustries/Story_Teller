'use client'

import { Card as CardType } from '@/lib/types'
import Card from './Card'

interface HandProps {
  cards: CardType[]
  selectedCardId?: string | null
  onSelectCard?: (cardId: string) => void
  disabled?: boolean
}

export default function Hand({
  cards,
  selectedCardId,
  onSelectCard,
  disabled = false,
}: HandProps) {
  return (
    <div className="bg-slate-800/50 rounded-xl p-4">
      <h3 className="text-lg font-semibold mb-3 text-gray-300">Your Hand</h3>
      <div className="flex flex-wrap gap-3 justify-center">
        {cards.map((card) => (
          <Card
            key={card.id}
            card={card}
            selected={selectedCardId === card.id}
            onClick={onSelectCard ? () => onSelectCard(card.id) : undefined}
            disabled={disabled}
            size="medium"
          />
        ))}
      </div>
      {cards.length === 0 && (
        <p className="text-gray-500 text-center py-8">No cards in hand</p>
      )}
    </div>
  )
}
