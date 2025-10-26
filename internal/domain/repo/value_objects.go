package repo

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// RepositoryID is a value object representing a repository's unique identifier
type RepositoryID struct {
	value uuid.UUID
}

// NewRepositoryID creates a new RepositoryID
func NewRepositoryID() RepositoryID {
	return RepositoryID{value: uuid.New()}
}

// ParseRepositoryID parses a string into a RepositoryID
func ParseRepositoryID(id string) (RepositoryID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return RepositoryID{}, fmt.Errorf("invalid repository ID format: %w", err)
	}
	return RepositoryID{value: uid}, nil
}

func (id RepositoryID) String() string {
	return id.value.String()
}

func (id RepositoryID) UUID() uuid.UUID {
	return id.value
}

func (id RepositoryID) Equals(other RepositoryID) bool {
	return id.value == other.value
}

// GitHubID is a value object representing a GitHub repository ID
type GitHubID struct {
	value int64
}

// NewGitHubID creates a new GitHubID with validation
func NewGitHubID(id int64) (GitHubID, error) {
	if id <= 0 {
		return GitHubID{}, fmt.Errorf("GitHub ID must be positive")
	}
	return GitHubID{value: id}, nil
}

func (g GitHubID) Int64() int64 {
	return g.value
}

func (g GitHubID) Equals(other GitHubID) bool {
	return g.value == other.value
}

// Name is a value object representing a repository name
type Name struct {
	value string
}

// NewName creates a new Name with validation
func NewName(name string) (Name, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return Name{}, fmt.Errorf("repository name cannot be empty")
	}

	if len(name) > 100 {
		return Name{}, fmt.Errorf("repository name too long (max 100 characters)")
	}

	return Name{value: name}, nil
}

func (n Name) String() string {
	return n.value
}

func (n Name) Equals(other Name) bool {
	return n.value == other.value
}

// URL is a value object representing a repository URL
type URL struct {
	value string
}

// NewURL creates a new URL with validation
func NewURL(url string) (URL, error) {
	url = strings.TrimSpace(url)

	if url == "" {
		return URL{}, fmt.Errorf("repository URL cannot be empty")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return URL{}, fmt.Errorf("repository URL must be a valid HTTP(S) URL")
	}

	return URL{value: url}, nil
}

func (u URL) String() string {
	return u.value
}

func (u URL) Equals(other URL) bool {
	return u.value == other.value
}
