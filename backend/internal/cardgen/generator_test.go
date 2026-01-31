package cardgen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidator_IsValidDeck_Empty(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validator := NewValidator(tmpDir)

	valid, err := validator.IsValidDeck()
	if err != nil {
		t.Fatalf("IsValidDeck returned error: %v", err)
	}
	if valid {
		t.Error("Expected invalid deck for empty directory")
	}
}

func TestValidator_IsValidDeck_PartialCards(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create cards.json with 100 cards
	imagesDir := filepath.Join(tmpDir, "images")
	os.MkdirAll(imagesDir, 0755)

	cards := make([]CardData, 100)
	for i := 0; i < 100; i++ {
		cards[i] = CardData{
			ID:    formatCardID(i + 1),
			Title: "Test Card",
			Image: "/cards/images/" + formatCardID(i+1) + ".png",
			Tags:  []string{"test"},
		}
	}

	cardsFile := CardsFile{Version: 1, Cards: cards}
	data, _ := json.Marshal(cardsFile)
	os.WriteFile(filepath.Join(tmpDir, "cards.json"), data, 0644)

	// Create only 50 images
	for i := 0; i < 50; i++ {
		os.WriteFile(filepath.Join(imagesDir, formatCardID(i+1)+".png"), []byte("test"), 0644)
	}

	validator := NewValidator(tmpDir)
	valid, err := validator.IsValidDeck()
	if err != nil {
		t.Fatalf("IsValidDeck returned error: %v", err)
	}
	if valid {
		t.Error("Expected invalid deck when images are missing")
	}
}

func TestValidator_IsValidDeck_Complete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	imagesDir := filepath.Join(tmpDir, "images")
	os.MkdirAll(imagesDir, 0755)

	cards := make([]CardData, 100)
	for i := 0; i < 100; i++ {
		cards[i] = CardData{
			ID:    formatCardID(i + 1),
			Title: "Test Card",
			Image: "/cards/images/" + formatCardID(i+1) + ".png",
			Tags:  []string{"test"},
		}
		// Create corresponding image
		os.WriteFile(filepath.Join(imagesDir, formatCardID(i+1)+".png"), []byte("test"), 0644)
	}

	cardsFile := CardsFile{Version: 1, Cards: cards}
	data, _ := json.Marshal(cardsFile)
	os.WriteFile(filepath.Join(tmpDir, "cards.json"), data, 0644)

	validator := NewValidator(tmpDir)
	valid, err := validator.IsValidDeck()
	if err != nil {
		t.Fatalf("IsValidDeck returned error: %v", err)
	}
	if !valid {
		t.Error("Expected valid deck when all cards and images exist")
	}
}

func TestLockFile_AcquireRelease(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	lock := NewLockFile(tmpDir)

	// Should be able to acquire
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Should be locked
	if !lock.IsLocked() {
		t.Error("Expected lock to be held")
	}

	// Second acquire should fail
	lock2 := NewLockFile(tmpDir)
	if err := lock2.Acquire(); err == nil {
		t.Error("Expected error when acquiring lock twice")
	}

	// Release
	lock.Release()

	// Should no longer be locked
	if lock.IsLocked() {
		t.Error("Expected lock to be released")
	}

	// Should be able to acquire again
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Failed to reacquire lock: %v", err)
	}
	lock.Release()
}

func TestLockFile_StaleLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a stale lock (old timestamp)
	lockPath := filepath.Join(tmpDir, LockFileName)
	staleInfo := LockInfo{
		PID:       99999,
		Hostname:  "old-host",
		StartedAt: time.Now().Add(-2 * time.Hour), // 2 hours ago (stale)
	}
	data, _ := json.Marshal(staleInfo)
	os.WriteFile(lockPath, data, 0644)

	lock := NewLockFile(tmpDir)

	// Should be able to acquire (stale lock removed)
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Failed to acquire lock over stale lock: %v", err)
	}
	lock.Release()
}

func TestGenerator_LoadState_Fresh(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := NewGenerator(Config{CardsPath: tmpDir})
	state, err := gen.loadOrInitState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if state.ConceptsGenerated {
		t.Error("Expected fresh state with ConceptsGenerated=false")
	}
	if len(state.ImagesGenerated) != 0 {
		t.Error("Expected empty ImagesGenerated for fresh state")
	}
}

func TestGenerator_LoadState_Resume(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create existing state
	existingState := GenerationState{
		Version:           1,
		StartedAt:         time.Now().Add(-1 * time.Hour),
		ConceptsGenerated: true,
		Concepts: []CardConcept{
			{ID: "card-001", Title: "Test", Description: "Test desc", Tags: []string{"test"}},
		},
		ImagesGenerated: []string{"card-001"},
	}
	data, _ := json.MarshalIndent(existingState, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, StateFileName), data, 0644)

	gen := NewGenerator(Config{CardsPath: tmpDir})
	state, err := gen.loadOrInitState()
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	if !state.ConceptsGenerated {
		t.Error("Expected ConceptsGenerated=true from loaded state")
	}
	if len(state.Concepts) != 1 {
		t.Errorf("Expected 1 concept, got %d", len(state.Concepts))
	}
	if len(state.ImagesGenerated) != 1 {
		t.Errorf("Expected 1 image generated, got %d", len(state.ImagesGenerated))
	}
}

func TestGenerator_FindRemainingConcepts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cardgen_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	imagesDir := filepath.Join(tmpDir, "images")
	os.MkdirAll(imagesDir, 0755)

	// Create state with 3 concepts, 1 already generated
	state := &GenerationState{
		Concepts: []CardConcept{
			{ID: "card-001", Title: "Test 1"},
			{ID: "card-002", Title: "Test 2"},
			{ID: "card-003", Title: "Test 3"},
		},
		ImagesGenerated: []string{"card-001"},
	}

	// Also create the image file for card-001
	os.WriteFile(filepath.Join(imagesDir, "card-001.png"), []byte("test"), 0644)

	gen := NewGenerator(Config{CardsPath: tmpDir})
	remaining := gen.findRemainingConcepts(state)

	if len(remaining) != 2 {
		t.Errorf("Expected 2 remaining concepts, got %d", len(remaining))
	}

	// Should be card-002 and card-003
	expectedIDs := map[string]bool{"card-002": true, "card-003": true}
	for _, c := range remaining {
		if !expectedIDs[c.ID] {
			t.Errorf("Unexpected remaining concept: %s", c.ID)
		}
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			input:    "  \n{\"key\": \"value\"}\n  ",
			expected: `{"key": "value"}`,
		},
	}

	for _, tc := range tests {
		result := cleanJSONResponse(tc.input)
		if result != tc.expected {
			t.Errorf("cleanJSONResponse(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatCardID(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{1, "card-001"},
		{10, "card-010"},
		{100, "card-100"},
		{99, "card-099"},
	}

	for _, tc := range tests {
		result := formatCardID(tc.index)
		if result != tc.expected {
			t.Errorf("formatCardID(%d) = %q, expected %q", tc.index, result, tc.expected)
		}
	}
}
