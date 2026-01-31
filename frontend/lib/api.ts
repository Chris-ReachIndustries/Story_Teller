import { CardSetInfo, CardData } from './types'

// Get base URL for API requests (handles both dev and prod)
function getBaseUrl(): string {
  if (typeof window !== 'undefined') {
    return '' // Same origin in browser
  }
  return process.env.API_URL || 'http://localhost:8080'
}

// Fetch all available card sets
export async function fetchCardSets(): Promise<CardSetInfo[]> {
  const response = await fetch(`${getBaseUrl()}/api/card-sets`)
  if (!response.ok) {
    throw new Error(`Failed to fetch card sets: ${response.statusText}`)
  }
  const data = await response.json()
  return data.sets || []
}

// Fetch all cards for a specific set
export async function fetchCardSetCards(setId: string): Promise<CardData[]> {
  const response = await fetch(`${getBaseUrl()}/api/card-sets/${encodeURIComponent(setId)}/cards`)
  if (!response.ok) {
    throw new Error(`Failed to fetch cards for set ${setId}: ${response.statusText}`)
  }
  const data = await response.json()
  return data.cards || []
}
