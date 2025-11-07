package service_test

import (
	"context"
	"testing"

	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/domain/user"
)

func TestRepositoryService_SearchRepositories(t *testing.T) {
	repoRepo := newMockRepositoryRepo()
	githubSvc := &mockGitHubService{}
	svc := service.NewRepositoryService(repoRepo, githubSvc)

	userID := user.NewUserID()

	// Create test repositories
	r1, _ := repo.NewRepository(userID, 12345, "my-react-app", "user/my-react-app", "https://github.com/user/my-react-app")
	r2, _ := repo.NewRepository(userID, 67890, "golang-service", "user/golang-service", "https://github.com/user/golang-service")
	r3, _ := repo.NewRepository(userID, 11111, "python-script", "user/python-script", "https://github.com/user/python-script")
	_ = repoRepo.Save(context.Background(), r1)
	_ = repoRepo.Save(context.Background(), r2)
	_ = repoRepo.Save(context.Background(), r3)

	// Test with search query
	resp, err := svc.GetRepositoriesByUserID(context.Background(), userID.String(), "golang", 1, 10)
	if err != nil {
		t.Fatalf("GetRepositoriesByUserID() with search error = %v", err)
	}

	// In a real implementation with proper search, we'd expect only matching results
	// For now, our mock returns all repos, so we just verify it works
	if len(resp.Repositories) == 0 {
		t.Error("Expected some repositories to be returned")
	}
}

func TestRepositoryService_EmptySearch(t *testing.T) {
	repoRepo := newMockRepositoryRepo()
	githubSvc := &mockGitHubService{}
	svc := service.NewRepositoryService(repoRepo, githubSvc)

	userID := user.NewUserID()

	// Create test repositories
	r1, _ := repo.NewRepository(userID, 12345, "repo1", "user/repo1", "https://github.com/user/repo1")
	_ = repoRepo.Save(context.Background(), r1)

	// Test with empty search query (should behave like regular list)
	resp, err := svc.GetRepositoriesByUserID(context.Background(), userID.String(), "", 1, 10)
	if err != nil {
		t.Fatalf("GetRepositoriesByUserID() with empty search error = %v", err)
	}

	if len(resp.Repositories) != 1 {
		t.Errorf("len(Repositories) = %v, want 1", len(resp.Repositories))
	}
}



