package user_test

import (
	"testing"
	"time"

	"snapdeploy-core/internal/domain/user"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		username    string
		clerkUserID string
		wantErr     bool
	}{
		{
			name:        "valid user",
			email:       "test@example.com",
			username:    "testuser",
			clerkUserID: "user_123",
			wantErr:     false,
		},
		{
			name:        "invalid email",
			email:       "invalid",
			username:    "testuser",
			clerkUserID: "user_123",
			wantErr:     true,
		},
		{
			name:        "invalid username",
			email:       "test@example.com",
			username:    "ab",
			clerkUserID: "user_123",
			wantErr:     true,
		},
		{
			name:        "invalid clerk ID",
			email:       "test@example.com",
			username:    "testuser",
			clerkUserID: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usr, err := user.NewUser(tt.email, tt.username, tt.clerkUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if usr == nil {
					t.Error("NewUser() returned nil user")
					return
				}
				if usr.Email().String() != tt.email {
					t.Errorf("Email = %v, want %v", usr.Email().String(), tt.email)
				}
				if usr.Username().String() != tt.username {
					t.Errorf("Username = %v, want %v", usr.Username().String(), tt.username)
				}
				if usr.ClerkUserID().String() != tt.clerkUserID {
					t.Errorf("ClerkUserID = %v, want %v", usr.ClerkUserID().String(), tt.clerkUserID)
				}
			}
		})
	}
}

func TestReconstitute(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	email := "test@example.com"
	username := "testuser"
	clerkID := "user_123"
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	usr, err := user.Reconstitute(id, email, username, clerkID, createdAt, updatedAt)
	if err != nil {
		t.Fatalf("Reconstitute() error = %v", err)
	}

	if usr.ID().String() != id {
		t.Errorf("ID = %v, want %v", usr.ID().String(), id)
	}
	if usr.Email().String() != email {
		t.Errorf("Email = %v, want %v", usr.Email().String(), email)
	}
	if usr.Username().String() != username {
		t.Errorf("Username = %v, want %v", usr.Username().String(), username)
	}
	if usr.ClerkUserID().String() != clerkID {
		t.Errorf("ClerkUserID = %v, want %v", usr.ClerkUserID().String(), clerkID)
	}
}

func TestUpdateEmail(t *testing.T) {
	usr, _ := user.NewUser("old@example.com", "testuser", "user_123")
	oldUpdatedAt := usr.UpdatedAt()

	time.Sleep(10 * time.Millisecond)

	newEmail := "new@example.com"
	err := usr.UpdateEmail(newEmail)
	if err != nil {
		t.Fatalf("UpdateEmail() error = %v", err)
	}

	if usr.Email().String() != newEmail {
		t.Errorf("Email = %v, want %v", usr.Email().String(), newEmail)
	}

	if !usr.UpdatedAt().After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated after changing email")
	}
}

func TestUpdateEmailInvalid(t *testing.T) {
	usr, _ := user.NewUser("old@example.com", "testuser", "user_123")

	err := usr.UpdateEmail("invalid")
	if err == nil {
		t.Error("UpdateEmail() should return error for invalid email")
	}
}

func TestUpdateUsername(t *testing.T) {
	usr, _ := user.NewUser("test@example.com", "olduser", "user_123")
	oldUpdatedAt := usr.UpdatedAt()

	time.Sleep(10 * time.Millisecond)

	newUsername := "newuser"
	err := usr.UpdateUsername(newUsername)
	if err != nil {
		t.Fatalf("UpdateUsername() error = %v", err)
	}

	if usr.Username().String() != newUsername {
		t.Errorf("Username = %v, want %v", usr.Username().String(), newUsername)
	}

	if !usr.UpdatedAt().After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated after changing username")
	}
}

func TestUpdateUsernameInvalid(t *testing.T) {
	usr, _ := user.NewUser("test@example.com", "olduser", "user_123")

	err := usr.UpdateUsername("ab")
	if err == nil {
		t.Error("UpdateUsername() should return error for invalid username")
	}
}
