package project

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ProjectID is a value object representing a project's unique identifier
type ProjectID struct {
	value uuid.UUID
}

// NewProjectID creates a new ProjectID
func NewProjectID() ProjectID {
	return ProjectID{value: uuid.New()}
}

// ParseProjectID parses a string into a ProjectID
func ParseProjectID(id string) (ProjectID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ProjectID{}, fmt.Errorf("invalid project ID format: %w", err)
	}
	return ProjectID{value: uid}, nil
}

func (id ProjectID) String() string {
	return id.value.String()
}

func (id ProjectID) UUID() uuid.UUID {
	return id.value
}

func (id ProjectID) Equals(other ProjectID) bool {
	return id.value == other.value
}

// RepositoryURL is a value object representing a repository URL
type RepositoryURL struct {
	value string
}

// NewRepositoryURL creates a new RepositoryURL with validation
func NewRepositoryURL(url string) (RepositoryURL, error) {
	url = strings.TrimSpace(url)

	if url == "" {
		return RepositoryURL{}, fmt.Errorf("repository URL cannot be empty")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return RepositoryURL{}, fmt.Errorf("repository URL must be a valid HTTP(S) URL")
	}

	return RepositoryURL{value: url}, nil
}

func (u RepositoryURL) String() string {
	return u.value
}

func (u RepositoryURL) Equals(other RepositoryURL) bool {
	return u.value == other.value
}

// Language represents the programming language/framework of a project
type Language string

const (
	LanguageNode   Language = "NODE"
	LanguageNodeTS Language = "NODE_TS"
	LanguageNextJS Language = "NEXTJS"
	LanguageGo     Language = "GO"
	LanguagePython Language = "PYTHON"
)

// NewLanguage creates a new Language with validation
func NewLanguage(lang string) (Language, error) {
	lang = strings.ToUpper(strings.TrimSpace(lang))

	switch Language(lang) {
	case LanguageNode, LanguageNodeTS, LanguageNextJS, LanguageGo, LanguagePython:
		return Language(lang), nil
	default:
		return "", fmt.Errorf("invalid language: %s (must be one of: NODE, NODE_TS, NEXTJS, GO, PYTHON)", lang)
	}
}

func (l Language) String() string {
	return string(l)
}

func (l Language) IsValid() bool {
	switch l {
	case LanguageNode, LanguageNodeTS, LanguageNextJS, LanguageGo, LanguagePython:
		return true
	default:
		return false
	}
}

// Command is a value object representing a build or run command
type Command struct {
	value string
}

// NewCommand creates a new Command with validation
func NewCommand(cmd string) (Command, error) {
	cmd = strings.TrimSpace(cmd)

	if cmd == "" {
		return Command{}, fmt.Errorf("command cannot be empty")
	}

	if len(cmd) > 500 {
		return Command{}, fmt.Errorf("command too long (max 500 characters)")
	}

	return Command{value: cmd}, nil
}

func (c Command) String() string {
	return c.value
}

func (c Command) Equals(other Command) bool {
	return c.value == other.value
}
