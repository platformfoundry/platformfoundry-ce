// Package crd provides Custom Resource Definition support for Platform Foundry.
package crd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser parses CRD and custom resource YAML files
type Parser struct {
	registry *Registry
}

// NewParser creates a new parser
func NewParser(registry *Registry) *Parser {
	if registry == nil {
		registry = DefaultRegistry
	}
	return &Parser{registry: registry}
}

// ParseFile parses a CRD or custom resource from a file
func (p *Parser) ParseFile(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ParseError{
			Path:    path,
			Message: "failed to read file",
			Cause:   err,
		}
	}

	return p.Parse(data)
}

// ParseDir parses all YAML files in a directory
func (p *Parser) ParseDir(dir string) ([]interface{}, error) {
	var results []interface{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		result, err := p.ParseFile(path)
		if err != nil {
			return err
		}

		results = append(results, result)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// Parse parses a CRD or custom resource from YAML data
func (p *Parser) Parse(data []byte) (interface{}, error) {
	// First, parse to get kind
	var base struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}

	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, &ParseError{
			Message: "failed to parse YAML",
			Cause:   err,
		}
	}

	// Parse based on kind
	if base.Kind == "CustomResourceDefinition" {
		return p.parseCRD(data)
	}

	return p.parseCustomResource(data, base.APIVersion, base.Kind)
}

// ParseMulti parses multiple documents from a YAML stream
func (p *Parser) ParseMulti(r io.Reader) ([]interface{}, error) {
	var results []interface{}

	decoder := yaml.NewDecoder(r)
	for {
		var raw map[string]interface{}
		if err := decoder.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, &ParseError{
				Message: "failed to parse YAML document",
				Cause:   err,
			}
		}

		// Convert back to YAML for reprocessing
		data, err := yaml.Marshal(raw)
		if err != nil {
			return nil, &ParseError{
				Message: "failed to marshal document",
				Cause:   err,
			}
		}

		result, err := p.Parse(data)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func (p *Parser) parseCRD(data []byte) (*CustomResourceDefinition, error) {
	var crd CustomResourceDefinition
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return nil, &ParseError{
			Message: "failed to parse CRD",
			Cause:   err,
		}
	}

	// Validate
	if crd.Kind != "CustomResourceDefinition" {
		return nil, &ParseError{
			Message: fmt.Sprintf("expected kind CustomResourceDefinition, got %s", crd.Kind),
		}
	}

	// Set defaults
	if crd.Spec.Plural == "" {
		crd.Spec.Plural = strings.ToLower(crd.Spec.Kind) + "s"
	}
	if crd.Spec.Singular == "" {
		crd.Spec.Singular = strings.ToLower(crd.Spec.Kind)
	}

	return &crd, nil
}

func (p *Parser) parseCustomResource(data []byte, apiVersion, kind string) (*CustomResource, error) {
	// Parse group and version
	group, version := parseAPIVersion(apiVersion)

	// Check if CRD is registered
	gvk := GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	}

	if !p.registry.HasGVK(gvk) {
		return nil, &ParseError{
			Message: fmt.Sprintf("unknown resource type: %s", gvk),
		}
	}

	// Parse custom resource
	var cr CustomResource
	if err := yaml.Unmarshal(data, &cr); err != nil {
		return nil, &ParseError{
			Message: "failed to parse custom resource",
			Cause:   err,
		}
	}

	return &cr, nil
}

// ParseCRD parses a CRD from YAML data
func ParseCRD(data []byte) (*CustomResourceDefinition, error) {
	parser := NewParser(nil)
	result, err := parser.Parse(data)
	if err != nil {
		return nil, err
	}
	crd, ok := result.(*CustomResourceDefinition)
	if !ok {
		return nil, &ParseError{
			Message: "expected CustomResourceDefinition",
		}
	}
	return crd, nil
}

// ParseCustomResource parses a custom resource from YAML data
func ParseCustomResource(data []byte) (*CustomResource, error) {
	var cr CustomResource
	if err := yaml.Unmarshal(data, &cr); err != nil {
		return nil, &ParseError{
			Message: "failed to parse custom resource",
			Cause:   err,
		}
	}
	return &cr, nil
}

// ParseError represents a parse error
type ParseError struct {
	Path    string
	Message string
	Cause   error
}

func (e *ParseError) Error() string {
	msg := e.Message
	if e.Path != "" {
		msg = e.Path + ": " + msg
	}
	if e.Cause != nil {
		msg = msg + ": " + e.Cause.Error()
	}
	return msg
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

// LoadCRDs loads all CRDs from a directory and registers them
func LoadCRDs(dir string, registry *Registry) error {
	if registry == nil {
		registry = DefaultRegistry
	}

	parser := NewParser(registry)
	results, err := parser.ParseDir(dir)
	if err != nil {
		return err
	}

	for _, result := range results {
		if crd, ok := result.(*CustomResourceDefinition); ok {
			if err := registry.Register(crd); err != nil {
				return err
			}
		}
	}

	return nil
}

// CRDBuilder helps build CRDs programmatically
type CRDBuilder struct {
	crd CustomResourceDefinition
}

// NewCRDBuilder creates a new CRD builder
func NewCRDBuilder(name string) *CRDBuilder {
	return &CRDBuilder{
		crd: CustomResourceDefinition{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "CustomResourceDefinition",
			Metadata: Metadata{
				Name: name,
			},
		},
	}
}

// Group sets the API group
func (b *CRDBuilder) Group(group string) *CRDBuilder {
	b.crd.Spec.Group = group
	return b
}

// Version sets the API version
func (b *CRDBuilder) Version(version string) *CRDBuilder {
	b.crd.Spec.Version = version
	return b
}

// Kind sets the resource kind
func (b *CRDBuilder) Kind(kind string) *CRDBuilder {
	b.crd.Spec.Kind = kind
	return b
}

// Plural sets the plural name
func (b *CRDBuilder) Plural(plural string) *CRDBuilder {
	b.crd.Spec.Plural = plural
	return b
}

// Scope sets the resource scope
func (b *CRDBuilder) Scope(scope ResourceScope) *CRDBuilder {
	b.crd.Spec.Scope = scope
	return b
}

// Schema sets the validation schema
func (b *CRDBuilder) Schema(schema *JSONSchemaProps) *CRDBuilder {
	b.crd.Spec.Schema = schema
	return b
}

// Handler sets the CRD handler
func (b *CRDBuilder) Handler(plugin, version string) *CRDBuilder {
	b.crd.Spec.Handler = &CRDHandler{
		Plugin:  plugin,
		Version: version,
	}
	return b
}

// Build returns the constructed CRD
func (b *CRDBuilder) Build() *CustomResourceDefinition {
	// Set defaults
	if b.crd.Spec.Plural == "" && b.crd.Spec.Kind != "" {
		b.crd.Spec.Plural = strings.ToLower(b.crd.Spec.Kind) + "s"
	}
	if b.crd.Spec.Singular == "" && b.crd.Spec.Kind != "" {
		b.crd.Spec.Singular = strings.ToLower(b.crd.Spec.Kind)
	}
	return &b.crd
}
