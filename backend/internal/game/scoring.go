package game

// RoundResult holds the results of a round's scoring
type RoundResult struct {
	StorytellerCardIndex int            `json:"storytellerCard"`
	Votes                map[string]int `json:"votes"`         // playerID -> submissionIndex they voted for
	PointsThisRound      map[string]int `json:"pointsThisRound"` // playerID -> points earned this round
}

// CalculateScores computes scores for a round based on Dixit rules:
// - If ALL or NOBODY guessed storyteller's card: storyteller=0, others=2 each
// - Otherwise: storyteller=3, each correct guesser=3
// - Additionally: each player gets +1 for each vote their submitted card received
func CalculateScores(
	storytellerID string,
	submissions []Submission, // submissions[0] is always storyteller's card
	votes map[string]int, // playerID -> submissionIndex
	players []*Player,
) *RoundResult {
	result := &RoundResult{
		StorytellerCardIndex: 0, // Storyteller's card is always at index 0 before shuffle
		Votes:                votes,
		PointsThisRound:      make(map[string]int),
	}

	// Initialize points to 0 for all players
	for _, p := range players {
		result.PointsThisRound[p.ID] = 0
	}

	// Count how many voted for storyteller's card (index 0)
	votesForStoryteller := 0
	totalVoters := 0
	for playerID, votedIdx := range votes {
		if playerID == storytellerID {
			continue // Storyteller doesn't vote
		}
		totalVoters++
		if votedIdx == 0 {
			votesForStoryteller++
		}
	}

	// Determine base scoring
	allGuessed := votesForStoryteller == totalVoters
	noneGuessed := votesForStoryteller == 0

	if allGuessed || noneGuessed {
		// Storyteller gets 0, all others get 2
		for _, p := range players {
			if p.ID != storytellerID {
				result.PointsThisRound[p.ID] += 2
			}
		}
	} else {
		// Storyteller gets 3
		result.PointsThisRound[storytellerID] += 3

		// Each correct guesser gets 3
		for playerID, votedIdx := range votes {
			if playerID == storytellerID {
				continue
			}
			if votedIdx == 0 { // Voted for storyteller's card
				result.PointsThisRound[playerID] += 3
			}
		}
	}

	// Bonus points: +1 for each vote received on your card
	// IMPORTANT: Storyteller does NOT get bonus points for votes on their card
	// Build a map of submissionIndex -> ownerPlayerID
	submissionOwners := make(map[int]string)
	for i, sub := range submissions {
		submissionOwners[i] = sub.PlayerID
	}

	// Count votes per submission
	votesPerSubmission := make(map[int]int)
	for playerID, votedIdx := range votes {
		if playerID == storytellerID {
			continue
		}
		votesPerSubmission[votedIdx]++
	}

	// Award bonus points (skip storyteller's card - they don't get bonus points)
	for submissionIdx, voteCount := range votesPerSubmission {
		ownerID := submissionOwners[submissionIdx]
		// Storyteller doesn't get bonus points for votes on their card
		if ownerID == storytellerID {
			continue
		}
		result.PointsThisRound[ownerID] += voteCount
	}

	return result
}

// ApplyScores adds the round points to player totals
func ApplyScores(result *RoundResult, players []*Player) {
	for _, p := range players {
		if points, ok := result.PointsThisRound[p.ID]; ok {
			p.Score += points
		}
	}
}
