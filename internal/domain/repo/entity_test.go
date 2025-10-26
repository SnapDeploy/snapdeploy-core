package repo_test

import (
	"testing"
	"time"

	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/domain/user"
)

func TestNewRepository(t *testing.T) {
	userID := user.NewUserID()

	tests := []struct {
		name     string
		githubID int64
		repoName string
		fullName string
		url      string
		wantErr  bool
	}{
		{
			name:     "valid repository",
			githubID: 12345,
			repoName: "my-repo",
			fullName: "user/my-repo",
			url:      "https://github.com/user/repo",
			wantErr:  false,
		},
		{
			name:     "invalid repository name",
			githubID: 12345,
			repoName: "",
			fullName: "user/my-repo",
			url:      "https://github.com/user/repo",
			wantErr:  true,
		},
		{
			name:     "invalid GitHub ID",
			githubID: 0,
			repoName: "my-repo",
			fullName: "user/my-repo",
			url:      "https://github.com/user/repo",
			wantErr:  true,
		},
		{
			name:     "invalid URL",
			githubID: 12345,
			repoName: "my-repo",
			fullName: "user/my-repo",
			url:      "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository, err := repo.NewRepository(userID, tt.githubID, tt.repoName, tt.fullName, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if repository == nil {
					t.Error("NewRepository() returned nil repository")
					return
				}
				if repository.Name().String() != tt.repoName {
					t.Errorf("Name = %v, want %v", repository.Name().String(), tt.repoName)
				}
				if repository.FullName() != tt.fullName {
					t.Errorf("FullName = %v, want %v", repository.FullName(), tt.fullName)
				}
			}
		})
	}
}

func TestUpdateMetadata(t *testing.T) {
	userID := user.NewUserID()
	repository, _ := repo.NewRepository(userID, 12345, "my-repo", "user/my-repo", "https://github.com/user/repo")

	desc := "A test repository"
	htmlURL := "https://github.com/user/repo"
	branch := "main"
	lang := "Go"

	oldUpdatedAt := repository.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	repository.UpdateMetadata(&desc, &htmlURL, true, false, 100, 50, 25, &branch, &lang)

	if repository.Description() == nil || *repository.Description() != desc {
		t.Errorf("Description = %v, want %v", repository.Description(), desc)
	}
	if repository.HTMLURL() == nil || *repository.HTMLURL() != htmlURL {
		t.Errorf("HTMLURL = %v, want %v", repository.HTMLURL(), htmlURL)
	}
	if !repository.IsPrivate() {
		t.Error("IsPrivate should be true")
	}
	if repository.IsFork() {
		t.Error("IsFork should be false")
	}
	if repository.StargazersCount() != 100 {
		t.Errorf("StargazersCount = %v, want 100", repository.StargazersCount())
	}
	if !repository.UpdatedAt().After(oldUpdatedAt) {
		t.Error("UpdatedAt should be updated after changing metadata")
	}
}

func TestBelongsToUser(t *testing.T) {
	userID1 := user.NewUserID()
	userID2 := user.NewUserID()

	repository, _ := repo.NewRepository(userID1, 12345, "my-repo", "user/my-repo", "https://github.com/user/repo")

	if !repository.BelongsToUser(userID1) {
		t.Error("Repository should belong to userID1")
	}

	if repository.BelongsToUser(userID2) {
		t.Error("Repository should not belong to userID2")
	}
}

func TestReconstitute(t *testing.T) {
	userID := user.NewUserID()
	id := "550e8400-e29b-41d4-a716-446655440000"
	name := "my-repo"
	fullName := "user/my-repo"
	url := "https://github.com/user/repo"
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	repository, err := repo.Reconstitute(
		id, userID, 12345, name, fullName, nil, url, nil,
		false, false, 0, 0, 0, nil, nil, createdAt, updatedAt,
	)

	if err != nil {
		t.Fatalf("Reconstitute() error = %v", err)
	}

	if repository.ID().String() != id {
		t.Errorf("ID = %v, want %v", repository.ID().String(), id)
	}
	if repository.Name().String() != name {
		t.Errorf("Name = %v, want %v", repository.Name().String(), name)
	}
	if repository.FullName() != fullName {
		t.Errorf("FullName = %v, want %v", repository.FullName(), fullName)
	}
}
