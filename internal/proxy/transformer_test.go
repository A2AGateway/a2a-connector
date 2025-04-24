// connector/internal/proxy/transformer_test.go
package proxy_test

import (
    "testing"
    "github.com/A2AGateway/a2agateway/connector/internal/proxy"
    "github.com/A2AGateway/a2agateway/saas/pkg/a2a"
)

func TestTransformA2AToLegacy(t *testing.T) {
    // Create a transformer
    transformer := proxy.NewTransformer()
    
    // Create an A2A message
    textPart := a2a.NewTextPart("Get customer data for ID: 12345")
    message := a2a.NewMessage(a2a.RoleUser, []a2a.Part{textPart})
    
    // Create a task using the message
    task := a2a.NewTask("task-123", a2a.TaskStateSubmitted)
    task.WithMessage(message)
    
    // Transform to legacy format (assuming your transformer returns a map)
    legacyRequest, err := transformer.TransformA2AToLegacy(*task)
    if err != nil {
        t.Fatalf("Failed to transform: %v", err)
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