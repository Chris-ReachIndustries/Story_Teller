package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"dixit-backend/internal/cardgen"
	"dixit-backend/internal/http"
)

func main() {
	// Get configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cardsBasePath := os.Getenv("CARDS_PATH")
	if cardsBasePath == "" {
		cardsBasePath = "/app/cards"
	}

	// Card set name - allows multiple named card sets
	cardSetName := getEnvOrDefault("CARD_SET_NAME", "default")

	// Card generation configuration
	genConfig := cardgen.Config{
		CardsBasePath:   cardsBasePath,
		SetName:         cardSetName,
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		AIEnabled:       os.Getenv("AI_ENABLED") == "true",
		ForceRegenerate: os.Getenv("CARDS_REGENERATE") == "true",
		Concurrency:     getEnvInt("DALLE_CONCURRENCY", 3),
		DALLEModel:      getEnvOrDefault("DALLE_MODEL", "dall-e-3"),
		ImageSize:       getEnvOrDefault("DALLE_SIZE", "1024x1792"),
		ImageQuality:    getEnvOrDefault("DALLE_QUALITY", "standard"),
	}

	// Ensure cards exist before starting server
	generator := cardgen.NewGenerator(genConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	if err := generator.EnsureCards(ctx); err != nil {
		log.Fatalf("Card generation failed: %v", err)
	}

	// The actual card set path includes the set name
	cardsSetPath := genConfig.GetSetPath()
	log.Printf("Using card set '%s' from %s", cardSetName, cardsSetPath)

	// Create and start server
	server := http.NewServer(cardsBasePath, cardSetName)

	addr := ":" + port
	log.Printf("Starting Dixit server on %s", addr)

	if err := server.Start(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}
