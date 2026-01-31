package cards

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_GetCards(t *testing.T) {
	// Create a temporary cards.json file
	tmpDir := t.TempDir()
	cardsPath := filepath.Join(tmpDir, "cards.json")

	cardsJSON := `{
		"version": 1,
		"cards": [
			{"id": "card-001", "title": "Test Card 1", "image": "/cards/images/card-001.svg", "tags": ["test"]},
			{"id": "card-002", "title": "Test Card 2", "image": "/cards/images/card-002.svg", "tags": ["test"]}
		]
	}`

	if err := os.WriteFile(cardsPath, []byte(cardsJSON), 0644); err != nil {
		t.Fatalf("Failed to write test cards.json: %v", err)
	}

	loader := NewLoader(cardsPath)

	cards, err := loader.GetCards()
	if err != nil {
		t.Fatalf("GetCards failed: %v", err)
	}

	if len(cards) != 2 {
		t.Errorf("Expected 2 cards, got %d", len(cards))
	}

	if cards[0].ID != "card-001" {
		t.Errorf("Expected card ID 'card-001', got '%s'", cards[0].ID)
	}

	if cards[1].Image != "/cards/images/card-002.svg" {
		t.Errorf("Expected image path '/cards/images/card-002.svg', got '%s'", cards[1].Image)
	}
}

func TestLoader_CacheRefresh(t *testing.T) {
	// Create a temporary cards.json file
	tmpDir := t.TempDir()
	cardsPath := filepath.Join(tmpDir, "cards.json")

	cardsJSON := `{
		"version": 1,
		"cards": [
			{"id": "card-001", "title": "Test Card 1", "image": "/cards/images/card-001.svg", "tags": ["test"]}
		]
	}`

	if err := os.WriteFile(cardsPath, []byte(cardsJSON), 0644); err != nil {
		t.Fatalf("Failed to write test cards.json: %v", err)
	}

	loader := NewLoader(cardsPath)
	loader.cacheTTL = 0 // Disable cache for this test

	// First load
	cards1, err := loader.GetCards()
	if err != nil {
		t.Fatalf("First GetCards failed: %v", err)
	}

	if len(cards1) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards1))
	}

	// Update the file
	cardsJSON2 := `{
		"version": 1,
		"cards": [
			{"id": "card-001", "title": "Test Card 1", "image": "/cards/images/card-001.svg", "tags": ["test"]},
			{"id": "card-002", "title": "Test Card 2", "image": "/cards/images/card-002.svg", "tags": ["test"]},
			{"id": "card-003", "title": "Test Card 3", "image": "/cards/images/card-003.svg", "tags": ["test"]}
		]
	}`

	if err := os.WriteFile(cardsPath, []byte(cardsJSON2), 0644); err != nil {
		t.Fatalf("Failed to update cards.json: %v", err)
	}

	// Second load - should see updated cards
	cards2, err := loader.GetCards()
	if err != nil {
		t.Fatalf("Second GetCards failed: %v", err)
	}

	if len(cards2) != 3 {
		t.Errorf("Expected 3 cards after update, got %d", len(cards2))
	}
}

func TestLoader_GetCardsData(t *testing.T) {
	// Create a temporary cards.json file
	tmpDir := t.TempDir()
	cardsPath := filepath.Join(tmpDir, "cards.json")

	cardsJSON := `{
		"version": 1,
		"cards": [
			{"id": "card-001", "title": "Dreamy Night", "image": "/cards/images/card-001.svg", "tags": ["dreamy", "night"]}
		]
	}`

	if err := os.WriteFile(cardsPath, []byte(cardsJSON), 0644); err != nil {
		t.Fatalf("Failed to write test cards.json: %v", err)
	}

	loader := NewLoader(cardsPath)

	cardsData, err := loader.GetCardsData()
	if err != nil {
		t.Fatalf("GetCardsData failed: %v", err)
	}

	if len(cardsData) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cardsData))
	}

	if cardsData[0].Title != "Dreamy Night" {
		t.Errorf("Expected title 'Dreamy Night', got '%s'", cardsData[0].Title)
	}

	if len(cardsData[0].Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(cardsData[0].Tags))
	}
}
