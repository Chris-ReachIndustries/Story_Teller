'use client'

interface AudioToggleProps {
  isMuted: boolean
  onToggle: () => void
}

export function AudioToggle({ isMuted, onToggle }: AudioToggleProps) {
  return (
    <button
      onClick={onToggle}
      className="p-2 rounded-lg bg-white/10 hover:bg-white/20 transition-colors"
      title={isMuted ? 'Unmute ambient music' : 'Mute ambient music'}
      aria-label={isMuted ? 'Unmute ambient music' : 'Mute ambient music'}
    >
      {isMuted ? (
        // Muted icon (speaker with X)
        <svg
          className="w-5 h-5 text-white/60"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z"
          />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2"
          />
        </svg>
      ) : (
        // Unmuted icon (speaker with sound waves)
        <svg
          className="w-5 h-5 text-white"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z"
          />
        </svg>
      )}
    </button>
  )
}
