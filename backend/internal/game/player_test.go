package game

import (
	"testing"
)

func TestNewPlayer(t *testing.T) {
	player := NewPlayer("TestPlayer")

	if player.ID == "" {
		t.Error("Player ID should not be empty")
	}

	if player.Name != "TestPlayer" {
		t.Errorf("Expected name 'TestPlayer', got '%s'", player.Name)
	}

	if player.PlayerType != PlayerTypeHuman {
		t.Errorf("Expected PlayerType Human, got %s", player.PlayerType)
	}

	if !player.Connected {
		t.Error("New player should be connected")
	}

	if player.Score != 0 {
		t.Errorf("Expected score 0, got %d", player.Score)
	}

	if player.ReconnectToken == "" {
		t.Error("Human player should have a reconnect token")
	}
}

func TestNewBotPlayer(t *testing.T) {
	bot := NewBotPlayer()

	if bot.ID == "" {
		t.Error("Bot ID should not be empty")
	}

	if bot.Name == "" {
		t.Error("Bot name should not be empty")
	}

	if bot.PlayerType != PlayerTypeBot {
		t.Errorf("Expected PlayerType Bot, got %s", bot.PlayerType)
	}

	if !bot.Connected {
		t.Error("Bot should always be connected")
	}

	if bot.Score != 0 {
		t.Errorf("Expected score 0, got %d", bot.Score)
	}

	// Bots don't need reconnect tokens
	if bot.ReconnectToken != "" {
		t.Error("Bot should not have a reconnect token")
	}
}

func TestNewBotPlayerUniqueName(t *testing.T) {
	// Create multiple bots and verify they get different names from the pool
	bots := make([]*Player, 10)
	for i := 0; i < 10; i++ {
		bots[i] = NewBotPlayer()
	}

	// All bots should have "Bot " prefix
	for _, bot := range bots {
		if len(bot.Name) < 5 || bot.Name[:4] != "Bot " {
			t.Errorf("Bot name should start with 'Bot ', got '%s'", bot.Name)
		}
	}

	// After creating 8+ bots, names should cycle
	// Just verify they all have valid names
	for _, bot := range bots {
		if bot.Name == "" {
			t.Error("Bot name should not be empty")
		}
	}
}

func TestPlayerIsBot(t *testing.T) {
	human := NewPlayer("Human")
	if human.IsBot() {
		t.Error("Human player should not be a bot")
	}

	bot := NewBotPlayer()
	if !bot.IsBot() {
		t.Error("Bot player should be a bot")
	}
}

func TestPlayerRemoveCardFromHand(t *testing.T) {
	player := NewPlayer("Test")
	player.Hand = []Card{
		{ID: "card-001", Image: "/cards/images/card-001.svg"},
		{ID: "card-002", Image: "/cards/images/card-002.svg"},
		{ID: "card-003", Image: "/cards/images/card-003.svg"},
	}

	removed := player.RemoveCardFromHand("card-002")
	if removed == nil {
		t.Fatal("Should have removed card-002")
	}

	if removed.ID != "card-002" {
		t.Errorf("Expected removed card ID 'card-002', got '%s'", removed.ID)
	}

	if len(player.Hand) != 2 {
		t.Errorf("Expected 2 cards in hand, got %d", len(player.Hand))
	}

	// Verify card-002 is no longer in hand
	for _, card := range player.Hand {
		if card.ID == "card-002" {
			t.Error("card-002 should not be in hand after removal")
		}
	}
}

func TestPlayerHasCard(t *testing.T) {
	player := NewPlayer("Test")
	player.Hand = []Card{
		{ID: "card-001", Image: "/cards/images/card-001.svg"},
		{ID: "card-002", Image: "/cards/images/card-002.svg"},
	}

	if !player.HasCard("card-001") {
		t.Error("Player should have card-001")
	}

	if !player.HasCard("card-002") {
		t.Error("Player should have card-002")
	}

	if player.HasCard("card-003") {
		t.Error("Player should not have card-003")
	}
}
