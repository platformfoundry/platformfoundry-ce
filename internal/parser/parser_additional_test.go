package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser_ParseMultiplePlatforms(t *testing.T) {
	yaml := `
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: platform-1
spec:
  cloud: aws
---
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: platform-2
spec:
  cloud: gcp
`

	parser := New()
	resources, err := parser.Parse([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, resources, 2)
	assert.Equal(t, "platform-1", resources[0].Metadata.Name)
	assert.Equal(t, "platform-2", resources[1].Metadata.Name)
}

func TestParser_ParseWithComments(t *testing.T) {
	yaml := `
# This is a comment
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: test-platform  # inline comment
spec:
  cloud: aws
  # Another comment
  region: us-east-1
`

	parser := New()
	resources, err := parser.Parse([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "test-platform", resources[0].Metadata.Name)
}

func TestParser_ParseInvalidYAML(t *testing.T) {
	yaml := `
this is not valid yaml {{{
`

	parser := New()
	_, err := parser.Parse([]byte(yaml))
	assert.Error(t, err)
}

func TestParser_ParseMissingFields(t *testing.T) {
	yaml := `
apiVersion: platformfoundry.io/v1
kind: Platform
metadata:
  name: test-platform
# Missing spec
`

	parser := New()
	_, err := parser.Parse([]byte(yaml))
	// Should error because spec is required
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing spec")
}

