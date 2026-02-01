'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import { CardSet } from '@/lib/types'

export default function HomePage() {
  const router = useRouter()
  const [sets, setSets] = useState<CardSet[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)

  // New set form
  const [showNewForm, setShowNewForm] = useState(false)
  const [newSetName, setNewSetName] = useState('')
  const [newSetTheme, setNewSetTheme] = useState('')
  const [cardCount, setCardCount] = useState(100)

  useEffect(() => {
    loadSets()
  }, [])

  async function loadSets() {
    try {
      const response = await api.listSets()
      setSets(response.sets || [])
    } catch (error) {
      console.error('Failed to load sets:', error)
    } finally {
      setLoading(false)
    }
  }

  async function handleCreateSet(e: React.FormEvent) {
    e.preventDefault()
    if (!newSetName.trim() || !newSetTheme.trim()) return

    setCreating(true)
    try {
      const newSet = await api.createSet({
        name: newSetName.trim(),
        theme: newSetTheme.trim(),
        cardCount,
      })
      router.push(`/concepts/${newSet.id}`)
    } catch (error) {
      console.error('Failed to create set:', error)
      alert('Failed to create set. Please try again.')
    } finally {
      setCreating(false)
    }
  }

  function getStatusBadge(status: string) {
    const styles: Record<string, string> = {
      concept: 'status-pending',
      generating: 'status-generating',
      reviewing: 'status-approved',
      exported: 'bg-purple-500/20 text-purple-400 border border-purple-500/30',
    }
    return styles[status] || styles.concept
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white">Your Card Sets</h1>
          <p className="text-slate-400 mt-1">Create and manage AI-generated Dixit card sets</p>
        </div>
        <button
          onClick={() => setShowNewForm(true)}
          className="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
        >
          + New Set
        </button>
      </div>

      {/* New Set Modal */}
      {showNewForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-800 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-xl font-bold text-white mb-4">Create New Card Set</h2>
            <form onSubmit={handleCreateSet} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Set Name
                </label>
                <input
                  type="text"
                  value={newSetName}
                  onChange={(e) => setNewSetName(e.target.value)}
                  placeholder="e.g., Fantasy Ocean"
                  className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Theme Description
                </label>
                <textarea
                  value={newSetTheme}
                  onChange={(e) => setNewSetTheme(e.target.value)}
                  placeholder="e.g., Underwater fantasy creatures, bioluminescent scenes, deep sea mysteries..."
                  rows={3}
                  className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500 resize-none"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Number of Cards
                </label>
                <input
                  type="number"
                  value={cardCount}
                  onChange={(e) => setCardCount(Number(e.target.value))}
                  min={10}
                  max={200}
                  className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
                />
              </div>
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowNewForm(false)}
                  className="flex-1 px-4 py-2 bg-slate-600 hover:bg-slate-500 text-white rounded-lg transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors disabled:opacity-50"
                >
                  {creating ? 'Creating...' : 'Create Set'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Loading state */}
      {loading && (
        <div className="text-center py-12">
          <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full mx-auto"></div>
          <p className="text-slate-400 mt-4">Loading sets...</p>
        </div>
      )}

      {/* Empty state */}
      {!loading && sets.length === 0 && (
        <div className="bg-slate-800/50 rounded-xl p-12 text-center">
          <div className="text-6xl mb-4">🎨</div>
          <h2 className="text-xl font-semibold text-white mb-2">No card sets yet</h2>
          <p className="text-slate-400 mb-6">
            Create your first AI-generated Dixit card set
          </p>
          <button
            onClick={() => setShowNewForm(true)}
            className="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
          >
            Create Your First Set
          </button>
        </div>
      )}

      {/* Sets grid */}
      {!loading && sets.length > 0 && (
        <div className="grid gap-4">
          {sets.map((set) => (
            <div
              key={set.id}
              onClick={() => {
                if (set.status === 'concept') {
                  router.push(`/concepts/${set.id}`)
                } else if (set.status === 'generating') {
                  router.push(`/generate/${set.id}`)
                } else {
                  router.push(`/review/${set.id}`)
                }
              }}
              className="bg-slate-800 hover:bg-slate-700 rounded-xl p-6 cursor-pointer transition-colors"
            >
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-lg font-semibold text-white">{set.name}</h3>
                  <p className="text-slate-400 text-sm mt-1 line-clamp-2">{set.theme}</p>
                </div>
                <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusBadge(set.status)}`}>
                  {set.status}
                </span>
              </div>
              <div className="flex gap-6 mt-4 text-sm">
                <div>
                  <span className="text-slate-500">Total: </span>
                  <span className="text-white">{set.stats.total}</span>
                </div>
                <div>
                  <span className="text-slate-500">Generated: </span>
                  <span className="text-blue-400">{set.stats.generated}</span>
                </div>
                <div>
                  <span className="text-slate-500">Approved: </span>
                  <span className="text-green-400">{set.stats.approved}</span>
                </div>
                <div>
                  <span className="text-slate-500">Rejected: </span>
                  <span className="text-red-400">{set.stats.rejected}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
