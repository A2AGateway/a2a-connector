// connector/internal/proxy/transformer_test.go
package proxy_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/A2AGateway/a2agateway/connector/internal/proxy"
	"github.com/A2AGateway/a2agateway/saas/pkg/a2a"
)

func TestTransformer(t *testing.T) {
	// Create a transformer
	transformer := proxy.NewTransformer()

	// Set up a request transform function
	transformer.SetRequestTransform(func(data []byte) ([]byte, error) {
		var taskData a2a.Task
		if err := json.Unmarshal(data, &taskData); err != nil {
			return nil, err
		}

		// Extract customer ID from the message
		var customerID string
		if taskData.Status.Message != nil && len(taskData.Status.Message.Parts) > 0 {
			if textPart, ok := taskData.Status.Message.Parts[0].(a2a.TextPart); ok {
				// Extract "12345" from "Get customer data for ID: 12345"
				text := textPart.Text
				if idx := strings.LastIndex(text, ":"); idx != -1 {
					customerID = strings.TrimSpace(text[idx+1:])
				}
			}
		}

		// Create a legacy request
		legacyRequest := map[string]interface{}{
			"action": "getCustomerData",
			"params": map[string]interface{}{
				"customerID": customerID,
			},
		}

		return json.Marshal(legacyRequest)
	})

	// Create an A2A message
	textPart := a2a.NewTextPart("Get customer data for ID: 12345")
	message := a2a.NewMessage(a2a.RoleUser, []a2a.Part{textPart})

	// Create a task using the message
	task := a2a.NewTask("task-123", a2a.TaskStateSubmitted)
	task.WithMessage(message)

	// Serialize the task
	taskData, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	// Create a mock HTTP request with the task data
	req, err := http.NewRequest("POST", "http://example.com/api", strings.NewReader(string(taskData)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Transform the request
	transformer.TransformRequest(req)

	// Read the transformed body
	body := make([]byte, req.ContentLength)
	req.Body.Read(body)

	// Parse the transformed body
	var legacyRequest map[string]interface{}
	if err := json.Unmarshal(body, &legacyRequest); err != nil {
		t.Fatalf("Failed to unmarshal transformed request: %v", err)
	}

	// Check the transformation
	action, ok := legacyRequest["action"].(string)
	if !ok {
		t.Fatal("Action is not a string")
	}

	if action != "getCustomerData" {
		t.Errorf("Action mismatch: expected %s, got %s", "getCustomerData", action)
	}

	params, ok := legacyRequest["params"].(map[string]interface{})
	if !ok {
		t.Fatal("Params is not a map")
	}

	customerID, ok := params["customerID"].(string)
	if !ok {
		t.Fatal("CustomerID is not a string")
	}

	if customerID != "12345" {
		t.Errorf("CustomerID mismatch: expected %s, got %s", "12345", customerID)
	}
}
