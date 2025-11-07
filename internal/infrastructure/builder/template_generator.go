package builder

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"snapdeploy-core/internal/domain/project"
)

//go:embed templates/*.tmpl
var templateFiles embed.FS

// TemplateGenerator generates Dockerfiles from templates
type TemplateGenerator struct {
	templates map[project.Language]string
}

// NewTemplateGenerator creates a new template generator
func NewTemplateGenerator() (*TemplateGenerator, error) {
	tg := &TemplateGenerator{
		templates: make(map[project.Language]string),
	}

	// Load templates from embedded files
	templateMap := map[project.Language]string{
		project.LanguageNode:   "templates/node.Dockerfile.tmpl",
		project.LanguageNodeTS: "templates/node_ts.Dockerfile.tmpl",
		project.LanguageNextJS: "templates/nextjs.Dockerfile.tmpl",
		project.LanguageGo:     "templates/go.Dockerfile.tmpl",
		project.LanguagePython: "templates/python.Dockerfile.tmpl",
	}

	for lang, path := range templateMap {
		content, err := templateFiles.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %w", path, err)
		}
		tg.templates[lang] = string(content)
	}

	return tg, nil
}

// TemplateData holds the data for Dockerfile template generation
type TemplateData struct {
	InstallCommand string
	BuildCommand   string
	RunCommand     string
	Port           string
}

// GenerateDockerfile generates a Dockerfile from a template
func (tg *TemplateGenerator) GenerateDockerfile(lang project.Language, data TemplateData) (string, error) {
	templateStr, exists := tg.templates[lang]
	if !exists {
		return "", fmt.Errorf("unsupported language: %s", lang)
	}

	// Set default port if not provided
	if data.Port == "" {
		data.Port = "8080"
	}

	tmpl, err := template.New("dockerfile").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
