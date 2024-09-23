package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// Redis client setup
var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

// Reverse Proxy handler
func reverseProxy(w http.ResponseWriter, req *http.Request) {
	cacheKey := req.URL.String()

	// Check if the response is already cached
	cachedResponse, err := rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		// Not in cache, forward the request to the backend
		fmt.Println("Cache miss, forwarding request to backend")

		// Forward request to backend
		backendURL := "http://backend-service" + req.URL.String() // Change this to your actual backend service URL
		resp, err := http.Get(backendURL)
		if err != nil {
			http.Error(w, "Failed to fetch from backend", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Read the backend response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read backend response", http.StatusInternalServerError)
			return
		}

		// Cache the response in Redis with a timeout of 60 seconds
		err = rdb.Set(ctx, cacheKey, body, 60*time.Second).Err()
		if err != nil {
			http.Error(w, "Failed to cache response", http.StatusInternalServerError)
			return
		}

		// Return the backend response to the client
		w.Write(body)
	} else if err != nil {
		// Handle Redis connection error
		http.Error(w, "Redis error", http.StatusInternalServerError)
	} else {
		// Cache hit: Serve the cached response
		fmt.Println("Cache hit, serving cached response")
		w.Write([]byte(cachedResponse))
	}
}

func main() {
	// Using http.HandleFunc to create a reverse proxy server
	http.HandleFunc("/", reverseProxy)

	// Start the server on port 8080
	fmt.Println("Starting reverse proxy on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
