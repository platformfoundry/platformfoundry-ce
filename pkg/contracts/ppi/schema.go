// Package ppi defines the Platform Provider Interface (PPI).
package ppi

// Schema defines the structure and validation rules for a resource or data source
type Schema struct {
	// Version is the schema version for upgrades
	Version int64

	// Attributes defines the attributes for this schema
	Attributes map[string]*Attribute

	// Blocks defines nested block structures
	Blocks map[string]*Block

	// Description provides documentation for this schema
	Description string
}

// Attribute defines a single attribute in a schema
type Attribute struct {
	// Type is the data type of this attribute
	Type AttributeType

	// Description provides documentation
	Description string

	// Required indicates this attribute must be provided
	Required bool

	// Optional indicates this attribute may be omitted
	Optional bool

	// Computed indicates this attribute is computed by the provider
	Computed bool

	// Sensitive indicates this attribute contains sensitive data
	Sensitive bool

	// Deprecated marks this attribute as deprecated
	Deprecated string

	// Default is the default value if not provided
	Default interface{}

	// Validators are validation functions for this attribute
	Validators []AttributeValidator
}

// AttributeType represents the type of an attribute
type AttributeType string

const (
	TypeString AttributeType = "string"
	TypeNumber AttributeType = "number"
	TypeBool   AttributeType = "bool"
	TypeList   AttributeType = "list"
	TypeSet    AttributeType = "set"
	TypeMap    AttributeType = "map"
	TypeObject AttributeType = "object"
	TypeAny    AttributeType = "any"
)

// Block defines a nested block structure in a schema
type Block struct {
	// Attributes defines the attributes within this block
	Attributes map[string]*Attribute

	// Blocks defines nested blocks within this block
	Blocks map[string]*Block

	// Description provides documentation
	Description string

	// MinItems is the minimum number of this block type
	MinItems int

	// MaxItems is the maximum number of this block type
	MaxItems int

	// Deprecated marks this block as deprecated
	Deprecated string
}

// AttributeValidator validates an attribute value
type AttributeValidator interface {
	// Validate checks if the value is valid
	Validate(value interface{}) *Diagnostics

	// Description returns a human-readable description
	Description() string
}

// StringLengthValidator validates string length
type StringLengthValidator struct {
	Min int
	Max int
}

func (v StringLengthValidator) Validate(value interface{}) *Diagnostics {
	diags := &Diagnostics{}
	if s, ok := value.(string); ok {
		if v.Min > 0 && len(s) < v.Min {
			diags.AddError("String too short", "Value must be at least %d characters", v.Min)
		}
		if v.Max > 0 && len(s) > v.Max {
			diags.AddError("String too long", "Value must be at most %d characters", v.Max)
		}
	}
	return diags
}

func (v StringLengthValidator) Description() string {
	return "validates string length"
}

// NumberRangeValidator validates numeric ranges
type NumberRangeValidator struct {
	Min *float64
	Max *float64
}

func (v NumberRangeValidator) Validate(value interface{}) *Diagnostics {
	diags := &Diagnostics{}
	var num float64
	switch n := value.(type) {
	case int:
		num = float64(n)
	case int64:
		num = float64(n)
	case float64:
		num = n
	default:
		return diags
	}
	if v.Min != nil && num < *v.Min {
		diags.AddError("Number too small", "Value must be at least %f", *v.Min)
	}
	if v.Max != nil && num > *v.Max {
		diags.AddError("Number too large", "Value must be at most %f", *v.Max)
	}
	return diags
}

func (v NumberRangeValidator) Description() string {
	return "validates numeric range"
}

// EnumValidator validates that a value is one of a set of allowed values
type EnumValidator struct {
	Values []interface{}
}

func (v EnumValidator) Validate(value interface{}) *Diagnostics {
	diags := &Diagnostics{}
	for _, allowed := range v.Values {
		if value == allowed {
			return diags
		}
	}
	diags.AddError("Invalid value", "Value must be one of the allowed values")
	return diags
}

func (v EnumValidator) Description() string {
	return "validates value is in allowed set"
}
