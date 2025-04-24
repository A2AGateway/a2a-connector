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
    
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    
    fileList := make([]map[string]interface{}, 0, len(files))
    for _, file := range files {
        fileInfo := map[string]interface{}{
            "name":  file.Name(),
            "size":  file.Size(),
            "isDir": file.IsDir(),
            "mode":  file.Mode().String(),
        }
        fileList = append(fileList, fileInfo)
    }
    
    return map[string]interface{}{
        "files": fileList,
    }, nil
}
    
