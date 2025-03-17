package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alcimerio/gopos-ratelimiter/pkg/limiter"
	"github.com/alcimerio/gopos-ratelimiter/pkg/middleware"
	"github.com/alcimerio/gopos-ratelimiter/pkg/storage"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Parse configuration
	ipLimit, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_IP"))
	tokenLimit, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_TOKEN"))
	blockDuration, _ := strconv.Atoi(os.Getenv("BLOCK_DURATION"))
	redisHost := os.Getenv("REDIS_HOST")
	redisPort, _ := strconv.Atoi(os.Getenv("REDIS_PORT"))
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	// Initialize Redis storage
	redisStorage, err := storage.NewRedisStorage(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		log.Fatalf("Failed to initialize Redis storage: %v", err)
	}
	defer redisStorage.Close()

	// Initialize rate limiter
	rateLimiter := limiter.NewRateLimiter(redisStorage, limiter.Config{
		IPLimit:       ipLimit,
		TokenLimit:    tokenLimit,
		BlockDuration: time.Duration(blockDuration) * time.Second,
	})

	// Create middleware
	rateLimiterMiddleware := middleware.NewRateLimiterMiddleware(rateLimiter)

	// Create a simple handler for testing
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello, World!"}`))
	})

	// Apply middleware to handler
	http.Handle("/", rateLimiterMiddleware.Handler(handler))

	// Start server
	log.Printf("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
