package plugin

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
)

var validate = validator.New()

// ValidateAndBind validates and binds a spec map to the plugin's config struct
// It performs the following steps:
// 1. Converts the map to the plugin's config struct type
// 2. Applies default values from struct tags
// 3. Validates the struct using validation tags
func ValidateAndBind(plugin Plugin, spec map[string]interface{}) (interface{}, error) {
	// Get the config struct type from the plugin
	configType := plugin.ConfigType()
	if configType == nil {
		return nil, fmt.Errorf("plugin %s does not provide a config type", plugin.Name())
	}

	// 1. Convert map to struct automatically
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           configType,
		WeaklyTypedInput: true,
		TagName:          "json",
		ErrorUnused:      false, // Allow extra fields in YAML
		ZeroFields:       true,  // Clear existing values
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(spec); err != nil {
		return nil, fmt.Errorf("failed to bind config: %w", err)
	}

	// 2. Apply defaults (from `default` tags)
	if err := applyDefaults(configType); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	// 3. Validate using `validate` tags
	if err := validate.Struct(configType); err != nil {
		return nil, formatValidationError(err)
	}

	return configType, nil
}

// applyDefaults applies default values from struct tags to zero-valued fields
func applyDefaults(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Check for default tag
		if defaultVal, ok := fieldType.Tag.Lookup("default"); ok {
			// If field is zero value, set the default
			if isZeroValue(field) {
				if err := setDefaultValue(field, defaultVal); err != nil {
					return fmt.Errorf("field %s: %w", fieldType.Name, err)
				}
			}
		}

		// Recursively handle nested structs and pointers to structs
		if field.Kind() == reflect.Struct {
			if err := applyDefaults(field.Addr().Interface()); err != nil {
				return err
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			if err := applyDefaults(field.Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}

// isZeroValue checks if a reflect.Value is the zero value for its type
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Struct:
		return v.IsZero()
	default:
		return false
	}
}

// setDefaultValue sets a default value on a field based on its type
func setDefaultValue(field reflect.Value, defaultVal string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultVal)
	case reflect.Bool:
		b, err := strconv.ParseBool(defaultVal)
		if err != nil {
			return fmt.Errorf("invalid bool default: %w", err)
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(defaultVal, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int default: %w", err)
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(defaultVal, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint default: %w", err)
		}
		field.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(defaultVal, 64)
		if err != nil {
			return fmt.Errorf("invalid float default: %w", err)
		}
		field.SetFloat(f)
	default:
		return fmt.Errorf("unsupported type for default value: %s", field.Kind())
	}
	return nil
}

// formatValidationError formats validator errors into a user-friendly message
func formatValidationError(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	var messages []string
	for _, e := range validationErrs {
		msg := formatFieldError(e)
		messages = append(messages, msg)
	}

	return fmt.Errorf("validation failed:\n  - %s",
		strings.Join(messages, "\n  - "))
}

// formatFieldError formats a single validation error
func formatFieldError(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("field '%s' is required", field)
	case "eq":
		return fmt.Sprintf("field '%s' must equal '%s'", field, e.Param())
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", field, e.Param())
	case "min":
		return fmt.Sprintf("field '%s' must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("field '%s' must be at most %s", field, e.Param())
	case "url":
		return fmt.Sprintf("field '%s' must be a valid URL", field)
	case "email":
		return fmt.Sprintf("field '%s' must be a valid email", field)
	default:
		return fmt.Sprintf("field '%s' failed validation: %s", field, tag)
	}
}
