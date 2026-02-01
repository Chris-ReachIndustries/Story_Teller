'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter, useParams } from 'next/navigation'
import api from '@/lib/api'
import { CardSet, GenerationProgress, QualityMode } from '@/lib/types'

export default function GeneratePage() {
  const params = useParams()
  const id = params.id as string
  const router = useRouter()
  const [set, setSet] = useState<CardSet | null>(null)
  const [progress, setProgress] = useState<GenerationProgress | null>(null)
  const [loading, setLoading] = useState(true)
  const [cancelling, setCancelling] = useState(false)
  const [qualityMode, setQualityMode] = useState<QualityMode>('high')
  const pollRef = useRef<NodeJS.Timeout | null>(null)

  useEffect(() => {
    loadData()
    return () => {
      if (pollRef.current) {
        clearInterval(pollRef.current)
      }
    }
  }, [id])

  async function loadData() {
    try {
      const [setData, progressData] = await Promise.all([
        api.getSet(id),
        api.getProgress(id).catch(() => null),
      ])
      setSet(setData)
      setProgress(progressData)

      // Start polling if generation is running
      if (progressData?.status === 'running') {
        startPolling()
      }
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  function startPolling() {
    if (pollRef.current) return
    pollRef.current = setInterval(async () => {
      try {
        const [setData, progressData] = await Promise.all([
          api.getSet(id),
          api.getProgress(id),
        ])
        setSet(setData)
        setProgress(progressData)

        if (progressData.status === 'complete' || progressData.status === 'error' || progressData.status === 'idle') {
          if (pollRef.current) {
            clearInterval(pollRef.current)
            pollRef.current = null
          }
        }
      } catch (error) {
        console.error('Polling error:', error)
      }
    }, 1000) // Poll every second for more responsive updates
  }

  async function handleStartGeneration() {
    try {
      await api.startGeneration(id, qualityMode)
      setProgress({ status: 'running', completed: 0, total: set?.concepts.length || 0, current: '' })
      startPolling()
    } catch (error) {
      console.error('Failed to start generation:', error)
      alert('Failed to start generation. Please try again.')
    }
  }

  async function handleCancel() {
    setCancelling(true)
    try {
      await api.cancelGeneration(id)
      if (pollRef.current) {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
      setProgress(null)
      // Reload to get fresh state
      await loadData()
    } catch (error) {
      console.error('Failed to cancel:', error)
      alert('Failed to cancel generation.')
    } finally {
      setCancelling(false)
    }
  }

  async function handlePause() {
    try {
      await api.pauseGeneration(id)
      if (pollRef.current) {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
      setProgress(prev => prev ? { ...prev, status: 'paused' } : null)
    } catch (error) {
      console.error('Failed to pause:', error)
    }
  }

  async function handleResume() {
    try {
      await api.startGeneration(id) // Resume by starting again
      setProgress(prev => prev ? { ...prev, status: 'running' } : null)
      startPolling()
    } catch (error) {
      console.error('Failed to resume:', error)
    }
  }

  function handleProceedToReview() {
    router.push(`/review/${id}`)
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

  const isRunning = progress?.status === 'running'
  const isPaused = progress?.status === 'paused'
  const isComplete = progress?.status === 'complete'
  const isIdle = !progress || progress.status === 'idle'
  const completedCount = progress?.completed || 0
  const totalCount = set.concepts.length
  const progressPercent = totalCount > 0 ? (completedCount / totalCount) * 100 : 0

  // Check which cards have images by their status
  const hasGeneratedImages = set.concepts.some(c =>
    c.status === 'generated' || c.status === 'approved' || c.status === 'rejected'
  )

  return (
    <div className="max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white">{set.name}</h1>
          <p className="text-slate-400 mt-1">Image Generation</p>
        </div>
        <div className="flex gap-3">
          {/* Show Start button only when idle */}
          {isIdle && !isComplete && (
            <button
              onClick={handleStartGeneration}
              className="px-6 py-3 bg-accent hover:bg-accent-dark text-black rounded-lg font-semibold transition-colors"
            >
              Start Generation
            </button>
          )}

          {/* Show Pause and Cancel when running */}
          {isRunning && (
            <>
              <button
                onClick={handlePause}
                className="px-6 py-3 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg font-semibold transition-colors"
              >
                Pause
              </button>
              <button
                onClick={handleCancel}
                disabled={cancelling}
                className="px-6 py-3 bg-red-600 hover:bg-red-700 text-white rounded-lg font-semibold transition-colors disabled:opacity-50"
              >
                {cancelling ? 'Cancelling...' : 'Cancel'}
              </button>
            </>
          )}

          {/* Show Resume and Cancel when paused */}
          {isPaused && (
            <>
              <button
                onClick={handleResume}
                className="px-6 py-3 bg-green-600 hover:bg-green-700 text-white rounded-lg font-semibold transition-colors"
              >
                Resume
              </button>
              <button
                onClick={handleCancel}
                disabled={cancelling}
                className="px-6 py-3 bg-red-600 hover:bg-red-700 text-white rounded-lg font-semibold transition-colors disabled:opacity-50"
              >
                {cancelling ? 'Cancelling...' : 'Cancel'}
              </button>
            </>
          )}

          {/* Review Cards button - only enabled when complete or has some images */}
          <button
            onClick={handleProceedToReview}
            disabled={isRunning || (!isComplete && !hasGeneratedImages)}
            className={`px-6 py-3 rounded-lg font-semibold transition-colors ${
              !isRunning && (isComplete || hasGeneratedImages)
                ? 'bg-primary-600 hover:bg-primary-700 text-white'
                : 'bg-slate-700 text-slate-500 cursor-not-allowed'
            }`}
          >
            Review Cards →
          </button>
        </div>
      </div>

      {/* Quality Mode Selector - only show when idle */}
      {isIdle && !isComplete && (
        <div className="bg-slate-800 rounded-xl p-6 mb-6">
          <h3 className="text-lg font-semibold text-white mb-4">Generation Settings</h3>

          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-400 mb-3">
              Image Quality
            </label>
            <div className="grid grid-cols-3 gap-3">
              {[
                { value: 'fast' as QualityMode, label: 'Fast', time: '~20s/image', desc: 'Quick preview, lower resolution' },
                { value: 'normal' as QualityMode, label: 'Normal', time: '~45s/image', desc: 'Good balance of quality and speed' },
                { value: 'high' as QualityMode, label: 'High', time: '~90s/image', desc: 'Best quality, recommended' },
              ].map((option) => (
                <button
                  key={option.value}
                  onClick={() => setQualityMode(option.value)}
                  className={`p-4 rounded-lg border-2 text-left transition-all ${
                    qualityMode === option.value
                      ? 'border-primary-500 bg-primary-500/20'
                      : 'border-slate-600 hover:border-slate-500 bg-slate-700/50'
                  }`}
                >
                  <div className="font-semibold text-white">{option.label}</div>
                  <div className="text-sm text-primary-400 mt-1">{option.time}</div>
                  <div className="text-xs text-slate-400 mt-2">{option.desc}</div>
                </button>
              ))}
            </div>
          </div>

          <p className="text-sm text-slate-500">
            High quality mode produces visually coherent illustrations with fewer artifacts.
            All cards in a set share a consistent visual style derived from the theme.
          </p>
        </div>
      )}

      {/* Progress bar */}
      <div className="bg-slate-800 rounded-xl p-6 mb-8">
        <div className="flex items-center justify-between mb-4">
          <div>
            <span className="text-2xl font-bold text-white">
              {completedCount} / {totalCount}
            </span>
            <span className="text-slate-400 ml-2">cards generated</span>
          </div>
          <div className="flex items-center gap-4">
            {isRunning && progress?.current && (
              <span className="text-sm text-slate-400">
                Generating: {progress.current}
              </span>
            )}
            <span className={`px-3 py-1 rounded-full text-xs font-medium ${
              isRunning ? 'bg-amber-500/20 text-amber-400' :
              isPaused ? 'bg-yellow-500/20 text-yellow-400' :
              isComplete ? 'bg-green-500/20 text-green-400' :
              'bg-slate-600 text-slate-300'
            }`}>
              {isRunning ? 'generating' : isPaused ? 'paused' : isComplete ? 'complete' : 'ready'}
            </span>
          </div>
        </div>
        <div className="h-4 bg-slate-700 rounded-full overflow-hidden">
          <div
            className={`h-full transition-all duration-300 ${
              isRunning ? 'bg-gradient-to-r from-amber-500 to-amber-400' :
              isComplete ? 'bg-gradient-to-r from-green-500 to-green-400' :
              'bg-gradient-to-r from-primary-600 to-primary-400'
            }`}
            style={{ width: `${progressPercent}%` }}
          />
        </div>
        {isRunning && completedCount > 0 && (
          <p className="text-sm text-slate-400 mt-2">
            {Math.round(progressPercent)}% complete
          </p>
        )}
      </div>

      {/* Card thumbnails grid */}
      <div className="grid grid-cols-6 md:grid-cols-8 lg:grid-cols-10 gap-3">
        {set.concepts.map((concept, index) => {
          const isCurrentlyGenerating = isRunning && progress?.current === concept.cardId
          const isGenerated = concept.status === 'generated' || concept.status === 'approved' || concept.status === 'rejected'

          return (
            <div
              key={concept.cardId}
              className={`aspect-[3/4] rounded-lg overflow-hidden relative ${
                isCurrentlyGenerating ? 'ring-2 ring-amber-500 animate-pulse' : ''
              }`}
            >
              {isGenerated ? (
                <img
                  src={api.getCardImageUrl(id, concept.cardId)}
                  alt={concept.title}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className={`w-full h-full flex items-center justify-center ${
                  isCurrentlyGenerating ? 'bg-amber-500/20' : 'bg-slate-700'
                }`}>
                  {isCurrentlyGenerating ? (
                    <div className="animate-spin w-6 h-6 border-2 border-amber-500 border-t-transparent rounded-full"></div>
                  ) : (
                    <span className="text-slate-500 text-xs">{index + 1}</span>
                  )}
                </div>
              )}
              {/* Status indicator */}
              <div className={`absolute top-1 right-1 w-2 h-2 rounded-full ${
                concept.status === 'approved' ? 'bg-green-500' :
                concept.status === 'rejected' ? 'bg-red-500' :
                isGenerated ? 'bg-blue-500' :
                isCurrentlyGenerating ? 'bg-amber-500' :
                'bg-slate-500'
              }`} />
            </div>
          )
        })}
      </div>

      {/* Stats summary */}
      <div className="mt-8 flex gap-6 text-sm">
        <div className="bg-slate-800 rounded-lg px-4 py-3">
          <span className="text-slate-400">Pending: </span>
          <span className="text-white font-semibold">{totalCount - completedCount}</span>
        </div>
        <div className="bg-slate-800 rounded-lg px-4 py-3">
          <span className="text-slate-400">Generated: </span>
          <span className="text-blue-400 font-semibold">{completedCount}</span>
        </div>
        <div className="bg-slate-800 rounded-lg px-4 py-3">
          <span className="text-slate-400">Approved: </span>
          <span className="text-green-400 font-semibold">{set.stats.approved}</span>
        </div>
        <div className="bg-slate-800 rounded-lg px-4 py-3">
          <span className="text-slate-400">Rejected: </span>
          <span className="text-red-400 font-semibold">{set.stats.rejected}</span>
        </div>
      </div>
    </div>
  )
}
