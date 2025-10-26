package service_test

import (
	"context"
	"errors"
	"testing"

	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/domain/user"
)

// Mock implementations
type mockRepositoryRepo struct {
	repos       map[string]*repo.Repository
	urlIndex    map[string]*repo.Repository
	shouldError bool
}

func newMockRepositoryRepo() *mockRepositoryRepo {
	return &mockRepositoryRepo{
		repos:    make(map[string]*repo.Repository),
		urlIndex: make(map[string]*repo.Repository),
	}
}

func (m *mockRepositoryRepo) Save(ctx context.Context, repository *repo.Repository) error {
	if m.shouldError {
		return errors.New("repository error")
	}
	m.repos[repository.ID().String()] = repository
	m.urlIndex[repository.URL().String()] = repository
	return nil
}

func (m *mockRepositoryRepo) FindByID(ctx context.Context, id repo.RepositoryID) (*repo.Repository, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	repository, ok := m.repos[id.String()]
	if !ok {
		return nil, repo.ErrRepositoryNotFound(id.String())
	}
	return repository, nil
}

func (m *mockRepositoryRepo) FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*repo.Repository, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	var result []*repo.Repository
	for _, repository := range m.repos {
		if repository.UserID().Equals(userID) {
			result = append(result, repository)
		}
	}
	return result, nil
}

func (m *mockRepositoryRepo) CountByUserID(ctx context.Context, userID user.UserID) (int64, error) {
	if m.shouldError {
		return 0, errors.New("repository error")
	}
	count := int64(0)
	for _, repository := range m.repos {
		if repository.UserID().Equals(userID) {
			count++
		}
	}
	return count, nil
}

func (m *mockRepositoryRepo) FindByURL(ctx context.Context, url repo.URL) (*repo.Repository, error) {
	if m.shouldError {
		return nil, errors.New("repository error")
	}
	repository, ok := m.urlIndex[url.String()]
	if !ok {
		return nil, repo.ErrRepositoryNotFound(url.String())
	}
	return repository, nil
}

func (m *mockRepositoryRepo) Delete(ctx context.Context, id repo.RepositoryID) error {
	if m.shouldError {
		return errors.New("repository error")
	}
	delete(m.repos, id.String())
	return nil
}

type mockGitHubService struct {
	repos       []*repo.GitHubRepository
	shouldError bool
}

func (m *mockGitHubService) FetchUserRepositories(ctx context.Context, accessToken string) ([]*repo.GitHubRepository, error) {
	if m.shouldError {
		return nil, errors.New("github error")
	}
	if m.repos == nil {
		desc := "Test repository"
		lang := "Go"
		return []*repo.GitHubRepository{
			{
				ID:              12345,
				Name:            "test-repo",
				FullName:        "user/test-repo",
				Description:     &desc,
				URL:             "https://github.com/user/test-repo",
				HTMLURL:         "https://github.com/user/test-repo",
				Private:         false,
				Fork:            false,
				StargazersCount: 10,
				WatchersCount:   5,
				ForksCount:      2,
				DefaultBranch:   "main",
				Language:        &lang,
			},
		}, nil
	}
	return m.repos, nil
}

func TestRepositoryService_SyncRepositoriesFromGitHub(t *testing.T) {
	repoRepo := newMockRepositoryRepo()
	githubSvc := &mockGitHubService{}
	svc := service.NewRepositoryService(repoRepo, githubSvc)

	userID := user.NewUserID()

	resp, err := svc.SyncRepositoriesFromGitHub(context.Background(), userID.String(), "token")
	if err != nil {
		t.Fatalf("SyncRepositoriesFromGitHub() error = %v", err)
	}

	if len(resp.Repositories) != 1 {
		t.Errorf("len(Repositories) = %v, want 1", len(resp.Repositories))
	}

	if resp.Repositories[0].Name != "test-repo" {
		t.Errorf("Name = %v, want test-repo", resp.Repositories[0].Name)
	}
}

func TestRepositoryService_SyncRepositoriesUpdate(t *testing.T) {
	repoRepo := newMockRepositoryRepo()
	githubSvc := &mockGitHubService{}
	svc := service.NewRepositoryService(repoRepo, githubSvc)

	userID := user.NewUserID()

	// First sync
	resp1, err := svc.SyncRepositoriesFromGitHub(context.Background(), userID.String(), "token")
	if err != nil {
		t.Fatalf("SyncRepositoriesFromGitHub() error = %v", err)
	}

	// Update GitHub data
	desc := "Updated repository"
	lang := "Go"
	githubSvc.repos = []*repo.GitHubRepository{
		{
			ID:              12345,
			Name:            "test-repo",
			FullName:        "user/test-repo",
			Description:     &desc,
			URL:             "https://github.com/user/test-repo",
			HTMLURL:         "https://github.com/user/test-repo",
			Private:         false,
			Fork:            false,
			StargazersCount: 20, // Updated
			WatchersCount:   10, // Updated
			ForksCount:      5,  // Updated
			DefaultBranch:   "main",
			Language:        &lang,
		},
	}

	// Second sync should update
	resp2, err := svc.SyncRepositoriesFromGitHub(context.Background(), userID.String(), "token")
	if err != nil {
		t.Fatalf("SyncRepositoriesFromGitHub() error = %v", err)
	}

	if resp2.Repositories[0].Stars != 20 {
		t.Errorf("Stars = %v, want 20", resp2.Repositories[0].Stars)
	}

	// Should be the same repository (updated, not duplicated)
	if resp1.Repositories[0].ID != resp2.Repositories[0].ID {
		t.Error("Repository should be updated, not duplicated")
	}
}

func TestRepositoryService_GetRepositoriesByUserID(t *testing.T) {
	repoRepo := newMockRepositoryRepo()
	githubSvc := &mockGitHubService{}
	svc := service.NewRepositoryService(repoRepo, githubSvc)

	userID := user.NewUserID()

	// Create repositories
	r1, _ := repo.NewRepository(userID, 12345, "repo1", "user/repo1", "https://github.com/user/repo1")
	r2, _ := repo.NewRepository(userID, 67890, "repo2", "user/repo2", "https://github.com/user/repo2")
	_ = repoRepo.Save(context.Background(), r1)
	_ = repoRepo.Save(context.Background(), r2)

	resp, err := svc.GetRepositoriesByUserID(context.Background(), userID.String(), 1, 10)
	if err != nil {
		t.Fatalf("GetRepositoriesByUserID() error = %v", err)
	}

	if len(resp.Repositories) != 2 {
		t.Errorf("len(Repositories) = %v, want 2", len(resp.Repositories))
	}
	if resp.Pagination.Total != 2 {
		t.Errorf("Total = %v, want 2", resp.Pagination.Total)
	}
}

func TestRepositoryService_GetRepositoriesWithPagination(t *testing.T) {
	repoRepo := newMockRepositoryRepo()
	githubSvc := &mockGitHubService{}
	svc := service.NewRepositoryService(repoRepo, githubSvc)

	userID := user.NewUserID()

	// Test with default values (should clamp)
	resp, err := svc.GetRepositoriesByUserID(context.Background(), userID.String(), 0, 0)
	if err != nil {
		t.Fatalf("GetRepositoriesByUserID() error = %v", err)
	}

	if resp.Pagination.Page != 1 {
		t.Errorf("Page = %v, want 1", resp.Pagination.Page)
	}
	if resp.Pagination.Limit != 20 {
		t.Errorf("Limit = %v, want 20", resp.Pagination.Limit)
	}

	// Test with too large limit (should clamp to 20)
	resp, err = svc.GetRepositoriesByUserID(context.Background(), userID.String(), 1, 200)
	if err != nil {
		t.Fatalf("GetRepositoriesByUserID() error = %v", err)
	}

	if resp.Pagination.Limit != 20 {
		t.Errorf("Limit = %v, want 20 (clamped)", resp.Pagination.Limit)
	}
}
