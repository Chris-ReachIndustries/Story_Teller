package ws

import (
	"dixit-backend/internal/game"
)

// MessageType identifies the type of WebSocket message
type MessageType string

// Client -> Server message types
const (
	MsgHello         MessageType = "hello"
	MsgCreateRoom    MessageType = "create_room"
	MsgJoinRoom      MessageType = "join_room"
	MsgStartGame     MessageType = "start_game"
	MsgStorySubmit   MessageType = "story_submit"
	MsgSubmitCard    MessageType = "submit_card"
	MsgVote          MessageType = "vote"
	MsgPing          MessageType = "ping"
	MsgReconnect     MessageType = "reconnect"
	MsgNextRound     MessageType = "next_round"
	MsgReturnToLobby MessageType = "return_to_lobby"
	MsgUpdateConfig  MessageType = "update_config"
	MsgAddBot        MessageType = "add_bot"
)

// Server -> Client message types
const (
	MsgState MessageType = "state"
	MsgError MessageType = "error"
	MsgPong  MessageType = "pong"
)

// Message is the base structure for all WebSocket messages
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// HelloPayload is the payload for hello messages
type HelloPayload struct {
	Name           string `json:"name"`
	ReconnectToken string `json:"reconnectToken,omitempty"`
	RoomCode       string `json:"roomCode,omitempty"`
}

// CreateRoomPayload is the payload for create_room messages
type CreateRoomPayload struct {
	Config *game.RoomConfig `json:"config,omitempty"`
}

// UpdateConfigPayload is the payload for update_config messages
type UpdateConfigPayload struct {
	Config game.RoomConfig `json:"config"`
}

// JoinRoomPayload is the payload for join_room messages
type JoinRoomPayload struct {
	Code string `json:"code"`
}

// StorySubmitPayload is the payload for story_submit messages
type StorySubmitPayload struct {
	Clue   string `json:"clue"`
	CardID string `json:"cardId"`
}

// SubmitCardPayload is the payload for submit_card messages
type SubmitCardPayload struct {
	CardID string `json:"cardId"`
}

// VotePayload is the payload for vote messages
type VotePayload struct {
	SubmissionIndex int `json:"submissionIndex"`
}

// ReconnectPayload is the payload for reconnect messages
type ReconnectPayload struct {
	RoomCode string `json:"roomCode"`
	PlayerID string `json:"playerId"`
}

// StatePayload is the full game state sent to clients
type StatePayload struct {
	Room                *RoomInfo                 `json:"room,omitempty"`
	You                 *YouInfo                  `json:"you,omitempty"`
	Phase               game.GamePhase            `json:"phase"`
	Players             []PlayerInfo              `json:"players"`
	Hand                []game.Card               `json:"hand,omitempty"`
	Clue                string                    `json:"clue,omitempty"`
	StorytellerID       string                    `json:"storytellerId,omitempty"`
	Submissions         []game.ShuffledSubmission `json:"submissions,omitempty"`
	RoundResults        *game.RoundResult         `json:"roundResults,omitempty"`
	Round               int                       `json:"round"`
	HasSubmitted        bool                      `json:"hasSubmitted,omitempty"`
	HasVoted            bool                      `json:"hasVoted,omitempty"`
	YourSubmissionIndex *int                      `json:"yourSubmissionIndex,omitempty"`
	GameEndReason       game.GameEndReason        `json:"gameEndReason,omitempty"`
	WinnerID            string                    `json:"winnerId,omitempty"`
	WinnerName          string                    `json:"winnerName,omitempty"`
}

// RoomInfo contains room metadata
type RoomInfo struct {
	Code   string          `json:"code"`
	HostID string          `json:"hostId"`
	Config game.RoomConfig `json:"config"`
}

// YouInfo contains the current player's info
type YouInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	IsHost         bool   `json:"isHost"`
	IsStoryteller  bool   `json:"isStoryteller"`
	ReconnectToken string `json:"reconnectToken,omitempty"`
}

// PlayerInfo contains public player information
type PlayerInfo struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	PlayerType   game.PlayerType `json:"playerType"`
	Connected    bool            `json:"connected"`
	Score        int             `json:"score"`
	HasSubmitted bool            `json:"hasSubmitted,omitempty"`
	HasVoted     bool            `json:"hasVoted,omitempty"`
}

// ErrorPayload is the payload for error messages
type ErrorPayload struct {
	Message string `json:"message"`
}
