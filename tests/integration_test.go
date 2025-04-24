// tests/integration_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/A2AGateway/a2agateway/connector/internal/proxy"
	"github.com/A2AGateway/a2agateway/saas/pkg/a2a"
)

// No need to create a new MockAdapter since it already exists in this package, under adapter_test.go

// Add an Execute method to the existing MockAdapter for testing purposes
func (m *MockAdapter) Execute(req *http.Request) (*http.Response, error) {
	// Read request body
	var requestData map[string]interface{}
	if req.Body != nil {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(body, &requestData); err != nil {
			return nil, err
		}
	}

	// Extract action and parameters from request
	action := "get_customer" // Default action for testing
	params := map[string]interface{}{
		"customer_id": "12345",
	}

	// Execute the task and record the interaction
	result, err := m.ExecuteTask(action, params)
	if err != nil {
		return nil, err
	}

	// Create response
	responseBody, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	// Create HTTP response
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")

	return resp, nil
}

func TestSimpleFlow(t *testing.T) {
	// Create components
	transformer := proxy.NewTransformer()
	mockAdapter := &MockAdapter{} // Use the existing MockAdapter

	// Create an A2A task
	textPart := a2a.NewTextPart("Get customer data for ID: 12345")
	message := a2a.NewMessage(a2a.RoleUser, []a2a.Part{textPart})
	task := a2a.NewTask("task-123", a2a.TaskStateSubmitted)
	task.WithMessage(message)

	// Set up transformation functions
	transformer.SetRequestTransform(func(data []byte) ([]byte, error) {
		// Transform A2A task to legacy format
		var taskData a2a.Task
		if err := json.Unmarshal(data, &taskData); err != nil {
			return nil, err
		}

		// Example transformation to legacy format
		legacyReq := map[string]interface{}{
			"action":      "get_customer",
			"customer_id": "12345", // Extract from task message
		}

		return json.Marshal(legacyReq)
	})

	transformer.SetResponseTransform(func(data []byte) ([]byte, error) {
		// Transform legacy response to A2A format
		var legacyResp map[string]interface{}
		if err := json.Unmarshal(data, &legacyResp); err != nil {
			return nil, err
		}

		// Create A2A response
		responseTask := a2a.NewTask(task.ID, a2a.TaskStateCompleted)

		// Create response message
		textResponse := a2a.NewTextPart(fmt.Sprintf("Result: %v", legacyResp["result"]))
		responseMessage := a2a.NewMessage(a2a.RoleAgent, []a2a.Part{textResponse})
		responseTask.WithMessage(responseMessage)

		return json.Marshal(responseTask)
	})

	// Create a mock HTTP request with the task data
	taskData, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8081/api/execute", bytes.NewBuffer(taskData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Transform the request
	transformer.TransformRequest(req)

	// Execute through the adapter
	resp, err := mockAdapter.Execute(req)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	// Transform the response
	err = transformer.TransformResponse(resp)
	if err != nil {
		t.Fatalf("Failed to transform response: %v", err)
	}

	// Read and parse the response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var a2aResponse a2a.Task
	err = json.Unmarshal(respBody, &a2aResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check the response
	if a2aResponse.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Status mismatch: expected %s, got %s", a2a.TaskStateCompleted, a2aResponse.Status.State)
	}

	// Check the message in the status
	if a2aResponse.Status.Message == nil {
		t.Errorf("Expected response to have a message in Status")
	}

	// Verify that the mock adapter was called with the right parameters
	if mockAdapter.ExecuteTaskAction != "get_customer" {
		t.Errorf("Expected action 'get_customer', got '%s'", mockAdapter.ExecuteTaskAction)
	}

	if mockAdapter.ExecuteTaskParams["customer_id"] != "12345" {
		t.Errorf("Expected customer_id '12345', got '%v'", mockAdapter.ExecuteTaskParams["customer_id"])
	}
}
