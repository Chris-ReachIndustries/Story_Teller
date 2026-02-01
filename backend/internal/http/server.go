package http

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"

	"dixit-backend/internal/ai"
	"dixit-backend/internal/cardgen"
	"dixit-backend/internal/cards"
	"dixit-backend/internal/cards/thumbs"
	"dixit-backend/internal/game"
	"dixit-backend/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Server is the HTTP server
type Server struct {
	hub           *ws.Hub
	handlers      *Handlers
	roomManager   *game.RoomManager
	cardLoader    *cards.Loader
	cardRegistry  *cards.Registry
	cardGenerator *cardgen.Generator
	cardsBasePath string
	cardSetName   string
	nextjsURL     string
	aiEnabled     bool
	thumbService  *thumbs.Service
}

// NewServer creates a new HTTP server
func NewServer(cardsBasePath, cardSetName string) *Server {
	roomManager := game.NewRoomManager()

	// Create card registry for multi-set support
	cardRegistry := cards.NewRegistry(cardsBasePath)

	// Card set path is base path + set name (for backwards compatibility)
	cardSetPath := filepath.Join(cardsBasePath, cardSetName)
	cardLoader := cards.NewLoader(filepath.Join(cardSetPath, "cards.json"))

	// Initialize AI services with base path (supports multiple card sets)
	aiClient, thumbService := initAIServices(cardsBasePath, cardSetName, cardLoader)

	// Pass registry to hub for per-room deck loading
	hub := ws.NewHub(roomManager, cardRegistry, cardsBasePath, aiClient, thumbService)
	go hub.Run()

	nextjsURL := os.Getenv("NEXTJS_URL")
	if nextjsURL == "" {
		nextjsURL = "http://localhost:3000"
	}

	// Create card generator for status endpoint (generation already done in main.go)
	cardGenerator := cardgen.NewGenerator(cardgen.Config{
		CardsBasePath: cardsBasePath,
		SetName:       cardSetName,
	})

	return &Server{
		hub:           hub,
		handlers:      NewHandlers(cardLoader, cardRegistry, cardGenerator),
		roomManager:   roomManager,
		cardLoader:    cardLoader,
		cardRegistry:  cardRegistry,
		cardGenerator: cardGenerator,
		cardsBasePath: cardsBasePath,
		cardSetName:   cardSetName,
		nextjsURL:     nextjsURL,
		aiEnabled:     aiClient != nil,
		thumbService:  thumbService,
	}
}

// initAIServices initializes AI client and thumbnail service from environment
func initAIServices(cardsBasePath string, defaultSetName string, cardLoader *cards.Loader) (ai.Client, *thumbs.Service) {
	aiEnabled := os.Getenv("AI_ENABLED") == "true"
	if !aiEnabled {
		log.Println("AI bots disabled (AI_ENABLED != true)")
		return nil, nil
	}

	// Determine provider (default to openai for backwards compatibility)
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = "openai"
	}

	// Parse common configuration
	thumbWidth := 320
	if val := os.Getenv("AI_THUMB_WIDTH"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			thumbWidth = parsed
		}
	}

	maxTokens := 250
	if val := os.Getenv("AI_MAX_TOKENS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			maxTokens = parsed
		}
	}

	temperature := 0.7
	if val := os.Getenv("AI_TEMPERATURE"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil && parsed >= 0 {
			temperature = parsed
		}
	}

	// Create thumbnail service with base path (supports all card sets)
	thumbService := thumbs.NewService(cardsBasePath, thumbWidth)

	// Prewarm thumbnail cache for the default card set
	go func() {
		cards, err := cardLoader.GetCards()
		if err != nil {
			log.Printf("Warning: failed to load cards for thumbnail prewarm: %v", err)
			return
		}
		cardIDs := make([]string, len(cards))
		for i, c := range cards {
			cardIDs[i] = c.ID
		}
		if err := thumbService.PrewarmCache(cardIDs, defaultSetName); err != nil {
			log.Printf("Warning: thumbnail prewarm failed: %v", err)
		} else {
			log.Printf("Thumbnail cache prewarmed with %d cards from set '%s'", len(cardIDs), defaultSetName)
		}
	}()

	// Initialize AI client based on provider
	var aiClient ai.Client
	switch provider {
	case "openai":
		aiClient = initOpenAIClient(maxTokens, temperature)
	case "local":
		aiClient = initOllamaClient(maxTokens, temperature)
	default:
		log.Printf("Unknown AI_PROVIDER=%s, defaulting to openai", provider)
		aiClient = initOpenAIClient(maxTokens, temperature)
	}

	if aiClient == nil {
		return nil, nil
	}

	// Perform health check if provider supports it
	if pinger, ok := aiClient.(ai.Pinger); ok {
		if err := pinger.Ping(); err != nil {
			log.Printf("WARNING: AI provider health check failed: %v", err)
			// Continue anyway - will retry on first use
		} else {
			log.Printf("AI provider health check passed")
		}
	}

	return aiClient, thumbService
}

// initOpenAIClient creates an OpenAI client from environment
func initOpenAIClient(maxTokens int, temperature float64) ai.Client {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("AI bots disabled (OPENAI_API_KEY not set for openai provider)")
		return nil
	}

	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	timeoutMs := 12000
	if val := os.Getenv("AI_TIMEOUT_MS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}

	log.Printf("AI bots enabled (provider: openai, model: %s, timeout: %dms)", model, timeoutMs)

	return ai.NewOpenAIClient(ai.OpenAIConfig{
		APIKey:      apiKey,
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Timeout:     time.Duration(timeoutMs) * time.Millisecond,
	})
}

// initOllamaClient creates an Ollama client from environment
func initOllamaClient(maxTokens int, temperature float64) ai.Client {
	baseURL := os.Getenv("LOCAL_AI_URL")
	if baseURL == "" {
		baseURL = "http://ollama:11434"
	}

	// Vision model for describing images
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "llava:7b"
	}

	// Text model for making decisions (faster, better at instructions)
	textModel := os.Getenv("AI_TEXT_MODEL")
	if textModel == "" {
		textModel = "llama3.2:3b"
	}

	// Longer default timeout for local inference
	timeoutMs := 60000
	if val := os.Getenv("AI_TIMEOUT_MS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}

	visionMaxTokens := 100
	if val := os.Getenv("AI_VISION_MAX_TOKENS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			visionMaxTokens = parsed
		}
	}

	describeConcurrency := 4
	if val := os.Getenv("AI_DESCRIBE_CONCURRENCY"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			describeConcurrency = parsed
		}
	}

	log.Printf("AI bots enabled (provider: local/ollama, vision: %s, text: %s, url: %s, timeout: %dms)", model, textModel, baseURL, timeoutMs)

	return ai.NewOllamaClient(ai.OllamaConfig{
		BaseURL:             baseURL,
		Model:               model,
		TextModel:           textModel,
		MaxTokens:           maxTokens,
		VisionMaxTokens:     visionMaxTokens,
		DescribeConcurrency: describeConcurrency,
		Temperature:         temperature,
		Timeout:             time.Duration(timeoutMs) * time.Millisecond,
	})
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", s.handlers.HealthHandler)
	mux.HandleFunc("/api/cards", s.handlers.CardsHandler)
	mux.HandleFunc("/api/cardgen/status", s.handlers.CardGenStatusHandler)
	mux.HandleFunc("/api/card-sets", s.handlers.CardSetsHandler)
	mux.HandleFunc("/api/card-sets/", s.handlers.CardSetCardsHandler)

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Serve card images statically from base path (URLs include set name: /cards/<setname>/images/...)
	cardsFS := http.FileServer(http.Dir(s.cardsBasePath))
	mux.Handle("/cards/", http.StripPrefix("/cards", cardsFS))

	// Reverse proxy to Next.js for all other routes
	nextjsProxy := s.createNextJSProxy()
	mux.Handle("/", nextjsProxy)

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(mux)

	log.Printf("Server starting on %s", addr)
	log.Printf("Card set: %s (path: %s)", s.cardSetName, filepath.Join(s.cardsBasePath, s.cardSetName))
	log.Printf("Next.js URL: %s", s.nextjsURL)
	log.Printf("AI bots: %v", s.aiEnabled)

	return http.ListenAndServe(addr, handler)
}

// handleWebSocket upgrades HTTP to WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := ws.NewClient(s.hub, conn)
	s.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// createNextJSProxy creates a reverse proxy to Next.js
func (s *Server) createNextJSProxy() http.Handler {
	target, err := url.Parse(s.nextjsURL)
	if err != nil {
		log.Fatalf("Invalid Next.js URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom director to handle the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)

		// Check if Next.js is not ready yet
		if strings.Contains(err.Error(), "connection refused") {
			http.Error(w, "Frontend is starting up, please wait...", http.StatusServiceUnavailable)
			return
		}

		http.Error(w, "Proxy error", http.StatusBadGateway)
	}

	return proxy
}
