package tests

import (
	"testing"
)

// MockAdapter for testing
type MockAdapter struct {
	InitializeCalled  bool
	CloseCalled       bool
	ExecuteTaskAction string
	ExecuteTaskParams map[string]interface{}
}

func (m *MockAdapter) Initialize() error {
	m.InitializeCalled = true
	return nil
}

func (m *MockAdapter) GetCapabilities() (map[string]interface{}, error) {
	return map[string]interface{}{
		"type": "mock",
	}, nil
}

func (m *MockAdapter) ExecuteTask(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.ExecuteTaskAction = action
	m.ExecuteTaskParams = params
	return map[string]interface{}{
		"result": "mock_result",
	}, nil
}

func (m *MockAdapter) Close() error {
	m.CloseCalled = true
	return nil
}

func TestMockAdapter(t *testing.T) {
	mock := &MockAdapter{}

	// Test Initialize
	err := mock.Initialize()
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
	if !mock.InitializeCalled {
		t.Error("Initialize was not called")
	}

	// Test GetCapabilities
	caps, err := mock.GetCapabilities()
	if err != nil {
		t.Errorf("GetCapabilities failed: %v", err)
	}
	if caps["type"] != "mock" {
		t.Errorf("Expected capability type 'mock', got '%v'", caps["type"])
	}

	// Test ExecuteTask
	result, err := mock.ExecuteTask("test_action", map[string]interface{}{"param": "value"})
	if err != nil {
		t.Errorf("ExecuteTask failed: %v", err)
	}
	if mock.ExecuteTaskAction != "test_action" {
		t.Errorf("Expected action 'test_action', got '%s'", mock.ExecuteTaskAction)
	}
	if mock.ExecuteTaskParams["param"] != "value" {
		t.Errorf("Expected param 'value', got '%v'", mock.ExecuteTaskParams["param"])
	}
	if result["result"] != "mock_result" {
		t.Errorf("Expected result 'mock_result', got '%v'", result["result"])
	}

	// Test Close
	err = mock.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
	if !mock.CloseCalled {
		t.Error("Close was not called")
	}
}
