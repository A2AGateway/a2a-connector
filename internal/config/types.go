// Package config provides structures and functions for configuration-driven integration
package config

import (
	"regexp"
	"strings"
	"text/template"
)

// ConnectorConfig represents the full configuration for a connector
type ConnectorConfig struct {
	Adapter    AdapterConfig     `yaml:"adapter" json:"adapter"`
	Mappings   []MappingConfig   `yaml:"mappings" json:"mappings"`
	Transforms TransformConfig   `yaml:"transforms" json:"transforms"`
	Variables  map[string]string `yaml:"variables" json:"variables,omitempty"`
}

// AdapterConfig represents the configuration for a specific adapter
type AdapterConfig struct {
	Type    string            `yaml:"type" json:"type"`
	Name    string            `yaml:"name" json:"name"`
	BaseURL string            `yaml:"baseUrl" json:"baseUrl"`
	Auth    AuthConfig        `yaml:"auth" json:"auth,omitempty"`
	Headers map[string]string `yaml:"headers" json:"headers,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string `yaml:"type" json:"type"`
	Username string `yaml:"username" json:"username,omitempty"`
	Password string `yaml:"password" json:"password,omitempty"`
	Token    string `yaml:"token" json:"token,omitempty"`
	KeyName  string `yaml:"keyName" json:"keyName,omitempty"`
}

// MappingConfig represents a mapping between A2A tasks and legacy endpoints
type MappingConfig struct {
	IntentPattern     string              `yaml:"intentPattern" json:"intentPattern"`
	Endpoint          string              `yaml:"endpoint" json:"endpoint"`
	Method            string              `yaml:"method" json:"method"`
	ParameterMappings []ParameterMapping  `yaml:"parameterMappings" json:"parameterMappings,omitempty"`
	ResponseTransform ResponseTransform   `yaml:"responseTransform" json:"responseTransform,omitempty"`
	compiledPattern   *regexp.Regexp      // Not exported, used internally
	compiledTemplate  *template.Template  // Not exported, used internally
}

// ParameterMapping represents how to extract parameters from A2A tasks
type ParameterMapping struct {
	Source   string `yaml:"source" json:"source"`
	Pattern  string `yaml:"pattern" json:"pattern"`
	Target   string `yaml:"target" json:"target"`
	Default  string `yaml:"default" json:"default,omitempty"`
	compiled *regexp.Regexp
}

// ResponseTransform defines how to transform legacy responses to A2A format
type ResponseTransform struct {
	Template   string            `yaml:"template" json:"template,omitempty"`
	Mappings   map[string]string `yaml:"mappings" json:"mappings,omitempty"`
	StatusPath string            `yaml:"statusPath" json:"statusPath,omitempty"`
	ErrorPath  string            `yaml:"errorPath" json:"errorPath,omitempty"`
	compiled   *template.Template
}

// TransformConfig defines global transformation rules
type TransformConfig struct {
	A2AToLegacy  []TransformRule `yaml:"a2aToLegacy" json:"a2aToLegacy,omitempty"`
	LegacyToA2A  []TransformRule `yaml:"legacyToA2a" json:"legacyToA2a,omitempty"`
}

// TransformRule defines a single transformation rule
type TransformRule struct {
	Source   string `yaml:"source" json:"source"`
	Target   string `yaml:"target" json:"target"`
	Regex    string `yaml:"regex" json:"regex,omitempty"`
	Template string `yaml:"template" json:"template,omitempty"`
	compiled *regexp.Regexp
}

// Compile compiles all regular expressions and templates in the configuration
func (c *ConnectorConfig) Compile() error {
	// Compile mappings
	for i := range c.Mappings {
		// Compile intent pattern
		pattern, err := regexp.Compile(strings.ToLower(c.Mappings[i].IntentPattern))
		if err != nil {
			return err
		}
		c.Mappings[i].compiledPattern = pattern

		// Compile parameter patterns
		for j := range c.Mappings[i].ParameterMappings {
			pattern, err := regexp.Compile(c.Mappings[i].ParameterMappings[j].Pattern)
			if err != nil {
				return err
			}
			c.Mappings[i].ParameterMappings[j].compiled = pattern
		}

		// Compile response template
		if c.Mappings[i].ResponseTransform.Template != "" {
			tmpl, err := template.New("response").Parse(c.Mappings[i].ResponseTransform.Template)
			if err != nil {
				return err
			}
			c.Mappings[i].ResponseTransform.compiled = tmpl
		}
	}

	// Compile transform rules
	for i := range c.Transforms.A2AToLegacy {
		if c.Transforms.A2AToLegacy[i].Regex != "" {
			pattern, err := regexp.Compile(c.Transforms.A2AToLegacy[i].Regex)
			if err != nil {
				return err
			}
			c.Transforms.A2AToLegacy[i].compiled = pattern
		}
	}

	for i := range c.Transforms.LegacyToA2A {
		if c.Transforms.LegacyToA2A[i].Regex != "" {
			pattern, err := regexp.Compile(c.Transforms.LegacyToA2A[i].Regex)
			if err != nil {
				return err
			}
			c.Transforms.LegacyToA2A[i].compiled = pattern
		}
	}

	return nil
}

// ResolveVariables replaces variable placeholders in config strings
func (c *ConnectorConfig) ResolveVariables() {
	// Resolve variables in various fields
	c.Adapter.BaseURL = resolveVariablesInString(c.Adapter.BaseURL, c.Variables)
	c.Adapter.Auth.Username = resolveVariablesInString(c.Adapter.Auth.Username, c.Variables)
	c.Adapter.Auth.Password = resolveVariablesInString(c.Adapter.Auth.Password, c.Variables)
	c.Adapter.Auth.Token = resolveVariablesInString(c.Adapter.Auth.Token, c.Variables)

	// Resolve variables in headers
	for key, value := range c.Adapter.Headers {
		c.Adapter.Headers[key] = resolveVariablesInString(value, c.Variables)
	}
}

// resolveVariablesInString replaces ${VAR} with the actual variable value
func resolveVariablesInString(s string, vars map[string]string) string {
	result := s
	for k, v := range vars {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
	}
	return result
}