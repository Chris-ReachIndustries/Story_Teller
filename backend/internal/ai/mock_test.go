package ai

import (
	"testing"
)

func TestMockClientStoryTell(t *testing.T) {
	client := NewMockClient()

	hand := []CardWithThumb{
		{CardID: "card-001", DataURL: "data:image/png;base64,test1"},
		{CardID: "card-002", DataURL: "data:image/png;base64,test2"},
		{CardID: "card-003", DataURL: "data:image/png;base64,test3"},
	}

	resp, err := client.Storytell(hand)
	if err != nil {
		t.Fatalf("Storytell failed: %v", err)
	}

	if resp.SelectedCard < 1 || resp.SelectedCard > len(hand) {
		t.Errorf("SelectedCard %d out of range [1, %d]", resp.SelectedCard, len(hand))
	}

	if resp.Clue == "" {
		t.Error("Clue should not be empty")
	}
}

func TestMockClientSubmit(t *testing.T) {
	client := NewMockClient()

	hand := []CardWithThumb{
		{CardID: "card-001", DataURL: "data:image/png;base64,test1"},
		{CardID: "card-002", DataURL: "data:image/png;base64,test2"},
		{CardID: "card-003", DataURL: "data:image/png;base64,test3"},
	}

	resp, err := client.Submit(hand, "test clue")
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if resp.SelectedCard < 1 || resp.SelectedCard > len(hand) {
		t.Errorf("SelectedCard %d out of range [1, %d]", resp.SelectedCard, len(hand))
	}
}

func TestMockClientVote(t *testing.T) {
	client := NewMockClient()

	submissions := []CardWithThumb{
		{CardID: "card-001", DataURL: "data:image/png;base64,test1"},
		{CardID: "card-002", DataURL: "data:image/png;base64,test2"},
		{CardID: "card-003", DataURL: "data:image/png;base64,test3"},
		{CardID: "card-004", DataURL: "data:image/png;base64,test4"},
	}

	ownIndex := 2 // Bot's own card is at index 2

	// Run multiple times to verify it never votes for own card
	for i := 0; i < 100; i++ {
		resp, err := client.Vote(submissions, "test clue", ownIndex)
		if err != nil {
			t.Fatalf("Vote failed: %v", err)
		}

		if resp.SelectedCard < 1 || resp.SelectedCard > len(submissions) {
			t.Errorf("SelectedCard %d out of range [1, %d]", resp.SelectedCard, len(submissions))
		}

		if resp.SelectedCard == ownIndex {
			t.Errorf("Bot voted for its own card (index %d)", ownIndex)
		}
	}
}

func TestMockClientVoteWithSingleOtherOption(t *testing.T) {
	client := NewMockClient()

	// Only 2 submissions - one is the bot's own card
	submissions := []CardWithThumb{
		{CardID: "card-001", DataURL: "data:image/png;base64,test1"},
		{CardID: "card-002", DataURL: "data:image/png;base64,test2"},
	}

	ownIndex := 1 // Bot's own card is at index 1

	resp, err := client.Vote(submissions, "test clue", ownIndex)
	if err != nil {
		t.Fatalf("Vote failed: %v", err)
	}

	// Should always vote for the only other option (index 2)
	if resp.SelectedCard != 2 {
		t.Errorf("Expected SelectedCard 2, got %d", resp.SelectedCard)
	}
}
