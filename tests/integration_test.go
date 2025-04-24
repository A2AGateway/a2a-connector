// tests/integration_test.go
package tests

import (
    "testing"
    "github.com/A2AGateway/a2agateway/connector/internal/adapter"
    "github.com/A2AGateway/a2agateway/connector/internal/proxy"
    "github.com/A2AGateway/a2agateway/saas/pkg/a2a"
)

func TestSimpleFlow(t *testing.T) {
    // Create components
    protocol := a2a.NewProtocol()
    transformer := proxy.NewTransformer()
    mockAdapter := adapter.NewMockAdapter()
    
    // Create an A2A task
    textPart := a2a.NewTextPart("Get customer data for ID: 12345")
    message := a2a.NewMessage(a2a.RoleUser, []a2a.Part{textPart})
    task := a2a.NewTask("task-123", a2a.TaskStateSubmitted)
    task.WithMessage(message)
    
    // Transform to legacy format
    legacyRequest, err := transformer.TransformA2AToLegacy(*task)
    if err != nil {
        t.Fatalf("Failed to transform to legacy: %v", err)
    }
    
    // Execute through the adapter
    legacyResponse, err := mockAdapter.Execute(legacyRequest)
    if err != nil {
        t.Fatalf("Failed to execute: %v", err)
    }
    
    // Transform back to A2A
    a2aResponse, err := transformer.TransformLegacyToA2A(legacyResponse)
    if err != nil {
        t.Fatalf("Failed to transform to A2A: %v", err)
    }
    
    // Check the response
    if a2aResponse.Status.State != a2a.TaskStateCompleted {
        t.Errorf("Status mismatch: expected %s, got %s", a2a.TaskStateCompleted, a2aResponse.Status.State)
    }
    
    // Verify the response contains the expected data
    // (Add appropriate checks based on your mock adapter implementation)
}