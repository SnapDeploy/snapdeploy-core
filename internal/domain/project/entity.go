package project

import (
	"fmt"
	"time"

	"snapdeploy-core/internal/domain/user"
)

// Project is a domain entity representing a deployment project
type Project struct {
	id             ProjectID
	userID         user.UserID
	repositoryURL  RepositoryURL
	installCommand Command
	buildCommand   Command
	runCommand     Command
	language       Language
	createdAt      time.Time
	updatedAt      time.Time
}

// NewProject creates a new Project entity
func NewProject(
	userID user.UserID,
	repositoryURL, installCommand, buildCommand, runCommand, language string,
) (*Project, error) {
	repoURL, err := NewRepositoryURL(repositoryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	installCmd, err := NewCommand(installCommand)
	if err != nil {
		return nil, fmt.Errorf("invalid install command: %w", err)
	}

	buildCmd, err := NewCommand(buildCommand)
	if err != nil {
		return nil, fmt.Errorf("invalid build command: %w", err)
	}

	runCmd, err := NewCommand(runCommand)
	if err != nil {
		return nil, fmt.Errorf("invalid run command: %w", err)
	}

	lang, err := NewLanguage(language)
	if err != nil {
		return nil, fmt.Errorf("invalid language: %w", err)
	}

	now := time.Now()
	return &Project{
		id:             NewProjectID(),
		userID:         userID,
		repositoryURL:  repoURL,
		installCommand: installCmd,
		buildCommand:   buildCmd,
		runCommand:     runCmd,
		language:       lang,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// Reconstitute recreates a Project entity from persistence
func Reconstitute(
	id string,
	userID user.UserID,
	repositoryURL, installCommand, buildCommand, runCommand, language string,
	createdAt, updatedAt time.Time,
) (*Project, error) {
	projectID, err := ParseProjectID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	repoURL, err := NewRepositoryURL(repositoryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	installCmd, err := NewCommand(installCommand)
	if err != nil {
		return nil, fmt.Errorf("invalid install command: %w", err)
	}

	buildCmd, err := NewCommand(buildCommand)
	if err != nil {
		return nil, fmt.Errorf("invalid build command: %w", err)
	}

	runCmd, err := NewCommand(runCommand)
	if err != nil {
		return nil, fmt.Errorf("invalid run command: %w", err)
	}

	lang, err := NewLanguage(language)
	if err != nil {
		return nil, fmt.Errorf("invalid language: %w", err)
	}

	return &Project{
		id:             projectID,
		userID:         userID,
		repositoryURL:  repoURL,
		installCommand: installCmd,
		buildCommand:   buildCmd,
		runCommand:     runCmd,
		language:       lang,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// Update updates project configuration
func (p *Project) Update(
	repositoryURL, installCommand, buildCommand, runCommand, language string,
) error {
	repoURL, err := NewRepositoryURL(repositoryURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	installCmd, err := NewCommand(installCommand)
	if err != nil {
		return fmt.Errorf("invalid install command: %w", err)
	}

	buildCmd, err := NewCommand(buildCommand)
	if err != nil {
		return fmt.Errorf("invalid build command: %w", err)
	}

	runCmd, err := NewCommand(runCommand)
	if err != nil {
		return fmt.Errorf("invalid run command: %w", err)
	}

	lang, err := NewLanguage(language)
	if err != nil {
		return fmt.Errorf("invalid language: %w", err)
	}

	p.repositoryURL = repoURL
	p.installCommand = installCmd
	p.buildCommand = buildCmd
	p.runCommand = runCmd
	p.language = lang
	p.updatedAt = time.Now()

	return nil
}

// BelongsToUser checks if the project belongs to the specified user
func (p *Project) BelongsToUser(userID user.UserID) bool {
	return p.userID.Equals(userID)
}

// Getters

func (p *Project) ID() ProjectID {
	return p.id
}

func (p *Project) UserID() user.UserID {
	return p.userID
}

func (p *Project) RepositoryURL() RepositoryURL {
	return p.repositoryURL
}

func (p *Project) InstallCommand() Command {
	return p.installCommand
}

func (p *Project) BuildCommand() Command {
	return p.buildCommand
}

func (p *Project) RunCommand() Command {
	return p.runCommand
}

func (p *Project) Language() Language {
	return p.language
}

func (p *Project) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}

// String returns string representation (for debugging)
func (p *Project) String() string {
	return fmt.Sprintf("Project{id: %s, userID: %s, language: %s}",
		p.id.String(), p.userID.String(), p.language.String())
}
