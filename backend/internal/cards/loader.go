package cards

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"dixit-backend/internal/game"
)

// CardsFile represents the structure of cards.json
type CardsFile struct {
	Version int        `json:"version"`
	Cards   []CardData `json:"cards"`
}

// CardData represents a card in the JSON file
type CardData struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Image string   `json:"image"`
	Tags  []string `json:"tags"`
}

// Loader loads and caches cards from JSON
type Loader struct {
	mu          sync.RWMutex
	path        string
	cards       []game.Card
	lastLoaded  time.Time
	cacheTTL    time.Duration
}

// NewLoader creates a new card loader
func NewLoader(path string) *Loader {
	return &Loader{
		path:     path,
		cacheTTL: 5 * time.Second, // Cache for 5 seconds
	}
}

// GetCards returns all cards, reloading from disk if cache expired
func (l *Loader) GetCards() ([]game.Card, error) {
	l.mu.RLock()
	if time.Since(l.lastLoaded) < l.cacheTTL && len(l.cards) > 0 {
		cards := make([]game.Card, len(l.cards))
		copy(cards, l.cards)
		l.mu.RUnlock()
		return cards, nil
	}
	l.mu.RUnlock()

	// Need to reload
	return l.reload()
}

// reload loads cards from disk
func (l *Loader) reload() ([]game.Card, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check in case another goroutine already reloaded
	if time.Since(l.lastLoaded) < l.cacheTTL && len(l.cards) > 0 {
		cards := make([]game.Card, len(l.cards))
		copy(cards, l.cards)
		return cards, nil
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return nil, err
	}

	l.cards = make([]game.Card, len(cardsFile.Cards))
	for i, c := range cardsFile.Cards {
		l.cards[i] = game.Card{
			ID:    c.ID,
			Image: c.Image,
		}
	}

	l.lastLoaded = time.Now()

	cards := make([]game.Card, len(l.cards))
	copy(cards, l.cards)
	return cards, nil
}

// GetCardsData returns full card data including titles and tags
func (l *Loader) GetCardsData() ([]CardData, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return nil, err
	}

	return cardsFile.Cards, nil
}
