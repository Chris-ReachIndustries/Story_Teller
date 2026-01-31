package cardgen

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// TargetCardCount is the number of cards to generate
	TargetCardCount = 100
	// MinValidCardCount is the minimum cards for an existing deck to be considered valid
	MinValidCardCount = 20
)

// Validator checks deck completeness
type Validator struct {
	cardsPath   string
	targetCount int
}

// NewValidator creates a deck validator
func NewValidator(cardsPath string) *Validator {
	return &Validator{
		cardsPath:   cardsPath,
		targetCount: TargetCardCount,
	}
}

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

// IsValidDeck checks if a valid deck exists
// A valid deck has cards.json with at least MinValidCardCount cards and all images present
func (v *Validator) IsValidDeck() (bool, error) {
	// 1. Check cards.json exists and is valid
	cardsJSONPath := filepath.Join(v.cardsPath, "cards.json")
	data, err := os.ReadFile(cardsJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File doesn't exist - not an error, just not valid
		}
		return false, err
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return false, nil // Invalid JSON format - treat as invalid deck
	}

	// Need at least some cards to be valid
	if len(cardsFile.Cards) < MinValidCardCount {
		return false, nil // Not enough cards
	}

	// 2. Check all listed images exist
	imagesPath := filepath.Join(v.cardsPath, "images")
	for _, card := range cardsFile.Cards {
		// Extract filename from image path (e.g., "/cards/images/card-001.png" -> "card-001.png")
		imageName := filepath.Base(card.Image)
		imagePath := filepath.Join(imagesPath, imageName)

		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			return false, nil // Image missing
		}
	}

	return true, nil
}

// IsCompleteDeck checks if a complete deck with TargetCardCount cards exists
func (v *Validator) IsCompleteDeck() (bool, error) {
	valid, err := v.IsValidDeck()
	if err != nil || !valid {
		return false, err
	}

	// Check we have the target count
	cardsJSONPath := filepath.Join(v.cardsPath, "cards.json")
	data, _ := os.ReadFile(cardsJSONPath)
	var cardsFile CardsFile
	json.Unmarshal(data, &cardsFile)

	return len(cardsFile.Cards) >= v.targetCount, nil
}

// GetMissingCardIDs returns list of card IDs that are missing images
func (v *Validator) GetMissingCardIDs(concepts []CardConcept) []string {
	missing := make([]string, 0)
	imagesPath := filepath.Join(v.cardsPath, "images")

	for _, concept := range concepts {
		imagePath := filepath.Join(imagesPath, concept.ID+".png")
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			missing = append(missing, concept.ID)
		}
	}

	return missing
}

// GetExistingCardCount returns the number of cards in the deck (from cards.json)
func (v *Validator) GetExistingCardCount() int {
	cardsJSONPath := filepath.Join(v.cardsPath, "cards.json")
	data, err := os.ReadFile(cardsJSONPath)
	if err != nil {
		return 0
	}

	var cardsFile CardsFile
	if err := json.Unmarshal(data, &cardsFile); err != nil {
		return 0
	}

	return len(cardsFile.Cards)
}

// cardIDFromIndex generates a card ID from an index (1-100)
func cardIDFromIndex(index int) string {
	return formatCardID(index)
}

// formatCardID formats an index as a card ID (e.g., 1 -> "card-001")
func formatCardID(index int) string {
	return "card-" + padZeros(index, 3)
}

// padZeros pads a number with leading zeros
func padZeros(n, width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s = string('0'+byte(n%10)) + s
		n /= 10
	}
	return s
}
