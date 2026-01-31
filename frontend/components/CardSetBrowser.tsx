'use client'

import { useEffect, useState } from 'react'
import { CardSetInfo } from '@/lib/types'
import { fetchCardSets } from '@/lib/api'
import CardGallery from './CardGallery'

interface CardSetBrowserProps {
  // Selection mode: if provided, shows select buttons and calls this when selected
  onSelect?: (setId: string) => void
  selectedSetId?: string
  // Browse mode: just shows sets without selection
  browseOnly?: boolean
}

export default function CardSetBrowser({
  onSelect,
  selectedSetId,
  browseOnly = false,
}: CardSetBrowserProps) {
  const [sets, setSets] = useState<CardSetInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [gallerySet, setGallerySet] = useState<CardSetInfo | null>(null)

  useEffect(() => {
    async function loadSets() {
      try {
        setLoading(true)
        const data = await fetchCardSets()
        setSets(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load card sets')
      } finally {
        setLoading(false)
      }
    }
    loadSets()
  }, [])

  if (loading) {
    return (
      <div className="bg-slate-800/50 rounded-xl p-6">
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          <span className="ml-3 text-gray-400">Loading card sets...</span>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-slate-800/50 rounded-xl p-6">
        <div className="text-red-400 text-center py-8">
          <p>{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-slate-800/50 rounded-xl p-6">
      <h2 className="text-xl font-bold text-white mb-4">
        {browseOnly ? 'Browse Card Sets' : 'Select Card Set'}
      </h2>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {sets.map((set) => {
          const isSelected = selectedSetId === set.id

          return (
            <div
              key={set.id}
              className={`bg-slate-900/50 rounded-lg p-4 border-2 transition-colors ${
                isSelected
                  ? 'border-primary'
                  : 'border-transparent hover:border-slate-600'
              }`}
            >
              {/* Set info */}
              <div className="flex items-start justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-white text-lg">{set.name}</h3>
                  <p className="text-gray-400 text-sm">{set.cardCount} cards</p>
                </div>
                {isSelected && (
                  <span className="bg-primary text-white text-xs px-2 py-1 rounded">
                    Selected
                  </span>
                )}
              </div>

              {/* Preview thumbnails */}
              <div className="grid grid-cols-4 gap-2 mb-3">
                {set.previewUrls.map((url, index) => (
                  <div
                    key={index}
                    className="aspect-[3/4] rounded overflow-hidden bg-slate-700"
                  >
                    <img
                      src={url}
                      alt={`Preview ${index + 1}`}
                      className="w-full h-full object-cover"
                    />
                  </div>
                ))}
              </div>

              {/* Actions */}
              <div className="flex gap-2">
                <button
                  onClick={() => setGallerySet(set)}
                  className="flex-1 px-3 py-2 rounded bg-slate-700 hover:bg-slate-600 text-gray-300 text-sm transition-colors"
                >
                  View All
                </button>
                {!browseOnly && onSelect && (
                  <button
                    onClick={() => onSelect(set.id)}
                    disabled={isSelected}
                    className={`flex-1 px-3 py-2 rounded text-sm transition-colors ${
                      isSelected
                        ? 'bg-primary/30 text-primary cursor-default'
                        : 'bg-primary hover:bg-primary/80 text-white'
                    }`}
                  >
                    {isSelected ? 'Selected' : 'Select'}
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {sets.length === 0 && (
        <p className="text-gray-500 text-center py-8">No card sets available</p>
      )}

      {/* Gallery modal */}
      {gallerySet && (
        <CardGallery
          setId={gallerySet.id}
          setName={gallerySet.name}
          onClose={() => setGallerySet(null)}
        />
      )}
    </div>
  )
}
