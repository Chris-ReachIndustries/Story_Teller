package game

import (
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMinPlayers    = 4
	MaxPlayers           = 6
	DefaultHandSize      = 6
	DefaultScoreToWin    = 30
	DisconnectTimeout    = 60 * time.Second
	ReconnectGracePeriod = 10 * time.Minute
)

// RoomConfig holds configurable room settings
type RoomConfig struct {
	ScoreToWin          int    `json:"scoreToWin"`
	HandSize            int    `json:"handSize"`
	StartingStoryteller string `json:"startingStoryteller"` // "random", "host", or a player ID
	DeckSetID           string `json:"deckSetId"`           // Card set to use (e.g., "default", "fantasy")
}

// DefaultRoomConfig returns the default room configuration
func DefaultRoomConfig() RoomConfig {
	return RoomConfig{
		ScoreToWin: DefaultScoreToWin,
		HandSize:   DefaultHandSize,
		DeckSetID:  "default",
	}
}

// Validate checks if the config values are within acceptable ranges
func (c RoomConfig) Validate() error {
	if c.ScoreToWin < 10 || c.ScoreToWin > 100 {
		return ErrInvalidScoreToWin
	}
	if c.HandSize < 4 || c.HandSize > 10 {
		return ErrInvalidHandSize
	}
	return nil
}

// GetMinPlayers returns the minimum players required (can be overridden for testing)
func GetMinPlayers() int {
	if val := os.Getenv("MIN_PLAYERS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n >= 3 && n <= MaxPlayers {
			return n
		}
	}
	return DefaultMinPlayers
}

// Room represents a game room
type Room struct {
	Mu sync.RWMutex

	Code      string     `json:"code"`
	HostID    string     `json:"hostId"`
	Players   []*Player  `json:"players"`
	State     *GameState `json:"state"`
	Config    RoomConfig `json:"config"`
	Deck      []Card     `json:"-"` // Available cards in deck
	AllCards  []Card     `json:"-"` // All cards (for refilling)
	CreatedAt time.Time  `json:"-"`

	// Disconnect tracking
	disconnectTimers map[string]*time.Timer
}

// RoomManager manages all game rooms
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewRoomManager creates a new room manager
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom creates a new room with a random code and optional config
func (rm *RoomManager) CreateRoom(host *Player, config *RoomConfig) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	roomConfig := DefaultRoomConfig()
	if config != nil {
		if err := config.Validate(); err == nil {
			roomConfig = *config
		}
	}

	code := rm.generateCode()
	room := &Room{
		Code:             code,
		HostID:           host.ID,
		Players:          []*Player{host},
		State:            NewGameState(),
		Config:           roomConfig,
		Deck:             make([]Card, 0),
		AllCards:         make([]Card, 0),
		CreatedAt:        time.Now(),
		disconnectTimers: make(map[string]*time.Timer),
	}

	rm.rooms[code] = room
	return room
}

// GetRoom retrieves a room by code
func (rm *RoomManager) GetRoom(code string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[strings.ToUpper(code)]
}

// RemoveRoom removes a room
func (rm *RoomManager) RemoveRoom(code string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, strings.ToUpper(code))
}

// generateCode creates a random 4-letter room code
func (rm *RoomManager) generateCode() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ" // Exclude I and O to avoid confusion
	for {
		code := make([]byte, 4)
		for i := range code {
			code[i] = letters[rand.Intn(len(letters))]
		}
		codeStr := string(code)
		if _, exists := rm.rooms[codeStr]; !exists {
			return codeStr
		}
	}
}

// AddPlayer adds a player to the room
func (r *Room) AddPlayer(player *Player) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if len(r.Players) >= MaxPlayers {
		return ErrRoomFull
	}

	if r.State.Phase != PhaseLobby {
		return ErrGameInProgress
	}

	r.Players = append(r.Players, player)
	return nil
}

// RemovePlayer removes a player from the room
func (r *Room) RemovePlayer(playerID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	for i, p := range r.Players {
		if p.ID == playerID {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
}

// GetPlayer gets a player by ID
func (r *Room) GetPlayer(playerID string) *Player {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	for _, p := range r.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

// GetPlayerByName gets a player by name
func (r *Room) GetPlayerByName(name string) *Player {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	for _, p := range r.Players {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// SetPlayerConnected sets a player's connection status
func (r *Room) SetPlayerConnected(playerID string, connected bool) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	for _, p := range r.Players {
		if p.ID == playerID {
			p.Connected = connected
			break
		}
	}
}

// CanStart checks if the game can be started
func (r *Room) CanStart() bool {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	if r.State.Phase != PhaseLobby {
		return false
	}

	connectedCount := 0
	for _, p := range r.Players {
		if p.Connected {
			connectedCount++
		}
	}

	return connectedCount >= GetMinPlayers() && connectedCount <= MaxPlayers
}

// StartGame initializes the game
func (r *Room) StartGame(allCards []Card) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.State.Phase != PhaseLobby {
		return ErrGameInProgress
	}

	connectedPlayers := r.getConnectedPlayers()
	if len(connectedPlayers) < GetMinPlayers() {
		return ErrNotEnoughPlayers
	}

	// Initialize deck with shuffled cards
	r.AllCards = make([]Card, len(allCards))
	copy(r.AllCards, allCards)
	r.Deck = make([]Card, len(allCards))
	copy(r.Deck, allCards)
	r.shuffleDeck()

	// Deal initial hands using config hand size
	handSize := r.Config.HandSize
	for _, p := range r.Players {
		if p.Connected {
			p.Hand = make([]Card, 0, handSize)
			for i := 0; i < handSize && len(r.Deck) > 0; i++ {
				p.Hand = append(p.Hand, r.Deck[0])
				r.Deck = r.Deck[1:]
			}
		}
	}

	// Determine starting storyteller
	startingIdx := r.determineStartingStoryteller(connectedPlayers)
	r.State.SetStartingStoryteller(startingIdx)

	// Start first round
	r.State.TransitionTo(PhaseStorytellerClue)
	r.State.StartNewRound(connectedPlayers)

	return nil
}

// determineStartingStoryteller returns the index of the starting storyteller
func (r *Room) determineStartingStoryteller(connectedPlayers []*Player) int {
	setting := r.Config.StartingStoryteller

	switch setting {
	case "", "host":
		// Default: host starts (find host in connected players)
		for i, p := range connectedPlayers {
			if p.ID == r.HostID {
				return i
			}
		}
		return 0 // Fallback to first player
	case "random":
		// Random player starts
		return rand.Intn(len(connectedPlayers))
	default:
		// Specific player ID
		for i, p := range connectedPlayers {
			if p.ID == setting {
				return i
			}
		}
		// If player not found, fallback to random
		return rand.Intn(len(connectedPlayers))
	}
}

// shuffleDeck shuffles the deck using Fisher-Yates
func (r *Room) shuffleDeck() {
	for i := len(r.Deck) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		r.Deck[i], r.Deck[j] = r.Deck[j], r.Deck[i]
	}
}

// getConnectedPlayers returns only connected players (internal, no lock)
func (r *Room) getConnectedPlayers() []*Player {
	connected := make([]*Player, 0)
	for _, p := range r.Players {
		if p.Connected {
			connected = append(connected, p)
		}
	}
	return connected
}

// GetConnectedPlayers returns only connected players (public, with lock)
func (r *Room) GetConnectedPlayers() []*Player {
	r.Mu.RLock()
	defer r.Mu.RUnlock()
	return r.getConnectedPlayers()
}

// SubmitStorytellerClue handles storyteller's clue and card submission
func (r *Room) SubmitStorytellerClue(playerID, clue, cardID string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.State.Phase != PhaseStorytellerClue {
		return ErrWrongPhase
	}

	if r.State.StorytellerID != playerID {
		return ErrNotStoryteller
	}

	player := r.getPlayerByID(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}

	if !player.HasCard(cardID) {
		return ErrCardNotInHand
	}

	// Remove card from hand and add to submissions
	card := player.RemoveCardFromHand(cardID)
	player.SubmittedCard = card

	r.State.Clue = clue
	r.State.Submissions = append(r.State.Submissions, Submission{
		PlayerID: playerID,
		Card:     *card,
	})

	r.State.TransitionTo(PhaseSubmissions)
	return nil
}

// SubmitCard handles non-storyteller card submission
func (r *Room) SubmitCard(playerID, cardID string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.State.Phase != PhaseSubmissions {
		return ErrWrongPhase
	}

	if r.State.StorytellerID == playerID {
		return ErrStorytellerCannotSubmit
	}

	player := r.getPlayerByID(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}

	if player.SubmittedCard != nil {
		return ErrAlreadySubmitted
	}

	if !player.HasCard(cardID) {
		return ErrCardNotInHand
	}

	// Remove card from hand and add to submissions
	card := player.RemoveCardFromHand(cardID)
	player.SubmittedCard = card

	r.State.Submissions = append(r.State.Submissions, Submission{
		PlayerID: playerID,
		Card:     *card,
	})

	// Check if all players have submitted
	if r.allPlayersSubmitted() {
		r.shuffleAndRevealSubmissions()
		r.State.TransitionTo(PhaseVoting)
	}

	return nil
}

// allPlayersSubmitted checks if all connected players have submitted
func (r *Room) allPlayersSubmitted() bool {
	for _, p := range r.Players {
		if p.Connected && p.SubmittedCard == nil {
			return false
		}
	}
	return true
}

// shuffleAndRevealSubmissions shuffles submissions for voting
func (r *Room) shuffleAndRevealSubmissions() {
	// Shuffle the submissions array
	for i := len(r.State.Submissions) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		r.State.Submissions[i], r.State.Submissions[j] = r.State.Submissions[j], r.State.Submissions[i]
	}

	// Create anonymized version for clients
	r.State.ShuffledSubmissions = make([]ShuffledSubmission, len(r.State.Submissions))
	for i, sub := range r.State.Submissions {
		r.State.ShuffledSubmissions[i] = ShuffledSubmission{
			Index: i,
			Image: sub.Card.Image,
		}
	}
}

// Vote handles a player's vote
func (r *Room) Vote(playerID string, submissionIndex int) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.State.Phase != PhaseVoting {
		return ErrWrongPhase
	}

	if r.State.StorytellerID == playerID {
		return ErrStorytellerCannotVote
	}

	player := r.getPlayerByID(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}

	if player.VotedFor >= 0 {
		return ErrAlreadyVoted
	}

	if submissionIndex < 0 || submissionIndex >= len(r.State.Submissions) {
		return ErrInvalidSubmission
	}

	// Check if voting for own card (not allowed)
	if r.State.Submissions[submissionIndex].PlayerID == playerID {
		return ErrCannotVoteOwnCard
	}

	player.VotedFor = submissionIndex

	// Check if all non-storyteller players have voted
	if r.allPlayersVoted() {
		r.State.TransitionTo(PhaseScoring)
	}

	return nil
}

// allPlayersVoted checks if all non-storyteller players have voted
func (r *Room) allPlayersVoted() bool {
	for _, p := range r.Players {
		if p.Connected && p.ID != r.State.StorytellerID && p.VotedFor < 0 {
			return false
		}
	}
	return true
}

// CalculateRoundScores calculates and applies scores for the round
func (r *Room) CalculateRoundScores() *RoundResult {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Build votes map
	votes := make(map[string]int)
	for _, p := range r.Players {
		if p.Connected && p.ID != r.State.StorytellerID {
			votes[p.ID] = p.VotedFor
		}
	}

	// Find storyteller's submission index in shuffled array
	storytellerCardIndex := -1
	for i, sub := range r.State.Submissions {
		if sub.PlayerID == r.State.StorytellerID {
			storytellerCardIndex = i
			break
		}
	}

	result := CalculateScores(r.State.StorytellerID, r.State.Submissions, votes, r.Players)
	result.StorytellerCardIndex = storytellerCardIndex

	ApplyScores(result, r.Players)

	r.State.TransitionTo(PhaseRoundEnd)
	return result
}

// CheckGameEnd checks if the game should end and returns the reason
// Returns (canContinue, endReason, winnerID)
func (r *Room) CheckGameEnd() (bool, GameEndReason, string) {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	// Check if any player has reached the winning score
	var winnerID string
	highestScore := 0
	for _, p := range r.Players {
		if p.Connected && p.Score >= r.Config.ScoreToWin {
			if p.Score > highestScore {
				highestScore = p.Score
				winnerID = p.ID
			}
		}
	}
	if winnerID != "" {
		return false, EndReasonScoreReached, winnerID
	}

	// Check if deck has enough cards to refill hands
	connectedPlayers := r.getConnectedPlayers()
	cardsNeeded := len(connectedPlayers) // Each player needs 1 card to refill

	if len(r.Deck) < cardsNeeded {
		// Find winner by highest score
		for _, p := range r.Players {
			if p.Connected && p.Score > highestScore {
				highestScore = p.Score
				winnerID = p.ID
			}
		}
		return false, EndReasonOutOfCards, winnerID
	}

	return true, EndReasonNone, ""
}

// StartNextRound begins the next round
func (r *Room) StartNextRound() error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.State.Phase != PhaseRoundEnd {
		return ErrWrongPhase
	}

	// Refill hands using config hand size
	handSize := r.Config.HandSize
	for _, p := range r.Players {
		if p.Connected {
			for len(p.Hand) < handSize && len(r.Deck) > 0 {
				p.Hand = append(p.Hand, r.Deck[0])
				r.Deck = r.Deck[1:]
			}
		}
	}

	r.State.TransitionTo(PhaseStorytellerClue)
	r.State.StartNewRound(r.getConnectedPlayers())

	return nil
}

// EndGame transitions to game end with reason
func (r *Room) EndGame(reason GameEndReason, winnerID string) {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	r.State.GameEndReason = reason
	r.State.WinnerID = winnerID
	r.State.TransitionTo(PhaseGameEnd)
}

// ReturnToLobby resets the room to lobby state
func (r *Room) ReturnToLobby() {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	r.State.ResetForNewGame()

	// Reset player scores
	for _, p := range r.Players {
		p.Score = 0
		p.Hand = nil
		p.SubmittedCard = nil
		p.VotedFor = -1
	}
}

// getPlayerByID gets player by ID (internal, no lock)
func (r *Room) getPlayerByID(playerID string) *Player {
	for _, p := range r.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

// Errors
type RoomError string

func (e RoomError) Error() string { return string(e) }

const (
	ErrRoomFull                RoomError = "room is full"
	ErrGameInProgress          RoomError = "game is already in progress"
	ErrNotEnoughPlayers        RoomError = "not enough players"
	ErrWrongPhase              RoomError = "wrong game phase"
	ErrNotStoryteller          RoomError = "you are not the storyteller"
	ErrStorytellerCannotSubmit RoomError = "storyteller cannot submit during this phase"
	ErrStorytellerCannotVote   RoomError = "storyteller cannot vote"
	ErrPlayerNotFound          RoomError = "player not found"
	ErrCardNotInHand           RoomError = "card not in hand"
	ErrAlreadySubmitted        RoomError = "already submitted a card"
	ErrAlreadyVoted            RoomError = "already voted"
	ErrInvalidSubmission       RoomError = "invalid submission index"
	ErrCannotVoteOwnCard       RoomError = "cannot vote for your own card"
	ErrRoomNotFound            RoomError = "room not found"
	ErrInvalidScoreToWin       RoomError = "scoreToWin must be between 10 and 100"
	ErrInvalidHandSize         RoomError = "handSize must be between 4 and 10"
	ErrNotHost                 RoomError = "only the host can change config"
	ErrInvalidReconnectToken   RoomError = "invalid reconnect token"
)

// UpdateConfig updates the room configuration (only in lobby by host)
func (r *Room) UpdateConfig(playerID string, config RoomConfig) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()

	if r.HostID != playerID {
		return ErrNotHost
	}

	if r.State.Phase != PhaseLobby {
		return ErrGameInProgress
	}

	if err := config.Validate(); err != nil {
		return err
	}

	r.Config = config
	return nil
}

// GetPlayerByToken finds a player by their reconnect token
func (r *Room) GetPlayerByToken(token string) *Player {
	r.Mu.RLock()
	defer r.Mu.RUnlock()

	for _, p := range r.Players {
		if p.ReconnectToken == token {
			return p
		}
	}
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
