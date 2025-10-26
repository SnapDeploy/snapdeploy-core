package repo

import (
	"fmt"
	"time"

	"snapdeploy-core/internal/domain/user"
)

// Repository is a domain entity representing a GitHub repository
type Repository struct {
	id              RepositoryID
	userID          user.UserID
	githubID        GitHubID
	name            Name
	fullName        string
	description     *string
	url             URL
	htmlURL         *string
	isPrivate       bool
	isFork          bool
	stargazersCount int32
	watchersCount   int32
	forksCount      int32
	defaultBranch   *string
	language        *string
	createdAt       time.Time
	updatedAt       time.Time
}

// NewRepository creates a new Repository entity
func NewRepository(
	userID user.UserID,
	githubID int64,
	name, fullName, url string,
) (*Repository, error) {
	repoName, err := NewName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid repository name: %w", err)
	}

	repoURL, err := NewURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	githubIDVO, err := NewGitHubID(githubID)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub ID: %w", err)
	}

	now := time.Now()
	return &Repository{
		id:        NewRepositoryID(),
		userID:    userID,
		githubID:  githubIDVO,
		name:      repoName,
		fullName:  fullName,
		url:       repoURL,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute recreates a Repository entity from persistence
func Reconstitute(
	id string,
	userID user.UserID,
	githubID int64,
	name, fullName string,
	description *string,
	url string,
	htmlURL *string,
	isPrivate, isFork bool,
	stars, watchers, forks int32,
	defaultBranch, language *string,
	createdAt, updatedAt time.Time,
) (*Repository, error) {
	repoID, err := ParseRepositoryID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid repository ID: %w", err)
	}

	repoName, err := NewName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid repository name: %w", err)
	}

	repoURL, err := NewURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	githubIDVO, err := NewGitHubID(githubID)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub ID: %w", err)
	}

	return &Repository{
		id:              repoID,
		userID:          userID,
		githubID:        githubIDVO,
		name:            repoName,
		fullName:        fullName,
		description:     description,
		url:             repoURL,
		htmlURL:         htmlURL,
		isPrivate:       isPrivate,
		isFork:          isFork,
		stargazersCount: stars,
		watchersCount:   watchers,
		forksCount:      forks,
		defaultBranch:   defaultBranch,
		language:        language,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}, nil
}

// UpdateMetadata updates repository metadata from GitHub sync
func (r *Repository) UpdateMetadata(
	description *string,
	htmlURL *string,
	isPrivate, isFork bool,
	stars, watchers, forks int32,
	defaultBranch, language *string,
) {
	r.description = description
	r.htmlURL = htmlURL
	r.isPrivate = isPrivate
	r.isFork = isFork
	r.stargazersCount = stars
	r.watchersCount = watchers
	r.forksCount = forks
	r.defaultBranch = defaultBranch
	r.language = language
	r.updatedAt = time.Now()
}

// BelongsToUser checks if the repository belongs to the specified user
func (r *Repository) BelongsToUser(userID user.UserID) bool {
	return r.userID.Equals(userID)
}

// Getters

func (r *Repository) ID() RepositoryID {
	return r.id
}

func (r *Repository) UserID() user.UserID {
	return r.userID
}

func (r *Repository) GitHubID() GitHubID {
	return r.githubID
}

func (r *Repository) Name() Name {
	return r.name
}

func (r *Repository) FullName() string {
	return r.fullName
}

func (r *Repository) Description() *string {
	return r.description
}

func (r *Repository) URL() URL {
	return r.url
}

func (r *Repository) HTMLURL() *string {
	return r.htmlURL
}

func (r *Repository) IsPrivate() bool {
	return r.isPrivate
}

func (r *Repository) IsFork() bool {
	return r.isFork
}

func (r *Repository) StargazersCount() int32 {
	return r.stargazersCount
}

func (r *Repository) WatchersCount() int32 {
	return r.watchersCount
}

func (r *Repository) ForksCount() int32 {
	return r.forksCount
}

func (r *Repository) DefaultBranch() *string {
	return r.defaultBranch
}

func (r *Repository) Language() *string {
	return r.language
}

func (r *Repository) CreatedAt() time.Time {
	return r.createdAt
}

func (r *Repository) UpdatedAt() time.Time {
	return r.updatedAt
}

// String returns string representation (for debugging)
func (r *Repository) String() string {
	return fmt.Sprintf("Repository{id: %s, name: %s, userID: %s}",
		r.id.String(), r.name.String(), r.userID.String())
}
