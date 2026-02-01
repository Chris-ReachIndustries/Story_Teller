'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

export default function GenerateIndexPage() {
  const router = useRouter()

  useEffect(() => {
    router.push('/')
  }, [router])

  return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <div className="text-center">
        <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full mx-auto"></div>
        <p className="text-slate-400 mt-4">Redirecting...</p>
      </div>
    </div>
  )
}
