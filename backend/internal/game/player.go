package game

import (
	"crypto/rand"
	"encoding/base64"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// PlayerType distinguishes between human and bot players
type PlayerType string

const (
	PlayerTypeHuman PlayerType = "human"
	PlayerTypeBot   PlayerType = "bot"
)

// Bot name pool for generating unique bot names
var botNames = []string{"Aurora", "Atlas", "Nova", "Echo", "Sage", "Luna", "Orion", "Iris"}
var botNameCounter uint64

// Player represents a connected player
type Player struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	PlayerType     PlayerType `json:"playerType"`
	Connected      bool       `json:"connected"`
	Score          int        `json:"score"`
	Hand           []Card     `json:"hand,omitempty"`
	SubmittedCard  *Card      `json:"-"` // Card submitted this round
	VotedFor       int        `json:"-"` // Index voted for (-1 = not voted)
	ReconnectToken string     `json:"-"` // Token for reconnecting
	LastSeen       time.Time  `json:"-"` // Last time player was seen connected
}

// NewPlayer creates a new human player with a unique ID and reconnect token
func NewPlayer(name string) *Player {
	return &Player{
		ID:             uuid.New().String(),
		Name:           name,
		PlayerType:     PlayerTypeHuman,
		Connected:      true,
		Score:          0,
		Hand:           make([]Card, 0),
		VotedFor:       -1,
		ReconnectToken: generateSecureToken(),
		LastSeen:       time.Now(),
	}
}

// NewBotPlayer creates a new AI bot player with a generated name
func NewBotPlayer() *Player {
	counter := atomic.AddUint64(&botNameCounter, 1) - 1
	name := "Bot " + botNames[counter%uint64(len(botNames))]
	return &Player{
		ID:         uuid.New().String(),
		Name:       name,
		PlayerType: PlayerTypeBot,
		Connected:  true, // Bots are always "connected"
		Score:      0,
		Hand:       make([]Card, 0),
		VotedFor:   -1,
		// No ReconnectToken needed for bots
		LastSeen: time.Now(),
	}
}

// IsBot returns true if the player is an AI bot
func (p *Player) IsBot() bool {
	return p.PlayerType == PlayerTypeBot
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to UUID if crypto/rand fails
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(b)
}

// UpdateLastSeen updates the player's last seen timestamp
func (p *Player) UpdateLastSeen() {
	p.LastSeen = time.Now()
}

// Card represents a game card
type Card struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

// RemoveCardFromHand removes a card from the player's hand by ID
func (p *Player) RemoveCardFromHand(cardID string) *Card {
	for i, card := range p.Hand {
		if card.ID == cardID {
			removed := p.Hand[i]
			p.Hand = append(p.Hand[:i], p.Hand[i+1:]...)
			return &removed
		}
	}
	return nil
}

// HasCard checks if the player has a specific card in their hand
func (p *Player) HasCard(cardID string) bool {
	for _, card := range p.Hand {
		if card.ID == cardID {
			return true
		}
	}
	return false
}
