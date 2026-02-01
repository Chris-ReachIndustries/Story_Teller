'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import api from '@/lib/api'
import { CardSet, Concept, CardStatus } from '@/lib/types'

type FilterType = 'all' | 'pending' | 'approved' | 'rejected' | 'generated'

export default function ReviewPage() {
  const params = useParams()
  const id = params.id as string
  const router = useRouter()
  const [set, setSet] = useState<CardSet | null>(null)
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<FilterType>('all')
  const [selectedCard, setSelectedCard] = useState<Concept | null>(null)
  const [editPrompt, setEditPrompt] = useState('')
  const [regenerating, setRegenerating] = useState(false)

  useEffect(() => {
    loadSet()
  }, [id])

  async function loadSet() {
    try {
      const data = await api.getSet(id)
      setSet(data)
    } catch (error) {
      console.error('Failed to load set:', error)
    } finally {
      setLoading(false)
    }
  }

  async function handleApprove(cardId: string) {
    try {
      await api.approveCard(id, cardId)
      loadSet()
      if (selectedCard?.cardId === cardId) {
        setSelectedCard(null)
      }
    } catch (error) {
      console.error('Failed to approve:', error)
    }
  }

  async function handleReject(cardId: string) {
    try {
      await api.rejectCard(id, cardId)
      loadSet()
      if (selectedCard?.cardId === cardId) {
        setSelectedCard(null)
      }
    } catch (error) {
      console.error('Failed to reject:', error)
    }
  }

  async function handleRegenerate() {
    if (!selectedCard) return
    setRegenerating(true)
    try {
      await api.regenerateCard(id, selectedCard.cardId, editPrompt || undefined)
      loadSet()
      setSelectedCard(null)
    } catch (error) {
      console.error('Failed to regenerate:', error)
    } finally {
      setRegenerating(false)
    }
  }

  async function handleApproveAllPending() {
    if (!set) return
    const pendingCards = set.concepts.filter(c =>
      c.status === 'generated' || c.status === 'pending'
    )
    for (const card of pendingCards) {
      await handleApprove(card.cardId)
    }
  }

  function openCard(concept: Concept) {
    setSelectedCard(concept)
    setEditPrompt(concept.prompt)
  }

  function filteredConcepts() {
    if (!set) return []
    if (filter === 'all') return set.concepts
    return set.concepts.filter(c => {
      if (filter === 'pending') return c.status === 'pending' || c.status === 'generated'
      return c.status === filter
    })
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="text-center">
          <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full mx-auto"></div>
          <p className="text-slate-400 mt-4">Loading...</p>
        </div>
      </div>
    )
  }

  if (!set) {
    return (
      <div className="text-center py-12">
        <p className="text-slate-400">Set not found</p>
      </div>
    )
  }

  const cards = filteredConcepts()
  const hasGeneratedCards = set.concepts.some(c =>
    ['generated', 'approved', 'rejected'].includes(c.status)
  )

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white">{set.name}</h1>
          <p className="text-slate-400 mt-1">Review and approve cards</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={handleApproveAllPending}
            className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg text-sm transition-colors"
          >
            Approve All Pending
          </button>
          <button
            onClick={() => router.push(`/export/${id}`)}
            className="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
          >
            Export Set →
          </button>
        </div>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-2 mb-6">
        {(['all', 'pending', 'approved', 'rejected'] as FilterType[]).map((f) => {
          const count = f === 'all'
            ? set.concepts.length
            : f === 'pending'
            ? set.stats.pending + (set.stats.generated - set.stats.approved - set.stats.rejected)
            : set.stats[f as keyof typeof set.stats] || 0

          return (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                filter === f
                  ? 'bg-primary-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {f.charAt(0).toUpperCase() + f.slice(1)} ({count})
            </button>
          )
        })}
      </div>

      {/* Cards grid */}
      {!hasGeneratedCards ? (
        <div className="bg-slate-800/50 rounded-xl p-12 text-center">
          <div className="text-6xl mb-4">🎨</div>
          <h2 className="text-xl font-semibold text-white mb-2">No images generated yet</h2>
          <p className="text-slate-400 mb-6">
            Generate images first before reviewing
          </p>
          <button
            onClick={() => router.push(`/generate/${id}`)}
            className="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
          >
            Go to Generation
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-4 md:grid-cols-5 lg:grid-cols-6 gap-4">
          {cards.map((concept) => {
            const hasImage = ['generated', 'approved', 'rejected'].includes(concept.status)

            return (
              <div
                key={concept.cardId}
                onClick={() => hasImage && openCard(concept)}
                className={`aspect-[3/4] rounded-xl overflow-hidden relative cursor-pointer card-hover ${
                  concept.status === 'approved' ? 'ring-2 ring-green-500' :
                  concept.status === 'rejected' ? 'ring-2 ring-red-500' : ''
                }`}
              >
                {hasImage ? (
                  <img
                    src={api.getCardImageUrl(id, concept.cardId)}
                    alt={concept.title}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="w-full h-full bg-slate-700 flex items-center justify-center">
                    <span className="text-slate-500">{concept.cardId}</span>
                  </div>
                )}
                {/* Status badge */}
                <div className={`absolute top-2 right-2 px-2 py-1 rounded text-xs font-medium ${
                  concept.status === 'approved' ? 'status-approved' :
                  concept.status === 'rejected' ? 'status-rejected' :
                  'status-pending'
                }`}>
                  {concept.status}
                </div>
                {/* Title overlay */}
                <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-3">
                  <p className="text-white text-sm font-medium truncate">{concept.title}</p>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Card detail modal */}
      {selectedCard && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-8">
          <div className="bg-slate-800 rounded-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex">
            {/* Image side */}
            <div className="flex-1 bg-black flex items-center justify-center">
              <img
                src={api.getCardImageUrl(id, selectedCard.cardId)}
                alt={selectedCard.title}
                className="max-w-full max-h-[80vh] object-contain"
              />
            </div>
            {/* Details side */}
            <div className="w-96 p-6 flex flex-col">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-xl font-bold text-white">{selectedCard.title}</h2>
                <button
                  onClick={() => setSelectedCard(null)}
                  className="text-slate-400 hover:text-white text-2xl"
                >
                  ×
                </button>
              </div>

              <div className="mb-4">
                <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                  selectedCard.status === 'approved' ? 'status-approved' :
                  selectedCard.status === 'rejected' ? 'status-rejected' :
                  'status-pending'
                }`}>
                  {selectedCard.status}
                </span>
              </div>

              <div className="mb-4">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Prompt
                </label>
                <textarea
                  value={editPrompt}
                  onChange={(e) => setEditPrompt(e.target.value)}
                  rows={4}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500 resize-none"
                />
              </div>

              <div className="mb-4">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Tags
                </label>
                <div className="flex flex-wrap gap-2">
                  {selectedCard.tags.map((tag) => (
                    <span
                      key={tag}
                      className="px-2 py-1 bg-slate-600 text-slate-300 rounded text-xs"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>

              <div className="flex-1" />

              {/* Actions */}
              <div className="space-y-3">
                <button
                  onClick={handleRegenerate}
                  disabled={regenerating}
                  className="w-full px-4 py-2 bg-accent hover:bg-accent-dark text-black rounded-lg font-medium transition-colors disabled:opacity-50"
                >
                  {regenerating ? 'Regenerating...' : 'Regenerate'}
                </button>
                <div className="flex gap-3">
                  <button
                    onClick={() => handleReject(selectedCard.cardId)}
                    className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium transition-colors"
                  >
                    Reject
                  </button>
                  <button
                    onClick={() => handleApprove(selectedCard.cardId)}
                    className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium transition-colors"
                  >
                    Approve
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Stats bar */}
      <div className="fixed bottom-0 left-64 right-0 bg-slate-900 border-t border-slate-700 p-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex gap-6 text-sm">
            <span>
              <span className="text-slate-400">Total: </span>
              <span className="text-white font-semibold">{set.stats.total}</span>
            </span>
            <span>
              <span className="text-slate-400">Generated: </span>
              <span className="text-blue-400 font-semibold">{set.stats.generated}</span>
            </span>
            <span>
              <span className="text-slate-400">Approved: </span>
              <span className="text-green-400 font-semibold">{set.stats.approved}</span>
            </span>
            <span>
              <span className="text-slate-400">Rejected: </span>
              <span className="text-red-400 font-semibold">{set.stats.rejected}</span>
            </span>
          </div>
          <span className="text-sm text-slate-400">
            {set.stats.approved} / {set.stats.total} cards approved ({Math.round(set.stats.approved / set.stats.total * 100)}%)
          </span>
        </div>
      </div>
    </div>
  )
}
