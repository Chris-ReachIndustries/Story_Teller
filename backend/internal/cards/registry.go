package cards

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dixit-backend/internal/game"
)

const (
	// RegistryCacheTTL is how long to cache the list of card sets
	RegistryCacheTTL = 30 * time.Second
	// PreviewCount is the number of preview images to include per set
	PreviewCount = 4
)

// CardSetInfo represents metadata about a card set
type CardSetInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	CardCount   int      `json:"cardCount"`
	PreviewURLs []string `json:"previewUrls"`
}

// Registry manages discovery and loading of multiple card sets
type Registry struct {
	basePath string
	mu       sync.RWMutex
	cache    []CardSetInfo
	cacheExp time.Time
}

// NewRegistry creates a new card set registry
func NewRegistry(basePath string) *Registry {
	return &Registry{
		basePath: basePath,
	}
}

// ListSets returns all available card sets, scanning the filesystem
func (r *Registry) ListSets() ([]CardSetInfo, error) {
	r.mu.RLock()
	if time.Now().Before(r.cacheExp) && r.cache != nil {
		sets := make([]CardSetInfo, len(r.cache))
		copy(sets, r.cache)
		r.mu.RUnlock()
		return sets, nil
	}
	r.mu.RUnlock()

	// Need to scan
	return r.scanSets()
}

// scanSets scans the filesystem for card sets
func (r *Registry) scanSets() ([]CardSetInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check cache after acquiring write lock
	if time.Now().Before(r.cacheExp) && r.cache != nil {
		sets := make([]CardSetInfo, len(r.cache))
		copy(sets, r.cache)
		return sets, nil
	}

	entries, err := os.ReadDir(r.basePath)
	if err != nil {
		return nil, err
	}

	var sets []CardSetInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		setID := entry.Name()
		// Skip hidden directories
		if strings.HasPrefix(setID, ".") {
			continue
		}

		cardsPath := filepath.Join(r.basePath, setID, "cards.json")
		if _, err := os.Stat(cardsPath); os.IsNotExist(err) {
			continue // No cards.json, skip
		}

		info, err := r.loadSetInfo(setID)
		if err != nil {
			continue // Skip invalid sets
		}

		sets = append(sets, *info)
	}

	r.cache = sets
	r.cacheExp = time.Now().Add(RegistryCacheTTL)

	result := make([]CardSetInfo, len(sets))
	copy(result, sets)
	return result, nil
}

// loadSetInfo loads metadata for a single card set
func (r *Registry) loadSetInfo(setID string) (*CardSetInfo, error) {
	cardsPath := filepath.Join(r.basePath, setID, "cards.json")

	data, err := os.ReadFile(cardsPath)
	if err != nil {
		return nil, err
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return nil, err
	}

	// Generate preview URLs (first N cards)
	previewURLs := make([]string, 0, PreviewCount)
	for i := 0; i < len(cardsFile.Cards) && i < PreviewCount; i++ {
		previewURLs = append(previewURLs, cardsFile.Cards[i].Image)
	}

	// Generate display name from ID (title case)
	name := strings.Title(strings.ReplaceAll(setID, "-", " "))

	return &CardSetInfo{
		ID:          setID,
		Name:        name,
		CardCount:   len(cardsFile.Cards),
		PreviewURLs: previewURLs,
	}, nil
}

// SetExists checks if a card set exists
func (r *Registry) SetExists(setID string) bool {
	cardsPath := filepath.Join(r.basePath, setID, "cards.json")
	_, err := os.Stat(cardsPath)
	return err == nil
}

// GetSetCards loads all cards from a specific set
func (r *Registry) GetSetCards(setID string) ([]game.Card, error) {
	cardsPath := filepath.Join(r.basePath, setID, "cards.json")

	data, err := os.ReadFile(cardsPath)
	if err != nil {
		return nil, err
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return nil, err
	}

	cards := make([]game.Card, len(cardsFile.Cards))
	for i, c := range cardsFile.Cards {
		cards[i] = game.Card{
			ID:    c.ID,
			Image: c.Image,
		}
	}

	return cards, nil
}

// GetSetCardsData loads full card data (with titles and tags) from a specific set
func (r *Registry) GetSetCardsData(setID string) ([]CardData, error) {
	cardsPath := filepath.Join(r.basePath, setID, "cards.json")

	data, err := os.ReadFile(cardsPath)
	if err != nil {
		return nil, err
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return nil, err
	}

	return cardsFile.Cards, nil
}

// GetSetInfo returns info for a specific set
func (r *Registry) GetSetInfo(setID string) (*CardSetInfo, error) {
	if !r.SetExists(setID) {
		return nil, os.ErrNotExist
	}
	return r.loadSetInfo(setID)
}

// ClearCache clears the registry cache
func (r *Registry) ClearCache() {
	r.mu.Lock()
	r.cache = nil
	r.cacheExp = time.Time{}
	r.mu.Unlock()
}
