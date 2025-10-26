package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/domain/user"
)

// RepositoryRepoImpl implements the domain repo.RepositoryRepo interface
type RepositoryRepoImpl struct {
	queries *database.Queries
}

// NewRepositoryRepository creates a new repository repository implementation
func NewRepositoryRepository(db *database.DB) repo.RepositoryRepo {
	return &RepositoryRepoImpl{
		queries: database.New(db.GetConnection()),
	}
}

// Save persists a repository (create or update via upsert)
func (r *RepositoryRepoImpl) Save(ctx context.Context, repository *repo.Repository) error {
	desc := sql.NullString{Valid: false}
	if repository.Description() != nil {
		desc = sql.NullString{String: *repository.Description(), Valid: true}
	}

	htmlURL := sql.NullString{Valid: false}
	if repository.HTMLURL() != nil {
		htmlURL = sql.NullString{String: *repository.HTMLURL(), Valid: true}
	}

	lang := sql.NullString{Valid: false}
	if repository.Language() != nil {
		lang = sql.NullString{String: *repository.Language(), Valid: true}
	}

	branch := sql.NullString{Valid: false}
	if repository.DefaultBranch() != nil {
		branch = sql.NullString{String: *repository.DefaultBranch(), Valid: true}
	}

	_, err := r.queries.UpsertRepository(ctx, &database.UpsertRepositoryParams{
		UserID:          repository.UserID().UUID(),
		GithubID:        repository.GitHubID().Int64(),
		Name:            repository.Name().String(),
		FullName:        repository.FullName(),
		Description:     desc,
		Url:             repository.URL().String(),
		HtmlUrl:         htmlURL,
		Private:         sql.NullBool{Bool: repository.IsPrivate(), Valid: true},
		Fork:            sql.NullBool{Bool: repository.IsFork(), Valid: true},
		StargazersCount: sql.NullInt32{Int32: repository.StargazersCount(), Valid: true},
		WatchersCount:   sql.NullInt32{Int32: repository.WatchersCount(), Valid: true},
		ForksCount:      sql.NullInt32{Int32: repository.ForksCount(), Valid: true},
		DefaultBranch:   branch,
		Language:        lang,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert repository: %w", err)
	}

	return nil
}

// FindByID retrieves a repository by its ID
func (r *RepositoryRepoImpl) FindByID(ctx context.Context, id repo.RepositoryID) (*repo.Repository, error) {
	// Note: We don't have a GetByID query, so this would need to be added to sqlc
	// For now, we'll return an error
	return nil, fmt.Errorf("FindByID not implemented - need to add to sqlc queries")
}

// FindByUserID retrieves repositories for a specific user with pagination
func (r *RepositoryRepoImpl) FindByUserID(ctx context.Context, userID user.UserID, limit, offset int32) ([]*repo.Repository, error) {
	dbRepos, err := r.queries.GetRepositoriesByUserID(ctx, &database.GetRepositoriesByUserIDParams{
		UserID: userID.UUID(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	repositories := make([]*repo.Repository, len(dbRepos))
	for i, dbRepo := range dbRepos {
		domainRepo, err := r.toDomain(dbRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to convert repository: %w", err)
		}
		repositories[i] = domainRepo
	}

	return repositories, nil
}

// CountByUserID returns the total number of repositories for a user
func (r *RepositoryRepoImpl) CountByUserID(ctx context.Context, userID user.UserID) (int64, error) {
	count, err := r.queries.CountRepositoriesByUserID(ctx, userID.UUID())
	if err != nil {
		return 0, fmt.Errorf("failed to count repositories: %w", err)
	}

	return count, nil
}

// FindByURL retrieves a repository by its URL
func (r *RepositoryRepoImpl) FindByURL(ctx context.Context, url repo.URL) (*repo.Repository, error) {
	dbRepo, err := r.queries.GetRepositoryByURL(ctx, url.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repo.ErrRepositoryNotFound(url.String())
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return r.toDomain(dbRepo)
}

// Delete removes a repository from persistence
func (r *RepositoryRepoImpl) Delete(ctx context.Context, id repo.RepositoryID) error {
	err := r.queries.DeleteRepository(ctx, id.UUID())
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	return nil
}

// toDomain converts database repository to domain repository
func (r *RepositoryRepoImpl) toDomain(dbRepo *database.Repository) (*repo.Repository, error) {
	var description *string
	if dbRepo.Description.Valid {
		description = &dbRepo.Description.String
	}

	var htmlURL *string
	if dbRepo.HtmlUrl.Valid {
		htmlURL = &dbRepo.HtmlUrl.String
	}

	var defaultBranch *string
	if dbRepo.DefaultBranch.Valid {
		defaultBranch = &dbRepo.DefaultBranch.String
	}

	var language *string
	if dbRepo.Language.Valid {
		language = &dbRepo.Language.String
	}

	userID, err := user.ParseUserID(dbRepo.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return repo.Reconstitute(
		dbRepo.ID.String(),
		userID,
		dbRepo.GithubID,
		dbRepo.Name,
		dbRepo.FullName,
		description,
		dbRepo.Url,
		htmlURL,
		dbRepo.Private.Bool,
		dbRepo.Fork.Bool,
		dbRepo.StargazersCount.Int32,
		dbRepo.WatchersCount.Int32,
		dbRepo.ForksCount.Int32,
		defaultBranch,
		language,
		dbRepo.CreatedAt.Time,
		dbRepo.UpdatedAt.Time,
	)
}
