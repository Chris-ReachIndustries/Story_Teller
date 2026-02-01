'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter, useParams } from 'next/navigation'
import api from '@/lib/api'
import { CardSet, Concept } from '@/lib/types'

type EditMode = 'title' | 'prompt' | null

export default function ConceptsPage() {
  const params = useParams()
  const id = params.id as string
  const router = useRouter()
  const [set, setSet] = useState<CardSet | null>(null)
  const [loading, setLoading] = useState(true)
  const [generating, setGenerating] = useState(false)
  const [editingCard, setEditingCard] = useState<string | null>(null)
  const [editMode, setEditMode] = useState<EditMode>(null)
  const [editTitle, setEditTitle] = useState('')
  const [editPrompt, setEditPrompt] = useState('')
  const [regeneratingCard, setRegeneratingCard] = useState<string | null>(null)
  const [menuOpen, setMenuOpen] = useState<string | null>(null)
  const [showFullTheme, setShowFullTheme] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    loadSet()
  }, [id])

  // Close menu when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

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
      const updates: Partial<Concept> = {}
      if (editMode === 'title') {
        updates.title = editTitle
      } else if (editMode === 'prompt') {
        updates.prompt = editPrompt
      }
      await api.updateConcept(id, cardId, updates)
      setEditingCard(null)
      setEditMode(null)
      loadSet()
    } catch (error) {
      console.error('Failed to update concept:', error)
    }
  }

  async function handleRegenerateConcept(cardId: string, regenerateTitle: boolean, regeneratePrompt: boolean) {
    setRegeneratingCard(cardId)
    setMenuOpen(null)
    try {
      await api.regenerateConceptText(id, cardId, { regenerateTitle, regeneratePrompt })
      loadSet()
    } catch (error) {
      console.error('Failed to regenerate concept:', error)
      alert('Failed to regenerate. Please try again.')
    } finally {
      setRegeneratingCard(null)
    }
  }

  function startEdit(concept: Concept, mode: EditMode) {
    setEditingCard(concept.cardId)
    setEditMode(mode)
    setEditTitle(concept.title)
    setEditPrompt(concept.prompt)
    setMenuOpen(null)
  }

  function cancelEdit() {
    setEditingCard(null)
    setEditMode(null)
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
  const themePreview = set.theme.length > 100 ? set.theme.slice(0, 100) + '...' : set.theme

  return (
    <div className="max-w-6xl mx-auto">
      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div className="flex-1 mr-4">
          <h1 className="text-3xl font-bold text-white">{set.name}</h1>
          <div className="mt-2">
            {set.theme.length > 100 ? (
              <div>
                <p className="text-slate-400 text-sm">
                  {showFullTheme ? set.theme : themePreview}
                </p>
                <button
                  onClick={() => setShowFullTheme(!showFullTheme)}
                  className="text-primary-400 hover:text-primary-300 text-xs mt-1"
                >
                  {showFullTheme ? 'Show less' : 'Show more'}
                </button>
              </div>
            ) : (
              <p className="text-slate-400 text-sm">{set.theme}</p>
            )}
          </div>
        </div>
        <div className="flex gap-3 flex-shrink-0">
          {/* Only show Regenerate All when we have concepts */}
          {hasConcepts && (
            <button
              onClick={handleGenerateConcepts}
              disabled={generating}
              className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50 text-sm"
            >
              {generating ? 'Generating...' : 'Regenerate All'}
            </button>
          )}
          {hasConcepts && (
            <button
              onClick={handleProceedToGeneration}
              className="px-6 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
            >
              Generate Images →
            </button>
          )}
        </div>
      </div>

      {/* Empty state - only button here when no concepts */}
      {!hasConcepts && !generating && (
        <div className="bg-slate-800/50 rounded-xl p-12 text-center">
          <div className="text-6xl mb-4">💡</div>
          <h2 className="text-xl font-semibold text-white mb-2">Ready to Generate Concepts</h2>
          <p className="text-slate-400 mb-6 max-w-md mx-auto">
            Create {set.stats.total} unique card concepts using AI based on your theme
          </p>
          <button
            onClick={handleGenerateConcepts}
            className="px-8 py-4 bg-accent hover:bg-accent-dark text-black rounded-lg font-semibold transition-colors text-lg"
          >
            Generate {set.stats.total} Concepts
          </button>
        </div>
      )}

      {/* Generating state */}
      {generating && (
        <div className="bg-slate-800 rounded-xl overflow-hidden">
          {/* Progress header */}
          <div className="bg-slate-900 px-6 py-4 border-b border-slate-700">
            <div className="flex items-center gap-4">
              <div className="animate-spin w-5 h-5 border-2 border-accent border-t-transparent rounded-full"></div>
              <div>
                <h3 className="text-white font-medium">Generating Concepts...</h3>
                <p className="text-slate-400 text-sm">Creating {set.stats.total} unique card ideas with AI</p>
              </div>
            </div>
          </div>

          {/* Skeleton table */}
          <table className="w-full">
            <thead>
              <tr className="bg-slate-900/50">
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400 w-16">#</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400 w-48">Title</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400">Prompt</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-400 w-32">Tags</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700">
              {[...Array(Math.min(10, set.stats.total))].map((_, index) => (
                <tr key={index} className="animate-pulse">
                  <td className="px-4 py-3">
                    <div className="h-4 w-6 bg-slate-700 rounded"></div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="h-4 w-32 bg-slate-700 rounded"></div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="h-4 w-full bg-slate-700 rounded"></div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1">
                      <div className="h-5 w-12 bg-slate-700 rounded"></div>
                      <div className="h-5 w-10 bg-slate-700 rounded"></div>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {set.stats.total > 10 && (
            <div className="px-6 py-3 text-center text-slate-500 text-sm border-t border-slate-700">
              And {set.stats.total - 10} more...
            </div>
          )}
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
                  <td className="px-4 py-3 text-sm text-white font-medium">
                    {editingCard === concept.cardId && editMode === 'title' ? (
                      <div className="flex gap-2">
                        <input
                          type="text"
                          value={editTitle}
                          onChange={(e) => setEditTitle(e.target.value)}
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
                          onClick={cancelEdit}
                          className="px-3 py-1 bg-slate-600 hover:bg-slate-500 text-white rounded text-xs"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      concept.title
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-300">
                    {editingCard === concept.cardId && editMode === 'prompt' ? (
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
                          onClick={cancelEdit}
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
                  <td className="px-4 py-3 text-right relative">
                    {regeneratingCard === concept.cardId ? (
                      <div className="flex items-center justify-end gap-2">
                        <div className="animate-spin w-4 h-4 border-2 border-primary-500 border-t-transparent rounded-full"></div>
                        <span className="text-xs text-slate-400">Regenerating...</span>
                      </div>
                    ) : editingCard !== concept.cardId ? (
                      <div className="relative" ref={menuOpen === concept.cardId ? menuRef : null}>
                        <button
                          onClick={() => setMenuOpen(menuOpen === concept.cardId ? null : concept.cardId)}
                          className="text-slate-400 hover:text-white text-sm px-2 py-1 rounded hover:bg-slate-700"
                        >
                          Actions ▾
                        </button>
                        {menuOpen === concept.cardId && (
                          <div className="absolute right-0 top-full mt-1 w-48 bg-slate-700 rounded-lg shadow-xl border border-slate-600 z-10">
                            <div className="py-1">
                              <div className="px-3 py-1 text-xs text-slate-400 uppercase tracking-wide">Manual Edit</div>
                              <button
                                onClick={() => startEdit(concept, 'title')}
                                className="w-full px-3 py-2 text-left text-sm text-white hover:bg-slate-600"
                              >
                                Edit Title
                              </button>
                              <button
                                onClick={() => startEdit(concept, 'prompt')}
                                className="w-full px-3 py-2 text-left text-sm text-white hover:bg-slate-600"
                              >
                                Edit Prompt
                              </button>
                              <div className="border-t border-slate-600 my-1"></div>
                              <div className="px-3 py-1 text-xs text-slate-400 uppercase tracking-wide">AI Regenerate</div>
                              <button
                                onClick={() => handleRegenerateConcept(concept.cardId, true, false)}
                                className="w-full px-3 py-2 text-left text-sm text-white hover:bg-slate-600"
                              >
                                Regenerate Title
                              </button>
                              <button
                                onClick={() => handleRegenerateConcept(concept.cardId, false, true)}
                                className="w-full px-3 py-2 text-left text-sm text-white hover:bg-slate-600"
                              >
                                Regenerate Prompt
                              </button>
                              <button
                                onClick={() => handleRegenerateConcept(concept.cardId, true, true)}
                                className="w-full px-3 py-2 text-left text-sm text-white hover:bg-slate-600"
                              >
                                Regenerate Both
                              </button>
                            </div>
                          </div>
                        )}
                      </div>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Summary footer */}
      {hasConcepts && !generating && (
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
