package adapter

// AdapterType represents the type of system being adapted
type AdapterType string

const (
	REST  AdapterType = "rest"
	SOAP  AdapterType = "soap"
	DB    AdapterType = "db"
	File  AdapterType = "file"
	Other AdapterType = "other"
)

// Adapter defines the interface that all system adapters must implement
type Adapter interface {
	// Initialize sets up the adapter
	Initialize() error
	
	// GetCapabilities returns the capabilities of the adapted system
	GetCapabilities() (map[string]interface{}, error)
	
	// ExecuteTask executes a task on the adapted system
	ExecuteTask(action string, params map[string]interface{}) (map[string]interface{}, error)
	
	// Close cleans up resources
	Close() error
}

// BaseAdapter provides common functionality for adapters
type BaseAdapter struct {
	Name        string
	Type        AdapterType
	Description string
	Config      map[string]interface{}
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name string, adapterType AdapterType, description string, config map[string]interface{}) *BaseAdapter {
	return &BaseAdapter{
		Name:        name,
		Type:        adapterType,
		Description: description,
		Config:      config,
	}
}
