package ai

// CardWithThumb represents a card with its base64 thumbnail for AI vision
type CardWithThumb struct {
	CardID  string
	DataURL string // base64 data URL (data:image/png;base64,...)
}

// StorytellerResponse is the AI response for storyteller actions
type StorytellerResponse struct {
	SelectedCard int    `json:"selectedCard"` // 1-indexed card number
	Clue         string `json:"clue"`
}

// SubmitResponse is the AI response for card submission
type SubmitResponse struct {
	SelectedCard int `json:"selectedCard"` // 1-indexed card number
}

// VoteResponse is the AI response for voting
type VoteResponse struct {
	SelectedCard int `json:"selectedCard"` // 1-indexed card number
}

// Client defines the interface for AI providers
type Client interface {
	// Storytell selects a card and generates a clue as the storyteller
	Storytell(hand []CardWithThumb) (*StorytellerResponse, error)

	// Submit selects a card that matches the given clue
	Submit(hand []CardWithThumb, clue string) (*SubmitResponse, error)

	// Vote selects which card is the storyteller's card
	// ownIndex is the 1-indexed position of the bot's own card (which cannot be voted for)
	Vote(submissions []CardWithThumb, clue string, ownIndex int) (*VoteResponse, error)
}

// Pinger is an optional interface for provider health checks
type Pinger interface {
	Ping() error
}
