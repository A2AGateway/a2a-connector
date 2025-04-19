package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// RESTAdapter adapts a REST API
type RESTAdapter struct {
	BaseAdapter
	BaseURL    string
	HTTPClient *http.Client
	Headers    map[string]string
}

// NewRESTAdapter creates a new REST adapter
func NewRESTAdapter(name, baseURL string, headers map[string]string, config map[string]interface{}) *RESTAdapter {
	base := NewBaseAdapter(name, REST, "REST API Adapter", config)
	return &RESTAdapter{
		BaseAdapter: *base,
		BaseURL:     baseURL,
		HTTPClient:  &http.Client{},
		Headers:     headers,
	}
}

// Initialize sets up the REST adapter
func (a *RESTAdapter) Initialize() error {
	// TODO: Validate base URL and set up auth if needed
	return nil
}

// GetCapabilities returns the capabilities of the REST API
func (a *RESTAdapter) GetCapabilities() (map[string]interface{}, error) {
	// TODO: Query API for capabilities or return static capabilities
	return map[string]interface{}{
		"type":    "rest",
		"methods": []string{"GET", "POST", "PUT", "DELETE"},
	}, nil
}

// ExecuteTask executes a REST request
func (a *RESTAdapter) ExecuteTask(action string, params map[string]interface{}) (map[string]interface{}, error) {
	// Parse action to determine HTTP method and endpoint
	method := "GET"
	endpoint := action
	
	if m, ok := params["method"].(string); ok {
		method = m
	}
	
	url := fmt.Sprintf("%s%s", a.BaseURL, endpoint)
	
	var req *http.Request
	var err error
	
	if method == "GET" {
		req, err = http.NewRequest(method, url, nil)
	} else {
		// Prepare request body for non-GET requests
		body, err := json.Marshal(params["body"])
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	}
	
	if err != nil {
		return nil, err
	}
	
	// Set headers
	for key, value := range a.Headers {
		req.Header.Set(key, value)
	}
	
	// Set content type if not already set
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Execute request
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Parse response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// Close cleans up resources
func (a *RESTAdapter) Close() error {
	// Nothing to clean up for HTTP client
	return nil
}
