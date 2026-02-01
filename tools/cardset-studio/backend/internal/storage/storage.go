package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CardStatus represents the approval status of a card
type CardStatus string

const (
	StatusConceptOnly CardStatus = ""          // Concept exists but no image generated yet
	StatusGenerated   CardStatus = "generated" // Image generated, awaiting review
	StatusApproved    CardStatus = "approved"
	StatusRejected    CardStatus = "rejected"
)

// Concept represents a card concept with its generation prompt
type Concept struct {
	CardID string   `json:"cardId"`
	Title  string   `json:"title"`
	Prompt string   `json:"prompt"`
	Tags   []string `json:"tags"`
	Status CardStatus `json:"status"`
}

// SetStats tracks generation and approval progress
type SetStats struct {
	Total     int `json:"total"`
	Generated int `json:"generated"`
	Approved  int `json:"approved"`
	Rejected  int `json:"rejected"`
	Pending   int `json:"pending"`
}

// SetStatus represents the overall status of a card set
type SetStatus string

const (
	SetStatusNew        SetStatus = "new"
	SetStatusConcepts   SetStatus = "concepts"
	SetStatusGenerating SetStatus = "generating"
	SetStatusPaused     SetStatus = "paused"
	SetStatusReviewing  SetStatus = "reviewing"
	SetStatusComplete   SetStatus = "complete"
	SetStatusExported   SetStatus = "exported"
)

// QualityMode represents the image generation quality level
type QualityMode string

const (
	QualityFast   QualityMode = "fast"   // Quick preview (~20s/image)
	QualityNormal QualityMode = "normal" // Balanced (~45s/image)
	QualityHigh   QualityMode = "high"   // Best quality (~90s/image)
)

// CardSet represents a card set being created
type CardSet struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Theme       string      `json:"theme"`
	ThemeStyle  string      `json:"themeStyle,omitempty"`  // Derived art style for visual coherence
	CardCount   int         `json:"cardCount"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Status      SetStatus   `json:"status"`
	QualityMode QualityMode `json:"qualityMode,omitempty"` // Image generation quality level
	Concepts    []Concept   `json:"concepts"`
	Stats       SetStats    `json:"stats"`
	SDSettings  SDSettings  `json:"sdSettings"`
}

// SDSettings contains Stable Diffusion generation settings
type SDSettings struct {
	Model          string  `json:"model"`
	Steps          int     `json:"steps"`
	CFGScale       float64 `json:"cfgScale"`
	Sampler        string  `json:"sampler"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	NegativePrompt string  `json:"negativePrompt"`
}

// DefaultSDSettings returns default high-quality Stable Diffusion settings
func DefaultSDSettings() SDSettings {
	return SDSettings{
		Model:          "juggernautXL_v9Rundiffusion", // SDXL model for high quality
		Steps:          45,                            // More steps for better quality
		CFGScale:       8.0,                           // Slightly higher for better prompt adherence
		Sampler:        "DPM++ 2M Karras",             // Excellent sampler for quality
		Width:          896,                           // SDXL native resolution
		Height:         1152,                          // ~2:3 ratio for Dixit cards
		NegativePrompt: "",                            // Built dynamically by PromptBuilder
	}
}

// GetSettingsForMode returns SD settings adjusted for the specified quality mode
func GetSettingsForMode(mode QualityMode) SDSettings {
	base := DefaultSDSettings()

	switch mode {
	case QualityFast:
		base.Steps = 20
		base.Width = 512
		base.Height = 768
	case QualityNormal:
		base.Steps = 30
		base.Width = 768
		base.Height = 1024
	case QualityHigh:
		// Use defaults (45 steps, 896x1152)
	default:
		// Default to high quality if not specified
	}

	return base
}

// Store manages card set persistence
type Store struct {
	baseDir string
	mu      sync.RWMutex
	sets    map[string]*CardSet
}

// NewStore creates a new storage instance
func NewStore(baseDir string) *Store {
	s := &Store{
		baseDir: baseDir,
		sets:    make(map[string]*CardSet),
	}
	s.loadAllSets()
	return s
}

// loadAllSets loads all existing sets from disk
func (s *Store) loadAllSets() {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			setPath := filepath.Join(s.baseDir, entry.Name(), "state.json")
			if data, err := os.ReadFile(setPath); err == nil {
				var set CardSet
				if err := json.Unmarshal(data, &set); err == nil {
					// Recalculate stats to ensure consistency
					s.UpdateStats(&set)
					s.sets[set.ID] = &set
				}
			}
		}
	}
}

// CreateSet creates a new card set
func (s *Store) CreateSet(name, theme string, cardCount int) (*CardSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()[:8]
	now := time.Now()

	set := &CardSet{
		ID:        id,
		Name:      name,
		Theme:     theme,
		CardCount: cardCount,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    SetStatusNew,
		Concepts:  make([]Concept, 0),
		Stats: SetStats{
			Total: cardCount,
		},
		SDSettings: DefaultSDSettings(),
	}

	// Create directory
	setDir := filepath.Join(s.baseDir, id)
	if err := os.MkdirAll(filepath.Join(setDir, "images"), 0755); err != nil {
		return nil, fmt.Errorf("failed to create set directory: %w", err)
	}

	// Save state
	if err := s.saveSet(set); err != nil {
		return nil, err
	}

	s.sets[id] = set
	return set, nil
}

// GetSet retrieves a card set by ID
func (s *Store) GetSet(id string) (*CardSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, ok := s.sets[id]
	if !ok {
		return nil, fmt.Errorf("set not found: %s", id)
	}
	return set, nil
}

// ListSets returns all card sets
func (s *Store) ListSets() []*CardSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sets := make([]*CardSet, 0, len(s.sets))
	for _, set := range s.sets {
		sets = append(sets, set)
	}
	return sets
}

// UpdateSet updates a card set
func (s *Store) UpdateSet(set *CardSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	set.UpdatedAt = time.Now()
	s.sets[set.ID] = set
	return s.saveSet(set)
}

// DeleteSet removes a card set
func (s *Store) DeleteSet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sets[id]; !ok {
		return fmt.Errorf("set not found: %s", id)
	}

	setDir := filepath.Join(s.baseDir, id)
	if err := os.RemoveAll(setDir); err != nil {
		return fmt.Errorf("failed to delete set directory: %w", err)
	}

	delete(s.sets, id)
	return nil
}

// saveSet writes the set state to disk
func (s *Store) saveSet(set *CardSet) error {
	setDir := filepath.Join(s.baseDir, set.ID)
	statePath := filepath.Join(setDir, "state.json")

	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal set: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// GetImagePath returns the path for a card's image
func (s *Store) GetImagePath(setID, cardID string) string {
	return filepath.Join(s.baseDir, setID, "images", cardID+".png")
}

// GetSetDir returns the directory for a card set
func (s *Store) GetSetDir(setID string) string {
	return filepath.Join(s.baseDir, setID)
}

// UpdateStats recalculates stats from concepts
func (s *Store) UpdateStats(set *CardSet) {
	stats := SetStats{Total: set.CardCount}

	for _, c := range set.Concepts {
		switch c.Status {
		case StatusApproved:
			stats.Approved++
			stats.Generated++
		case StatusRejected:
			stats.Rejected++
			stats.Generated++
		case StatusGenerated:
			stats.Pending++ // "Pending" in stats means "pending review"
			stats.Generated++
		case StatusConceptOnly:
			// Concept only, no image yet - don't count as generated
		}
	}

	set.Stats = stats
}
