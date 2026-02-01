// Card set status
export type SetStatus = 'concept' | 'generating' | 'reviewing' | 'exported'

// Individual card/concept status
export type CardStatus = 'pending' | 'generating' | 'generated' | 'approved' | 'rejected'

// A single card concept
export interface Concept {
  cardId: string
  title: string
  prompt: string
  tags: string[]
  status: CardStatus
  imageUrl?: string
  generatedAt?: string
  rejectionReason?: string
}

// Card set statistics
export interface SetStats {
  total: number
  generated: number
  approved: number
  rejected: number
  pending: number
}

// A complete card set
export interface CardSet {
  id: string
  name: string
  theme: string
  createdAt: string
  updatedAt: string
  status: SetStatus
  concepts: Concept[]
  stats: SetStats
}

// Create set request
export interface CreateSetRequest {
  name: string
  theme: string
  cardCount: number
}

// Concept generation request
export interface GenerateConceptsRequest {
  regenerateAll?: boolean
  cardIds?: string[]
}

// Image generation request
export interface GenerateImagesRequest {
  cardIds?: string[]
  settings?: SDSettings
}

// Stable Diffusion settings
export interface SDSettings {
  model?: string
  steps?: number
  cfgScale?: number
  sampler?: string
  width?: number
  height?: number
  negativePrompt?: string
}

// Generation progress
export interface GenerationProgress {
  setId: string
  total: number
  completed: number
  current?: string
  status: 'idle' | 'running' | 'paused' | 'complete' | 'error'
  error?: string
  eta?: string
}

// Export request
export interface ExportRequest {
  outputPath?: string
  includeMetadata?: boolean
}

// Export result
export interface ExportResult {
  success: boolean
  path: string
  cardCount: number
  gitCommands?: string[]
}

// API error response
export interface APIError {
  error: string
  details?: string
}

// List sets response
export interface ListSetsResponse {
  sets: CardSet[]
}

// Health check response
export interface HealthResponse {
  status: string
  services: {
    ollama: boolean
    stableDiffusion: boolean
  }
}
