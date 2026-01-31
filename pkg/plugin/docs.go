package plugin

import (
	"fmt"
	"reflect"
	"strings"
)

// GenerateDocs generates markdown documentation from a plugin's config struct tags
func GenerateDocs(plugin Plugin) string {
	configType := plugin.ConfigType()
	if configType == nil {
		return fmt.Sprintf("# %s Plugin\n\nNo configuration available.\n", plugin.Name())
	}

	val := reflect.TypeOf(configType)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("# %s Plugin\n\n", strings.Title(plugin.Name())))
	doc.WriteString(fmt.Sprintf("**Type**: %s\n", plugin.Type()))
	doc.WriteString(fmt.Sprintf("**Version**: %s\n\n", plugin.Version()))
	doc.WriteString("## Configuration\n\n")

	generateFieldDocs(&doc, val, "", 0)

	return doc.String()
}

// generateFieldDocs recursively generates documentation for struct fields
func generateFieldDocs(doc *strings.Builder, t reflect.Type, prefix string, depth int) {
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag (e.g., "name,omitempty")
		jsonParts := strings.Split(jsonTag, ",")
		fieldName := jsonParts[0]
		omitempty := len(jsonParts) > 1 && contains(jsonParts[1:], "omitempty")

		validateTag := field.Tag.Get("validate")
		description := field.Tag.Get("description")
		defaultVal := field.Tag.Get("default")

		// Determine if required
		required := strings.Contains(validateTag, "required") && !omitempty
		reqStr := ""
		if required {
			reqStr = " **(required)**"
		}

		// Build field path
		fullPath := fieldName
		if prefix != "" {
			fullPath = prefix + "." + fieldName
		}

		// Determine type string
		typeStr := getTypeString(field.Type, validateTag)

		// Write field documentation
		indent := strings.Repeat("  ", depth)
		doc.WriteString(fmt.Sprintf("%s- `%s` (%s)%s",
			indent, fullPath, typeStr, reqStr))

		if description != "" {
			doc.WriteString(fmt.Sprintf(": %s", description))
		}
		doc.WriteString("\n")

		// Add default value
		if defaultVal != "" {
			doc.WriteString(fmt.Sprintf("%s  - Default: `%s`\n", indent, defaultVal))
		}

		// Add validation constraints
		if constraints := extractValidationConstraints(validateTag); constraints != "" {
			doc.WriteString(fmt.Sprintf("%s  - Constraints: %s\n", indent, constraints))
		}

		// Recursively handle nested structs
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			doc.WriteString("\n")
			generateFieldDocs(doc, fieldType, fullPath, depth+1)
		} else if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Array {
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Struct {
				doc.WriteString(fmt.Sprintf("%s  - Array items:\n", indent))
				generateFieldDocs(doc, elemType, fullPath+"[]", depth+2)
			}
		}
	}
}

// getTypeString returns a human-readable type string for a field
func getTypeString(t reflect.Type, validateTag string) string {
	kind := t.Kind()

	// Handle pointers
	if kind == reflect.Ptr {
		return getTypeString(t.Elem(), validateTag) + " (optional)"
	}

	// Handle basic types
	switch kind {
	case reflect.String:
		// Check for enum constraint
		if oneof := extractTag(validateTag, "oneof"); oneof != "" {
			values := strings.Fields(oneof)
			return fmt.Sprintf("enum: %s", strings.Join(values, " | "))
		}
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "unsigned integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		elemType := t.Elem()
		return fmt.Sprintf("array of %s", getTypeString(elemType, ""))
	case reflect.Map:
		keyType := getTypeString(t.Key(), "")
		valType := getTypeString(t.Elem(), "")
		return fmt.Sprintf("map[%s]%s", keyType, valType)
	case reflect.Struct:
		return "object"
	default:
		return kind.String()
	}
}

// extractValidationConstraints extracts human-readable constraints from validate tag
func extractValidationConstraints(validateTag string) string {
	if validateTag == "" {
		return ""
	}

	var constraints []string

	// Split by comma to get individual validations
	parts := strings.Split(validateTag, ",")
	for _, part := range parts {
		// Skip 'required' and 'omitempty' as they're shown separately
		if part == "required" || part == "omitempty" {
			continue
		}

		// Handle constraints with parameters
		if strings.Contains(part, "=") {
			tagParts := strings.SplitN(part, "=", 2)
			tag := tagParts[0]
			param := tagParts[1]

			switch tag {
			case "eq":
				constraints = append(constraints, fmt.Sprintf("must equal '%s'", param))
			case "ne":
				constraints = append(constraints, fmt.Sprintf("must not equal '%s'", param))
			case "min":
				constraints = append(constraints, fmt.Sprintf("minimum: %s", param))
			case "max":
				constraints = append(constraints, fmt.Sprintf("maximum: %s", param))
			case "oneof":
				values := strings.Fields(param)
				constraints = append(constraints, fmt.Sprintf("one of: %s", strings.Join(values, ", ")))
			default:
				constraints = append(constraints, fmt.Sprintf("%s=%s", tag, param))
			}
		} else {
			// Simple constraints without parameters
			switch part {
			case "url":
				constraints = append(constraints, "must be a valid URL")
			case "email":
				constraints = append(constraints, "must be a valid email")
			case "uuid":
				constraints = append(constraints, "must be a valid UUID")
			default:
				constraints = append(constraints, part)
			}
		}
	}

	return strings.Join(constraints, ", ")
}

// extractTag extracts a specific tag value from a validation tag string
func extractTag(validateTag, tagName string) string {
	parts := strings.Split(validateTag, ",")
	for _, part := range parts {
		if strings.HasPrefix(part, tagName+"=") {
			return strings.TrimPrefix(part, tagName+"=")
		}
	}
	return ""
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
