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

// NewOptionalCommand creates a new Command that allows empty strings (for optional commands like build)
func NewOptionalCommand(cmd string) Command {
	cmd = strings.TrimSpace(cmd)

	if len(cmd) > 500 {
		// Truncate if too long (shouldn't happen, but handle gracefully)
		cmd = cmd[:500]
	}

	return Command{value: cmd}
}

// IsEmpty checks if the command is empty
func (c Command) IsEmpty() bool {
	return c.value == ""
}

func (c Command) String() string {
	return c.value
}

func (c Command) Equals(other Command) bool {
	return c.value == other.value
}

// CustomDomain is a value object representing a custom subdomain prefix
// The full domain will be: <custom-domain>.<base-domain>
// e.g., "my-app" becomes "my-app.snapdeploy.app"
type CustomDomain struct {
	value string
}

// NewCustomDomain creates a new CustomDomain with validation
func NewCustomDomain(domain string) (CustomDomain, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// If empty, generate a random UUID-based subdomain
	if domain == "" {
		domain = generateRandomSubdomain()
		return CustomDomain{value: domain}, nil
	}

	// Validate subdomain format (RFC 1123)
	// Must be lowercase alphanumeric with hyphens, start/end with alphanumeric
	if len(domain) < 1 || len(domain) > 63 {
		return CustomDomain{}, fmt.Errorf("custom domain must be between 1 and 63 characters")
	}

	// Check first and last character
	if !isAlphanumeric(rune(domain[0])) || !isAlphanumeric(rune(domain[len(domain)-1])) {
		return CustomDomain{}, fmt.Errorf("custom domain must start and end with alphanumeric characters")
	}

	// Check all characters are valid
	for _, c := range domain {
		if !isAlphanumeric(c) && c != '-' {
			return CustomDomain{}, fmt.Errorf("custom domain can only contain lowercase letters, numbers, and hyphens")
		}
	}

	// Reserved subdomains
	reserved := []string{"www", "api", "admin", "app", "dashboard", "console", "staging", "prod", "production", "dev", "development", "test", "testing"}
	for _, r := range reserved {
		if domain == r {
			return CustomDomain{}, fmt.Errorf("subdomain '%s' is reserved", domain)
		}
	}

	return CustomDomain{value: domain}, nil
}

// NewCustomDomainFromExisting creates a CustomDomain from an existing value (skips generation)
func NewCustomDomainFromExisting(domain string) (CustomDomain, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	if domain == "" {
		return CustomDomain{}, fmt.Errorf("custom domain cannot be empty when reconstituting")
	}
	
	return CustomDomain{value: domain}, nil
}

// generateRandomSubdomain generates a random subdomain using a short UUID
func generateRandomSubdomain() string {
	// Use first 8 characters of UUID (short, unique enough)
	id := uuid.New().String()
	return strings.Split(id, "-")[0]
}

// isAlphanumeric checks if a rune is alphanumeric
func isAlphanumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func (d CustomDomain) String() string {
	return d.value
}

func (d CustomDomain) Equals(other CustomDomain) bool {
	return d.value == other.value
}

// IsEmpty checks if the custom domain is empty
func (d CustomDomain) IsEmpty() bool {
	return d.value == ""
}
