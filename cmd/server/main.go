package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"devflow-backend/internal/api"
	"devflow-backend/internal/config"
	"devflow-backend/internal/database"
	"devflow-backend/internal/repository"
	"devflow-backend/internal/kafka"
    "devflow-backend/internal/kafka/workers"
	ws "devflow-backend/internal/websocket"

	"github.com/joho/godotenv"
)

func init() {
    if err := godotenv.Load(".env.local"); err != nil {
        log.Println("No .env.local file found, using OS env vars")
    }
}

func main() {
	// Load env vars (you can replace with godotenv if needed)
	cfg := config.Load()

	if cfg.MongoURI == "" {
		log.Fatal("MONGODB_URI is required")
	}
	
	// Connect to MongoDB
	if err := database.Connect(cfg.MongoURI); err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	defer database.Disconnect()
	database.ConnectRedis()
	kafka.Init()
	defer kafka.Close()

	// Initialize WebSocket Hub
	pairRepo := repository.NewPairRepo(database.GetDB())
	ws.GlobalHub = ws.NewHub(pairRepo)
	go ws.GlobalHub.Run()
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	reviewRepo := repository.NewReviewRepo(database.GetDB())
	ws.GlobalReviewHub = ws.NewReviewHub(reviewRepo)
	go ws.GlobalReviewHub.Run()
	workers.StartActivityWorker(rootCtx)

	router := api.NewRouter()

	// HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start in goroutine for graceful shutdown
	go func() {
		log.Printf("🚀 DevFlow API running on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Forced shutdown: %v", err)
	}
}
