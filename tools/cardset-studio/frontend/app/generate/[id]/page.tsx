'use client'

import { useState, useEffect, useRef, use } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import { CardSet, GenerationProgress } from '@/lib/types'

interface PageProps {
  params: Promise<{ id: string }>
}

export default function GeneratePage({ params }: PageProps) {
  const { id } = use(params)
  const router = useRouter()
  const [set, setSet] = useState<CardSet | null>(null)
  const [progress, setProgress] = useState<GenerationProgress | null>(null)
  const [loading, setLoading] = useState(true)
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

        if (progressData.status === 'complete' || progressData.status === 'error') {
          if (pollRef.current) {
            clearInterval(pollRef.current)
            pollRef.current = null
          }
        }
      } catch (error) {
        console.error('Polling error:', error)
      }
    }, 2000)
  }

  async function handleStartGeneration() {
    try {
      const progressData = await api.startGeneration(id)
      setProgress(progressData)
      startPolling()
    } catch (error) {
      console.error('Failed to start generation:', error)
      alert('Failed to start generation. Please try again.')
    }
  }

  async function handlePause() {
    try {
      const progressData = await api.pauseGeneration(id)
      setProgress(progressData)
      if (pollRef.current) {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
    } catch (error) {
      console.error('Failed to pause:', error)
    }
  }

  async function handleResume() {
    try {
      const progressData = await api.resumeGeneration(id)
      setProgress(progressData)
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
  const progressPercent = progress ? (progress.completed / progress.total) * 100 : 0

  return (
    <div className="max-w-6xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-white">{set.name}</h1>
          <p className="text-slate-400 mt-1">Image Generation</p>
        </div>
        <div className="flex gap-3">
          {!isRunning && !isComplete && (
            <button
              onClick={handleStartGeneration}
              className="px-6 py-3 bg-accent hover:bg-accent-dark text-black rounded-lg font-semibold transition-colors"
            >
              {isPaused ? 'Resume' : 'Start Generation'}
            </button>
          )}
          {isRunning && (
            <button
              onClick={handlePause}
              className="px-6 py-3 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg font-semibold transition-colors"
            >
              Pause
            </button>
          )}
          {isPaused && (
            <button
              onClick={handleResume}
              className="px-6 py-3 bg-green-600 hover:bg-green-700 text-white rounded-lg font-semibold transition-colors"
            >
              Resume
            </button>
          )}
          {(isComplete || set.stats.generated > 0) && (
            <button
              onClick={handleProceedToReview}
              className="px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-semibold transition-colors"
            >
              Review Cards →
            </button>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="bg-slate-800 rounded-xl p-6 mb-8">
        <div className="flex items-center justify-between mb-4">
          <div>
            <span className="text-2xl font-bold text-white">
              {progress?.completed || set.stats.generated} / {progress?.total || set.stats.total}
            </span>
            <span className="text-slate-400 ml-2">cards generated</span>
          </div>
          <div className="flex items-center gap-4">
            {isRunning && progress?.current && (
              <span className="text-sm text-slate-400">
                Currently: {progress.current}
              </span>
            )}
            <span className={`px-3 py-1 rounded-full text-xs font-medium ${
              isRunning ? 'status-generating' :
              isPaused ? 'status-pending' :
              isComplete ? 'status-approved' :
              'bg-slate-600 text-slate-300'
            }`}>
              {progress?.status || 'idle'}
            </span>
          </div>
        </div>
        <div className="h-4 bg-slate-700 rounded-full overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-primary-600 to-primary-400 transition-all duration-500"
            style={{ width: `${progressPercent}%` }}
          />
        </div>
        {progress?.eta && (
          <p className="text-sm text-slate-400 mt-2">
            Estimated time remaining: {progress.eta}
          </p>
        )}
      </div>

      {/* Card thumbnails grid */}
      <div className="grid grid-cols-6 md:grid-cols-8 lg:grid-cols-10 gap-3">
        {set.concepts.map((concept, index) => {
          const isGenerating = isRunning && progress?.current === concept.cardId
          const isGenerated = concept.status === 'generated' || concept.status === 'approved' || concept.status === 'rejected'

          return (
            <div
              key={concept.cardId}
              className={`aspect-[3/4] rounded-lg overflow-hidden relative ${
                isGenerating ? 'ring-2 ring-accent animate-pulse' : ''
              }`}
            >
              {isGenerated && concept.imageUrl ? (
                <img
                  src={api.getCardImageUrl(id, concept.cardId)}
                  alt={concept.title}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className={`w-full h-full flex items-center justify-center ${
                  isGenerating ? 'bg-accent/20' : 'bg-slate-700'
                }`}>
                  {isGenerating ? (
                    <div className="animate-spin w-6 h-6 border-2 border-accent border-t-transparent rounded-full"></div>
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
          <span className="text-white font-semibold">{set.stats.pending}</span>
        </div>
        <div className="bg-slate-800 rounded-lg px-4 py-3">
          <span className="text-slate-400">Generated: </span>
          <span className="text-blue-400 font-semibold">{set.stats.generated}</span>
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
