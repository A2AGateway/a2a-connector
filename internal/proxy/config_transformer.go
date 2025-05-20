package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/A2AGateway/a2agateway/connector/internal/config"
	"github.com/A2AGateway/a2agateway/saas/pkg/a2a"
)

// ConfigTransformer is a transformer that uses configuration to transform requests and responses
type ConfigTransformer struct {
	Config     *config.ConnectorConfig
	Transformer
}

// NewConfigTransformer creates a new transformer based on configuration
func NewConfigTransformer(cfg *config.ConnectorConfig) *ConfigTransformer {
	t := &ConfigTransformer{
		Config:     cfg,
		Transformer: *NewTransformer(),
	}

	// Set up transformation functions
	t.SetRequestTransform(t.transformRequest)
	t.SetResponseTransform(t.transformResponse)

	// Set up headers from config
	for key, value := range cfg.Adapter.Headers {
		t.SetRequestHeader(key, value)
	}

	return t
}

// transformRequest transforms an A2A task to a legacy request format
func (t *ConfigTransformer) transformRequest(data []byte) ([]byte, error) {
	// Parse A2A task from JSON
	var taskMap map[string]interface{}
	if err := json.Unmarshal(data, &taskMap); err != nil {
		log.Printf("Error unmarshaling A2A task: %v", err)
		return nil, err
	}

	// Extract text from the message parts
	text, err := extractTextFromTask(taskMap)
	if err != nil {
		return nil, err
	}

	// Find matching mapping configuration
	mappingConfig, err := t.findMatchingMapping(text)
	if err != nil {
		return nil, err
	}

	// Extract parameters from the task
	params, err := t.extractParameters(mappingConfig, taskMap, text)
	if err != nil {
		return nil, err
	}

	// Get task ID for tracking
	taskID := getTaskID(taskMap)

	// Create the legacy request
	legacyRequest := map[string]interface{}{
		"action": mappingConfig.Method,
		"params": params,
		"meta": map[string]interface{}{
			"taskId":     taskID,
			"timestamp":  time.Now().Format(time.RFC3339),
			"endpoint":   renderEndpoint(mappingConfig.Endpoint, params),
			"mappingId":  mappingConfig.IntentPattern,
		},
	}

	// Apply global transformation rules
	for _, rule := range t.Config.Transforms.A2AToLegacy {
		applyTransformRule(rule, taskMap, legacyRequest)
	}

	return json.Marshal(legacyRequest)
}

// transformResponse transforms a legacy response to an A2A task
func (t *ConfigTransformer) transformResponse(data []byte) ([]byte, error) {
	// Parse legacy response
	var legacyResponse map[string]interface{}
	if err := json.Unmarshal(data, &legacyResponse); err != nil {
		log.Printf("Error unmarshaling legacy response: %v", err)
		return nil, err
	}

	// Get task ID from metadata
	taskID := "unknown-task"
	if meta, ok := legacyResponse["meta"].(map[string]interface{}); ok {
		if id, ok := meta["taskId"].(string); ok {
			taskID = id
		}
	}

	// Get mapping config ID
	mappingID := ""
	if meta, ok := legacyResponse["meta"].(map[string]interface{}); ok {
		if id, ok := meta["mappingId"].(string); ok {
			mappingID = id
		}
	}

	// Find mapping config
	var responseTransform config.ResponseTransform
	for _, mapping := range t.Config.Mappings {
		if mapping.IntentPattern == mappingID {
			responseTransform = mapping.ResponseTransform
			break
		}
	}

	// Determine task state
	taskState := string(a2a.TaskStateCompleted)
	if status, ok := legacyResponse["status"].(string); ok {
		if status != "success" {
			taskState = string(a2a.TaskStateFailed)
		}
	}
	if err, ok := legacyResponse["error"].(string); ok && err != "" {
		taskState = string(a2a.TaskStateFailed)
	}

	// Build parts array
	parts := []map[string]interface{}{}

	// Add text part if we have a template
	if responseTransform.Template != "" && responseTransform.compiled != nil {
		var buf bytes.Buffer
		if err := responseTransform.compiled.Execute(&buf, legacyResponse); err == nil {
			textPart := map[string]interface{}{
				"type": "text",
				"text": buf.String(),
			}
			parts = append(parts, textPart)
		}
	} else {
		// Default text response
		textContent := ""
		if status, ok := legacyResponse["status"].(string); ok {
			textContent += "Status: " + status + "\n"
		}
		if error, ok := legacyResponse["error"].(string); ok && error != "" {
			textContent += "Error: " + error + "\n"
		}
		
		if textContent != "" {
			parts = append(parts, map[string]interface{}{
				"type": "text",
				"text": textContent,
			})
		}
	}

	// Add data part with the result
	if result, ok := legacyResponse["result"].(map[string]interface{}); ok {
		parts = append(parts, map[string]interface{}{
			"type": "data",
			"data": result,
		})
	}

	// Create a message with the parts
	message := map[string]interface{}{
		"role":  "agent",
		"parts": parts,
	}

	// Create the task
	task := map[string]interface{}{
		"id": taskID,
		"status": map[string]interface{}{
			"state":     taskState,
			"message":   message,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// Add metadata from the legacy response
	if meta, ok := legacyResponse["meta"].(map[string]interface{}); ok {
		task["metadata"] = meta
	}

	// Apply global transformation rules
	for _, rule := range t.Config.Transforms.LegacyToA2A {
		applyTransformRule(rule, legacyResponse, task)
	}

	return json.Marshal(task)
}

// findMatchingMapping finds the mapping configuration that matches the text
func (t *ConfigTransformer) findMatchingMapping(text string) (*config.MappingConfig, error) {
	text = strings.ToLower(text)
	
	for i := range t.Config.Mappings {
		mapping := &t.Config.Mappings[i]
		if mapping.compiledPattern != nil && mapping.compiledPattern.MatchString(text) {
			return mapping, nil
		}
	}
	
	return nil, fmt.Errorf("no matching mapping found for text: %s", text)
}

// extractParameters extracts parameters from the task using parameter mappings
func (t *ConfigTransformer) extractParameters(mapping *config.MappingConfig, taskMap map[string]interface{}, text string) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	
	// Apply parameter mappings
	for _, paramMapping := range mapping.ParameterMappings {
		if paramMapping.Source == "text" {
			if paramMapping.compiled != nil && paramMapping.compiled.MatchString(text) {
				matches := paramMapping.compiled.FindStringSubmatch(text)
				if len(matches) > 1 {
					// Extract captured group and set in params
					setValue(params, paramMapping.Target, matches[1])
				}
			} else if paramMapping.Default != "" {
				// Use default value if no match
				setValue(params, paramMapping.Target, paramMapping.Default)
			}
		} else {
			// Extract value from task using path
			value := getValueByPath(taskMap, paramMapping.Source)
			if value != nil {
				setValue(params, paramMapping.Target, value)
			} else if paramMapping.Default != "" {
				setValue(params, paramMapping.Target, paramMapping.Default)
			}
		}
	}
	
	return params, nil
}

// Helper functions

// extractTextFromTask extracts text from the message parts
func extractTextFromTask(taskMap map[string]interface{}) (string, error) {
	text := ""
	
	status, ok := taskMap["status"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("status field not found or not an object")
	}
	
	message, ok := status["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("message field not found or not an object")
	}
	
	parts, ok := message["parts"].([]interface{})
	if !ok {
		return "", fmt.Errorf("parts field not found or not an array")
	}
	
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		
		partType, ok := partMap["type"].(string)
		if !ok {
			continue
		}
		
		if partType == "text" {
			if partText, ok := partMap["text"].(string); ok {
				if text != "" {
					text += " "
				}
				text += partText
			}
		}
	}
	
	return text, nil
}

// getTaskID gets the task ID from the task map
func getTaskID(taskMap map[string]interface{}) string {
	if id, ok := taskMap["id"].(string); ok {
		return id
	}
	return fmt.Sprintf("task-%d", time.Now().Unix())
}

// renderEndpoint renders the endpoint with parameter values
func renderEndpoint(endpoint string, params map[string]interface{}) string {
	result := endpoint
	
	// Replace {param} placeholders
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(endpoint, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			placeholder := "{" + paramName + "}"
			
			// Find param value
			if value, ok := params[paramName].(string); ok {
				result = strings.ReplaceAll(result, placeholder, value)
			} else if valueNum, ok := params[paramName].(float64); ok {
				result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", valueNum))
			} else if valueBool, ok := params[paramName].(bool); ok {
				result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", valueBool))
			}
		}
	}
	
	return result
}

// getValueByPath gets a value from a nested map using a dot-notation path
func getValueByPath(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		
		if nextMap, ok := current[part].(map[string]interface{}); ok {
			current = nextMap
		} else if nextMap, ok := current[part].(map[string]string); ok {
			// Convert map[string]string to map[string]interface{}
			converted := make(map[string]interface{})
			for k, v := range nextMap {
				converted[k] = v
			}
			current = converted
		} else {
			return nil
		}
	}
	
	return nil
}

// setValue sets a value in a nested map using a dot-notation path
func setValue(data map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}
		
		if nextMap, ok := current[part].(map[string]interface{}); ok {
			current = nextMap
		} else {
			// Convert to map if possible
			if converted, ok := current[part].(map[string]string); ok {
				newMap := make(map[string]interface{})
				for k, v := range converted {
					newMap[k] = v
				}
				current[part] = newMap
				current = newMap
			} else {
				// Cannot traverse further, reset the path
				current[part] = make(map[string]interface{})
				current = current[part].(map[string]interface{})
			}
		}
	}
}

// applyTransformRule applies a transformation rule to convert between data formats
func applyTransformRule(rule config.TransformRule, source, target map[string]interface{}) {
	sourceValue := getValueByPath(source, rule.Source)
	if sourceValue == nil {
		return
	}
	
	var targetValue interface{} = sourceValue
	
	// Apply regex if provided
	if rule.Regex != "" && rule.compiled != nil {
		if sourceStr, ok := sourceValue.(string); ok {
			if rule.compiled.MatchString(sourceStr) {
				matches := rule.compiled.FindStringSubmatch(sourceStr)
				if len(matches) > 1 {
					targetValue = matches[1]
				}
			}
		}
	}
	
	// Apply template if provided
	if rule.Template != "" {
		if sourceStr, ok := sourceValue.(string); ok {
			targetValue = strings.ReplaceAll(rule.Template, "{value}", sourceStr)
		}
	}
	
	// Set the transformed value
	setValue(target, rule.Target, targetValue)
}