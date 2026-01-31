package bot

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"dixit-backend/internal/ai"
	"dixit-backend/internal/cards/thumbs"
	"dixit-backend/internal/game"
)

const (
	maxRetries     = 3
	minJitterMs    = 600
	maxJitterMs    = 1200
	retryDelayBase = 500 * time.Millisecond
)

// Controller manages bot actions for a room
type Controller struct {
	room         *game.Room
	aiClient     ai.Client
	thumbService *thumbs.Service

	mu         sync.Mutex
	processing bool
	rng        *rand.Rand
}

// NewController creates a new bot controller for a room
func NewController(room *game.Room, aiClient ai.Client, thumbService *thumbs.Service) *Controller {
	return &Controller{
		room:         room,
		aiClient:     aiClient,
		thumbService: thumbService,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// OnPhaseChange handles phase transitions and triggers bot actions
// Returns a channel that closes when all bot actions are complete
func (c *Controller) OnPhaseChange(phase game.GamePhase) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		c.mu.Lock()
		if c.processing {
			c.mu.Unlock()
			return // Already processing, skip
		}
		c.processing = true
		c.mu.Unlock()

		defer func() {
			c.mu.Lock()
			c.processing = false
			c.mu.Unlock()
		}()

		switch phase {
		case game.PhaseStorytellerClue:
			c.handleStorytellerPhase()
		case game.PhaseSubmissions:
			c.handleSubmissionsPhase()
		case game.PhaseVoting:
			c.handleVotingPhase()
		}
	}()

	return done
}

// handleStorytellerPhase handles bot storyteller actions
func (c *Controller) handleStorytellerPhase() {
	c.room.Mu.RLock()
	storytellerID := c.room.State.StorytellerID
	storyteller := c.room.GetPlayer(storytellerID)
	c.room.Mu.RUnlock()

	if storyteller == nil || !storyteller.IsBot() {
		return // Human storyteller
	}

	c.jitterDelay()

	// Get hand with thumbnails
	hand, err := c.getHandWithThumbs(storyteller)
	if err != nil {
		log.Printf("Bot controller: failed to get hand thumbs: %v", err)
		c.fallbackStorytell(storyteller)
		return
	}

	// Try AI with retries
	var response *ai.StorytellerResponse
	for attempt := 0; attempt < maxRetries; attempt++ {
		response, err = c.aiClient.Storytell(hand)
		if err == nil && c.validateStorytellerResponse(response, len(hand)) {
			break
		}
		if err != nil {
			log.Printf("Bot controller: storytell attempt %d failed: %v", attempt+1, err)
		}
		if attempt < maxRetries-1 {
			time.Sleep(retryDelayBase * time.Duration(attempt+1))
		}
	}

	if response == nil || !c.validateStorytellerResponse(response, len(hand)) {
		log.Printf("Bot controller: storytell failed after %d attempts, using fallback", maxRetries)
		c.fallbackStorytell(storyteller)
		return
	}

	// Submit the storyteller's clue and card
	cardID := storyteller.Hand[response.SelectedCard-1].ID
	if err := c.room.SubmitStorytellerClue(storytellerID, response.Clue, cardID); err != nil {
		log.Printf("Bot controller: failed to submit storyteller clue: %v", err)
	}
}

// handleSubmissionsPhase handles bot card submissions
func (c *Controller) handleSubmissionsPhase() {
	c.room.Mu.RLock()
	bots := c.getBotsNeedingSubmission()
	clue := c.room.State.Clue
	c.room.Mu.RUnlock()

	for _, bot := range bots {
		c.jitterDelay()
		c.submitBotCard(bot, clue)
	}
}

// handleVotingPhase handles bot voting
func (c *Controller) handleVotingPhase() {
	c.room.Mu.RLock()
	bots := c.getBotsNeedingVote()
	clue := c.room.State.Clue
	submissions := c.room.State.Submissions // Use full submissions, not shuffled
	c.room.Mu.RUnlock()

	for _, bot := range bots {
		c.jitterDelay()
		c.submitBotVote(bot, clue, submissions)
	}
}

// getBotsNeedingSubmission returns bots that haven't submitted yet (excluding storyteller)
func (c *Controller) getBotsNeedingSubmission() []*game.Player {
	var bots []*game.Player
	for _, p := range c.room.Players {
		if p.IsBot() && p.ID != c.room.State.StorytellerID && p.SubmittedCard == nil {
			bots = append(bots, p)
		}
	}
	return bots
}

// getBotsNeedingVote returns bots that haven't voted yet (excluding storyteller)
func (c *Controller) getBotsNeedingVote() []*game.Player {
	var bots []*game.Player
	for _, p := range c.room.Players {
		if p.IsBot() && p.ID != c.room.State.StorytellerID && p.VotedFor < 0 {
			bots = append(bots, p)
		}
	}
	return bots
}

// submitBotCard submits a card for a bot player
func (c *Controller) submitBotCard(bot *game.Player, clue string) {
	hand, err := c.getHandWithThumbs(bot)
	if err != nil {
		log.Printf("Bot controller: failed to get hand thumbs for %s: %v", bot.Name, err)
		c.fallbackSubmit(bot)
		return
	}

	var response *ai.SubmitResponse
	for attempt := 0; attempt < maxRetries; attempt++ {
		response, err = c.aiClient.Submit(hand, clue)
		if err == nil && c.validateSubmitResponse(response, len(hand)) {
			break
		}
		if err != nil {
			log.Printf("Bot controller: submit attempt %d failed for %s: %v", attempt+1, bot.Name, err)
		}
		if attempt < maxRetries-1 {
			time.Sleep(retryDelayBase * time.Duration(attempt+1))
		}
	}

	if response == nil || !c.validateSubmitResponse(response, len(hand)) {
		log.Printf("Bot controller: submit failed for %s after %d attempts, using fallback", bot.Name, maxRetries)
		c.fallbackSubmit(bot)
		return
	}

	cardID := bot.Hand[response.SelectedCard-1].ID
	if err := c.room.SubmitCard(bot.ID, cardID); err != nil {
		log.Printf("Bot controller: failed to submit card for %s: %v", bot.Name, err)
	}
}

// submitBotVote submits a vote for a bot player
func (c *Controller) submitBotVote(bot *game.Player, clue string, submissions []game.Submission) {
	// Find the bot's own submission index
	ownIndex := -1
	for i, sub := range submissions {
		if sub.PlayerID == bot.ID {
			ownIndex = i + 1 // 1-indexed
			break
		}
	}

	// Get submission thumbnails
	thumbs, err := c.getSubmissionThumbs(submissions)
	if err != nil {
		log.Printf("Bot controller: failed to get submission thumbs for %s: %v", bot.Name, err)
		c.fallbackVote(bot, ownIndex, len(submissions))
		return
	}

	var response *ai.VoteResponse
	for attempt := 0; attempt < maxRetries; attempt++ {
		response, err = c.aiClient.Vote(thumbs, clue, ownIndex)
		if err == nil && c.validateVoteResponse(response, len(submissions), ownIndex) {
			break
		}
		if err != nil {
			log.Printf("Bot controller: vote attempt %d failed for %s: %v", attempt+1, bot.Name, err)
		}
		if attempt < maxRetries-1 {
			time.Sleep(retryDelayBase * time.Duration(attempt+1))
		}
	}

	if response == nil || !c.validateVoteResponse(response, len(submissions), ownIndex) {
		log.Printf("Bot controller: vote failed for %s after %d attempts, using fallback", bot.Name, maxRetries)
		c.fallbackVote(bot, ownIndex, len(submissions))
		return
	}

	// Convert to 0-indexed for the game
	if err := c.room.Vote(bot.ID, response.SelectedCard-1); err != nil {
		log.Printf("Bot controller: failed to submit vote for %s: %v", bot.Name, err)
	}
}

// getHandWithThumbs converts a player's hand to cards with thumbnails
func (c *Controller) getHandWithThumbs(player *game.Player) ([]ai.CardWithThumb, error) {
	result := make([]ai.CardWithThumb, len(player.Hand))
	for i, card := range player.Hand {
		dataURL, err := c.thumbService.GetCardThumbDataURL(card.ID)
		if err != nil {
			return nil, err
		}
		result[i] = ai.CardWithThumb{
			CardID:  card.ID,
			DataURL: dataURL,
		}
	}
	return result, nil
}

// getSubmissionThumbs converts submissions to cards with thumbnails
func (c *Controller) getSubmissionThumbs(submissions []game.Submission) ([]ai.CardWithThumb, error) {
	result := make([]ai.CardWithThumb, len(submissions))
	for i, sub := range submissions {
		dataURL, err := c.thumbService.GetCardThumbDataURL(sub.Card.ID)
		if err != nil {
			return nil, err
		}
		result[i] = ai.CardWithThumb{
			CardID:  sub.Card.ID,
			DataURL: dataURL,
		}
	}
	return result, nil
}

// Validation helpers
func (c *Controller) validateStorytellerResponse(r *ai.StorytellerResponse, handSize int) bool {
	return r != nil && r.SelectedCard >= 1 && r.SelectedCard <= handSize && r.Clue != ""
}

func (c *Controller) validateSubmitResponse(r *ai.SubmitResponse, handSize int) bool {
	return r != nil && r.SelectedCard >= 1 && r.SelectedCard <= handSize
}

func (c *Controller) validateVoteResponse(r *ai.VoteResponse, numSubmissions int, ownIndex int) bool {
	return r != nil && r.SelectedCard >= 1 && r.SelectedCard <= numSubmissions && r.SelectedCard != ownIndex
}

// Fallback actions (random selection)
func (c *Controller) fallbackStorytell(player *game.Player) {
	if len(player.Hand) == 0 {
		return
	}
	cardIdx := c.rng.Intn(len(player.Hand))
	clues := []string{"Mystery", "Journey", "Dreams", "Echoes", "Shadows"}
	clue := clues[c.rng.Intn(len(clues))]

	if err := c.room.SubmitStorytellerClue(player.ID, clue, player.Hand[cardIdx].ID); err != nil {
		log.Printf("Bot controller: fallback storytell failed: %v", err)
	}
}

func (c *Controller) fallbackSubmit(player *game.Player) {
	if len(player.Hand) == 0 {
		return
	}
	cardIdx := c.rng.Intn(len(player.Hand))
	if err := c.room.SubmitCard(player.ID, player.Hand[cardIdx].ID); err != nil {
		log.Printf("Bot controller: fallback submit failed: %v", err)
	}
}

func (c *Controller) fallbackVote(player *game.Player, ownIndex int, numSubmissions int) {
	// Pick a random valid index (not own card)
	validIndices := make([]int, 0, numSubmissions-1)
	for i := 0; i < numSubmissions; i++ {
		if i+1 != ownIndex { // ownIndex is 1-indexed
			validIndices = append(validIndices, i)
		}
	}
	if len(validIndices) == 0 {
		return
	}
	voteIdx := validIndices[c.rng.Intn(len(validIndices))]
	if err := c.room.Vote(player.ID, voteIdx); err != nil {
		log.Printf("Bot controller: fallback vote failed: %v", err)
	}
}

// jitterDelay adds a random delay before bot actions for natural feel
func (c *Controller) jitterDelay() {
	delay := minJitterMs + c.rng.Intn(maxJitterMs-minJitterMs)
	time.Sleep(time.Duration(delay) * time.Millisecond)
}
