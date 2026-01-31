package cardgen

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// StateFileName is the name of the generation state file
	StateFileName = ".cardgen.state"
	// DefaultConcurrency is the default number of parallel DALL-E calls
	DefaultConcurrency = 3
)

// Config holds card generation configuration
type Config struct {
	CardsBasePath   string // Base path for all card sets (e.g., /app/cards)
	SetName         string // Name of the card set (e.g., "default", "fantasy", "ocean")
	OpenAIAPIKey    string
	AIEnabled       bool
	ForceRegenerate bool
	Concurrency     int
	DALLEModel      string
	ImageSize       string
	ImageQuality    string
}

// GetSetPath returns the full path for the current card set
func (c *Config) GetSetPath() string {
	if c.SetName == "" {
		return c.CardsBasePath
	}
	return filepath.Join(c.CardsBasePath, c.SetName)
}

// GenerationState tracks partial generation progress
type GenerationState struct {
	Version           int           `json:"version"`
	StartedAt         time.Time     `json:"startedAt"`
	ConceptsGenerated bool          `json:"conceptsGenerated"`
	Concepts          []CardConcept `json:"concepts"`
	ImagesGenerated   []string      `json:"imagesGenerated"` // Card IDs completed
	LastError         string        `json:"lastError,omitempty"`
	LastUpdated       time.Time     `json:"lastUpdated"`
}

// Generator handles automatic card generation
type Generator struct {
	config           Config
	conceptGenerator *ConceptGenerator
	dalleClient      *DALLEClient
	validator        *Validator
	lockFile         *LockFile
}

// NewGenerator creates a new card generator
func NewGenerator(config Config) *Generator {
	if config.Concurrency <= 0 {
		config.Concurrency = DefaultConcurrency
	}
	if config.SetName == "" {
		config.SetName = "default"
	}

	setPath := config.GetSetPath()
	return &Generator{
		config:           config,
		conceptGenerator: NewConceptGenerator(config.OpenAIAPIKey),
		dalleClient:      NewDALLEClient(config.OpenAIAPIKey, config.DALLEModel, config.ImageSize, config.ImageQuality),
		validator:        NewValidator(setPath),
		lockFile:         NewLockFile(setPath),
	}
}

// EnsureCards is the main entry point - called during server startup
func (g *Generator) EnsureCards(ctx context.Context) error {
	setPath := g.config.GetSetPath()
	log.Printf("[CardGen] Checking card set '%s' at %s...", g.config.SetName, setPath)

	// 1. Check if valid deck exists (unless force regenerate)
	if !g.config.ForceRegenerate {
		valid, err := g.validator.IsValidDeck()
		if err != nil {
			return fmt.Errorf("failed to validate deck: %w", err)
		}
		if valid {
			existingCount := g.validator.GetExistingCardCount()
			log.Printf("[CardGen] Valid deck found in set '%s' (%d cards), skipping generation", g.config.SetName, existingCount)
			return nil
		}
		log.Printf("[CardGen] Set '%s' incomplete or missing, generation required", g.config.SetName)
	} else {
		log.Printf("[CardGen] CARDS_REGENERATE=true, forcing regeneration of set '%s'", g.config.SetName)
	}

	// 2. Check if AI is enabled
	if !g.config.AIEnabled {
		return fmt.Errorf("card generation required but AI_ENABLED=false. " +
			"Set AI_ENABLED=true and provide OPENAI_API_KEY, or provide pre-generated cards in %s",
			setPath)
	}

	if g.config.OpenAIAPIKey == "" {
		return fmt.Errorf("card generation required but OPENAI_API_KEY is not set")
	}

	// 3. Ensure directories exist (before acquiring lock)
	imagesPath := filepath.Join(setPath, "images")
	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	// 4. Acquire lock
	log.Println("[CardGen] Acquiring generation lock...")
	if err := g.lockFile.Acquire(); err != nil {
		return fmt.Errorf("another generation may be in progress: %w", err)
	}
	defer g.lockFile.Release()

	// 5. Load or initialize state
	state, err := g.loadOrInitState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// 6. Generate concepts if needed
	if !state.ConceptsGenerated || g.config.ForceRegenerate {
		if g.config.SetName != "default" {
			log.Printf("[CardGen] Generating 100 card concepts via GPT-4o with theme '%s'...", g.config.SetName)
		} else {
			log.Println("[CardGen] Generating 100 card concepts via GPT-4o...")
		}
		concepts, err := g.conceptGenerator.GenerateConcepts(ctx, g.config.SetName)
		if err != nil {
			state.LastError = err.Error()
			g.saveState(state)
			return fmt.Errorf("failed to generate concepts: %w", err)
		}
		state.Concepts = concepts
		state.ConceptsGenerated = true
		state.ImagesGenerated = nil // Reset if regenerating
		if err := g.saveState(state); err != nil {
			return err
		}
		log.Printf("[CardGen] Generated %d concepts successfully", len(concepts))
	} else {
		log.Printf("[CardGen] Resuming with %d existing concepts", len(state.Concepts))
	}

	// 7. Find remaining cards to generate
	remaining := g.findRemainingConcepts(state)
	if len(remaining) > 0 {
		log.Printf("[CardGen] Generating %d images (already have %d)...",
			len(remaining), len(state.ImagesGenerated))
		if err := g.generateImages(ctx, state, remaining); err != nil {
			return fmt.Errorf("failed to generate images: %w", err)
		}
	} else {
		log.Println("[CardGen] All images already generated")
	}

	// 8. Generate cards.json
	log.Println("[CardGen] Writing cards.json...")
	if err := g.generateCardsJSON(state.Concepts); err != nil {
		return fmt.Errorf("failed to generate cards.json: %w", err)
	}

	// 9. Final validation
	valid, err := g.validator.IsValidDeck()
	if err != nil || !valid {
		return fmt.Errorf("final validation failed after generation")
	}

	// 10. Clean up state file (successful completion)
	g.cleanupState()

	log.Printf("[CardGen] Card generation complete! %d cards ready.", TargetCardCount)
	return nil
}

// loadOrInitState loads existing state or creates new
func (g *Generator) loadOrInitState() (*GenerationState, error) {
	statePath := filepath.Join(g.config.GetSetPath(), StateFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize new state
			return &GenerationState{
				Version:         1,
				StartedAt:       time.Now(),
				ImagesGenerated: make([]string, 0),
			}, nil
		}
		return nil, err
	}

	var state GenerationState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupt state, start fresh
		return &GenerationState{
			Version:         1,
			StartedAt:       time.Now(),
			ImagesGenerated: make([]string, 0),
		}, nil
	}

	return &state, nil
}

// saveState saves the current generation state
func (g *Generator) saveState(state *GenerationState) error {
	state.LastUpdated = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	statePath := filepath.Join(g.config.GetSetPath(), StateFileName)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// cleanupState removes the state file after successful completion
func (g *Generator) cleanupState() {
	statePath := filepath.Join(g.config.GetSetPath(), StateFileName)
	os.Remove(statePath)
}

// findRemainingConcepts returns concepts that still need images
func (g *Generator) findRemainingConcepts(state *GenerationState) []CardConcept {
	generated := make(map[string]bool)
	for _, id := range state.ImagesGenerated {
		generated[id] = true
	}

	remaining := make([]CardConcept, 0)
	setPath := g.config.GetSetPath()
	for _, concept := range state.Concepts {
		if !generated[concept.ID] {
			// Also check if file exists on disk
			imagePath := filepath.Join(setPath, "images", concept.ID+".png")
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				remaining = append(remaining, concept)
			}
		}
	}

	return remaining
}

// generateImages generates images with controlled concurrency and retries
func (g *Generator) generateImages(ctx context.Context, state *GenerationState, concepts []CardConcept) error {
	sem := make(chan struct{}, g.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	imagesPath := filepath.Join(g.config.GetSetPath(), "images")
	completed := 0

	for _, concept := range concepts {
		wg.Add(1)
		go func(c CardConcept) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			outputPath := filepath.Join(imagesPath, c.ID+".png")

			// Retry logic with exponential backoff
			var lastErr error
			for attempt := 1; attempt <= 3; attempt++ {
				if attempt > 1 {
					backoff := time.Duration(attempt*attempt) * 5 * time.Second
					log.Printf("[CardGen] Retry %d for %s after %v", attempt, c.ID, backoff)
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						mu.Lock()
						errors = append(errors, ctx.Err())
						mu.Unlock()
						return
					}
				}

				if err := g.dalleClient.GenerateImage(ctx, c, outputPath); err != nil {
					lastErr = err
					log.Printf("[CardGen] Error generating %s (attempt %d): %v", c.ID, attempt, err)
					continue
				}

				// Success
				mu.Lock()
				state.ImagesGenerated = append(state.ImagesGenerated, c.ID)
				completed++
				currentCompleted := completed

				// Save state every 5 cards
				if currentCompleted%5 == 0 {
					g.saveState(state)
					log.Printf("[CardGen] Progress: %d/%d images generated", currentCompleted, len(concepts)+len(state.ImagesGenerated)-len(concepts))
				}
				mu.Unlock()

				log.Printf("[CardGen] Generated %s: %s", c.ID, c.Title)
				return
			}

			mu.Lock()
			errors = append(errors, fmt.Errorf("failed to generate %s after 3 attempts: %w", c.ID, lastErr))
			mu.Unlock()
		}(concept)
	}

	wg.Wait()

	// Save final state
	g.saveState(state)

	if len(errors) > 0 {
		return fmt.Errorf("failed to generate %d images, first error: %v", len(errors), errors[0])
	}

	log.Printf("[CardGen] All %d images generated successfully", len(concepts))
	return nil
}

// generateCardsJSON creates the cards.json file
func (g *Generator) generateCardsJSON(concepts []CardConcept) error {
	cards := make([]CardData, len(concepts))
	// Image path includes the set name for proper routing
	imageBasePath := "/cards/" + g.config.SetName + "/images/"
	for i, c := range concepts {
		cards[i] = CardData{
			ID:    c.ID,
			Title: c.Title,
			Image: imageBasePath + c.ID + ".png",
			Tags:  c.Tags,
		}
	}

	cardsFile := CardsFile{
		Version: 1,
		Cards:   cards,
	}

	data, err := json.MarshalIndent(cardsFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cards.json: %w", err)
	}

	cardsPath := filepath.Join(g.config.GetSetPath(), "cards.json")
	if err := os.WriteFile(cardsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cards.json: %w", err)
	}

	return nil
}

// GetStatus returns the current generation status
func (g *Generator) GetStatus() map[string]interface{} {
	valid, _ := g.validator.IsValidDeck()
	status := map[string]interface{}{
		"deckValid":     valid,
		"targetCards":   TargetCardCount,
		"existingCards": g.validator.GetExistingCardCount(),
		"setName":       g.config.SetName,
	}

	// Check if generation is in progress
	if g.lockFile.IsLocked() {
		statePath := filepath.Join(g.config.GetSetPath(), StateFileName)
		if data, err := os.ReadFile(statePath); err == nil {
			var state GenerationState
			if json.Unmarshal(data, &state) == nil {
				status["generation"] = map[string]interface{}{
					"inProgress":        true,
					"conceptsGenerated": state.ConceptsGenerated,
					"imagesCompleted":   len(state.ImagesGenerated),
					"imagesTotal":       TargetCardCount,
					"startedAt":         state.StartedAt,
					"lastUpdated":       state.LastUpdated,
					"lastError":         state.LastError,
				}
			}
		}
	}

	return status
}
