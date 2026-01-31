'use client'

import { useEffect, useState } from 'react'
import { CardData } from '@/lib/types'
import { fetchCardSetCards } from '@/lib/api'

interface CardGalleryProps {
  setId: string
  setName: string
  onClose: () => void
}

export default function CardGallery({ setId, setName, onClose }: CardGalleryProps) {
  const [cards, setCards] = useState<CardData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [hoveredCard, setHoveredCard] = useState<CardData | null>(null)

  useEffect(() => {
    async function loadCards() {
      try {
        setLoading(true)
        const data = await fetchCardSetCards(setId)
        setCards(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load cards')
      } finally {
        setLoading(false)
      }
    }
    loadCards()
  }, [setId])

  // Close on escape key
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  return (
    <div
      className="fixed inset-0 bg-black/80 z-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div
        className="bg-slate-800 rounded-xl max-w-6xl w-full max-h-[90vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div>
            <h2 className="text-2xl font-bold text-white">{setName}</h2>
            <p className="text-gray-400">{cards.length} cards</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-white text-3xl font-light"
          >
            &times;
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
              <span className="ml-3 text-gray-400">Loading cards...</span>
            </div>
          )}

          {error && (
            <div className="text-red-400 text-center py-12">
              <p>{error}</p>
              <button
                onClick={() => window.location.reload()}
                className="mt-4 text-primary hover:underline"
              >
                Try again
              </button>
            </div>
          )}

          {!loading && !error && (
            <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-4">
              {cards.map((card) => (
                <div
                  key={card.id}
                  className="relative group cursor-pointer"
                  onMouseEnter={() => setHoveredCard(card)}
                  onMouseLeave={() => setHoveredCard(null)}
                >
                  <div className="aspect-[3/4] rounded-lg overflow-hidden bg-slate-700 border-2 border-transparent group-hover:border-primary transition-colors">
                    <img
                      src={card.image}
                      alt={card.title || card.id}
                      className="w-full h-full object-cover"
                      loading="lazy"
                    />
                  </div>
                  {/* Title overlay on hover */}
                  {hoveredCard?.id === card.id && card.title && (
                    <div className="absolute bottom-0 left-0 right-0 bg-black/80 text-white text-xs p-2 rounded-b-lg">
                      {card.title}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
