package ai

import (
	"math/rand"
	"time"
)

// MockClient implements the Client interface with random selections
// Used when AI is disabled or for testing
type MockClient struct {
	rng *rand.Rand
}

// NewMockClient creates a new mock AI client
func NewMockClient() *MockClient {
	return &MockClient{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Storytell implements Client.Storytell with random selection
func (c *MockClient) Storytell(hand []CardWithThumb) (*StorytellerResponse, error) {
	clues := []string{
		"A journey begins",
		"Dreams of tomorrow",
		"The hidden path",
		"Between worlds",
		"Silent echoes",
		"Whispers in the wind",
		"The last light",
		"Endless possibilities",
	}

	return &StorytellerResponse{
		SelectedCard: c.rng.Intn(len(hand)) + 1,
		Clue:         clues[c.rng.Intn(len(clues))],
	}, nil
}

// Submit implements Client.Submit with random selection
func (c *MockClient) Submit(hand []CardWithThumb, clue string) (*SubmitResponse, error) {
	return &SubmitResponse{
		SelectedCard: c.rng.Intn(len(hand)) + 1,
	}, nil
}

// Vote implements Client.Vote with random selection (avoiding own card)
func (c *MockClient) Vote(submissions []CardWithThumb, clue string, ownIndex int) (*VoteResponse, error) {
	// Build list of valid choices (excluding own card)
	validChoices := make([]int, 0, len(submissions)-1)
	for i := 1; i <= len(submissions); i++ {
		if i != ownIndex {
			validChoices = append(validChoices, i)
		}
	}

	if len(validChoices) == 0 {
		// Fallback (shouldn't happen in normal play)
		return &VoteResponse{SelectedCard: 1}, nil
	}

	return &VoteResponse{
		SelectedCard: validChoices[c.rng.Intn(len(validChoices))],
	}, nil
}
