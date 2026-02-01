'use client'

import { useState, useEffect, use } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import { CardSet, Concept } from '@/lib/types'

interface PageProps {
  params: Promise<{ id: string }>
}

export default function ConceptsPage({ params }: PageProps) {
  const { id } = use(params)
  const router = useRouter()
  const [set, setSet] = useState<CardSet | null>(null)
  const [loading, setLoading] = useState(true)
  const [generating, setGenerating] = useState(false)
  const [editingCard, setEditingCard] = useState<string | null>(null)
  const [editPrompt, setEditPrompt] = useState('')

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

  async function handleGenerateConcepts() {
    if (!set) return
    setGenerating(true)
    try {
      const updatedSet = await api.generateConcepts(id)
      setSet(updatedSet)
    } catch (error) {
      console.error('Failed to generate concepts:', error)
      alert('Failed to generate concepts. Please try again.')
    } finally {
      setGenerating(false)
    }
  }

  async function handleUpdateConcept(cardId: string) {
    if (!set) return
    try {
      await api.updateConcept(id, cardId, { prompt: editPrompt })
      setEditingCard(null)
      loadSet()
    } catch (error) {
      console.error('Failed to update concept:', error)
    }
  }

  function startEdit(concept: Concept) {
    setEditingCard(concept.cardId)
    setEditPrompt(concept.prompt)
  }

  function handleProceedToGeneration() {
    router.push(`/generate/${id}`)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="text-center">
          <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full mx-auto"></div>
          <p className="text-slate-400 mt-4">Loading set...</p>
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

  const hasConcepts = set.concepts && set.concepts.length > 0

  return (
    <div className="max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white">{set.name}</h1>
          <p className="text-slate-400 mt-1">{set.theme}</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={handleGenerateConcepts}
            disabled={generating}
            className="px-6 py-3 bg-accent hover:bg-accent-dark text-black rounded-lg font-semibold transition-colors disabled:opacity-50"
          >
            {generating ? 'Generating...' : hasConcepts ? 'Regenerate All' : 'Generate Concepts'}
          </button>
          {hasConcepts && (
            <button
              onClick={handleProceedToGeneration}
              className="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
            >
              Generate Images →
            </button>
          )}
        </div>
      </div>

      {/* Empty state */}
      {!hasConcepts && !generating && (
        <div className="bg-slate-800/50 rounded-xl p-12 text-center">
          <div className="text-6xl mb-4">💡</div>
          <h2 className="text-xl font-semibold text-white mb-2">No concepts yet</h2>
          <p className="text-slate-400 mb-6">
            Generate AI-powered card concepts based on your theme
          </p>
          <button
            onClick={handleGenerateConcepts}
            className="px-6 py-3 bg-accent hover:bg-accent-dark text-black rounded-lg font-semibold transition-colors"
          >
            Generate {set.stats.total} Concepts
          </button>
        </div>
      )}

      {/* Generating state */}
      {generating && (
        <div className="bg-slate-800/50 rounded-xl p-12 text-center">
          <div className="animate-spin w-12 h-12 border-4 border-accent border-t-transparent rounded-full mx-auto mb-4"></div>
          <h2 className="text-xl font-semibold text-white mb-2">Generating Concepts</h2>
          <p className="text-slate-400">Using AI to create {set.stats.total} unique card ideas...</p>
        </div>
      )}

      {/* Concepts table */}
      {hasConcepts && !generating && (
        <div className="bg-slate-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="bg-slate-900">
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400 w-16">#</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400 w-48">Title</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400">Prompt</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400 w-32">Tags</th>
                <th className="px-4 py-3 text-right text-sm font-medium text-slate-400 w-24">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700">
              {set.concepts.map((concept, index) => (
                <tr key={concept.cardId} className="hover:bg-slate-700/50">
                  <td className="px-4 py-3 text-sm text-slate-500">{index + 1}</td>
                  <td className="px-4 py-3 text-sm text-white font-medium">{concept.title}</td>
                  <td className="px-4 py-3 text-sm text-slate-300">
                    {editingCard === concept.cardId ? (
                      <div className="flex gap-2">
                        <input
                          type="text"
                          value={editPrompt}
                          onChange={(e) => setEditPrompt(e.target.value)}
                          className="flex-1 px-3 py-1 bg-slate-600 border border-slate-500 rounded text-white text-sm focus:outline-none focus:border-primary-500"
                          autoFocus
                        />
                        <button
                          onClick={() => handleUpdateConcept(concept.cardId)}
                          className="px-3 py-1 bg-green-600 hover:bg-green-700 text-white rounded text-xs"
                        >
                          Save
                        </button>
                        <button
                          onClick={() => setEditingCard(null)}
                          className="px-3 py-1 bg-slate-600 hover:bg-slate-500 text-white rounded text-xs"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <span className="line-clamp-2">{concept.prompt}</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {concept.tags.slice(0, 3).map((tag) => (
                        <span
                          key={tag}
                          className="px-2 py-0.5 bg-slate-600 text-slate-300 rounded text-xs"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right">
                    {editingCard !== concept.cardId && (
                      <button
                        onClick={() => startEdit(concept)}
                        className="text-slate-400 hover:text-white text-sm"
                      >
                        Edit
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Summary */}
      {hasConcepts && (
        <div className="mt-6 flex items-center justify-between text-sm text-slate-400">
          <span>{set.concepts.length} concepts ready for image generation</span>
          <button
            onClick={handleProceedToGeneration}
            className="text-primary-400 hover:text-primary-300"
          >
            Continue to Image Generation →
          </button>
        </div>
      )}
    </div>
  )
}
