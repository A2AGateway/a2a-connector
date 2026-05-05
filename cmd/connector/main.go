package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	a2a "github.com/A2AGateway/a2a-protocol"
	"github.com/A2AGateway/a2a-connector/internal/adapter"
	"github.com/A2AGateway/a2a-connector/internal/config"
	"github.com/A2AGateway/a2a-connector/internal/gateway"
	"github.com/A2AGateway/a2a-connector/internal/proxy"
)

func main() {
	var (
		saasEndpoint  = flag.String("saas-endpoint", "", "A2A Gateway base URL for registration (e.g. http://gateway:8080)")
		connectorID   = flag.String("connector-id", "my-connector", "Unique connector ID registered with the gateway")
		connectorHost = flag.String("connector-host", "http://localhost:8082", "Public URL of this connector (included in agent card)")
		legacyBaseURL = flag.String("legacy-url", "http://localhost:8081", "Legacy system base URL")
		connectorPort = flag.String("port", "8082", "Port this connector listens on")
		configFile    = flag.String("config", "", "Path to YAML/JSON config file")
		useConfig     = flag.Bool("use-config", false, "Use config file instead of flags")
	)
	flag.Parse()

	log.Println("Starting A2A Connector...")

	// --- build adapter + transformer ---
	var adptr adapter.Adapter
	var transformer *proxy.Transformer
	var legacyURL string

	if *useConfig && *configFile != "" {
		cfg, err := config.LoadFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		if err := config.ValidateConfig(cfg); err != nil {
			log.Fatalf("Invalid config: %v", err)
		}

		headers := make(map[string]string)
		for k, v := range cfg.Adapter.Headers {
			headers[k] = v
		}
		restAdptr := adapter.NewRESTAdapter(cfg.Adapter.Name, cfg.Adapter.BaseURL, headers, nil)
		if err := restAdptr.Initialize(); err != nil {
			log.Fatalf("Failed to initialize adapter: %v", err)
		}
		adptr = restAdptr

		ct := proxy.NewConfigTransformer(cfg)
		transformer = &ct.Transformer
		legacyURL = cfg.Adapter.BaseURL
		log.Println("Connecting to legacy system at:", legacyURL)
	} else {
		headers := make(map[string]string)
		restAdptr := adapter.NewRESTAdapter("Legacy REST", *legacyBaseURL, headers, nil)
		if err := restAdptr.Initialize(); err != nil {
			log.Fatalf("Failed to initialize adapter: %v", err)
		}
		adptr = restAdptr

		transformer = proxy.NewTransformer()
		transformer.SetRequestTransform(defaultRequestTransform)
		transformer.SetResponseTransform(defaultResponseTransform)
		legacyURL = *legacyBaseURL
		log.Println("Connecting to legacy system at:", legacyURL)
	}

	defer func() {
		if err := adptr.Close(); err != nil {
			log.Printf("Error closing adapter: %v", err)
		}
	}()

	// --- agent card ---
	card := buildAgentCard(*connectorID, *connectorHost, adptr)

	// --- gateway registration ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *saasEndpoint != "" {
		gwClient := gateway.NewClient(*saasEndpoint, *connectorID, *connectorHost)
		if err := gwClient.Register(card); err != nil {
			log.Printf("Warning: gateway registration failed: %v", err)
		} else {
			log.Printf("Registered connector %q with gateway at %s", *connectorID, *saasEndpoint)
		}
		gwClient.StartHeartbeat(ctx, 30*time.Second)
	} else {
		log.Println("Warning: --saas-endpoint not set; running standalone (not registered with gateway)")
	}

	// --- HTTP routes ---
	mux := http.NewServeMux()

	// A2A discovery: gateway and other agents fetch this to learn what the connector can do
	mux.HandleFunc("/.well-known/agent.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(card)
	})

	// A2A JSON-RPC endpoint: gateway forwards tasks here
	mux.HandleFunc("/", a2aHandler(transformer, adptr))

	server := &http.Server{
		Addr:         ":" + *connectorPort,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Connector listening on :%s", *connectorPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down...")
	cancel()
	if err := server.Close(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}
	log.Println("Connector stopped.")
}

// a2aHandler handles incoming A2A JSON-RPC requests from the gateway.
func a2aHandler(transformer *proxy.Transformer, adptr adapter.Adapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeRPCError(w, nil, a2a.ErrCodeParseError, "Failed to read request body", nil)
			return
		}

		var rpcReq a2a.JSONRPCRequest
		if err := json.Unmarshal(body, &rpcReq); err != nil {
			writeRPCError(w, nil, a2a.ErrCodeParseError, "Invalid JSON", nil)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch rpcReq.Method {
		case "tasks/send":
			handleTaskSend(w, rpcReq, transformer, adptr)
		default:
			writeRPCError(w, rpcReq.ID, a2a.ErrCodeMethodNotFound, "Method not found", nil)
		}
	}
}

func handleTaskSend(w http.ResponseWriter, rpcReq a2a.JSONRPCRequest, transformer *proxy.Transformer, adptr adapter.Adapter) {
	paramsBytes, err := json.Marshal(rpcReq.Params)
	if err != nil {
		writeRPCError(w, rpcReq.ID, a2a.ErrCodeInvalidParams, "Failed to parse params", nil)
		return
	}

	// A2A task params → legacy request format
	legacyData, err := transformer.TransformRequestData(paramsBytes)
	if err != nil {
		writeRPCError(w, rpcReq.ID, a2a.ErrCodeInternalError, "Request transform failed", err.Error())
		return
	}

	var legacyReq map[string]interface{}
	if err := json.Unmarshal(legacyData, &legacyReq); err != nil {
		writeRPCError(w, rpcReq.ID, a2a.ErrCodeInternalError, "Bad legacy request format", err.Error())
		return
	}

	action, _ := legacyReq["action"].(string)
	params, _ := legacyReq["params"].(map[string]interface{})
	result, execErr := adptr.ExecuteTask(action, params)

	legacyResp := map[string]interface{}{
		"result": result,
		"meta":   legacyReq["meta"],
	}
	if execErr != nil {
		legacyResp["status"] = "error"
		legacyResp["error"] = execErr.Error()
	} else {
		legacyResp["status"] = "success"
	}

	legacyRespBytes, _ := json.Marshal(legacyResp)

	// Legacy response → A2A task
	a2aRespBytes, err := transformer.TransformResponseData(legacyRespBytes)
	if err != nil {
		writeRPCError(w, rpcReq.ID, a2a.ErrCodeInternalError, "Response transform failed", err.Error())
		return
	}

	var task interface{}
	json.Unmarshal(a2aRespBytes, &task)

	json.NewEncoder(w).Encode(a2a.JSONRPCResponse{
		JSONRPC: a2a.JSONRPCVersion,
		ID:      rpcReq.ID,
		Result:  task,
	})
}

func writeRPCError(w http.ResponseWriter, id interface{}, code int, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a2a.JSONRPCResponse{
		JSONRPC: a2a.JSONRPCVersion,
		ID:      id,
		Error:   &a2a.JSONRPCError{Code: code, Message: msg, Data: data},
	})
}

// buildAgentCard constructs the A2A agent card that describes this connector.
func buildAgentCard(id, url string, adptr adapter.Adapter) *a2a.AgentCard {
	caps, _ := adptr.GetCapabilities()
	adapterType := "rest"
	if t, ok := caps["type"].(string); ok {
		adapterType = t
	}

	desc := "A2A Connector bridging a legacy " + adapterType + " system"
	skillDesc := "Execute a task on the connected legacy system"
	skill := a2a.AgentSkill{
		ID:          "legacy-execute",
		Name:        "Execute Legacy Task",
		Description: &skillDesc,
		Tags:        []string{"legacy", adapterType},
		InputModes:  []string{"text"},
		OutputModes: []string{"text", "data"},
	}

	card := a2a.NewAgentCard(
		id, url, "1.0.0",
		a2a.AgentCapabilities{Streaming: false, PushNotifications: false},
		[]a2a.AgentSkill{skill},
	)
	card.WithDescription(desc)
	return card
}

// defaultRequestTransform converts an A2A task payload to a generic legacy request.
// Replace this with config-driven mappings for production use.
func defaultRequestTransform(data []byte) ([]byte, error) {
	var taskMap map[string]interface{}
	if err := json.Unmarshal(data, &taskMap); err != nil {
		return nil, err
	}

	action := "query"
	params := map[string]interface{}{}

	if status, ok := taskMap["status"].(map[string]interface{}); ok {
		if msg, ok := status["message"].(map[string]interface{}); ok {
			if parts, ok := msg["parts"].([]interface{}); ok {
				for _, p := range parts {
					if part, ok := p.(map[string]interface{}); ok {
						if part["type"] == "text" {
							if text, ok := part["text"].(string); ok {
								params["text"] = text
								action = "execute"
							}
						}
					}
				}
			}
		}
	}

	taskID := ""
	if id, ok := taskMap["id"].(string); ok {
		taskID = id
	}

	return json.Marshal(map[string]interface{}{
		"action": action,
		"params": params,
		"meta":   map[string]interface{}{"taskId": taskID},
	})
}

// defaultResponseTransform converts a legacy response to an A2A task.
func defaultResponseTransform(data []byte) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	taskID := "unknown"
	if meta, ok := resp["meta"].(map[string]interface{}); ok {
		if id, ok := meta["taskId"].(string); ok {
			taskID = id
		}
	}

	state := string(a2a.TaskStateCompleted)
	if _, hasErr := resp["error"]; hasErr {
		state = string(a2a.TaskStateFailed)
	}

	parts := []map[string]interface{}{}
	if result, ok := resp["result"].(map[string]interface{}); ok {
		parts = append(parts, map[string]interface{}{"type": "data", "data": result})
	}
	if errMsg, ok := resp["error"].(string); ok {
		parts = append(parts, map[string]interface{}{"type": "text", "text": "Error: " + errMsg})
	}

	return json.Marshal(map[string]interface{}{
		"id": taskID,
		"status": map[string]interface{}{
			"state":     state,
			"timestamp": time.Now().Format(time.RFC3339),
			"message": map[string]interface{}{
				"role":  "agent",
				"parts": parts,
			},
		},
	})
}
