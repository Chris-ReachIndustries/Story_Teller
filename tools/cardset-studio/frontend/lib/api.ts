import {
  CardSet,
  CreateSetRequest,
  GenerateConceptsRequest,
  GenerateImagesRequest,
  ExportRequest,
  ExportResult,
  GenerationProgress,
  HealthResponse,
  ListSetsResponse,
  Concept,
  QualityMode,
} from './types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api'

class APIClient {
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${API_BASE}${endpoint}`
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new Error(error.error || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // Health check
  async health(): Promise<HealthResponse> {
    return this.request('/health')
  }

  // List all sets
  async listSets(): Promise<ListSetsResponse> {
    return this.request('/sets')
  }

  // Get single set
  async getSet(id: string): Promise<CardSet> {
    return this.request(`/sets/${id}`)
  }

  // Create new set
  async createSet(data: CreateSetRequest): Promise<CardSet> {
    return this.request('/sets', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // Delete set
  async deleteSet(id: string): Promise<void> {
    await this.request(`/sets/${id}`, {
      method: 'DELETE',
    })
  }

  // Generate concepts
  async generateConcepts(
    setId: string,
    data: GenerateConceptsRequest = {}
  ): Promise<CardSet> {
    return this.request(`/sets/${setId}/concepts`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // Update single concept
  async updateConcept(
    setId: string,
    cardId: string,
    data: Partial<Concept>
  ): Promise<Concept> {
    return this.request(`/sets/${setId}/concepts/${cardId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  // Regenerate concept text (title/prompt) using AI
  async regenerateConceptText(
    setId: string,
    cardId: string,
    options: { regenerateTitle?: boolean; regeneratePrompt?: boolean }
  ): Promise<Concept> {
    return this.request(`/sets/${setId}/concepts/${cardId}/regenerate`, {
      method: 'POST',
      body: JSON.stringify(options),
    })
  }

  // Start image generation
  async startGeneration(
    setId: string,
    qualityMode: QualityMode = 'high',
    data: GenerateImagesRequest = {}
  ): Promise<GenerationProgress> {
    return this.request(`/sets/${setId}/generate`, {
      method: 'POST',
      body: JSON.stringify({ ...data, qualityMode }),
    })
  }

  // Pause generation
  async pauseGeneration(setId: string): Promise<GenerationProgress> {
    return this.request(`/sets/${setId}/generate/pause`, {
      method: 'POST',
    })
  }

  // Cancel generation
  async cancelGeneration(setId: string): Promise<{ status: string }> {
    return this.request(`/sets/${setId}/generate/cancel`, {
      method: 'POST',
    })
  }

  // Resume generation
  async resumeGeneration(setId: string): Promise<GenerationProgress> {
    return this.request(`/sets/${setId}/generate/resume`, {
      method: 'POST',
    })
  }

  // Get generation progress
  async getProgress(setId: string): Promise<GenerationProgress> {
    return this.request(`/sets/${setId}/generate/progress`)
  }

  // Approve card
  async approveCard(setId: string, cardId: string): Promise<Concept> {
    return this.request(`/sets/${setId}/cards/${cardId}/approve`, {
      method: 'POST',
    })
  }

  // Reject card
  async rejectCard(
    setId: string,
    cardId: string,
    reason?: string
  ): Promise<Concept> {
    return this.request(`/sets/${setId}/cards/${cardId}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
  }

  // Regenerate card
  async regenerateCard(
    setId: string,
    cardId: string,
    newPrompt?: string
  ): Promise<Concept> {
    return this.request(`/sets/${setId}/cards/${cardId}/regenerate`, {
      method: 'POST',
      body: JSON.stringify({ prompt: newPrompt }),
    })
  }

  // Export set
  async exportSet(setId: string, data: ExportRequest = {}): Promise<ExportResult> {
    return this.request(`/sets/${setId}/export`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // Get card image URL
  getCardImageUrl(setId: string, cardId: string): string {
    return `${API_BASE}/sets/${setId}/cards/${cardId}/image`
  }
}

export const api = new APIClient()
export default api
