// Game phases matching backend
export type GamePhase =
  | 'LOBBY'
  | 'STORYTELLER_CLUE'
  | 'SUBMISSIONS'
  | 'VOTING'
  | 'SCORING'
  | 'ROUND_END'
  | 'GAME_END'

// Card type
export interface Card {
  id: string
  image: string
}

// Card set info from API (for browsing available sets)
export interface CardSetInfo {
  id: string
  name: string
  cardCount: number
  previewUrls: string[]
}

// Full card data for gallery view
export interface CardData {
  id: string
  title: string
  image: string
  tags: string[]
}

// Player type (human or bot)
export type PlayerType = 'human' | 'bot'

// Player info for public display
export interface PlayerInfo {
  id: string
  name: string
  playerType: PlayerType
  connected: boolean
  score: number
  hasSubmitted?: boolean
  hasVoted?: boolean
}

// Room configuration
export interface RoomConfig {
  scoreToWin: number
  handSize: number
  startingStoryteller: string // "host", "random", or a player ID
  deckSetId: string // Card set to use (e.g., "default", "fantasy")
}

// Room info
export interface RoomInfo {
  code: string
  hostId: string
  config: RoomConfig
}

// Current player info
export interface YouInfo {
  id: string
  name: string
  isHost: boolean
  isStoryteller: boolean
  reconnectToken?: string
}

// Submission for display
export interface ShuffledSubmission {
  index: number
  image: string
}

// Round results
export interface RoundResult {
  storytellerCard: number
  votes: Record<string, number>
  pointsThisRound: Record<string, number>
}

// Game end reasons
export type GameEndReason = '' | 'score_reached' | 'out_of_cards' | 'player_disconnected'

// Full game state from server
export interface GameState {
  room?: RoomInfo
  you?: YouInfo
  phase: GamePhase
  players: PlayerInfo[]
  hand?: Card[]
  clue?: string
  storytellerId?: string
  submissions?: ShuffledSubmission[]
  roundResults?: RoundResult
  round: number
  hasSubmitted?: boolean
  hasVoted?: boolean
  yourSubmissionIndex?: number
  gameEndReason?: GameEndReason
  winnerId?: string
  winnerName?: string
}

// Message types
export type ClientMessageType =
  | 'hello'
  | 'create_room'
  | 'join_room'
  | 'start_game'
  | 'story_submit'
  | 'submit_card'
  | 'vote'
  | 'ping'
  | 'reconnect'
  | 'next_round'
  | 'return_to_lobby'
  | 'update_config'
  | 'add_bot'

export type ServerMessageType = 'state' | 'error' | 'pong'

export interface ClientMessage {
  type: ClientMessageType
  payload?: unknown
}

export interface ServerMessage {
  type: ServerMessageType
  payload: unknown
}

export interface ErrorPayload {
  message: string
}
