'use client'

import { useState, useEffect, use } from 'react'
import { useRouter } from 'next/navigation'
import api from '@/lib/api'
import { CardSet, ExportResult } from '@/lib/types'

interface PageProps {
  params: Promise<{ id: string }>
}

export default function ExportPage({ params }: PageProps) {
  const { id } = use(params)
  const router = useRouter()
  const [set, setSet] = useState<CardSet | null>(null)
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [exportResult, setExportResult] = useState<ExportResult | null>(null)

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

  async function handleExport() {
    setExporting(true)
    try {
      const result = await api.exportSet(id)
      setExportResult(result)
    } catch (error) {
      console.error('Failed to export:', error)
      alert('Failed to export. Please try again.')
    } finally {
      setExporting(false)
    }
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

  const pendingCount = set.stats.total - set.stats.approved
  const canExport = set.stats.approved > 0

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-white">{set.name}</h1>
        <p className="text-slate-400 mt-1">Export to Dixit cards directory</p>
      </div>

      {/* Summary card */}
      <div className="bg-slate-800 rounded-xl p-6 mb-8">
        <h2 className="text-lg font-semibold text-white mb-4">Export Summary</h2>
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-slate-700 rounded-lg p-4">
            <div className="text-3xl font-bold text-green-400">{set.stats.approved}</div>
            <div className="text-sm text-slate-400">Cards to export</div>
          </div>
          <div className="bg-slate-700 rounded-lg p-4">
            <div className="text-3xl font-bold text-yellow-400">{pendingCount}</div>
            <div className="text-sm text-slate-400">Not approved</div>
          </div>
        </div>

        {pendingCount > 0 && (
          <div className="mt-4 p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
            <p className="text-yellow-400 text-sm">
              <strong>Warning:</strong> {pendingCount} cards are not approved and will not be included in the export.
              <button
                onClick={() => router.push(`/review/${id}`)}
                className="ml-2 underline hover:no-underline"
              >
                Review cards
              </button>
            </p>
          </div>
        )}
      </div>

      {/* Export destination */}
      <div className="bg-slate-800 rounded-xl p-6 mb-8">
        <h2 className="text-lg font-semibold text-white mb-4">Export Destination</h2>
        <div className="bg-slate-900 rounded-lg p-4 font-mono text-sm text-slate-300">
          /app/dixit-cards/{set.id}/
        </div>
        <p className="text-sm text-slate-400 mt-2">
          This will create the card set in the main Dixit repository&apos;s cards directory.
        </p>
      </div>

      {/* Export button */}
      {!exportResult && (
        <button
          onClick={handleExport}
          disabled={!canExport || exporting}
          className="w-full px-6 py-4 bg-primary-600 hover:bg-primary-700 text-white rounded-xl font-semibold text-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {exporting ? (
            <span className="flex items-center justify-center gap-3">
              <div className="animate-spin w-5 h-5 border-2 border-white border-t-transparent rounded-full"></div>
              Exporting...
            </span>
          ) : (
            `Export ${set.stats.approved} Cards`
          )}
        </button>
      )}

      {/* Export result */}
      {exportResult && (
        <div className="bg-green-500/10 border border-green-500/30 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="text-3xl">✅</div>
            <div>
              <h2 className="text-xl font-bold text-green-400">Export Complete!</h2>
              <p className="text-slate-400">
                {exportResult.cardCount} cards exported to {exportResult.path}
              </p>
            </div>
          </div>

          {exportResult.gitCommands && exportResult.gitCommands.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-semibold text-white mb-3">
                Git Commands to commit your new card set:
              </h3>
              <div className="bg-slate-900 rounded-lg p-4 font-mono text-sm space-y-2">
                {exportResult.gitCommands.map((cmd, i) => (
                  <div key={i} className="text-slate-300">
                    <span className="text-slate-500">$ </span>
                    {cmd}
                  </div>
                ))}
              </div>
              <p className="text-xs text-slate-500 mt-3">
                Run these commands in the Dixit repository root to add the card set to Git LFS.
              </p>
            </div>
          )}

          <div className="mt-6 flex gap-3">
            <button
              onClick={() => router.push('/')}
              className="flex-1 px-4 py-2 bg-slate-600 hover:bg-slate-500 text-white rounded-lg transition-colors"
            >
              Back to Home
            </button>
            <button
              onClick={() => {
                setExportResult(null)
                handleExport()
              }}
              className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
            >
              Re-export
            </button>
          </div>
        </div>
      )}

      {/* Help section */}
      <div className="mt-8 p-6 bg-slate-800/50 rounded-xl">
        <h3 className="font-semibold text-white mb-3">About Git LFS</h3>
        <p className="text-sm text-slate-400 mb-4">
          Card images are stored using Git Large File Storage (LFS) to keep the repository size manageable.
          The export will create all necessary files, but you&apos;ll need to run the git commands manually
          to commit them.
        </p>
        <div className="text-sm text-slate-400">
          <p className="mb-2">Requirements:</p>
          <ul className="list-disc list-inside space-y-1 text-slate-500">
            <li>Git LFS installed (<code className="text-slate-400">git lfs install</code>)</li>
            <li>Write access to the Dixit repository</li>
            <li>Sufficient disk space for images</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
