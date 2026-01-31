package ws

import (
	"encoding/json"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"dixit-backend/internal/ai"
	"dixit-backend/internal/cards"
	"dixit-backend/internal/cards/thumbs"
	"dixit-backend/internal/game"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients     map[*Client]bool
	roomClients map[string]map[*Client]bool // roomCode -> clients
	register    chan *Client
	unregister  chan *Client
	mu          sync.RWMutex

	roomManager   *game.RoomManager
	cardRegistry  *cards.Registry
	cardsBasePath string

	// Round results stored temporarily for state broadcasts
	roundResults map[string]*game.RoundResult // roomCode -> result

	// AI bot support
	aiClient     ai.Client
	thumbService *thumbs.Service
	aiEnabled    bool
}

// NewHub creates a new Hub
func NewHub(roomManager *game.RoomManager, cardRegistry *cards.Registry, cardsBasePath string, aiClient ai.Client, thumbService *thumbs.Service) *Hub {
	aiEnabled := aiClient != nil && thumbService != nil
	return &Hub{
		clients:       make(map[*Client]bool),
		roomClients:   make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		roomManager:   roomManager,
		cardRegistry:  cardRegistry,
		cardsBasePath: cardsBasePath,
		roundResults:  make(map[string]*game.RoundResult),
		aiClient:      aiClient,
		thumbService:  thumbService,
		aiEnabled:     aiEnabled,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.handleDisconnect(client)
		}
	}
}

// Register registers a client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// handleDisconnect handles client disconnection
func (h *Hub) handleDisconnect(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}

	roomCode := client.GetRoom()
	if roomCode != "" {
		if roomClients, ok := h.roomClients[roomCode]; ok {
			delete(roomClients, client)
		}
	}
	h.mu.Unlock()

	// Mark player as disconnected
	if player := client.GetPlayer(); player != nil && roomCode != "" {
		room := h.roomManager.GetRoom(roomCode)
		if room != nil {
			room.SetPlayerConnected(player.ID, false)

			// Start disconnect timer
			go h.startDisconnectTimer(room, player.ID)

			// Broadcast updated state
			h.broadcastRoomState(roomCode)
		}
	}
}

// startDisconnectTimer starts a timer to remove disconnected player
func (h *Hub) startDisconnectTimer(room *game.Room, playerID string) {
	time.Sleep(game.DisconnectTimeout)

	player := room.GetPlayer(playerID)
	if player != nil && !player.Connected {
		// Player didn't reconnect, handle based on game state
		if room.State.Phase != game.PhaseLobby {
			// Game in progress - end the game due to player disconnect
			// Find current leader as "winner"
			var winnerID string
			highestScore := 0
			for _, p := range room.Players {
				if p.Connected && p.Score > highestScore {
					highestScore = p.Score
					winnerID = p.ID
				}
			}
			room.EndGame(game.EndReasonPlayerDisconnected, winnerID)
			h.broadcastRoomState(room.Code)
		} else {
			// In lobby - just remove the player
			room.RemovePlayer(playerID)
			h.broadcastRoomState(room.Code)
		}
	}
}

// handleMessage handles an incoming message from a client
func (h *Hub) handleMessage(client *Client, msg *Message) {
	switch msg.Type {
	case MsgHello:
		h.handleHello(client, msg)
	case MsgCreateRoom:
		h.handleCreateRoom(client, msg)
	case MsgJoinRoom:
		h.handleJoinRoom(client, msg)
	case MsgStartGame:
		h.handleStartGame(client)
	case MsgStorySubmit:
		h.handleStorySubmit(client, msg)
	case MsgSubmitCard:
		h.handleSubmitCard(client, msg)
	case MsgVote:
		h.handleVote(client, msg)
	case MsgNextRound:
		h.handleNextRound(client)
	case MsgReturnToLobby:
		h.handleReturnToLobby(client)
	case MsgUpdateConfig:
		h.handleUpdateConfig(client, msg)
	case MsgAddBot:
		h.handleAddBot(client)
	case MsgPing:
		client.Send(&Message{Type: MsgPong, Payload: nil})
	case MsgReconnect:
		h.handleReconnect(client, msg)
	default:
		client.sendError("Unknown message type")
	}
}

// handleHello handles the hello message, including reconnection attempts
func (h *Hub) handleHello(client *Client, msg *Message) {
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload HelloPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid hello payload")
		return
	}

	// Try to reconnect if token and room code provided
	if payload.ReconnectToken != "" && payload.RoomCode != "" {
		room := h.roomManager.GetRoom(payload.RoomCode)
		if room != nil {
			player := room.GetPlayerByToken(payload.ReconnectToken)
			if player != nil {
				// Successful reconnection
				player.Connected = true
				player.UpdateLastSeen()
				client.SetPlayer(player)
				client.SetRoom(payload.RoomCode)

				h.mu.Lock()
				if h.roomClients[payload.RoomCode] == nil {
					h.roomClients[payload.RoomCode] = make(map[*Client]bool)
				}
				h.roomClients[payload.RoomCode][client] = true
				h.mu.Unlock()

				// Mark this as a reconnection (don't send token again)
				h.sendClientStateWithToken(client, room, false)
				h.broadcastRoomState(payload.RoomCode)
				return
			}
		}
		// Token invalid or room not found - fall through to create new player
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		client.sendError("Name cannot be empty")
		return
	}

	if len(name) > 20 {
		name = name[:20]
	}

	player := game.NewPlayer(name)
	client.SetPlayer(player)

	// Send initial state (not in a room yet)
	h.sendClientState(client, nil)
}

// handleCreateRoom handles room creation
func (h *Hub) handleCreateRoom(client *Client, msg *Message) {
	player := client.GetPlayer()
	if player == nil {
		client.sendError("Must send hello first")
		return
	}

	// Parse optional config from payload
	var config *game.RoomConfig
	if msg != nil && msg.Payload != nil {
		payloadBytes, err := json.Marshal(msg.Payload)
		if err == nil {
			var payload CreateRoomPayload
			if json.Unmarshal(payloadBytes, &payload) == nil && payload.Config != nil {
				config = payload.Config
			}
		}
	}

	room := h.roomManager.CreateRoom(player, config)
	client.SetRoom(room.Code)

	h.mu.Lock()
	if h.roomClients[room.Code] == nil {
		h.roomClients[room.Code] = make(map[*Client]bool)
	}
	h.roomClients[room.Code][client] = true
	clientCount := len(h.roomClients[room.Code])
	h.mu.Unlock()

	log.Printf("[CreateRoom] Player %s created room %s, now %d clients in room", player.Name, room.Code, clientCount)

	// Send state with reconnect token for the creator
	h.sendClientStateWithToken(client, room, true)
}

// handleJoinRoom handles joining a room
func (h *Hub) handleJoinRoom(client *Client, msg *Message) {
	player := client.GetPlayer()
	if player == nil {
		client.sendError("Must send hello first")
		return
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload JoinRoomPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid join_room payload")
		return
	}

	code := strings.ToUpper(strings.TrimSpace(payload.Code))
	room := h.roomManager.GetRoom(code)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	if err := room.AddPlayer(player); err != nil {
		client.sendError(err.Error())
		return
	}

	client.SetRoom(code)

	h.mu.Lock()
	if h.roomClients[code] == nil {
		h.roomClients[code] = make(map[*Client]bool)
	}
	h.roomClients[code][client] = true
	clientCount := len(h.roomClients[code])
	h.mu.Unlock()

	log.Printf("[JoinRoom] Player %s joined room %s, now %d clients in room", player.Name, code, clientCount)

	// Send state with reconnect token for the joiner
	h.sendClientStateWithToken(client, room, true)
	// Broadcast to others without token
	h.broadcastRoomStateExcept(code, client)
}

// handleStartGame handles starting the game
func (h *Hub) handleStartGame(client *Client) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	if room.HostID != player.ID {
		client.sendError("Only the host can start the game")
		return
	}

	log.Printf("[StartGame] Player %s starting game in room %s, current phase: %s", player.Name, roomCode, room.State.Phase)

	// Get the deck set from room config
	deckSetID := room.Config.DeckSetID
	if deckSetID == "" {
		deckSetID = "default"
	}

	// Check if the selected set still exists
	if !h.cardRegistry.SetExists(deckSetID) {
		client.sendError("Selected card set is no longer available. Please choose another set.")
		log.Printf("[StartGame] Card set '%s' not found", deckSetID)
		return
	}

	// Load cards from the selected set
	allCards, err := h.cardRegistry.GetSetCards(deckSetID)
	if err != nil {
		client.sendError("Failed to load cards from selected set")
		log.Printf("Failed to load cards from set '%s': %v", deckSetID, err)
		return
	}

	log.Printf("[StartGame] Loaded %d cards from set '%s'", len(allCards), deckSetID)

	if err := room.StartGame(allCards); err != nil {
		log.Printf("[StartGame] Error: %v", err)
		client.sendError(err.Error())
		return
	}

	log.Printf("[StartGame] Game started successfully, new phase: %s", room.State.Phase)
	h.processBotsAndBroadcast(roomCode)
}

// handleStorySubmit handles storyteller's clue and card
func (h *Hub) handleStorySubmit(client *Client, msg *Message) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload StorySubmitPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid story_submit payload")
		return
	}

	clue := strings.TrimSpace(payload.Clue)
	if clue == "" {
		client.sendError("Clue cannot be empty")
		return
	}

	if err := room.SubmitStorytellerClue(player.ID, clue, payload.CardID); err != nil {
		client.sendError(err.Error())
		return
	}

	// After storyteller submits, bots may need to submit cards
	h.processBotsAndBroadcast(roomCode)
}

// handleSubmitCard handles non-storyteller card submission
func (h *Hub) handleSubmitCard(client *Client, msg *Message) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload SubmitCardPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid submit_card payload")
		return
	}

	if err := room.SubmitCard(player.ID, payload.CardID); err != nil {
		client.sendError(err.Error())
		return
	}

	// After card submission, bots may need to submit or vote
	h.processBotsAndBroadcast(roomCode)
}

// handleVote handles voting
func (h *Hub) handleVote(client *Client, msg *Message) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload VotePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid vote payload")
		return
	}

	if err := room.Vote(player.ID, payload.SubmissionIndex); err != nil {
		client.sendError(err.Error())
		return
	}

	// Check if we need to calculate scores (human's vote may have been the last one)
	h.checkAndCalculateScores(room)

	// After vote, bots may need to vote
	h.processBotsAndBroadcast(roomCode)
}

// handleNextRound handles transitioning to the next round
func (h *Hub) handleNextRound(client *Client) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	if room.HostID != player.ID {
		client.sendError("Only the host can advance to next round")
		return
	}

	canContinue, endReason, winnerID := room.CheckGameEnd()
	if !canContinue {
		room.EndGame(endReason, winnerID)
	} else {
		if err := room.StartNextRound(); err != nil {
			client.sendError(err.Error())
			return
		}
	}

	// Clear round results
	h.mu.Lock()
	delete(h.roundResults, roomCode)
	h.mu.Unlock()

	// New round may have bot storyteller
	h.processBotsAndBroadcast(roomCode)
}

// handleReturnToLobby handles returning to lobby after game end
func (h *Hub) handleReturnToLobby(client *Client) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	if room.HostID != player.ID {
		client.sendError("Only the host can return to lobby")
		return
	}

	room.ReturnToLobby()

	// Clear round results
	h.mu.Lock()
	delete(h.roundResults, roomCode)
	h.mu.Unlock()

	h.broadcastRoomState(roomCode)
}

// handleReconnect handles reconnecting to a room (legacy - use hello with token instead)
func (h *Hub) handleReconnect(client *Client, msg *Message) {
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload ReconnectPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid reconnect payload")
		return
	}

	room := h.roomManager.GetRoom(payload.RoomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	player := room.GetPlayer(payload.PlayerID)
	if player == nil {
		client.sendError("Player not found in room")
		return
	}

	// Reconnect the player
	player.Connected = true
	player.UpdateLastSeen()
	client.SetPlayer(player)
	client.SetRoom(payload.RoomCode)

	h.mu.Lock()
	if h.roomClients[payload.RoomCode] == nil {
		h.roomClients[payload.RoomCode] = make(map[*Client]bool)
	}
	h.roomClients[payload.RoomCode][client] = true
	h.mu.Unlock()

	h.broadcastRoomState(payload.RoomCode)
}

// handleUpdateConfig handles config updates from host in lobby
func (h *Hub) handleUpdateConfig(client *Client, msg *Message) {
	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		client.sendError("Invalid payload")
		return
	}

	var payload UpdateConfigPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		client.sendError("Invalid update_config payload")
		return
	}

	if err := room.UpdateConfig(player.ID, payload.Config); err != nil {
		client.sendError(err.Error())
		return
	}

	h.broadcastRoomState(roomCode)
}

// handleAddBot handles adding an AI bot to the room (host only, lobby only)
func (h *Hub) handleAddBot(client *Client) {
	if !h.aiEnabled {
		client.sendError("AI bots are not enabled on this server")
		return
	}

	player := client.GetPlayer()
	roomCode := client.GetRoom()

	if player == nil || roomCode == "" {
		client.sendError("Not in a room")
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		client.sendError("Room not found")
		return
	}

	if room.HostID != player.ID {
		client.sendError("Only the host can add bots")
		return
	}

	if room.State.Phase != game.PhaseLobby {
		client.sendError("Can only add bots in the lobby")
		return
	}

	// Create and add bot player
	botPlayer := game.NewBotPlayer()
	if err := room.AddPlayer(botPlayer); err != nil {
		client.sendError(err.Error())
		return
	}

	log.Printf("[AddBot] Bot %s added to room %s, total players: %d", botPlayer.Name, roomCode, len(room.Players))
	h.broadcastRoomState(roomCode)
}

// broadcastRoomState sends the current state to all clients in a room
func (h *Hub) broadcastRoomState(roomCode string) {
	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		return
	}

	// Make a copy of the clients map while holding the lock
	// This prevents issues if the map is modified during iteration
	h.mu.RLock()
	originalClients := h.roomClients[roomCode]
	clients := make([]*Client, 0, len(originalClients))
	for c := range originalClients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		h.sendClientStateWithToken(client, room, false)
	}
}

// broadcastRoomStateExcept sends state to all clients except one
func (h *Hub) broadcastRoomStateExcept(roomCode string, excludeClient *Client) {
	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		return
	}

	// Make a copy of the clients while holding the lock
	h.mu.RLock()
	originalClients := h.roomClients[roomCode]
	clients := make([]*Client, 0, len(originalClients))
	for c := range originalClients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if client != excludeClient {
			h.sendClientStateWithToken(client, room, false)
		}
	}
}

// sendClientState sends personalized state to a client (without token)
func (h *Hub) sendClientState(client *Client, room *game.Room) {
	h.sendClientStateWithToken(client, room, false)
}

// sendClientStateWithToken sends personalized state with optional reconnect token
func (h *Hub) sendClientStateWithToken(client *Client, room *game.Room, includeToken bool) {
	player := client.GetPlayer()
	if player == nil {
		return
	}

	var state StatePayload

	if room == nil {
		// Not in a room yet
		state = StatePayload{
			Phase:   game.PhaseLobby,
			Players: []PlayerInfo{},
			Round:   0,
		}
		state.You = &YouInfo{
			ID:   player.ID,
			Name: player.Name,
		}
	} else {
		room.Mu.RLock()
		defer room.Mu.RUnlock()

		// Build player list
		players := make([]PlayerInfo, len(room.Players))
		for i, p := range room.Players {
			players[i] = PlayerInfo{
				ID:           p.ID,
				Name:         p.Name,
				PlayerType:   p.PlayerType,
				Connected:    p.Connected,
				Score:        p.Score,
				HasSubmitted: p.SubmittedCard != nil,
				HasVoted:     p.VotedFor >= 0,
			}
		}

		// Get winner name if there is one
		var winnerName string
		if room.State.WinnerID != "" {
			for _, p := range room.Players {
				if p.ID == room.State.WinnerID {
					winnerName = p.Name
					break
				}
			}
		}

		state = StatePayload{
			Room: &RoomInfo{
				Code:   room.Code,
				HostID: room.HostID,
				Config: room.Config,
			},
			You: &YouInfo{
				ID:            player.ID,
				Name:          player.Name,
				IsHost:        room.HostID == player.ID,
				IsStoryteller: room.State.StorytellerID == player.ID,
			},
			Phase:         room.State.Phase,
			Players:       players,
			Hand:          player.Hand,
			Clue:          room.State.Clue,
			StorytellerID: room.State.StorytellerID,
			Submissions:   room.State.ShuffledSubmissions,
			Round:         room.State.Round,
			HasSubmitted:  player.SubmittedCard != nil,
			HasVoted:      player.VotedFor >= 0,
			GameEndReason: room.State.GameEndReason,
			WinnerID:      room.State.WinnerID,
			WinnerName:    winnerName,
		}

		// Include reconnect token only on first join/create
		if includeToken {
			state.You.ReconnectToken = player.ReconnectToken
		}

		// During VOTING phase, find player's submission index
		if room.State.Phase == game.PhaseVoting {
			for i, sub := range room.State.Submissions {
				if sub.PlayerID == player.ID {
					idx := i
					state.YourSubmissionIndex = &idx
					break
				}
			}
		}

		// Add round results if in ROUND_END or GAME_END phase
		if room.State.Phase == game.PhaseRoundEnd || room.State.Phase == game.PhaseGameEnd {
			h.mu.RLock()
			if result, ok := h.roundResults[room.Code]; ok {
				state.RoundResults = result
			}
			h.mu.RUnlock()
		}
	}

	client.Send(&Message{
		Type:    MsgState,
		Payload: state,
	})
}

// processBotsAndBroadcast handles any pending bot actions, then broadcasts
// This is called after any human action that might require bot responses
func (h *Hub) processBotsAndBroadcast(roomCode string) {
	if !h.aiEnabled {
		h.broadcastRoomState(roomCode)
		return
	}

	room := h.roomManager.GetRoom(roomCode)
	if room == nil {
		return
	}

	// Process bot actions in the background, broadcasting after each
	go h.processBotActionsLoop(roomCode)

	// Broadcast current state immediately so humans see the update
	h.broadcastRoomState(roomCode)
}

// processBotActionsLoop processes all pending bot actions with delays between each
func (h *Hub) processBotActionsLoop(roomCode string) {
	for {
		room := h.roomManager.GetRoom(roomCode)
		if room == nil {
			return
		}

		acted := h.processSingleBotAction(room)
		if !acted {
			return // No more bots need to act
		}

		// Broadcast after bot action so humans see progress
		h.broadcastRoomState(roomCode)

		// Small delay before next bot action for natural feel
		time.Sleep(time.Duration(600+rand.Intn(600)) * time.Millisecond)
	}
}

// processSingleBotAction handles one bot action if any bot needs to act
// Returns true if a bot acted, false if no bots need to act
func (h *Hub) processSingleBotAction(room *game.Room) bool {
	switch room.State.Phase {
	case game.PhaseStorytellerClue:
		storyteller := room.GetPlayer(room.State.StorytellerID)
		if storyteller != nil && storyteller.IsBot() {
			return h.botStorytell(room, storyteller)
		}
	case game.PhaseSubmissions:
		for _, p := range room.Players {
			if p.IsBot() && p.ID != room.State.StorytellerID && p.SubmittedCard == nil {
				return h.botSubmitCard(room, p)
			}
		}
	case game.PhaseVoting:
		for _, p := range room.Players {
			if p.IsBot() && p.ID != room.State.StorytellerID && p.VotedFor < 0 {
				return h.botVote(room, p)
			}
		}
	}
	return false
}

// botStorytell has a bot submit a storyteller clue
func (h *Hub) botStorytell(room *game.Room, bot *game.Player) bool {
	cardSetID := room.Config.DeckSetID
	if cardSetID == "" {
		cardSetID = "default"
	}
	hand, err := h.getHandWithThumbs(bot, cardSetID)
	if err != nil {
		return h.botStorytellFallback(room, bot)
	}

	resp, err := h.aiClient.Storytell(hand)
	if err != nil || resp.SelectedCard < 1 || resp.SelectedCard > len(bot.Hand) || resp.Clue == "" {
		return h.botStorytellFallback(room, bot)
	}

	cardID := bot.Hand[resp.SelectedCard-1].ID
	if err := room.SubmitStorytellerClue(bot.ID, resp.Clue, cardID); err != nil {
		log.Printf("[Bot] ERROR: %s failed to submit clue: %v", bot.Name, err)
		return false
	}
	log.Printf("[Bot] %s → clue: \"%s\"", bot.Name, resp.Clue)
	return true
}

func (h *Hub) botStorytellFallback(room *game.Room, bot *game.Player) bool {
	if len(bot.Hand) == 0 {
		return false
	}
	cardIdx := rand.Intn(len(bot.Hand))
	clues := []string{"Mystery", "Journey", "Dreams", "Echoes", "Shadows"}
	clue := clues[rand.Intn(len(clues))]

	if err := room.SubmitStorytellerClue(bot.ID, clue, bot.Hand[cardIdx].ID); err != nil {
		log.Printf("[Bot] ERROR: %s fallback failed: %v", bot.Name, err)
		return false
	}
	log.Printf("[Bot] %s → clue: \"%s\" (fallback)", bot.Name, clue)
	return true
}

// botSubmitCard has a bot submit a card for the clue
func (h *Hub) botSubmitCard(room *game.Room, bot *game.Player) bool {
	cardSetID := room.Config.DeckSetID
	if cardSetID == "" {
		cardSetID = "default"
	}
	hand, err := h.getHandWithThumbs(bot, cardSetID)
	if err != nil {
		return h.botSubmitFallback(room, bot)
	}

	resp, err := h.aiClient.Submit(hand, room.State.Clue)
	if err != nil || resp.SelectedCard < 1 || resp.SelectedCard > len(bot.Hand) {
		return h.botSubmitFallback(room, bot)
	}

	cardID := bot.Hand[resp.SelectedCard-1].ID
	if err := room.SubmitCard(bot.ID, cardID); err != nil {
		log.Printf("[Bot] ERROR: %s failed to submit: %v", bot.Name, err)
		return false
	}
	log.Printf("[Bot] %s → submitted card %d", bot.Name, resp.SelectedCard)
	return true
}

func (h *Hub) botSubmitFallback(room *game.Room, bot *game.Player) bool {
	if len(bot.Hand) == 0 {
		return false
	}
	cardIdx := rand.Intn(len(bot.Hand))
	if err := room.SubmitCard(bot.ID, bot.Hand[cardIdx].ID); err != nil {
		log.Printf("[Bot] ERROR: %s fallback submit failed: %v", bot.Name, err)
		return false
	}
	log.Printf("[Bot] %s → submitted card (fallback)", bot.Name)
	return true
}

// botVote has a bot vote for a card
func (h *Hub) botVote(room *game.Room, bot *game.Player) bool {
	cardSetID := room.Config.DeckSetID
	if cardSetID == "" {
		cardSetID = "default"
	}

	// Find bot's own submission index (can't vote for self)
	ownIndex := -1
	for i, sub := range room.State.Submissions {
		if sub.PlayerID == bot.ID {
			ownIndex = i + 1 // 1-indexed for AI
			break
		}
	}

	thumbs, err := h.getSubmissionThumbs(room.State.Submissions, cardSetID)
	if err != nil {
		return h.botVoteFallback(room, bot, ownIndex)
	}

	resp, err := h.aiClient.Vote(thumbs, room.State.Clue, ownIndex)
	if err != nil || resp.SelectedCard < 1 || resp.SelectedCard > len(room.State.Submissions) || resp.SelectedCard == ownIndex {
		return h.botVoteFallback(room, bot, ownIndex)
	}

	// Convert to 0-indexed
	if err := room.Vote(bot.ID, resp.SelectedCard-1); err != nil {
		log.Printf("[Bot] ERROR: %s failed to vote: %v", bot.Name, err)
		return false
	}
	log.Printf("[Bot] %s → voted for card %d", bot.Name, resp.SelectedCard)

	// Check if we need to calculate scores (bot's vote may have been the last one)
	h.checkAndCalculateScores(room)

	return true
}

func (h *Hub) botVoteFallback(room *game.Room, bot *game.Player, ownIndex int) bool {
	// Pick a random valid index (not own card)
	validIndices := make([]int, 0)
	for i := range room.State.Submissions {
		if i+1 != ownIndex { // ownIndex is 1-indexed
			validIndices = append(validIndices, i)
		}
	}
	if len(validIndices) == 0 {
		return false
	}
	voteIdx := validIndices[rand.Intn(len(validIndices))]
	if err := room.Vote(bot.ID, voteIdx); err != nil {
		log.Printf("[Bot] ERROR: %s fallback vote failed: %v", bot.Name, err)
		return false
	}
	log.Printf("[Bot] %s → voted for card %d (fallback)", bot.Name, voteIdx+1)

	// Check if we need to calculate scores (bot's vote may have been the last one)
	h.checkAndCalculateScores(room)

	return true
}

// checkAndCalculateScores calculates round scores if the phase just transitioned to SCORING
func (h *Hub) checkAndCalculateScores(room *game.Room) {
	if room.State.Phase == game.PhaseScoring {
		result := room.CalculateRoundScores()
		h.mu.Lock()
		h.roundResults[room.Code] = result
		h.mu.Unlock()
	}
}

// getHandWithThumbs converts a player's hand to cards with thumbnails
func (h *Hub) getHandWithThumbs(player *game.Player, cardSetID string) ([]ai.CardWithThumb, error) {
	result := make([]ai.CardWithThumb, len(player.Hand))
	for i, card := range player.Hand {
		dataURL, err := h.thumbService.GetCardThumbDataURL(card.ID, cardSetID)
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
func (h *Hub) getSubmissionThumbs(submissions []game.Submission, cardSetID string) ([]ai.CardWithThumb, error) {
	result := make([]ai.CardWithThumb, len(submissions))
	for i, sub := range submissions {
		dataURL, err := h.thumbService.GetCardThumbDataURL(sub.Card.ID, cardSetID)
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
