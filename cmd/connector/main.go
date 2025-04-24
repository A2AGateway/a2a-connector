package main

import (
    "flag"
    "log"
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
    )
    flag.Parse()

    log.Println("Starting A2A Gateway Connector...")
    log.Println("Connecting to SaaS at:", *saasEndpoint)
    log.Println("Connector ID:", *connectorID)
    log.Println("Legacy system URL:", *legacyBaseURL)

    // Create a REST adapter for the legacy system
    restAdapter := adapter.NewRESTAdapter(*legacyBaseURL)

    // Create a transformer
    transformer := proxy.NewTransformer()

    // Create a proxy
    p := proxy.NewProxy(restAdapter, transformer)

    // Set up a channel to listen for interrupt signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Start the proxy in a goroutine
    go func() {
        err := p.Start()
        if err != nil {
            log.Fatalf("Failed to start proxy: %v", err)
        }
    }()

    log.Println("Connector started. Press Ctrl+C to exit.")

    // Wait for interrupt signal
    <-sigChan
    log.Println("Shutting down...")

    // Stop the proxy
    err := p.Stop()
    if err != nil {
        log.Printf("Error stopping proxy: %v", err)
    }

    log.Println("Connector stopped.")
}