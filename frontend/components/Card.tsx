'use client'

import { Card as CardType } from '@/lib/types'

interface CardProps {
  card: CardType | { index: number; image: string }
  selected?: boolean
  onClick?: () => void
  disabled?: boolean
  showOverlay?: string
  size?: 'small' | 'medium' | 'large'
}

export default function Card({
  card,
  selected = false,
  onClick,
  disabled = false,
  showOverlay,
  size = 'medium',
}: CardProps) {
  const sizeClasses = {
    small: 'w-24 h-32',
    medium: 'w-32 h-44',
    large: 'w-40 h-56',
  }

  return (
    <div
      onClick={disabled ? undefined : onClick}
      className={`
        relative rounded-lg overflow-hidden bg-slate-700 shadow-lg
        ${sizeClasses[size]}
        ${onClick && !disabled ? 'card-hover cursor-pointer' : ''}
        ${selected ? 'card-selected' : ''}
        ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
      `}
    >
      <img
        src={card.image}
        alt="Card"
        className="w-full h-full object-cover"
      />
      {showOverlay && (
        <div className="absolute inset-0 bg-black/60 flex items-center justify-center">
          <span className="text-white font-bold text-lg">{showOverlay}</span>
        </div>
      )}
      {selected && (
        <div className="absolute top-2 right-2 w-6 h-6 bg-primary rounded-full flex items-center justify-center">
          <span className="text-white text-sm">✓</span>
        </div>
      )}
    </div>
  )
}
