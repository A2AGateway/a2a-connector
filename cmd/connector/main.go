// connector/cmd/connector/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/A2AGateway/a2agateway/connector/internal/adapter"
	"github.com/A2AGateway/a2agateway/connector/internal/config"
	"github.com/A2AGateway/a2agateway/connector/internal/proxy"
	"github.com/A2AGateway/a2agateway/saas/pkg/a2a"
)

func main() {
	// Parse command-line flags
	var (
		saasEndpoint  = flag.String("saas-endpoint", "http://localhost:8080", "SaaS component endpoint")
		connectorID   = flag.String("connector-id", "test-connector", "Connector ID")
		legacyBaseURL = flag.String("legacy-url", "http://localhost:8081", "Legacy system base URL")
		connectorPort = flag.String("port", "8082", "Connector listening port")
		configFile    = flag.String("config", "", "Path to configuration file (YAML or JSON)")
		useConfig     = flag.Bool("use-config", false, "Use configuration file instead of command line parameters")
	)
	flag.Parse()

	log.Println("Starting A2A Gateway Connector...")
	
	var cfg *config.ConnectorConfig
	var err error
	
	// Load configuration if specified
	if *useConfig && *configFile != "" {
		log.Println("Loading configuration from:", *configFile)
		cfg, err = config.LoadFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
		
		// Validate configuration
		if err := config.ValidateConfig(cfg); err != nil {
			log.Fatalf("Invalid configuration: %v", err)
		}
		
		log.Println("Configuration loaded successfully")
		log.Println("Connecting to legacy system at:", cfg.Adapter.BaseURL)
	} else {
		log.Println("Using command line parameters")
		log.Println("Connecting to SaaS at:", *saasEndpoint)
		log.Println("Connector ID:", *connectorID)
		log.Println("Legacy system URL:", *legacyBaseURL)
	}
	
	log.Println("Listening on port:", *connectorPort)

	var transformer *proxy.Transformer
	var legacyUrl string

	// Initialize adapter and transformer based on configuration or command line parameters
	if *useConfig && cfg != nil {
		// Create appropriate adapter based on config
		var adapterInstance adapter.Adapter
		adapterConfig := make(map[string]interface{})
		
		switch cfg.Adapter.Type {
		case "rest":
			headers := make(map[string]string)
			for k, v := range cfg.Adapter.Headers {
				headers[k] = v
			}
			adapterInstance = adapter.NewRESTAdapter(cfg.Adapter.Name, cfg.Adapter.BaseURL, headers, adapterConfig)
		case "soap":
			// Create SOAP adapter
			log.Println("SOAP adapter not fully implemented yet, using REST adapter")
			headers := make(map[string]string)
			adapterInstance = adapter.NewRESTAdapter(cfg.Adapter.Name, cfg.Adapter.BaseURL, headers, adapterConfig)
		case "db":
			// Create DB adapter
			log.Println("DB adapter not fully implemented yet, using REST adapter")
			headers := make(map[string]string)
			adapterInstance = adapter.NewRESTAdapter(cfg.Adapter.Name, cfg.Adapter.BaseURL, headers, adapterConfig)
		case "file":
			// Create File adapter
			log.Println("File adapter not fully implemented yet, using REST adapter")
			headers := make(map[string]string)
			adapterInstance = adapter.NewRESTAdapter(cfg.Adapter.Name, cfg.Adapter.BaseURL, headers, adapterConfig)
		default:
			log.Fatalf("Unsupported adapter type: %s", cfg.Adapter.Type)
		}
		
		// Initialize the adapter
		if err := adapterInstance.Initialize(); err != nil {
			log.Fatalf("Failed to initialize adapter: %v", err)
		}
		
		// Create config-driven transformer
		configTransformer := proxy.NewConfigTransformer(cfg)
		transformer = &configTransformer.Transformer
		legacyUrl = cfg.Adapter.BaseURL
		
	} else {
		// Using command-line parameters
		// Create a REST adapter for the legacy system
		headers := make(map[string]string)
		adapterConfig := make(map[string]interface{})
		restAdapter := adapter.NewRESTAdapter("Legacy REST", *legacyBaseURL, headers, adapterConfig)
		
		// Initialize the adapter
		if err := restAdapter.Initialize(); err != nil {
			log.Fatalf("Failed to initialize adapter: %v", err)
		}
		
		// Create a traditional hardcoded transformer
		transformer = proxy.NewTransformer()
		
		// Define transformation functions - these will convert between A2A and legacy formats
		transformer.SetRequestTransform(func(data []byte) ([]byte, error) {
			// Transform A2A format to legacy format
			var taskData a2a.Task
			if err := json.Unmarshal(data, &taskData); err != nil {
				log.Printf("Error unmarshaling A2A task: %v", err)
				return nil, err
			}
	
			// Extract information from the A2A task for the legacy system
			// This logic depends on what the legacy system expects
			
			// Example: Extract customer ID from the message
			var action string
			var params map[string]interface{}
			
			// Initialize params
			params = make(map[string]interface{})
			
			// Default action if we can't determine one
			action = "query"
			
			// Check if we have a message to extract information from
			if taskData.Status.Message != nil && len(taskData.Status.Message.Parts) > 0 {
				// Process each part (using the updated Part interface)
				for _, part := range taskData.Status.Message.Parts {
					if part.GetType() == "text" {
						if textPart, ok := part.(a2a.TextPart); ok {
							text := textPart.Text
							
							// Try to determine the action based on the text
							text = strings.ToLower(text)
							
							if strings.Contains(text, "get") || strings.Contains(text, "retrieve") || strings.Contains(text, "query") {
								action = "getEntity"
								
								// Try to extract entity ID from formats like "ID: 12345" or similar patterns
								if idx := strings.LastIndex(text, ":"); idx != -1 {
									idStr := strings.TrimSpace(text[idx+1:])
									params["id"] = idStr
								}
								
								// Try to determine entity type
								if strings.Contains(text, "customer") {
									params["entityType"] = "customer"
								} else if strings.Contains(text, "order") {
									params["entityType"] = "order"
								} else if strings.Contains(text, "product") || strings.Contains(text, "inventory") {
									params["entityType"] = "product"
								}
							} else if strings.Contains(text, "update") || strings.Contains(text, "change") {
								action = "updateEntity"
								
								// For update actions, we would need more sophisticated parsing
								// of the text to extract entity type, ID, and fields to update
								// This is a simplified example
								if strings.Contains(text, "customer") {
									params["entityType"] = "customer"
								} else if strings.Contains(text, "order") {
									params["entityType"] = "order"
								} else if strings.Contains(text, "product") || strings.Contains(text, "inventory") {
									params["entityType"] = "product"
								}
							}
						}
					} else if part.GetType() == "data" {
						// Handle structured data if present
						if dataPart, ok := part.(a2a.DataPart); ok {
							// Extract fields from the data part
							for k, v := range dataPart.Data {
								params[k] = v
							}
						}
					}
				}
			}
			
			// Create a legacy request format
			legacyRequest := map[string]interface{}{
				"action": action,
				"params": params,
				"meta": map[string]interface{}{
					"taskId": taskData.ID,
				},
			}
			
			// Add any task metadata that might be useful for the legacy system
			if taskData.Metadata != nil {
				for k, v := range taskData.Metadata {
					if _, exists := legacyRequest["meta"].(map[string]interface{})[k]; !exists {
						legacyRequest["meta"].(map[string]interface{})[k] = v
					}
				}
			}
	
			return json.Marshal(legacyRequest)
		})
	
		transformer.SetResponseTransform(func(data []byte) ([]byte, error) {
			// Transform legacy format to A2A format
			var legacyResponse map[string]interface{}
			if err := json.Unmarshal(data, &legacyResponse); err != nil {
				log.Printf("Error unmarshaling legacy response: %v", err)
				return nil, err
			}
			
			// Get the task ID from the metadata if available
			taskId := "unknown-task"
			if meta, ok := legacyResponse["meta"].(map[string]interface{}); ok {
				if id, ok := meta["taskId"].(string); ok {
					taskId = id
				}
			}
			
			// Determine the task state based on the legacy response
			taskState := a2a.TaskStateCompleted
			if errorMsg, ok := legacyResponse["error"].(string); ok && errorMsg != "" {
				taskState = a2a.TaskStateFailed
			}
			
			// Create a new A2A task
			task := a2a.NewTask(taskId, taskState)
			
			// Create parts for the response message
			var parts []a2a.Part
			
			// Add a text part with a summary of the response
			var textContent string
			if status, ok := legacyResponse["status"].(string); ok {
				textContent += "Status: " + status + "\n"
			}
			
			// Add result data
			if result, ok := legacyResponse["result"].(map[string]interface{}); ok {
				// For structured data, we can create a data part
				dataPart := a2a.NewDataPart(result)
				parts = append(parts, dataPart)
				
				// Also add a summary as text
				textContent += "Results:\n"
				for k, v := range result {
					textContent += k + ": " + fmt.Sprintf("%v", v) + "\n"
				}
			}
			
			// Add any error message
			if errorMsg, ok := legacyResponse["error"].(string); ok && errorMsg != "" {
				textContent += "Error: " + errorMsg + "\n"
			}
			
			// Add the text part if we have text content
			if textContent != "" {
				textPart := a2a.NewTextPart(textContent)
				parts = append(parts, textPart)
			}
			
			// Create a message with the parts
			message := a2a.NewMessage(a2a.RoleAgent, parts)
			
			// Add the message to the task
			task.WithMessage(message)
			
			// Add metadata from the legacy response
			if meta, ok := legacyResponse["meta"].(map[string]interface{}); ok {
				task.Metadata = meta
			}
			
			// Marshal the task to JSON
			return json.Marshal(task)
		})
		
		legacyUrl = *legacyBaseURL
	}

	// Create a proxy
	p, err := proxy.NewProxy(legacyUrl, transformer)
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
