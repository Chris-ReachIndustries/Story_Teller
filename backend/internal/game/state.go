package game

// GamePhase represents the current phase of the game
type GamePhase string

const (
	PhaseLobby          GamePhase = "LOBBY"
	PhaseStorytellerClue GamePhase = "STORYTELLER_CLUE"
	PhaseSubmissions    GamePhase = "SUBMISSIONS"
	PhaseVoting         GamePhase = "VOTING"
	PhaseScoring        GamePhase = "SCORING"
	PhaseRoundEnd       GamePhase = "ROUND_END"
	PhaseGameEnd        GamePhase = "GAME_END"
)

// GameEndReason explains why the game ended
type GameEndReason string

const (
	EndReasonNone              GameEndReason = ""
	EndReasonScoreReached      GameEndReason = "score_reached"
	EndReasonOutOfCards        GameEndReason = "out_of_cards"
	EndReasonPlayerDisconnected GameEndReason = "player_disconnected"
)

// GameState holds the current state of a game
type GameState struct {
	Phase              GamePhase            `json:"phase"`
	Round              int                  `json:"round"`
	StorytellerIdx     int                  `json:"-"`
	StorytellerOffset  int                  `json:"-"` // Starting offset for storyteller rotation
	StorytellerID      string               `json:"storytellerId"`
	Clue               string               `json:"clue"`
	Submissions        []Submission         `json:"-"`          // Internal tracking
	ShuffledSubmissions []ShuffledSubmission `json:"submissions"` // Anonymized for clients
	GameEndReason      GameEndReason        `json:"gameEndReason,omitempty"`
	WinnerID           string               `json:"winnerId,omitempty"`
}

// Submission tracks who submitted what card
type Submission struct {
	PlayerID string
	Card     Card
}

// ShuffledSubmission is the anonymized version shown to clients
type ShuffledSubmission struct {
	Index int    `json:"index"`
	Image string `json:"image"`
}

// NewGameState creates a new game state in lobby phase
func NewGameState() *GameState {
	return &GameState{
		Phase:          PhaseLobby,
		Round:          0,
		StorytellerIdx: 0,
		Submissions:    make([]Submission, 0),
	}
}

// CanTransitionTo checks if a transition to the target phase is valid
func (gs *GameState) CanTransitionTo(target GamePhase) bool {
	switch gs.Phase {
	case PhaseLobby:
		return target == PhaseStorytellerClue
	case PhaseStorytellerClue:
		return target == PhaseSubmissions
	case PhaseSubmissions:
		return target == PhaseVoting
	case PhaseVoting:
		return target == PhaseScoring
	case PhaseScoring:
		return target == PhaseRoundEnd
	case PhaseRoundEnd:
		return target == PhaseStorytellerClue || target == PhaseGameEnd
	case PhaseGameEnd:
		return target == PhaseLobby
	}
	return false
}

// TransitionTo transitions to a new phase if valid
func (gs *GameState) TransitionTo(target GamePhase) bool {
	if !gs.CanTransitionTo(target) {
		return false
	}
	gs.Phase = target
	return true
}

// StartNewRound prepares state for a new round
func (gs *GameState) StartNewRound(players []*Player) {
	gs.Round++
	gs.Clue = ""
	gs.Submissions = make([]Submission, 0)
	gs.ShuffledSubmissions = nil

	// Rotate storyteller using the stored offset
	gs.StorytellerIdx = (gs.StorytellerOffset + gs.Round - 1) % len(players)
	gs.StorytellerID = players[gs.StorytellerIdx].ID

	// Reset player round state
	for _, p := range players {
		p.SubmittedCard = nil
		p.VotedFor = -1
	}
}

// SetStartingStoryteller sets the storyteller offset for the game
func (gs *GameState) SetStartingStoryteller(playerIdx int) {
	gs.StorytellerOffset = playerIdx
}

// ResetForNewGame resets state for a new game
func (gs *GameState) ResetForNewGame() {
	gs.Phase = PhaseLobby
	gs.Round = 0
	gs.StorytellerIdx = 0
	gs.StorytellerOffset = 0
	gs.StorytellerID = ""
	gs.Clue = ""
	gs.Submissions = nil
	gs.ShuffledSubmissions = nil
	gs.GameEndReason = EndReasonNone
	gs.WinnerID = ""
}
