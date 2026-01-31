// Package crd provides Custom Resource Definition support for Platform Foundry.
package crd

// JSONSchemaProps defines the JSON schema for validation
type JSONSchemaProps struct {
	// ID is the schema ID
	ID string `yaml:"id,omitempty" json:"id,omitempty"`

	// Schema is the JSON schema version
	Schema string `yaml:"$schema,omitempty" json:"$schema,omitempty"`

	// Ref is a reference to another schema
	Ref *string `yaml:"$ref,omitempty" json:"$ref,omitempty"`

	// Description describes this schema
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Type is the JSON type
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Format is the format (e.g., "date-time", "email")
	Format string `yaml:"format,omitempty" json:"format,omitempty"`

	// Title is a title for this schema
	Title string `yaml:"title,omitempty" json:"title,omitempty"`

	// Default is the default value
	Default interface{} `yaml:"default,omitempty" json:"default,omitempty"`

	// Maximum is the maximum for numbers
	Maximum *float64 `yaml:"maximum,omitempty" json:"maximum,omitempty"`

	// ExclusiveMaximum indicates if maximum is exclusive
	ExclusiveMaximum bool `yaml:"exclusiveMaximum,omitempty" json:"exclusiveMaximum,omitempty"`

	// Minimum is the minimum for numbers
	Minimum *float64 `yaml:"minimum,omitempty" json:"minimum,omitempty"`

	// ExclusiveMinimum indicates if minimum is exclusive
	ExclusiveMinimum bool `yaml:"exclusiveMinimum,omitempty" json:"exclusiveMinimum,omitempty"`

	// MaxLength is the maximum string length
	MaxLength *int64 `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`

	// MinLength is the minimum string length
	MinLength *int64 `yaml:"minLength,omitempty" json:"minLength,omitempty"`

	// Pattern is a regex pattern for strings
	Pattern string `yaml:"pattern,omitempty" json:"pattern,omitempty"`

	// MaxItems is the maximum array items
	MaxItems *int64 `yaml:"maxItems,omitempty" json:"maxItems,omitempty"`

	// MinItems is the minimum array items
	MinItems *int64 `yaml:"minItems,omitempty" json:"minItems,omitempty"`

	// UniqueItems requires array items to be unique
	UniqueItems bool `yaml:"uniqueItems,omitempty" json:"uniqueItems,omitempty"`

	// MaxProperties is the maximum number of properties
	MaxProperties *int64 `yaml:"maxProperties,omitempty" json:"maxProperties,omitempty"`

	// MinProperties is the minimum number of properties
	MinProperties *int64 `yaml:"minProperties,omitempty" json:"minProperties,omitempty"`

	// Required lists required properties
	Required []string `yaml:"required,omitempty" json:"required,omitempty"`

	// Items is the schema for array items
	Items *JSONSchemaPropsOrArray `yaml:"items,omitempty" json:"items,omitempty"`

	// AllOf requires all schemas to match
	AllOf []JSONSchemaProps `yaml:"allOf,omitempty" json:"allOf,omitempty"`

	// OneOf requires exactly one schema to match
	OneOf []JSONSchemaProps `yaml:"oneOf,omitempty" json:"oneOf,omitempty"`

	// AnyOf requires at least one schema to match
	AnyOf []JSONSchemaProps `yaml:"anyOf,omitempty" json:"anyOf,omitempty"`

	// Not negates a schema
	Not *JSONSchemaProps `yaml:"not,omitempty" json:"not,omitempty"`

	// Properties defines object properties
	Properties map[string]JSONSchemaProps `yaml:"properties,omitempty" json:"properties,omitempty"`

	// AdditionalProperties controls additional properties
	AdditionalProperties *JSONSchemaPropsOrBool `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`

	// PatternProperties defines properties by pattern
	PatternProperties map[string]JSONSchemaProps `yaml:"patternProperties,omitempty" json:"patternProperties,omitempty"`

	// Dependencies defines property dependencies
	Dependencies map[string]JSONSchemaPropsOrStringArray `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// Enum lists allowed values
	Enum []interface{} `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Nullable allows null values
	Nullable bool `yaml:"nullable,omitempty" json:"nullable,omitempty"`

	// XPreserveUnknownFields preserves unknown fields
	XPreserveUnknownFields *bool `yaml:"x-kubernetes-preserve-unknown-fields,omitempty" json:"x-kubernetes-preserve-unknown-fields,omitempty"`

	// XEmbeddedResource indicates this is an embedded resource
	XEmbeddedResource bool `yaml:"x-kubernetes-embedded-resource,omitempty" json:"x-kubernetes-embedded-resource,omitempty"`

	// XIntOrString allows int or string
	XIntOrString bool `yaml:"x-kubernetes-int-or-string,omitempty" json:"x-kubernetes-int-or-string,omitempty"`

	// Example is an example value
	Example interface{} `yaml:"example,omitempty" json:"example,omitempty"`
}

// JSONSchemaPropsOrArray is either a schema or an array of schemas
type JSONSchemaPropsOrArray struct {
	// Schema is a single schema
	Schema *JSONSchemaProps

	// JSONSchemas is an array of schemas
	JSONSchemas []JSONSchemaProps
}

// JSONSchemaPropsOrBool is either a schema or a boolean
type JSONSchemaPropsOrBool struct {
	// Allows indicates if additional properties are allowed
	Allows bool

	// Schema is the schema for additional properties
	Schema *JSONSchemaProps
}

// JSONSchemaPropsOrStringArray is either a schema or a string array
type JSONSchemaPropsOrStringArray struct {
	// Schema is the schema
	Schema *JSONSchemaProps

	// Property lists required properties
	Property []string
}

// SchemaBuilder helps build JSON schemas
type SchemaBuilder struct {
	schema JSONSchemaProps
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		schema: JSONSchemaProps{
			Type:       "object",
			Properties: make(map[string]JSONSchemaProps),
		},
	}
}

// Type sets the schema type
func (b *SchemaBuilder) Type(t string) *SchemaBuilder {
	b.schema.Type = t
	return b
}

// Description sets the schema description
func (b *SchemaBuilder) Description(desc string) *SchemaBuilder {
	b.schema.Description = desc
	return b
}

// Required sets required properties
func (b *SchemaBuilder) Required(props ...string) *SchemaBuilder {
	b.schema.Required = append(b.schema.Required, props...)
	return b
}

// Property adds a property
func (b *SchemaBuilder) Property(name string, prop JSONSchemaProps) *SchemaBuilder {
	b.schema.Properties[name] = prop
	return b
}

// StringProperty adds a string property
func (b *SchemaBuilder) StringProperty(name, description string) *SchemaBuilder {
	b.schema.Properties[name] = JSONSchemaProps{
		Type:        "string",
		Description: description,
	}
	return b
}

// IntegerProperty adds an integer property
func (b *SchemaBuilder) IntegerProperty(name, description string) *SchemaBuilder {
	b.schema.Properties[name] = JSONSchemaProps{
		Type:        "integer",
		Description: description,
	}
	return b
}

// BooleanProperty adds a boolean property
func (b *SchemaBuilder) BooleanProperty(name, description string) *SchemaBuilder {
	b.schema.Properties[name] = JSONSchemaProps{
		Type:        "boolean",
		Description: description,
	}
	return b
}

// ArrayProperty adds an array property
func (b *SchemaBuilder) ArrayProperty(name, description string, items *JSONSchemaProps) *SchemaBuilder {
	b.schema.Properties[name] = JSONSchemaProps{
		Type:        "array",
		Description: description,
		Items:       &JSONSchemaPropsOrArray{Schema: items},
	}
	return b
}

// ObjectProperty adds an object property
func (b *SchemaBuilder) ObjectProperty(name, description string, props map[string]JSONSchemaProps) *SchemaBuilder {
	b.schema.Properties[name] = JSONSchemaProps{
		Type:        "object",
		Description: description,
		Properties:  props,
	}
	return b
}

// EnumProperty adds an enum property
func (b *SchemaBuilder) EnumProperty(name, description string, values ...interface{}) *SchemaBuilder {
	b.schema.Properties[name] = JSONSchemaProps{
		Type:        "string",
		Description: description,
		Enum:        values,
	}
	return b
}

// Build returns the constructed schema
func (b *SchemaBuilder) Build() *JSONSchemaProps {
	return &b.schema
}
