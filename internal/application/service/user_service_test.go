package service_test

import (
	"context"
	"errors"
	"testing"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/domain/user"
)

// Mock implementations
type mockUserRepository struct {
	users         map[string]*user.User
	emailIndex    map[string]*user.User
	clerkIDIndex  map[string]*user.User
	shouldError   bool
	existsByEmail bool
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:        make(map[string]*user.User),
		emailIndex:   make(map[string]*user.User),
		clerkIDIndex: make(map[string]*user.User),
	}
}

func (m *mockUserRepository) Save(ctx context.Context, usr *user.User) error {
	if m.shouldError {
		return errors.New("repository error")
	}
	m.users[usr.ID().String()] = usr
	m.emailIndex[usr.Email().String()] = usr
	m.clerkIDIndex[usr.ClerkUserID().String()] = usr
	return nil
}

func (m *mockUserRepository) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	usr, ok := m.users[id.String()]
	if !ok {
		return nil, user.ErrUserNotFound(id.String())
	}
	return usr, nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	usr, ok := m.emailIndex[email.String()]
	if !ok {
		return nil, user.ErrUserNotFound(email.String())
	}
	return usr, nil
}

func (m *mockUserRepository) FindByClerkID(ctx context.Context, clerkID user.ClerkUserID) (*user.User, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	usr, ok := m.clerkIDIndex[clerkID.String()]
	if !ok {
		return nil, user.ErrUserNotFound(clerkID.String())
	}
	return usr, nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id user.UserID) error {
	if m.shouldError {
		return errors.New("repository error")
	}
	delete(m.users, id.String())
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, limit, offset int32) ([]*user.User, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	var result []*user.User
	for _, usr := range m.users {
		result = append(result, usr)
	}
	return result, nil
}

func (m *mockUserRepository) Count(ctx context.Context) (int64, error) {
	if m.shouldError {
		return 0, errors.New("repository error")
	}
	return int64(len(m.users)), nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email user.Email) (bool, error) {
	if m.shouldError {
		return false, errors.New("repository error")
	}
	return m.existsByEmail, nil
}

type mockClerkService struct {
	userData    *service.ClerkUserData
	shouldError bool
}

func (m *mockClerkService) GetUser(ctx context.Context, clerkUserID string) (*service.ClerkUserData, error) {
	if m.shouldError {
		return nil, errors.New("clerk error")
	}
	if m.userData == nil {
		return &service.ClerkUserData{
			ID:       clerkUserID,
			Email:    "test@example.com",
			Username: "testuser",
		}, nil
	}
	return m.userData, nil
}

func TestUserService_CreateUser(t *testing.T) {
	repo := newMockUserRepository()
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	req := &dto.CreateUserRequest{
		Email:       "test@example.com",
		Username:    "testuser",
		ClerkUserID: "user_123",
	}

	resp, err := svc.CreateUser(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if resp.Email != req.Email {
		t.Errorf("Email = %v, want %v", resp.Email, req.Email)
	}
	if resp.Username != req.Username {
		t.Errorf("Username = %v, want %v", resp.Username, req.Username)
	}
}

func TestUserService_CreateUserDuplicate(t *testing.T) {
	repo := newMockUserRepository()
	repo.existsByEmail = true
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	req := &dto.CreateUserRequest{
		Email:       "test@example.com",
		Username:    "testuser",
		ClerkUserID: "user_123",
	}

	_, err := svc.CreateUser(context.Background(), req)
	if err == nil {
		t.Error("CreateUser() should return error for duplicate email")
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	repo := newMockUserRepository()
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	// Create a user first
	usr, _ := user.NewUser("test@example.com", "testuser", "user_123")
	_ = repo.Save(context.Background(), usr)

	resp, err := svc.GetUserByID(context.Background(), usr.ID().String())
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if resp.ID != usr.ID().String() {
		t.Errorf("ID = %v, want %v", resp.ID, usr.ID().String())
	}
}

func TestUserService_GetOrCreateUserByClerkID(t *testing.T) {
	repo := newMockUserRepository()
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	// User doesn't exist, should create
	resp, err := svc.GetOrCreateUserByClerkID(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("GetOrCreateUserByClerkID() error = %v", err)
	}

	if resp.Email != "test@example.com" {
		t.Errorf("Email = %v, want %v", resp.Email, "test@example.com")
	}

	// User exists, should retrieve
	resp2, err := svc.GetOrCreateUserByClerkID(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("GetOrCreateUserByClerkID() error = %v", err)
	}

	if resp.ID != resp2.ID {
		t.Error("Should return the same user when called again")
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	repo := newMockUserRepository()
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	// Create a user first
	usr, _ := user.NewUser("test@example.com", "testuser", "user_123")
	_ = repo.Save(context.Background(), usr)

	newEmail := "new@example.com"
	newUsername := "newuser"
	req := &dto.UpdateUserRequest{
		Email:    &newEmail,
		Username: &newUsername,
	}

	resp, err := svc.UpdateUser(context.Background(), usr.ID().String(), req)
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	if resp.Email != newEmail {
		t.Errorf("Email = %v, want %v", resp.Email, newEmail)
	}
	if resp.Username != newUsername {
		t.Errorf("Username = %v, want %v", resp.Username, newUsername)
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	repo := newMockUserRepository()
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	// Create a user first
	usr, _ := user.NewUser("test@example.com", "testuser", "user_123")
	_ = repo.Save(context.Background(), usr)

	err := svc.DeleteUser(context.Background(), usr.ID().String())
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	// Verify user is deleted
	_, err = repo.FindByID(context.Background(), usr.ID())
	if err == nil {
		t.Error("User should be deleted")
	}
}

func TestUserService_ListUsers(t *testing.T) {
	repo := newMockUserRepository()
	clerkSvc := &mockClerkService{}
	svc := service.NewUserService(repo, clerkSvc)

	// Create multiple users
	usr1, _ := user.NewUser("test1@example.com", "user1", "user_1")
	usr2, _ := user.NewUser("test2@example.com", "user2", "user_2")
	_ = repo.Save(context.Background(), usr1)
	_ = repo.Save(context.Background(), usr2)

	resp, err := svc.ListUsers(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}

	if len(resp.Users) != 2 {
		t.Errorf("len(Users) = %v, want 2", len(resp.Users))
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("Total = %v, want 2", resp.Pagination.Total)
	}
}
