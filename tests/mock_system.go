package main

import (
    "encoding/json"
    "flag"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Parse command-line flags
    port := flag.String("port", "8081", "Port to listen on")
    flag.Parse()

    // Create and start the mock system
    mockSystem := NewMockLegacySystem(*port)

    // Set up a channel to listen for interrupt signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start the server in a goroutine
    go func() {
        log.Println("Starting mock legacy system on port", *port)
        err := mockSystem.Start()
        if err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start mock system: %v", err)
        }
    }()

    // Wait for interrupt signal
    <-sigChan
    log.Println("Shutting down...")

    // Stop the server
    err := mockSystem.Stop()
    if err != nil {
        log.Printf("Error stopping server: %v", err)
    }

    log.Println("Server stopped.")
}

// MockLegacySystem is a simple HTTP server that mimics a legacy system
type MockLegacySystem struct {
    server *http.Server
}

// NewMockLegacySystem creates a new mock legacy system
func NewMockLegacySystem(port string) *MockLegacySystem {
    mux := http.NewServeMux()
    
    // Add routes for the mock system
    mux.HandleFunc("/api/customers", handleCustomers)
    
    server := &http.Server{
        Addr:    ":" + port,
        Handler: mux,
    }
    
    return &MockLegacySystem{
        server: server,
    }
}

// Start starts the mock legacy system
func (m *MockLegacySystem) Start() error {
    log.Println("Starting mock legacy system on", m.server.Addr)
    return m.server.ListenAndServe()
}

// Stop stops the mock legacy system
func (m *MockLegacySystem) Stop() error {
    return m.server.Close()
}

// handleCustomers handles customer-related requests
func handleCustomers(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Get customer ID from query parameters
    customerID := r.URL.Query().Get("id")
    if customerID == "" {
        http.Error(w, "Customer ID is required", http.StatusBadRequest)
        return
    }
    
    // Return mock customer data
    customer := map[string]interface{}{
        "id":        customerID,
        "name":      "Test Customer",
        "email":     "test@example.com",
        "createdAt": "2025-04-01T00:00:00Z",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(customer)
}