package project

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EnvironmentVariable represents a project environment variable
type EnvironmentVariable struct {
	id        EnvVarID
	projectID ProjectID
	key       EnvVarKey
	value     EnvVarValue
	createdAt time.Time
	updatedAt time.Time
}

// NewEnvironmentVariable creates a new environment variable
func NewEnvironmentVariable(
	projectID ProjectID,
	key, value string,
) (*EnvironmentVariable, error) {
	envKey, err := NewEnvVarKey(key)
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	envValue, err := NewEnvVarValue(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	now := time.Now()
	return &EnvironmentVariable{
		id:        NewEnvVarID(),
		projectID: projectID,
		key:       envKey,
		value:     envValue,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ReconstituteEnvVar recreates an environment variable from persistence
func ReconstituteEnvVar(
	id string,
	projectID ProjectID,
	key, encryptedValue string,
	createdAt, updatedAt time.Time,
) (*EnvironmentVariable, error) {
	envID, err := ParseEnvVarID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid env var ID: %w", err)
	}

	envKey, err := NewEnvVarKey(key)
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	envValue := NewEnvVarValueFromEncrypted(encryptedValue)

	return &EnvironmentVariable{
		id:        envID,
		projectID: projectID,
		key:       envKey,
		value:     envValue,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

// UpdateValue updates the environment variable value
func (e *EnvironmentVariable) UpdateValue(newValue string) error {
	envValue, err := NewEnvVarValue(newValue)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}

	e.value = envValue
	e.updatedAt = time.Now()
	return nil
}

// Getters

func (e *EnvironmentVariable) ID() EnvVarID {
	return e.id
}

func (e *EnvironmentVariable) ProjectID() ProjectID {
	return e.projectID
}

func (e *EnvironmentVariable) Key() EnvVarKey {
	return e.key
}

func (e *EnvironmentVariable) Value() EnvVarValue {
	return e.value
}

func (e *EnvironmentVariable) CreatedAt() time.Time {
	return e.createdAt
}

func (e *EnvironmentVariable) UpdatedAt() time.Time {
	return e.updatedAt
}

// EnvVarID is a value object for environment variable ID
type EnvVarID struct {
	value uuid.UUID
}

func NewEnvVarID() EnvVarID {
	return EnvVarID{value: uuid.New()}
}

func ParseEnvVarID(id string) (EnvVarID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return EnvVarID{}, fmt.Errorf("invalid env var ID format: %w", err)
	}
	return EnvVarID{value: uid}, nil
}

func (id EnvVarID) String() string {
	return id.value.String()
}

func (id EnvVarID) UUID() uuid.UUID {
	return id.value
}

// EnvVarKey is a value object for environment variable key
type EnvVarKey struct {
	value string
}

func NewEnvVarKey(key string) (EnvVarKey, error) {
	key = strings.TrimSpace(key)

	if key == "" {
		return EnvVarKey{}, fmt.Errorf("environment variable key cannot be empty")
	}

	// Validate key format (Unix env var rules)
	// Must start with letter or underscore, contain only alphanumeric and underscores
	if !isValidEnvVarKey(key) {
		return EnvVarKey{}, fmt.Errorf("invalid key format: must start with letter/underscore and contain only alphanumeric and underscores")
	}

	if len(key) > 255 {
		return EnvVarKey{}, fmt.Errorf("key too long (max 255 characters)")
	}

	return EnvVarKey{value: key}, nil
}

func (k EnvVarKey) String() string {
	return k.value
}

func (k EnvVarKey) Equals(other EnvVarKey) bool {
	return k.value == other.value
}

// isValidEnvVarKey validates environment variable key format
func isValidEnvVarKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	// First character must be letter or underscore
	first := rune(key[0])
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	// Rest can be alphanumeric or underscore
	for _, c := range key {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// EnvVarValue is a value object for environment variable value
// Stores encrypted value and provides masked representation
type EnvVarValue struct {
	encryptedValue string
}

func NewEnvVarValue(plaintext string) (EnvVarValue, error) {
	// Value will be encrypted by the application service before storage
	// This constructor is used when creating/updating
	return EnvVarValue{encryptedValue: plaintext}, nil
}

func NewEnvVarValueFromEncrypted(encrypted string) EnvVarValue {
	return EnvVarValue{encryptedValue: encrypted}
}

func (v EnvVarValue) EncryptedValue() string {
	return v.encryptedValue
}

// Masked returns a masked version of the value for display
// Format: first_char + ******* + last_char
// Example: "my_secret_value" -> "m*******e"
func (v EnvVarValue) Masked() string {
	if v.encryptedValue == "" {
		return ""
	}

	// For very short values (1-2 chars), mask completely
	if len(v.encryptedValue) <= 2 {
		return "***"
	}

	// For longer values: first char + ******* + last char
	first := string(v.encryptedValue[0])
	last := string(v.encryptedValue[len(v.encryptedValue)-1])
	
	return fmt.Sprintf("%s*******%s", first, last)
}

// IsEmpty checks if value is empty
func (v EnvVarValue) IsEmpty() bool {
	return v.encryptedValue == ""
}

