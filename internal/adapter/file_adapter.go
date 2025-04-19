package adapter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// FileAdapter adapts a file system
type FileAdapter struct {
	BaseAdapter
	BasePath string
}

// NewFileAdapter creates a new file system adapter
func NewFileAdapter(name, basePath string, config map[string]interface{}) *FileAdapter {
	base := NewBaseAdapter(name, File, "File System Adapter", config)
	return &FileAdapter{
		BaseAdapter: *base,
		BasePath:    basePath,
	}
}

// Initialize sets up the file adapter
func (a *FileAdapter) Initialize() error {
	// Check if base path exists
	_, err := os.Stat(a.BasePath)
	return err
}

// GetCapabilities returns the capabilities of the file system
func (a *FileAdapter) GetCapabilities() (map[string]interface{}, error) {
	// List files in the base directory
	files, err := ioutil.ReadDir(a.BasePath)
	if err != nil {
		return nil, err
	}
	
	var fileList []string
	for _, file := range files {
		fileList = append(fileList, file.Name())
	}
	
	return map[string]interface{}{
		"type":  "file",
		"files": fileList,
	}, nil
}

// ExecuteTask executes a file system operation
func (a *FileAdapter) ExecuteTask(action string, params map[string]interface{}) (map[string]interface{}, error) {
	switch action {
	case "read":
		return a.readFile(params)
	case "write":
		return a.writeFile(params)
	case "delete":
		return a.deleteFile(params)
	case "list":
		return a.listFiles(params)
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// readFile reads a file
func (a *FileAdapter) readFile(params map[string]interface{}) (map[string]interface{}, error) {
	filename, ok := params["filename"].(string)
	if !ok {
		return nil, fmt.Errorf("filename parameter is required")
	}
	
	path := filepath.Join(a.BasePath, filename)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"content": string(content),
	}, nil
}

// writeFile writes a file
func (a *FileAdapter) writeFile(params map[string]interface{}) (map[string]interface{}, error) {
	filename, ok := params["filename"].(string)
	if !ok {
		return nil, fmt.Errorf("filename parameter is required")
	}
	
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content parameter is required")
	}
	
	path := filepath.Join(a.BasePath, filename)
	err := ioutil.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"success": true,
	}, nil
}

// deleteFile deletes a file
func (a *FileAdapter) deleteFile(params map[string]interface{}) (map[string]interface{}, error) {
	filename, ok := params["filename"].(string)
	if !ok {
		return nil, fmt.Errorf("filename parameter is required")
	}
	
	path := filepath.Join(a.BasePath, filename)
	err := os.Remove(path)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"success": true,
	}, nil
}

// listFiles lists files in a directory
func (a *FileAdapter) listFiles(params map[string]interface{}) (map[string]interface{}, error) {
	dir := a.BasePath
	
	if dirParam, ok := params["directory"].(string); ok {
		dir = filepath.Join(a.BasePath, dirParam)
	}
	
	files,
    
cat > docs/deployment/connector_deployment.md << 'cat > docs/user/agent_configuration.md << 'EOF'
# Agent Configuration Guide

This guide explains how to configure and manage agents in the A2AGateway platform.

## Understanding Agents

In A2AGateway, an agent represents a system or service that can be discovered and used by A2A-compatible clients. Agents expose capabilities that define what actions they can perform.

## Agent Structure

An agent has the following components:

- **ID**: Unique identifier for the agent
- **Name**: Display name for the agent
- **Description**: Short description of the agent's purpose
- **Connector ID**: ID of the connector that manages the agent
- **Organization ID**: ID of the organization that owns the agent
- **Capabilities**: Definition of the agent's abilities

## Creating an Agent

Agents can be created in two ways:

1. **Automatically**: The connector can automatically generate agents based on its adapters
2. **Manually**: You can create agents manually through the API or dashboard

### Automatic Agent Creation

To enable automatic agent creation:

1. Configure the adapters in your connector
2. Set `auto_create_agents: true` in the connector configuration
3. Start the connector
4. The connector will scan the adapters and create agents for each system

### Manual Agent Creation

To create an agent manually:

1. Log in to the A2AGateway dashboard
2. Go to "Agents" and click "Create Agent"
3. Enter the agent details
4. Define the agent's capabilities
5. Link the agent to a connector
6. Save the agent

## Defining Capabilities

Agent capabilities define what actions the agent can perform. A capability has:

- **Name**: Name of the action (e.g., "query", "create", "update")
- **Description**: Description of what the action does
- **Parameters**: Definition of the input parameters
- **Result**: Definition of the output format

Example capability definition:

```json
{
  "operations": [
    {
      "name": "query",
      "description": "Execute a SQL query",
      "params": [
        {
          "name": "query",
          "type": "string",
          "required": true,
          "description": "SQL query to execute"
        }
      ]
    }
  ],
  "data_types": ["string", "number", "boolean", "date", "array", "object"],
  "schemas": {
    "CUSTOMERS": {
      "ID": "number",
      "NAME": "string",
      "STATUS": "string"
    }
  }
}
```

## Agent Cards

Agent cards are public descriptions of an agent's capabilities that follow the A2A protocol. A2AGateway generates agent cards automatically based on your agent configuration.

## Managing Agents

### Updating an Agent

To update an agent:

1. Log in to the A2AGateway dashboard
2. Go to "Agents" and select the agent
3. Click "Edit"
4. Make your changes
5. Save the agent

### Deleting an Agent

To delete an agent:

1. Log in to the A2AGateway dashboard
2. Go to "Agents" and select the agent
3. Click "Delete"
4. Confirm the deletion

## Agent Discovery

Agents can be discovered by A2A-compatible clients through:

1. **Registry API**: Clients can query the registry API to find agents
2. **Agent Cards**: Clients can retrieve agent cards to understand capabilities

## Agent Security

Secure your agents by:

1. **Access Control**: Define who can access each agent
2. **Authentication**: Require authentication for agent access
3. **Authorization**: Define what actions are allowed for each user/role
4. **Logging**: Monitor agent usage

## Testing an Agent

To test an agent:

1. Log in to the A2AGateway dashboard
2. Go to "Agents" and select the agent
3. Click "Test"
4. Select an operation to test
5. Enter test parameters
6. Run the test
7. View the results

## Best Practices

1. **Clear Naming**: Use clear, descriptive names for agents and operations
2. **Complete Documentation**: Provide detailed descriptions for all capabilities
3. **Proper Schemas**: Define data schemas accurately
4. **Regular Updates**: Keep agent capabilities up to date
5. **Security First**: Always consider security implications
6. **Performance Monitoring**: Monitor agent performance
7. **Version Management**: Consider versioning for agent capabilities
