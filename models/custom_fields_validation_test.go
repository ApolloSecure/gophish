package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCustomFieldsMySQLTextByteLimit(t *testing.T) {
	atLimit := customFieldsWithEncodedSize(t, maxCustomFieldsSize)
	if err := atLimit.Validate(); err != nil {
		t.Fatalf("validate %d-byte custom fields: %v", maxCustomFieldsSize, err)
	}

	overLimit := customFieldsWithEncodedSize(t, maxCustomFieldsSize+1)
	if err := overLimit.Validate(); err == nil {
		t.Fatalf("expected %d-byte custom fields to be rejected", maxCustomFieldsSize+1)
	}
}

func customFieldsWithEncodedSize(t *testing.T, size int) CustomFields {
	t.Helper()
	fields := CustomFields{}
	for i := 0; i < 15; i++ {
		fields[string(rune('A'+i))] = strings.Repeat("x", maxCustomFieldValLen)
	}
	fields["Final"] = ""
	encoded, err := json.Marshal(map[string]string(fields))
	if err != nil {
		t.Fatalf("marshal custom fields: %v", err)
	}
	remaining := size - len(encoded)
	if remaining < 0 || remaining > maxCustomFieldValLen {
		t.Fatalf("cannot construct %d-byte custom fields; final value length would be %d", size, remaining)
	}
	fields["Final"] = strings.Repeat("x", remaining)
	encoded, err = json.Marshal(map[string]string(fields))
	if err != nil {
		t.Fatalf("marshal sized custom fields: %v", err)
	}
	if len(encoded) != size {
		t.Fatalf("custom fields encoded size = %d, want %d", len(encoded), size)
	}
	return fields
}
