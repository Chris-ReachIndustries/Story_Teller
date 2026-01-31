package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"dixit-backend/internal/cardgen"
	"dixit-backend/internal/cards"
)

// Handlers contains HTTP handlers
type Handlers struct {
	cardLoader    *cards.Loader
	cardRegistry  *cards.Registry
	cardGenerator *cardgen.Generator
}

// NewHandlers creates new handlers
func NewHandlers(cardLoader *cards.Loader, cardRegistry *cards.Registry, cardGenerator *cardgen.Generator) *Handlers {
	return &Handlers{
		cardLoader:    cardLoader,
		cardRegistry:  cardRegistry,
		cardGenerator: cardGenerator,
	}
}

// HealthHandler returns a simple health check
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// CardsHandler returns the list of all cards
func (h *Handlers) CardsHandler(w http.ResponseWriter, r *http.Request) {
	cardsData, err := h.cardLoader.GetCardsData()
	if err != nil {
		http.Error(w, "Failed to load cards", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cards": cardsData,
	})
}

// CardGenStatusHandler returns the card generation status
func (h *Handlers) CardGenStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := h.cardGenerator.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// CardSetsHandler returns the list of all available card sets
func (h *Handlers) CardSetsHandler(w http.ResponseWriter, r *http.Request) {
	sets, err := h.cardRegistry.ListSets()
	if err != nil {
		http.Error(w, "Failed to list card sets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sets": sets,
	})
}

// CardSetCardsHandler returns cards for a specific set
// URL: /api/card-sets/{setId}/cards
func (h *Handlers) CardSetCardsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract setId from path: /api/card-sets/{setId}/cards
	path := strings.TrimPrefix(r.URL.Path, "/api/card-sets/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "cards" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	setID := parts[0]

	if !h.cardRegistry.SetExists(setID) {
		http.Error(w, "Card set not found", http.StatusNotFound)
		return
	}

	cardsData, err := h.cardRegistry.GetSetCardsData(setID)
	if err != nil {
		http.Error(w, "Failed to load cards", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cards": cardsData,
		"total": len(cardsData),
	})
}
