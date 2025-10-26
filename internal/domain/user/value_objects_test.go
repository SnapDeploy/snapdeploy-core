package user_test

import (
	"testing"

	"snapdeploy-core/internal/domain/user"
)

func TestNewEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"empty email", "", true},
		{"invalid format no @", "notanemail", true},
		{"invalid format no domain", "test@", true},
		{"too long email", "a" + string(make([]byte, 256)) + "@example.com", true},
		{"email with spaces trimmed", "  test@example.com  ", false},
		{"uppercase converted to lowercase", "TEST@EXAMPLE.COM", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := user.NewEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && email.String() == "" {
				t.Errorf("NewEmail() returned empty string for valid email")
			}
		})
	}
}

func TestNewUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "testuser", false},
		{"valid with underscore", "test_user", false},
		{"valid with hyphen", "test-user", false},
		{"valid with numbers", "user123", false},
		{"too short", "ab", true},
		{"too long", string(make([]byte, 51)), true},
		{"empty username", "", true},
		{"invalid characters", "test@user", true},
		{"spaces", "test user", true},
		{"username with spaces trimmed", "  testuser  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := user.NewUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && username.String() == "" {
				t.Errorf("NewUsername() returned empty string for valid username")
			}
		})
	}
}

func TestNewClerkUserID(t *testing.T) {
	tests := []struct {
		name    string
		clerkID string
		wantErr bool
	}{
		{"valid clerk ID", "user_2abcdefg123456", false},
		{"empty clerk ID", "", true},
		{"too long", string(make([]byte, 256)), true},
		{"clerk ID with spaces trimmed", "  user_123  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clerkID, err := user.NewClerkUserID(tt.clerkID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClerkUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && clerkID.String() == "" {
				t.Errorf("NewClerkUserID() returned empty string for valid clerk ID")
			}
		})
	}
}

func TestUserIDEquals(t *testing.T) {
	id1 := user.NewUserID()
	id2 := user.NewUserID()

	if id1.Equals(id2) {
		t.Error("Different UserIDs should not be equal")
	}

	if !id1.Equals(id1) {
		t.Error("Same UserID should be equal to itself")
	}
}

func TestEmailEquals(t *testing.T) {
	email1, _ := user.NewEmail("test@example.com")
	email2, _ := user.NewEmail("test@example.com")
	email3, _ := user.NewEmail("other@example.com")

	if !email1.Equals(email2) {
		t.Error("Same emails should be equal")
	}

	if email1.Equals(email3) {
		t.Error("Different emails should not be equal")
	}
}

func TestUsernameEquals(t *testing.T) {
	username1, _ := user.NewUsername("testuser")
	username2, _ := user.NewUsername("testuser")
	username3, _ := user.NewUsername("otheruser")

	if !username1.Equals(username2) {
		t.Error("Same usernames should be equal")
	}

	if username1.Equals(username3) {
		t.Error("Different usernames should not be equal")
	}
}
