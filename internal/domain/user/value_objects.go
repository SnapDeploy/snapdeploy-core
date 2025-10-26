package user

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// UserID is a value object representing a user's unique identifier
type UserID struct {
	value uuid.UUID
}

// NewUserID creates a new UserID
func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

// ParseUserID parses a string into a UserID
func ParseUserID(id string) (UserID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return UserID{}, fmt.Errorf("invalid user ID format: %w", err)
	}
	return UserID{value: uid}, nil
}

func (id UserID) String() string {
	return id.value.String()
}

func (id UserID) UUID() uuid.UUID {
	return id.value
}

func (id UserID) Equals(other UserID) bool {
	return id.value == other.value
}

// Email is a value object representing a valid email address
type Email struct {
	value string
}

// NewEmail creates a new Email with validation
func NewEmail(email string) (Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		return Email{}, fmt.Errorf("email cannot be empty")
	}

	if len(email) > 255 {
		return Email{}, fmt.Errorf("email too long (max 255 characters)")
	}

	if !emailRegex.MatchString(email) {
		return Email{}, fmt.Errorf("invalid email format")
	}

	return Email{value: email}, nil
}

func (e Email) String() string {
	return e.value
}

func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// Username is a value object representing a valid username
type Username struct {
	value string
}

// NewUsername creates a new Username with validation
func NewUsername(username string) (Username, error) {
	username = strings.TrimSpace(username)

	if username == "" {
		return Username{}, fmt.Errorf("username cannot be empty")
	}

	if len(username) < 3 {
		return Username{}, fmt.Errorf("username too short (min 3 characters)")
	}

	if len(username) > 50 {
		return Username{}, fmt.Errorf("username too long (max 50 characters)")
	}

	// Username should contain only alphanumeric characters, underscores, and hyphens
	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validUsername.MatchString(username) {
		return Username{}, fmt.Errorf("username can only contain letters, numbers, underscores, and hyphens")
	}

	return Username{value: username}, nil
}

func (u Username) String() string {
	return u.value
}

func (u Username) Equals(other Username) bool {
	return u.value == other.value
}

// ClerkUserID is a value object representing a Clerk user identifier
type ClerkUserID struct {
	value string
}

// NewClerkUserID creates a new ClerkUserID with validation
func NewClerkUserID(clerkID string) (ClerkUserID, error) {
	clerkID = strings.TrimSpace(clerkID)

	if clerkID == "" {
		return ClerkUserID{}, fmt.Errorf("clerk user ID cannot be empty")
	}

	if len(clerkID) > 255 {
		return ClerkUserID{}, fmt.Errorf("clerk user ID too long")
	}

	return ClerkUserID{value: clerkID}, nil
}

func (c ClerkUserID) String() string {
	return c.value
}

func (c ClerkUserID) Equals(other ClerkUserID) bool {
	return c.value == other.value
}
