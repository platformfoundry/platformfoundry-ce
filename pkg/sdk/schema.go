// Package sdk provides the Plugin SDK for Platform Foundry.
package sdk

import (
	"github.com/platformfoundry/pf-ce/pkg/contracts/ppi"
)

// SchemaBuilder helps construct schemas with a fluent API
type SchemaBuilder struct {
	schema *ppi.Schema
}

// NewSchemaBuilder creates a new schema builder
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		schema: &ppi.Schema{
			Attributes: make(map[string]*ppi.Attribute),
			Blocks:     make(map[string]*ppi.Block),
		},
	}
}

// Version sets the schema version
func (b *SchemaBuilder) Version(version int64) *SchemaBuilder {
	b.schema.Version = version
	return b
}

// Description sets the schema description
func (b *SchemaBuilder) Description(desc string) *SchemaBuilder {
	b.schema.Description = desc
	return b
}

// Attribute adds an attribute to the schema
func (b *SchemaBuilder) Attribute(name string, attr *ppi.Attribute) *SchemaBuilder {
	b.schema.Attributes[name] = attr
	return b
}

// Block adds a block to the schema
func (b *SchemaBuilder) Block(name string, block *ppi.Block) *SchemaBuilder {
	b.schema.Blocks[name] = block
	return b
}

// Build returns the constructed schema
func (b *SchemaBuilder) Build() *ppi.Schema {
	return b.schema
}

// AttributeBuilder helps construct attributes with a fluent API
type AttributeBuilder struct {
	attr *ppi.Attribute
}

// NewAttributeBuilder creates a new attribute builder
func NewAttributeBuilder(attrType ppi.AttributeType) *AttributeBuilder {
	return &AttributeBuilder{
		attr: &ppi.Attribute{
			Type: attrType,
		},
	}
}

// String creates a string attribute builder
func String() *AttributeBuilder {
	return NewAttributeBuilder(ppi.TypeString)
}

// Number creates a number attribute builder
func Number() *AttributeBuilder {
	return NewAttributeBuilder(ppi.TypeNumber)
}

// Bool creates a bool attribute builder
func Bool() *AttributeBuilder {
	return NewAttributeBuilder(ppi.TypeBool)
}

// List creates a list attribute builder
func List() *AttributeBuilder {
	return NewAttributeBuilder(ppi.TypeList)
}

// Set creates a set attribute builder
func Set() *AttributeBuilder {
	return NewAttributeBuilder(ppi.TypeSet)
}

// Map creates a map attribute builder
func Map() *AttributeBuilder {
	return NewAttributeBuilder(ppi.TypeMap)
}

// Required marks the attribute as required
func (b *AttributeBuilder) Required() *AttributeBuilder {
	b.attr.Required = true
	b.attr.Optional = false
	return b
}

// Optional marks the attribute as optional
func (b *AttributeBuilder) Optional() *AttributeBuilder {
	b.attr.Optional = true
	b.attr.Required = false
	return b
}

// Computed marks the attribute as computed
func (b *AttributeBuilder) Computed() *AttributeBuilder {
	b.attr.Computed = true
	return b
}

// Sensitive marks the attribute as sensitive
func (b *AttributeBuilder) Sensitive() *AttributeBuilder {
	b.attr.Sensitive = true
	return b
}

// Deprecated marks the attribute as deprecated
func (b *AttributeBuilder) Deprecated(message string) *AttributeBuilder {
	b.attr.Deprecated = message
	return b
}

// Description sets the attribute description
func (b *AttributeBuilder) Description(desc string) *AttributeBuilder {
	b.attr.Description = desc
	return b
}

// Default sets the attribute default value
func (b *AttributeBuilder) Default(value interface{}) *AttributeBuilder {
	b.attr.Default = value
	return b
}

// Validator adds a validator to the attribute
func (b *AttributeBuilder) Validator(v ppi.AttributeValidator) *AttributeBuilder {
	b.attr.Validators = append(b.attr.Validators, v)
	return b
}

// Build returns the constructed attribute
func (b *AttributeBuilder) Build() *ppi.Attribute {
	return b.attr
}

// BlockBuilder helps construct blocks with a fluent API
type BlockBuilder struct {
	block *ppi.Block
}

// NewBlockBuilder creates a new block builder
func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{
		block: &ppi.Block{
			Attributes: make(map[string]*ppi.Attribute),
			Blocks:     make(map[string]*ppi.Block),
		},
	}
}

// Description sets the block description
func (b *BlockBuilder) Description(desc string) *BlockBuilder {
	b.block.Description = desc
	return b
}

// MinItems sets the minimum number of blocks
func (b *BlockBuilder) MinItems(min int) *BlockBuilder {
	b.block.MinItems = min
	return b
}

// MaxItems sets the maximum number of blocks
func (b *BlockBuilder) MaxItems(max int) *BlockBuilder {
	b.block.MaxItems = max
	return b
}

// Deprecated marks the block as deprecated
func (b *BlockBuilder) Deprecated(message string) *BlockBuilder {
	b.block.Deprecated = message
	return b
}

// Attribute adds an attribute to the block
func (b *BlockBuilder) Attribute(name string, attr *ppi.Attribute) *BlockBuilder {
	b.block.Attributes[name] = attr
	return b
}

// Block adds a nested block
func (b *BlockBuilder) Block(name string, block *ppi.Block) *BlockBuilder {
	b.block.Blocks[name] = block
	return b
}

// Build returns the constructed block
func (b *BlockBuilder) Build() *ppi.Block {
	return b.block
}

// Validators provides common validators

// StringLengthBetween creates a string length validator
func StringLengthBetween(min, max int) ppi.AttributeValidator {
	return ppi.StringLengthValidator{Min: min, Max: max}
}

// NumberBetween creates a number range validator
func NumberBetween(min, max float64) ppi.AttributeValidator {
	return ppi.NumberRangeValidator{Min: &min, Max: &max}
}

// NumberAtLeast creates a minimum number validator
func NumberAtLeast(min float64) ppi.AttributeValidator {
	return ppi.NumberRangeValidator{Min: &min}
}

// NumberAtMost creates a maximum number validator
func NumberAtMost(max float64) ppi.AttributeValidator {
	return ppi.NumberRangeValidator{Max: &max}
}

// OneOf creates an enum validator
func OneOf(values ...interface{}) ppi.AttributeValidator {
	return ppi.EnumValidator{Values: values}
}
