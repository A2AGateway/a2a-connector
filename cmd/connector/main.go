// connector/cmd/connector/main.go
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/A2AGateway/a2agateway/connector/internal/adapter"
	"github.com/A2AGateway/a2agateway/connector/internal/proxy"
)

func main() {
	// Parse command-line flags
	var (
		saasEndpoint  = flag.String("saas-endpoint", "http://localhost:8080", "SaaS component endpoint")
		connectorID   = flag.String("connector-id", "test-connector", "Connector ID")
		legacyBaseURL = flag.String("legacy-url", "http://localhost:8081", "Legacy system base URL")
		connectorPort = flag.String("port", "8082", "Connector listening port")
	)
	flag.Parse()

	log.Println("Starting A2A Gateway Connector...")
	log.Println("Connecting to SaaS at:", *saasEndpoint)
	log.Println("Connector ID:", *connectorID)
	log.Println("Legacy system URL:", *legacyBaseURL)
	log.Println("Listening on port:", *connectorPort)

	// Create a REST adapter for the legacy system
	headers := make(map[string]string)
	config := make(map[string]interface{})
	restAdapter := adapter.NewRESTAdapter("Legacy REST", *legacyBaseURL, headers, config)

	// Initialize the adapter
	err := restAdapter.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Create a transformer
	transformer := proxy.NewTransformer()

	// Define transformation functions - these will convert between A2A and legacy formats
	transformer.SetRequestTransform(func(data []byte) ([]byte, error) {
		// Transform A2A format to legacy format
		// Your transformation logic here
		return data, nil
	})

	transformer.SetResponseTransform(func(data []byte) ([]byte, error) {
		// Transform legacy format to A2A format
		// Your transformation logic here
		return data, nil
	})

	// Create a proxy
	p, err := proxy.NewProxy(*legacyBaseURL, transformer)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	// Set up a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + *connectorPort,
		Handler: p, // Use the proxy as the handler
	}

	// Start the server in a goroutine
	go func() {
		log.Println("Server listening on port", *connectorPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Connector started. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down...")

	// Stop the server
	if err := server.Close(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	// Close the adapter
	if err := restAdapter.Close(); err != nil {
		log.Printf("Error closing adapter: %v", err)
	}

	log.Println("Connector stopped.")
}
