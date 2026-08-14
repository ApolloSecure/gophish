package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

const (
	maxCustomFields      = 50
	maxCustomFieldKeyLen = 64
	maxCustomFieldValLen = 4096
	maxCustomFieldsSize  = 64 * 1024
)

var customFieldKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// CustomFields contains group-scoped values that can be referenced from
// phishing templates using {{.Custom.FieldName}}.
type CustomFields map[string]string

func cloneCustomFields(fields CustomFields) CustomFields {
	cloned := make(CustomFields, len(fields))
	for key, value := range fields {
		cloned[key] = value
	}
	return cloned
}

// Validate ensures custom fields are safe to persist and reference from a
// template.
func (f CustomFields) Validate() error {
	if len(f) > maxCustomFields {
		return fmt.Errorf("custom_fields cannot contain more than %d fields", maxCustomFields)
	}
	for key, value := range f {
		if len(key) > maxCustomFieldKeyLen || !customFieldKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid custom field key %q", key)
		}
		if len(value) > maxCustomFieldValLen {
			return fmt.Errorf("custom field %q cannot exceed %d characters", key, maxCustomFieldValLen)
		}
	}
	encoded, err := json.Marshal(map[string]string(f))
	if err != nil {
		return err
	}
	if len(encoded) > maxCustomFieldsSize {
		return fmt.Errorf("custom_fields cannot exceed %d bytes", maxCustomFieldsSize)
	}
	return nil
}

// MarshalJSON represents an unset field collection as an empty object instead
// of null in API responses.
func (f CustomFields) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]string(f))
}

// UnmarshalJSON only accepts an object containing string values. A missing
// custom_fields property remains nil so PUT requests can preserve existing
// membership values, while an explicit empty object clears them.
func (f *CustomFields) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return errors.New("custom_fields must be a JSON object")
	}
	fields := map[string]string{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return errors.New("custom_fields must be an object containing string values")
	}
	customFields := CustomFields(fields)
	if err := customFields.Validate(); err != nil {
		return err
	}
	*f = customFields
	return nil
}

// Value serializes custom fields for storage by database/sql.
func (f CustomFields) Value() (driver.Value, error) {
	if f == nil {
		f = CustomFields{}
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(map[string]string(f))
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

// Scan deserializes custom fields loaded by database/sql.
func (f *CustomFields) Scan(value interface{}) error {
	if value == nil {
		*f = CustomFields{}
		return nil
	}
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("unsupported custom_fields database value %T", value)
	}
	if len(data) == 0 {
		*f = CustomFields{}
		return nil
	}
	return f.UnmarshalJSON(data)
}
