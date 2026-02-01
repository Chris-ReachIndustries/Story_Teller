'use client'

import { Concept } from '@/lib/types'

interface CardPreviewProps {
  concept: Concept
  imageUrl?: string
  onClick?: () => void
  size?: 'small' | 'medium' | 'large'
  showStatus?: boolean
}

const sizeClasses = {
  small: 'w-24',
  medium: 'w-40',
  large: 'w-56',
}

export default function CardPreview({
  concept,
  imageUrl,
  onClick,
  size = 'medium',
  showStatus = true,
}: CardPreviewProps) {
  const hasImage = imageUrl && ['generated', 'approved', 'rejected'].includes(concept.status)

  return (
    <div
      onClick={onClick}
      className={`${sizeClasses[size]} aspect-[3/4] rounded-xl overflow-hidden relative ${
        onClick ? 'cursor-pointer card-hover' : ''
      } ${
        concept.status === 'approved' ? 'ring-2 ring-green-500' :
        concept.status === 'rejected' ? 'ring-2 ring-red-500' : ''
      }`}
    >
      {hasImage ? (
        <img
          src={imageUrl}
          alt={concept.title}
          className="w-full h-full object-cover"
        />
      ) : (
        <div className={`w-full h-full flex items-center justify-center ${
          concept.status === 'generating' ? 'bg-accent/20' : 'bg-slate-700'
        }`}>
          {concept.status === 'generating' ? (
            <div className="animate-spin w-6 h-6 border-2 border-accent border-t-transparent rounded-full"></div>
          ) : (
            <span className="text-slate-500 text-xs">{concept.cardId}</span>
          )}
        </div>
      )}

      {/* Status badge */}
      {showStatus && (
        <div className={`absolute top-2 right-2 px-2 py-1 rounded text-xs font-medium ${
          concept.status === 'approved' ? 'status-approved' :
          concept.status === 'rejected' ? 'status-rejected' :
          concept.status === 'generating' ? 'status-generating' :
          'status-pending'
        }`}>
          {concept.status}
        </div>
      )}

      {/* Title overlay */}
      <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-3">
        <p className="text-white text-sm font-medium truncate">{concept.title}</p>
      </div>
    </div>
  )
}
