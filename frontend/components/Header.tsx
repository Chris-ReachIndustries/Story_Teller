'use client'

import Link from 'next/link'
import Image from 'next/image'

interface HeaderProps {
  roomCode?: string
  playerName?: string
  isHost?: boolean
  connected?: boolean
}

export default function Header({ roomCode, playerName, isHost, connected = true }: HeaderProps) {
  return (
    <header className="flex justify-between items-center mb-6 animate-fade-in">
      <div className="flex items-center gap-4">
        <Link href="/" className="hover:opacity-80 transition-opacity">
          <Image
            src="/logo.svg"
            alt="Story Teller"
            width={180}
            height={32}
            priority
          />
        </Link>
        {roomCode && (
          <>
            <div className="h-8 w-px bg-slate-600"></div>
            <span className="font-mono text-white text-lg bg-slate-700/50 px-3 py-1 rounded">
              {roomCode}
            </span>
          </>
        )}
      </div>

      <div className="text-right">
        {playerName && (
          <p className="text-gray-400">
            Playing as <span className="text-white font-semibold">{playerName}</span>
            {isHost && <span className="ml-2 text-accent">(Host)</span>}
          </p>
        )}
        <div className={`inline-flex items-center gap-2 mt-1 ${connected ? 'text-green-400' : 'text-red-400'}`}>
          <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-400 animate-pulse-soft' : 'bg-red-400'}`}></span>
          {connected ? 'Connected' : 'Disconnected'}
        </div>
      </div>
    </header>
  )
}
