'use client'

import { useEffect, useRef, useCallback, useState } from 'react'
import { useGame } from './gameContext'

const FADE_DURATION = 2000 // 2 seconds
const MUSIC_VOLUME = 0.3 // 30% volume

export function useAmbientMusic() {
  const { state } = useGame()
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const fadeIntervalRef = useRef<NodeJS.Timeout | null>(null)
  const [isMuted, setIsMuted] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('ambientMuted') === 'true'
    }
    return false
  })

  // Initialize audio element
  useEffect(() => {
    const audio = new Audio('/audio/ambient-lofi.mp3')
    audio.loop = true
    audio.volume = 0
    audioRef.current = audio

    return () => {
      audio.pause()
      audio.src = ''
    }
  }, [])

  // Determine if player is waiting
  const isWaiting = useCallback(() => {
    if (!state) return false

    const { phase, you, hasSubmitted, hasVoted } = state

    switch (phase) {
      case 'STORYTELLER_CLUE':
        // Non-storytellers wait for the storyteller to pick a card and clue
        return !you?.isStoryteller
      case 'SUBMISSIONS':
        // Players wait after they've submitted their card
        return !!hasSubmitted
      case 'VOTING':
        // Players wait after voting, or storyteller can't vote at all
        return !!hasVoted || !!you?.isStoryteller
      case 'ROUND_END':
      case 'GAME_END':
      case 'SCORING':
        // Everyone is just viewing results
        return true
      default:
        return false
    }
  }, [state])

  // Fade audio in/out
  const fadeAudio = useCallback(
    (fadeIn: boolean) => {
      const audio = audioRef.current
      if (!audio || isMuted) return

      // Clear existing fade
      if (fadeIntervalRef.current) {
        clearInterval(fadeIntervalRef.current)
      }

      const targetVolume = fadeIn ? MUSIC_VOLUME : 0
      const stepCount = FADE_DURATION / 50
      const step = (MUSIC_VOLUME / stepCount) * (fadeIn ? 1 : -1)

      // Start playing if fading in
      if (fadeIn && audio.paused) {
        audio.play().catch(() => {
          // Autoplay blocked - will work after user interaction
        })
      }

      fadeIntervalRef.current = setInterval(() => {
        const newVolume = audio.volume + step

        if ((fadeIn && newVolume >= targetVolume) || (!fadeIn && newVolume <= 0)) {
          audio.volume = Math.max(0, Math.min(targetVolume, newVolume))
          if (fadeIntervalRef.current) {
            clearInterval(fadeIntervalRef.current)
            fadeIntervalRef.current = null
          }

          // Pause when fully faded out to save resources
          if (!fadeIn && audio.volume === 0) {
            audio.pause()
          }
        } else {
          audio.volume = Math.max(0, Math.min(MUSIC_VOLUME, newVolume))
        }
      }, 50)
    },
    [isMuted]
  )

  // React to waiting state changes
  useEffect(() => {
    fadeAudio(isWaiting())
  }, [isWaiting, fadeAudio])

  // Stop music when muted
  useEffect(() => {
    if (isMuted && audioRef.current) {
      if (fadeIntervalRef.current) {
        clearInterval(fadeIntervalRef.current)
        fadeIntervalRef.current = null
      }
      audioRef.current.pause()
      audioRef.current.volume = 0
    }
  }, [isMuted])

  // Handle mute toggle
  const toggleMute = useCallback(() => {
    setIsMuted((prev) => {
      const newValue = !prev
      localStorage.setItem('ambientMuted', String(newValue))
      return newValue
    })
  }, [])

  return { isMuted, toggleMute }
}
