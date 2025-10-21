package models

import (
	"fmt"
)

// ValidateDocument validates a document's data against a schema
func ValidateDocument(data map[string]interface{}, schema *Schema) error {
	// Check that all fields in data match the schema
	for fieldName, value := range data {
		fieldType, exists := schema.Fields[fieldName]
		if !exists {
			return fmt.Errorf("field '%s' is not defined in schema", fieldName)
		}

		if err := validateFieldValue(fieldName, value, fieldType); err != nil {
			return err
		}
	}

	// All fields must be present (no optional fields for now)
	for fieldName := range schema.Fields {
		if _, exists := data[fieldName]; !exists {
			return fmt.Errorf("required field '%s' is missing", fieldName)
		}
	}

	return nil
}

// validateFieldValue validates a single field value against its type
func validateFieldValue(fieldName string, value interface{}, expectedType FieldType) error {
	switch expectedType {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", fieldName, value)
		}
	case FieldTypeNumber:
		// JSON numbers can be float64 or int
		switch value.(type) {
		case float64, int, int64, float32:
			return nil
		default:
			return fmt.Errorf("field '%s' must be a number, got %T", fieldName, value)
		}
	case FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %T", fieldName, value)
		}
	default:
		return fmt.Errorf("unknown field type: %s", expectedType)
	}

	return nil
}
