package repo_test

import (
	"testing"

	"snapdeploy-core/internal/domain/repo"
)

func TestNewName(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		wantErr  bool
	}{
		{"valid name", "my-repo", false},
		{"valid with underscore", "my_repo", false},
		{"empty name", "", true},
		{"too long", string(make([]byte, 101)), true},
		{"name with spaces trimmed", "  my-repo  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := repo.NewName(tt.repoName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && name.String() == "" {
				t.Errorf("NewName() returned empty string for valid name")
			}
		})
	}
}

func TestNewURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https URL", "https://github.com/user/repo", false},
		{"valid http URL", "http://github.com/user/repo", false},
		{"empty URL", "", true},
		{"invalid URL no protocol", "github.com/user/repo", true},
		{"URL with spaces trimmed", "  https://github.com/user/repo  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := repo.NewURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && url.String() == "" {
				t.Errorf("NewURL() returned empty string for valid URL")
			}
		})
	}
}

func TestNewGitHubID(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{"valid positive ID", 12345, false},
		{"valid large ID", 999999999, false},
		{"invalid zero", 0, true},
		{"invalid negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ghID, err := repo.NewGitHubID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGitHubID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ghID.Int64() != tt.id {
				t.Errorf("NewGitHubID() = %v, want %v", ghID.Int64(), tt.id)
			}
		})
	}
}

func TestRepositoryIDEquals(t *testing.T) {
	id1 := repo.NewRepositoryID()
	id2 := repo.NewRepositoryID()

	if id1.Equals(id2) {
		t.Error("Different RepositoryIDs should not be equal")
	}

	if !id1.Equals(id1) {
		t.Error("Same RepositoryID should be equal to itself")
	}
}

func TestNameEquals(t *testing.T) {
	name1, _ := repo.NewName("my-repo")
	name2, _ := repo.NewName("my-repo")
	name3, _ := repo.NewName("other-repo")

	if !name1.Equals(name2) {
		t.Error("Same names should be equal")
	}

	if name1.Equals(name3) {
		t.Error("Different names should not be equal")
	}
}

func TestURLEquals(t *testing.T) {
	url1, _ := repo.NewURL("https://github.com/user/repo")
	url2, _ := repo.NewURL("https://github.com/user/repo")
	url3, _ := repo.NewURL("https://github.com/user/other")

	if !url1.Equals(url2) {
		t.Error("Same URLs should be equal")
	}

	if url1.Equals(url3) {
		t.Error("Different URLs should not be equal")
	}
}
