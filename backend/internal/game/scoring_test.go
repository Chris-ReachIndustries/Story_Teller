package game

import "testing"

// Helper to create test players
func createTestPlayers(count int) []*Player {
	players := make([]*Player, count)
	for i := 0; i < count; i++ {
		players[i] = &Player{
			ID:    string(rune('A' + i)), // A, B, C, D, E, F
			Name:  string(rune('A' + i)),
			Score: 0,
		}
	}
	return players
}

// Helper to create test submissions
func createTestSubmissions(players []*Player) []Submission {
	submissions := make([]Submission, len(players))
	for i, p := range players {
		submissions[i] = Submission{
			PlayerID: p.ID,
			Card:     Card{ID: "card-" + p.ID, Image: "/cards/images/card-" + p.ID + ".jpg"},
		}
	}
	return submissions
}

func TestScoring_NobodyGuessesStoryteller(t *testing.T) {
	// Setup: 4 players, A is storyteller
	players := createTestPlayers(4)
	storytellerID := players[0].ID

	// Submissions: A (storyteller), B, C, D
	submissions := createTestSubmissions(players)

	// Votes: nobody votes for storyteller's card (index 0)
	votes := map[string]int{
		"B": 1, // B votes for C's card
		"C": 1, // C votes for B's card (wait, that's same index - let's fix)
		"D": 2, // D votes for C's card
	}
	// Let's make them vote for non-storyteller cards
	votes = map[string]int{
		"B": 1, // B's card is at index 1, so B votes for C's card (index 2)
		"C": 3, // C votes for D's card
		"D": 1, // D votes for B's card
	}

	result := CalculateScores(storytellerID, 0, submissions, votes, players)

	// Nobody guessed: storyteller=0, others=2 each
	if result.PointsThisRound["A"] != 0 {
		t.Errorf("Storyteller should get 0, got %d", result.PointsThisRound["A"])
	}

	// Others get 2 base + bonus for votes received
	// B: 2 base + 1 vote from D = 3
	// C: 2 base + 1 vote from B = 3  (wait, B voted for index 1 which is B's own card... let me reconsider)

	// Actually: submissions[0]=A, submissions[1]=B, submissions[2]=C, submissions[3]=D
	// B votes for 1 (B's own card - but they can't vote for their own in real game, for test let's allow)
	// Let me rewrite this more carefully:

	// In reality, the rule is you can't vote for your own card
	// But for this test of "nobody guesses", let's make it clear:
	votes = map[string]int{
		"B": 2, // B votes for C's card (index 2)
		"C": 3, // C votes for D's card (index 3)
		"D": 1, // D votes for B's card (index 1)
	}

	result = CalculateScores(storytellerID, 0, submissions, votes, players)

	// Nobody guessed storyteller (index 0)
	// Storyteller A: 0 base + 0 bonus = 0
	if result.PointsThisRound["A"] != 0 {
		t.Errorf("Storyteller should get 0, got %d", result.PointsThisRound["A"])
	}

	// B: 2 base + 1 vote from D = 3
	if result.PointsThisRound["B"] != 3 {
		t.Errorf("B should get 3, got %d", result.PointsThisRound["B"])
	}

	// C: 2 base + 1 vote from B = 3
	if result.PointsThisRound["C"] != 3 {
		t.Errorf("C should get 3, got %d", result.PointsThisRound["C"])
	}

	// D: 2 base + 1 vote from C = 3
	if result.PointsThisRound["D"] != 3 {
		t.Errorf("D should get 3, got %d", result.PointsThisRound["D"])
	}
}

func TestScoring_EverybodyGuessesStoryteller(t *testing.T) {
	// Setup: 4 players, A is storyteller
	players := createTestPlayers(4)
	storytellerID := players[0].ID
	submissions := createTestSubmissions(players)

	// Everyone votes for storyteller's card (index 0)
	votes := map[string]int{
		"B": 0, // All vote for storyteller
		"C": 0,
		"D": 0,
	}

	result := CalculateScores(storytellerID, 0, submissions, votes, players)

	// Everybody guessed: storyteller=0, others=2 each
	// Storyteller does NOT get bonus points for votes on their card
	if result.PointsThisRound["A"] != 0 {
		t.Errorf("Storyteller should get 0 (no bonus for storyteller), got %d", result.PointsThisRound["A"])
	}

	// Others get 2 base, no bonus (nobody voted for their cards)
	if result.PointsThisRound["B"] != 2 {
		t.Errorf("B should get 2, got %d", result.PointsThisRound["B"])
	}
	if result.PointsThisRound["C"] != 2 {
		t.Errorf("C should get 2, got %d", result.PointsThisRound["C"])
	}
	if result.PointsThisRound["D"] != 2 {
		t.Errorf("D should get 2, got %d", result.PointsThisRound["D"])
	}
}

func TestScoring_MixedGuesses(t *testing.T) {
	// Setup: 4 players, A is storyteller
	players := createTestPlayers(4)
	storytellerID := players[0].ID
	submissions := createTestSubmissions(players)

	// Some guess correctly, some don't
	// B guesses correctly (votes 0), C and D don't
	votes := map[string]int{
		"B": 0, // Correct
		"C": 1, // Wrong - votes for B's card
		"D": 1, // Wrong - votes for B's card
	}

	result := CalculateScores(storytellerID, 0, submissions, votes, players)

	// Mixed: storyteller=3, correct guessers=3
	// A (storyteller): 3 base, no bonus (storyteller doesn't get vote bonus)
	if result.PointsThisRound["A"] != 3 {
		t.Errorf("Storyteller should get 3 (no bonus for storyteller), got %d", result.PointsThisRound["A"])
	}

	// B: 3 (correct guess) + 2 bonus (C and D voted for B's card) = 5
	if result.PointsThisRound["B"] != 5 {
		t.Errorf("B should get 5 (3 + 2 bonus), got %d", result.PointsThisRound["B"])
	}

	// C: 0 (wrong guess) + 0 bonus = 0
	if result.PointsThisRound["C"] != 0 {
		t.Errorf("C should get 0, got %d", result.PointsThisRound["C"])
	}

	// D: 0 (wrong guess) + 0 bonus = 0
	if result.PointsThisRound["D"] != 0 {
		t.Errorf("D should get 0, got %d", result.PointsThisRound["D"])
	}
}

func TestScoring_VoteBonuses(t *testing.T) {
	// Test that vote bonuses are calculated correctly
	players := createTestPlayers(5) // A, B, C, D, E
	storytellerID := players[0].ID
	submissions := createTestSubmissions(players)

	// Mixed voting pattern to test bonuses
	// 2 correct guessers, 2 wrong
	votes := map[string]int{
		"B": 0, // Correct - votes for storyteller
		"C": 0, // Correct - votes for storyteller
		"D": 1, // Wrong - votes for B
		"E": 2, // Wrong - votes for C
	}

	result := CalculateScores(storytellerID, 0, submissions, votes, players)

	// A (storyteller): 3 base, no bonus (storyteller doesn't get vote bonus)
	if result.PointsThisRound["A"] != 3 {
		t.Errorf("A should get 3, got %d", result.PointsThisRound["A"])
	}

	// B: 3 (correct) + 1 bonus (D voted for index 1) = 4
	if result.PointsThisRound["B"] != 4 {
		t.Errorf("B should get 4, got %d", result.PointsThisRound["B"])
	}

	// C: 3 (correct) + 1 bonus (E voted for index 2) = 4
	if result.PointsThisRound["C"] != 4 {
		t.Errorf("C should get 4, got %d", result.PointsThisRound["C"])
	}

	// D: 0 (wrong) + 0 bonus = 0
	if result.PointsThisRound["D"] != 0 {
		t.Errorf("D should get 0, got %d", result.PointsThisRound["D"])
	}

	// E: 0 (wrong) + 0 bonus = 0
	if result.PointsThisRound["E"] != 0 {
		t.Errorf("E should get 0, got %d", result.PointsThisRound["E"])
	}
}

func TestScoring_ThreePlayerMinimum(t *testing.T) {
	// Test with 3 players (minimum for test mode)
	players := createTestPlayers(3) // A, B, C
	storytellerID := players[0].ID
	submissions := createTestSubmissions(players)

	// One correct, one wrong
	votes := map[string]int{
		"B": 0, // Correct
		"C": 1, // Wrong - votes for B
	}

	result := CalculateScores(storytellerID, 0, submissions, votes, players)

	// Mixed result: A=3, B=3 (correct)
	// A: 3 base, no bonus (storyteller doesn't get vote bonus)
	if result.PointsThisRound["A"] != 3 {
		t.Errorf("A should get 3, got %d", result.PointsThisRound["A"])
	}

	// B: 3 (correct) + 1 bonus = 4
	if result.PointsThisRound["B"] != 4 {
		t.Errorf("B should get 4, got %d", result.PointsThisRound["B"])
	}

	// C: 0 (wrong) + 0 bonus = 0
	if result.PointsThisRound["C"] != 0 {
		t.Errorf("C should get 0, got %d", result.PointsThisRound["C"])
	}
}

func TestApplyScores(t *testing.T) {
	players := createTestPlayers(3)
	players[0].Score = 10
	players[1].Score = 5
	players[2].Score = 0

	result := &RoundResult{
		PointsThisRound: map[string]int{
			"A": 3,
			"B": 5,
			"C": 2,
		},
	}

	ApplyScores(result, players)

	if players[0].Score != 13 {
		t.Errorf("A score should be 13, got %d", players[0].Score)
	}
	if players[1].Score != 10 {
		t.Errorf("B score should be 10, got %d", players[1].Score)
	}
	if players[2].Score != 2 {
		t.Errorf("C score should be 2, got %d", players[2].Score)
	}
}
