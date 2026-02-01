package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cardset-studio/internal/concepts"
	"cardset-studio/internal/images"
	"cardset-studio/internal/storage"

	"github.com/go-chi/chi/v5"
)

const maxImageRetries = 3 // Maximum retry attempts per image

// Handler implements the API endpoints
type Handler struct {
	store      *storage.Store
	conceptGen *concepts.Generator
	imageGen   *images.Generator
	exportDir  string

	// Generation state
	genMu       sync.Mutex
	genCancel   map[string]chan struct{} // setID -> cancel channel
	genProgress map[string]int           // setID -> current card index
}

// NewHandler creates a new API handler
func NewHandler(store *storage.Store, conceptGen *concepts.Generator, imageGen *images.Generator, exportDir string) *Handler {
	return &Handler{
		store:       store,
		conceptGen:  conceptGen,
		imageGen:    imageGen,
		exportDir:   exportDir,
		genCancel:   make(map[string]chan struct{}),
		genProgress: make(map[string]int),
	}
}

// JSON helper
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error helper
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// HealthCheck returns API status
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	sdAvailable := h.imageGen.IsAvailable()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":              "ok",
		"stableDiffusionReady": sdAvailable,
	})
}

// ListSets returns all card sets
func (h *Handler) ListSets(w http.ResponseWriter, r *http.Request) {
	sets := h.store.ListSets()
	writeJSON(w, http.StatusOK, sets)
}

// CreateSet creates a new card set
func (h *Handler) CreateSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Theme     string `json:"theme"`
		CardCount int    `json:"cardCount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Theme == "" {
		writeError(w, http.StatusBadRequest, "theme is required")
		return
	}
	if req.CardCount <= 0 {
		req.CardCount = 100
	}

	set, err := h.store.CreateSet(req.Name, req.Theme, req.CardCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, set)
}

// GetSet returns a specific card set
func (h *Handler) GetSet(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Add generation progress if active
	h.genMu.Lock()
	if progress, ok := h.genProgress[setID]; ok {
		set.Stats.Generated = progress
	}
	h.genMu.Unlock()

	writeJSON(w, http.StatusOK, set)
}

// DeleteSet removes a card set
func (h *Handler) DeleteSet(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	// Cancel any ongoing generation
	h.genMu.Lock()
	if cancel, ok := h.genCancel[setID]; ok {
		close(cancel)
		delete(h.genCancel, setID)
	}
	h.genMu.Unlock()

	if err := h.store.DeleteSet(setID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateConcepts creates card concepts for a set
func (h *Handler) GenerateConcepts(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Generate concepts
	conceptList, err := h.conceptGen.GenerateConcepts(set.Theme, set.CardCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate concepts: %v", err))
		return
	}

	// Convert to storage format
	set.Concepts = make([]storage.Concept, len(conceptList))
	for i, c := range conceptList {
		set.Concepts[i] = storage.Concept{
			CardID: c.CardID,
			Title:  c.Title,
			Prompt: c.Prompt,
			Tags:   c.Tags,
			Status: storage.StatusConceptOnly,
		}
	}

	set.Status = storage.SetStatusConcepts
	h.store.UpdateStats(set)

	if err := h.store.UpdateSet(set); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, set)
}

// UpdateConcept modifies a specific concept
func (h *Handler) UpdateConcept(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")
	cardID := chi.URLParam(r, "cardID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		Title  string   `json:"title"`
		Prompt string   `json:"prompt"`
		Tags   []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Find and update the concept
	found := false
	for i := range set.Concepts {
		if set.Concepts[i].CardID == cardID {
			if req.Title != "" {
				set.Concepts[i].Title = req.Title
			}
			if req.Prompt != "" {
				set.Concepts[i].Prompt = req.Prompt
			}
			if len(req.Tags) > 0 {
				set.Concepts[i].Tags = req.Tags
			}
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "concept not found")
		return
	}

	if err := h.store.UpdateSet(set); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, set)
}

// RegenerateConceptText regenerates concept title/prompt using AI
func (h *Handler) RegenerateConceptText(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")
	cardID := chi.URLParam(r, "cardID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req struct {
		RegenerateTitle  bool `json:"regenerateTitle"`
		RegeneratePrompt bool `json:"regeneratePrompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !req.RegenerateTitle && !req.RegeneratePrompt {
		writeError(w, http.StatusBadRequest, "must specify regenerateTitle or regeneratePrompt")
		return
	}

	// Find the concept
	var conceptIdx int = -1
	for i := range set.Concepts {
		if set.Concepts[i].CardID == cardID {
			conceptIdx = i
			break
		}
	}

	if conceptIdx == -1 {
		writeError(w, http.StatusNotFound, "concept not found")
		return
	}

	concept := &set.Concepts[conceptIdx]

	// Regenerate based on request
	if req.RegenerateTitle && req.RegeneratePrompt {
		// Regenerate both
		newConcept, err := h.conceptGen.RegenerateSingleConcept(set.Theme, concept.Title)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to regenerate concept: %v", err))
			return
		}
		concept.Title = newConcept.Title
		concept.Prompt = newConcept.Prompt
		concept.Tags = newConcept.Tags
	} else if req.RegenerateTitle {
		// Regenerate just title
		newTitle, err := h.conceptGen.RegenerateTitle(set.Theme, concept.Prompt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to regenerate title: %v", err))
			return
		}
		concept.Title = newTitle
	} else if req.RegeneratePrompt {
		// Regenerate just prompt
		newPrompt, err := h.conceptGen.RegenerateConceptPrompt(concept.Title, set.Theme)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to regenerate prompt: %v", err))
			return
		}
		concept.Prompt = newPrompt
	}

	if err := h.store.UpdateSet(set); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, concept)
}

// StartGeneration begins image generation for a set
func (h *Handler) StartGeneration(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if len(set.Concepts) == 0 {
		writeError(w, http.StatusBadRequest, "no concepts to generate - run concept generation first")
		return
	}

	// Parse request for quality mode
	var req struct {
		QualityMode string `json:"qualityMode"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Set quality mode (default to high if not specified)
	if req.QualityMode != "" {
		set.QualityMode = storage.QualityMode(req.QualityMode)
	} else if set.QualityMode == "" {
		set.QualityMode = storage.QualityHigh
	}

	// Check if generation is already running
	h.genMu.Lock()
	if _, ok := h.genCancel[setID]; ok {
		h.genMu.Unlock()
		writeError(w, http.StatusConflict, "generation already in progress")
		return
	}

	// Create cancel channel
	cancel := make(chan struct{})
	h.genCancel[setID] = cancel
	h.genProgress[setID] = 0
	h.genMu.Unlock()

	// Start generation in background
	go h.runGeneration(set, cancel)

	set.Status = storage.SetStatusGenerating
	h.store.UpdateSet(set)

	writeJSON(w, http.StatusOK, map[string]string{"status": "generation started"})
}

// runGeneration generates images in the background with theme-aware prompts
func (h *Handler) runGeneration(set *storage.CardSet, cancel chan struct{}) {
	defer func() {
		h.genMu.Lock()
		delete(h.genCancel, set.ID)
		delete(h.genProgress, set.ID)
		h.genMu.Unlock()
	}()

	fmt.Println("========================================")
	fmt.Printf("[PHASE: PREPARATION] Set: %s\n", set.Name)
	fmt.Printf("[PHASE: PREPARATION] Theme: %s\n", set.Theme)
	fmt.Printf("[PHASE: PREPARATION] Cards to generate: %d\n", len(set.Concepts))
	fmt.Println("========================================")

	// Derive theme style if not already set (for visual coherence across set)
	if set.ThemeStyle == "" {
		fmt.Println("[PHASE: THEME ANALYSIS] Deriving art style from theme...")
		themeStyle, err := h.conceptGen.DeriveThemeStyle(set.Theme)
		if err != nil {
			fmt.Printf("[PHASE: THEME ANALYSIS] Warning: %v, using fallback style\n", err)
			themeStyle = "fantasy illustration, detailed art, atmospheric lighting, painterly style"
		}
		set.ThemeStyle = themeStyle
		h.store.UpdateSet(set)
		fmt.Printf("[PHASE: THEME ANALYSIS] Derived style: %s\n", themeStyle)
	} else {
		fmt.Printf("[PHASE: THEME ANALYSIS] Using existing style: %s\n", set.ThemeStyle)
	}

	// Get quality-adjusted settings
	sdSettings := storage.GetSettingsForMode(set.QualityMode)
	if set.QualityMode == "" {
		set.QualityMode = storage.QualityHigh // Default to high quality
	}
	fmt.Printf("[PHASE: PREPARATION] Quality mode: %s (Steps: %d, Resolution: %dx%d)\n",
		set.QualityMode, sdSettings.Steps, sdSettings.Width, sdSettings.Height)

	// Create prompt builder for enhanced prompts
	promptBuilder := images.NewPromptBuilder()

	fmt.Println("========================================")
	fmt.Println("[PHASE: IMAGE GENERATION] Starting image generation...")
	fmt.Println("========================================")

	successCount := 0
	failCount := 0

	for i, concept := range set.Concepts {
		select {
		case <-cancel:
			fmt.Println("[PHASE: CANCELLED] Generation stopped by user")
			set.Status = storage.SetStatusPaused
			h.store.UpdateSet(set)
			return
		default:
		}

		// Skip already generated cards
		imagePath := h.store.GetImagePath(set.ID, concept.CardID)
		if _, err := os.Stat(imagePath); err == nil {
			fmt.Printf("[PHASE: IMAGE GENERATION] %d/%d: %s - Skipping (already exists)\n",
				i+1, len(set.Concepts), concept.Title)
			h.genMu.Lock()
			h.genProgress[set.ID] = i + 1
			h.genMu.Unlock()
			successCount++
			continue
		}

		fmt.Printf("[PHASE: IMAGE GENERATION] %d/%d: %s\n", i+1, len(set.Concepts), concept.Title)

		// Build enhanced prompt with theme context
		enhancedPrompt := promptBuilder.Build(concept.Prompt, set.Theme, set.ThemeStyle)

		// Build generation settings
		settings := images.GenerationSettings{
			Prompt:         enhancedPrompt,
			NegativePrompt: promptBuilder.BuildNegativePrompt(),
			Steps:          sdSettings.Steps,
			CFGScale:       sdSettings.CFGScale,
			Width:          sdSettings.Width,
			Height:         sdSettings.Height,
			Sampler:        sdSettings.Sampler,
			Model:          sdSettings.Model,
			Seed:           -1,
		}

		// Generate with retry logic
		err := h.generateWithRetry(&concept, settings, imagePath)
		if err != nil {
			fmt.Printf("[PHASE: IMAGE GENERATION] FAILED: %s - %v\n", concept.CardID, err)
			failCount++
		} else {
			successCount++
			set.Concepts[i].Status = storage.StatusGenerated
		}

		// Update progress
		h.genMu.Lock()
		h.genProgress[set.ID] = i + 1
		h.genMu.Unlock()

		h.store.UpdateStats(set)
		h.store.UpdateSet(set)
	}

	fmt.Println("========================================")
	fmt.Printf("[PHASE: COMPLETE] Generated: %d, Failed: %d, Total: %d\n",
		successCount, failCount, len(set.Concepts))
	fmt.Println("========================================")

	set.Status = storage.SetStatusReviewing
	h.store.UpdateSet(set)
}

// generateWithRetry attempts to generate an image with retry logic
func (h *Handler) generateWithRetry(concept *storage.Concept, settings images.GenerationSettings, imagePath string) error {
	var lastErr error

	for attempt := 1; attempt <= maxImageRetries; attempt++ {
		fmt.Printf("  [Attempt %d/%d] Generating...\n", attempt, maxImageRetries)

		// Use different seed on retry
		if attempt > 1 {
			settings.Seed = time.Now().UnixNano() % 1000000000
			fmt.Printf("  [Attempt %d/%d] Retrying with seed %d\n", attempt, maxImageRetries, settings.Seed)
		}

		err := h.imageGen.Generate(settings.Prompt, settings, imagePath)
		if err == nil {
			fmt.Printf("  [Attempt %d/%d] Success!\n", attempt, maxImageRetries)
			return nil
		}

		lastErr = err
		fmt.Printf("  [Attempt %d/%d] Failed: %v\n", attempt, maxImageRetries, err)

		// Brief pause before retry
		if attempt < maxImageRetries {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("all %d attempts failed: %w", maxImageRetries, lastErr)
}

// PauseGeneration stops ongoing generation
func (h *Handler) PauseGeneration(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	h.genMu.Lock()
	cancel, ok := h.genCancel[setID]
	h.genMu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "no generation in progress")
		return
	}

	close(cancel)
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

// CancelGeneration stops and cancels ongoing generation
func (h *Handler) CancelGeneration(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	h.genMu.Lock()
	cancel, ok := h.genCancel[setID]
	h.genMu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "no generation in progress")
		return
	}

	close(cancel)

	// Update set status
	set, err := h.store.GetSet(setID)
	if err == nil {
		set.Status = storage.SetStatusConcepts // Go back to concepts status
		h.store.UpdateSet(set)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// GetProgress returns the current generation progress
func (h *Handler) GetProgress(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	h.genMu.Lock()
	progress, isRunning := h.genProgress[setID]
	_, hasCancel := h.genCancel[setID]
	h.genMu.Unlock()

	// Determine status
	var status string
	switch {
	case isRunning && hasCancel:
		status = "running"
	case set.Status == storage.SetStatusPaused:
		status = "paused"
	case set.Status == storage.SetStatusGenerating && !hasCancel:
		// Was generating but cancel channel closed (paused/cancelled)
		status = "paused"
	case set.Status == storage.SetStatusReviewing || set.Status == storage.SetStatusComplete:
		status = "complete"
		progress = len(set.Concepts)
	default:
		status = "idle"
		// Only show generated count if we've actually been through image generation
		// If still in concepts phase, show 0
		if set.Status == storage.SetStatusConcepts || set.Status == storage.SetStatusNew {
			progress = 0
		} else {
			progress = set.Stats.Generated
		}
	}

	// Find current card being generated
	var currentCard string
	if isRunning && progress < len(set.Concepts) {
		currentCard = set.Concepts[progress].CardID
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    status,
		"completed": progress,
		"total":     len(set.Concepts),
		"current":   currentCard,
	})
}

// ListCards returns all cards in a set
func (h *Handler) ListCards(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, set.Concepts)
}

// GetCardImage serves a card's image
func (h *Handler) GetCardImage(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")
	cardID := chi.URLParam(r, "cardID")

	imagePath := h.store.GetImagePath(setID, cardID)
	http.ServeFile(w, r, imagePath)
}

// ApproveCard marks a card as approved
func (h *Handler) ApproveCard(w http.ResponseWriter, r *http.Request) {
	h.setCardStatus(w, r, storage.StatusApproved)
}

// RejectCard marks a card as rejected
func (h *Handler) RejectCard(w http.ResponseWriter, r *http.Request) {
	h.setCardStatus(w, r, storage.StatusRejected)
}

func (h *Handler) setCardStatus(w http.ResponseWriter, r *http.Request, status storage.CardStatus) {
	setID := chi.URLParam(r, "setID")
	cardID := chi.URLParam(r, "cardID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	found := false
	for i := range set.Concepts {
		if set.Concepts[i].CardID == cardID {
			set.Concepts[i].Status = status
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}

	h.store.UpdateStats(set)
	if err := h.store.UpdateSet(set); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, set)
}

// RegenerateCard creates a new image for a card
func (h *Handler) RegenerateCard(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")
	cardID := chi.URLParam(r, "cardID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Optionally update the prompt
	var req struct {
		Prompt string `json:"prompt"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var concept *storage.Concept
	for i := range set.Concepts {
		if set.Concepts[i].CardID == cardID {
			concept = &set.Concepts[i]
			if req.Prompt != "" {
				concept.Prompt = req.Prompt
			}
			break
		}
	}

	if concept == nil {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}

	// Generate new image
	settings := images.GenerationSettings{
		Prompt:         concept.Prompt,
		NegativePrompt: set.SDSettings.NegativePrompt,
		Steps:          set.SDSettings.Steps,
		CFGScale:       set.SDSettings.CFGScale,
		Width:          set.SDSettings.Width,
		Height:         set.SDSettings.Height,
		Sampler:        set.SDSettings.Sampler,
		Seed:           -1,
	}

	imagePath := h.store.GetImagePath(setID, cardID)
	if err := h.imageGen.Generate(concept.Prompt, settings, imagePath); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("generation failed: %v", err))
		return
	}

	concept.Status = storage.StatusGenerated
	h.store.UpdateStats(set)
	h.store.UpdateSet(set)

	writeJSON(w, http.StatusOK, set)
}

// ExportSet exports the completed set to the Dixit cards directory
func (h *Handler) ExportSet(w http.ResponseWriter, r *http.Request) {
	setID := chi.URLParam(r, "setID")

	set, err := h.store.GetSet(setID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Check all cards are approved
	for _, c := range set.Concepts {
		if c.Status != storage.StatusApproved {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("card %s is not approved", c.CardID))
			return
		}
	}

	// Create export directory
	exportPath := filepath.Join(h.exportDir, set.Name)
	imagesPath := filepath.Join(exportPath, "images")
	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create export directory: %v", err))
		return
	}

	// Copy images
	for _, c := range set.Concepts {
		srcPath := h.store.GetImagePath(setID, c.CardID)
		dstPath := filepath.Join(imagesPath, c.CardID+".png")

		src, err := os.Open(srcPath)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to open %s: %v", c.CardID, err))
			return
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create %s: %v", c.CardID, err))
			return
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()

		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to copy %s: %v", c.CardID, err))
			return
		}
	}

	// Generate cards.json
	cardsJSON := make([]map[string]interface{}, len(set.Concepts))
	for i, c := range set.Concepts {
		cardsJSON[i] = map[string]interface{}{
			"id":    c.CardID,
			"title": c.Title,
			"image": fmt.Sprintf("/cards/%s/images/%s.png", set.Name, c.CardID),
			"tags":  c.Tags,
		}
	}

	cardsData, _ := json.MarshalIndent(cardsJSON, "", "  ")
	cardsJSONPath := filepath.Join(exportPath, "cards.json")
	if err := os.WriteFile(cardsJSONPath, cardsData, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write cards.json: %v", err))
		return
	}

	set.Status = storage.SetStatusExported
	h.store.UpdateSet(set)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "exported",
		"exportPath": exportPath,
		"gitCommands": []string{
			fmt.Sprintf("cd %s", h.exportDir),
			fmt.Sprintf("git lfs track \"cards/%s/images/*.png\"", set.Name),
			fmt.Sprintf("git add cards/%s/", set.Name),
			fmt.Sprintf("git commit -m \"feat: add %s card set (%d cards)\"", set.Name, len(set.Concepts)),
			"git push",
		},
	})
}
