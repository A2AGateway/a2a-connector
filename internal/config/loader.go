package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a file in YAML or JSON format
func LoadFromFile(filePath string) (*ConnectorConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Determine file type based on extension
	ext := strings.ToLower(filepath.Ext(filePath))
	var config ConnectorConfig

	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &config)
	case ".json":
		err = json.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported file format: %s. Please use .yaml, .yml, or .json", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	// Process environment variables
	processEnvironmentVariables(&config)

	// Resolve variable references
	config.ResolveVariables()

	// Compile regular expressions and templates
	if err := config.Compile(); err != nil {
		return nil, fmt.Errorf("error compiling regular expressions: %v", err)
	}

	return &config, nil
}

// processEnvironmentVariables loads environment variables into the configuration
func processEnvironmentVariables(config *ConnectorConfig) {
	if config.Variables == nil {
		config.Variables = make(map[string]string)
	}

	// Load environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			// Only add environment variables that start with specified prefixes
			if strings.HasPrefix(parts[0], "A2A_") ||
				strings.HasPrefix(parts[0], "CONNECTOR_") {
				config.Variables[parts[0]] = parts[1]
			}
		}
	}
}

// ValidateConfig validates that the configuration is complete and usable
func ValidateConfig(config *ConnectorConfig) error {
	// Validate adapter configuration
	if config.Adapter.Type == "" {
		return fmt.Errorf("adapter type is required")
	}
	if config.Adapter.BaseURL == "" {
		return fmt.Errorf("adapter baseUrl is required")
	}

	// Validate mappings
	if len(config.Mappings) == 0 {
		return fmt.Errorf("at least one mapping is required")
	}
	for i, mapping := range config.Mappings {
		if mapping.IntentPattern == "" {
			return fmt.Errorf("mapping %d is missing intentPattern", i)
		}
		if mapping.Endpoint == "" {
			return fmt.Errorf("mapping %d is missing endpoint", i)
		}
		if mapping.Method == "" {
			return fmt.Errorf("mapping %d is missing method", i)
		}
	}

	return nil
}