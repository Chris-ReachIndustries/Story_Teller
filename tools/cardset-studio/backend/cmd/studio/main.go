package main

import (
	"log"
	"net/http"
	"os"

	"cardset-studio/internal/api"
	"cardset-studio/internal/concepts"
	"cardset-studio/internal/images"
	"cardset-studio/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	sdURL := os.Getenv("SD_URL")
	if sdURL == "" {
		sdURL = "http://localhost:7860"
	}

	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "./output"
	}

	exportDir := os.Getenv("EXPORT_DIR")
	if exportDir == "" {
		exportDir = "/app/dixit-cards"
	}

	// Initialize services
	store := storage.NewStore(outputDir)
	conceptGen := concepts.NewGenerator(ollamaURL)
	imageGen := images.NewGenerator(sdURL)

	// Create API handler
	handler := api.NewHandler(store, conceptGen, imageGen, exportDir)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With"},
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handler.HealthCheck)

		r.Route("/sets", func(r chi.Router) {
			r.Get("/", handler.ListSets)
			r.Post("/", handler.CreateSet)
			r.Get("/{setID}", handler.GetSet)
			r.Delete("/{setID}", handler.DeleteSet)

			r.Post("/{setID}/concepts", handler.GenerateConcepts)
			r.Put("/{setID}/concepts/{cardID}", handler.UpdateConcept)
			r.Post("/{setID}/concepts/{cardID}/regenerate", handler.RegenerateConceptText)

			r.Post("/{setID}/generate", handler.StartGeneration)
			r.Get("/{setID}/generate/progress", handler.GetProgress)
			r.Post("/{setID}/generate/pause", handler.PauseGeneration)
			r.Post("/{setID}/generate/cancel", handler.CancelGeneration)

			r.Get("/{setID}/cards", handler.ListCards)
			r.Get("/{setID}/cards/{cardID}/image", handler.GetCardImage)
			r.Post("/{setID}/cards/{cardID}/approve", handler.ApproveCard)
			r.Post("/{setID}/cards/{cardID}/reject", handler.RejectCard)
			r.Post("/{setID}/cards/{cardID}/regenerate", handler.RegenerateCard)

			r.Post("/{setID}/export", handler.ExportSet)
		})
	})

	// Serve static files (generated images)
	r.Handle("/output/*", http.StripPrefix("/output/", http.FileServer(http.Dir(outputDir))))

	log.Printf("Card Set Studio starting on port %s", port)
	log.Printf("Ollama URL: %s", ollamaURL)
	log.Printf("Stable Diffusion URL: %s", sdURL)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
