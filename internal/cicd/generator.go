package cicd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/gogo/internal/templates"
)

// Config represents CI/CD configuration options
type Config struct {
	ProjectName   string
	GoVersion     string
	CoverageMin   float64
	TestFramework string
	HasDocker     bool
	HasDatabase   bool
	DatabaseType  string
	LintTimeout   string
	BuildTargets  []string
}

// Generator handles CI/CD configuration generation
type Generator struct {
	templateEngine templates.TemplateRenderer
}

// NewGenerator creates a new CI/CD generator
func NewGenerator() *Generator {
	return &Generator{
		templateEngine: templates.NewEngine(),
	}
}

// GenerateAll generates all CI/CD configurations
func (g *Generator) GenerateAll(ctx context.Context, outputDir string, config Config) error {
	// Set defaults
	if config.GoVersion == "" {
		config.GoVersion = "1.25.1"
	}
	if config.CoverageMin == 0 {
		config.CoverageMin = 0.80
	}
	if config.TestFramework == "" {
		config.TestFramework = "testify"
	}
	if config.LintTimeout == "" {
		config.LintTimeout = "5m"
	}
	if len(config.BuildTargets) == 0 {
		config.BuildTargets = []string{"linux", "darwin", "windows"}
	}

	// Generate .golangci.yml
	if err := g.GenerateGolangCIConfig(ctx, outputDir, config); err != nil {
		return fmt.Errorf("failed to generate .golangci.yml: %w", err)
	}

	// Generate GitHub Actions workflow
	if err := g.GenerateGitHubActions(ctx, outputDir, config); err != nil {
		return fmt.Errorf("failed to generate GitHub Actions: %w", err)
	}

	// Generate pre-commit config
	if err := g.GeneratePreCommitConfig(ctx, outputDir, config); err != nil {
		return fmt.Errorf("failed to generate pre-commit config: %w", err)
	}

	return nil
}

// GenerateGolangCIConfig generates .golangci.yml configuration
func (g *Generator) GenerateGolangCIConfig(ctx context.Context, outputDir string, config Config) error {
	template := `run:
  timeout: {{ LintTimeout }}
  issues-exit-code: 1
  tests: true

output:
  format: colored-line-number
  print-issued-lines: true
  print-linter-name: true

linters-settings:
  revive:
    min-confidence: 0
  gocyclo:
    min-complexity: 15
  maligned:
    suggest-new: true
  dupl:
    threshold: 100
  goconst:
    min-len: 2
    min-occurrences: 2

linters:
  disable-all: true
  enable:
    - bodyclose
    - deadcode
    - depguard
    - dogsled
    - dupl
    - errcheck
    - exportloopref
    - exhaustive
    - gochecknoinits
    - goconst
    - gocyclo
    - gofmt
    - goimports
    - gomnd
    - goprintffuncname
    - gosec
    - gosimple
    - govet
    - ineffassign
    - lll
    - misspell
    - nakedret
    - noctx
    - nolintlint
    - revive
    - staticcheck
    - structcheck
    - stylecheck
    - typecheck
    - unconvert
    - unparam
    - unused
    - varcheck
    - whitespace

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gomnd
        - goconst
        - dupl
    - path: cmd/
      linters:
        - gochecknoinits
  exclude-use-default: false
  fix: false

severity:
  default-severity: error
  rules:
    - linters:
        - revive
      severity: warning`

	variables := map[string]any{
		"LintTimeout": config.LintTimeout,
	}

	outputPath := filepath.Join(outputDir, ".golangci.yml")
	return g.templateEngine.RenderToFile(ctx, template, variables, outputPath)
}

// GenerateGitHubActions generates GitHub Actions workflow
func (g *Generator) GenerateGitHubActions(ctx context.Context, outputDir string, config Config) error {
	template := `name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ["{{ GoVersion }}", "stable"]
{% if HasDatabase %}
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: test_db
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
{% endif %}
    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ "{{" }} matrix.go-version {{ "}}" }}

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ "{{" }} runner.os {{ "}}" }}-go-${{ "{{" }} matrix.go-version {{ "}}" }}-${{ "{{" }} hashFiles('**/go.sum') {{ "}}" }}
        restore-keys: |
          ${{ "{{" }} runner.os {{ "}}" }}-go-${{ "{{" }} matrix.go-version {{ "}}" }}-

    - name: Download dependencies
      run: go mod download

    - name: Verify dependencies
      run: go mod verify

    - name: Run tests
      run: |{% if HasDatabase %}
        export DATABASE_URL="postgres://postgres:postgres@localhost:5432/test_db?sslmode=disable"{% endif %}
        go test -race -covermode atomic -coverprofile=coverage.out ./...

    - name: Check coverage
      run: |
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
        echo "Coverage: $COVERAGE%"
        if (( $(echo "$COVERAGE < {{ CoverageMin }}" | bc -l) )); then
          echo "Coverage $COVERAGE% is below minimum {{ CoverageMin }}%"
          exit 1
        fi

    - name: Upload coverage reports to Codecov
      uses: codecov/codecov-action@v3
      env:
        CODECOV_TOKEN: ${{ "{{" }} secrets.CODECOV_TOKEN {{ "}}" }}

  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: "{{ GoVersion }}"
    - name: golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  build:
    runs-on: ubuntu-latest
    needs: [test, lint]
    strategy:
      matrix:
        goos: [{% for target in BuildTargets %}"{{ target }}"{% if not loop.last %}, {% endif %}{% endfor %}]
        goarch: ["amd64"]
        
    steps:
    - uses: actions/checkout@v4
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: "{{ GoVersion }}"
        
    - name: Build
      env:
        GOOS: ${{ "{{" }} matrix.goos {{ "}}" }}
        GOARCH: ${{ "{{" }} matrix.goarch {{ "}}" }}
      run: |
        mkdir -p dist
        go build -ldflags "-s -w" -o dist/{{ ProjectName }}-${{ "{{" }} matrix.goos {{ "}}" }}-${{ "{{" }} matrix.goarch {{ "}}" }}{% if matrix.goos == "windows" %}.exe{% endif %} ./cmd/{{ ProjectName }}
        
    - name: Upload build artifacts
      uses: actions/upload-artifact@v3
      with:
        name: {{ ProjectName }}-${{ "{{" }} matrix.goos {{ "}}" }}-${{ "{{" }} matrix.goarch {{ "}}" }}
        path: dist/{{ ProjectName }}-${{ "{{" }} matrix.goos {{ "}}" }}-${{ "{{" }} matrix.goarch {{ "}}" }}*`

	variables := map[string]any{
		"ProjectName":  config.ProjectName,
		"GoVersion":    config.GoVersion,
		"CoverageMin":  config.CoverageMin * 100, // Convert to percentage
		"HasDatabase":  config.HasDatabase,
		"DatabaseType": config.DatabaseType,
		"BuildTargets": config.BuildTargets,
	}

	// Ensure .github/workflows directory exists
	workflowDir := filepath.Join(outputDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	outputPath := filepath.Join(workflowDir, "ci.yml")
	return g.templateEngine.RenderToFile(ctx, template, variables, outputPath)
}

// GeneratePreCommitConfig generates .pre-commit-config.yaml
func (g *Generator) GeneratePreCommitConfig(ctx context.Context, outputDir string, config Config) error {
	template := `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
      - id: check-case-conflict
      - id: check-merge-conflict
      - id: check-toml
      - id: detect-private-key

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.61.0
    hooks:
      - id: golangci-lint

  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: gofmt -l -s -w .
        language: system
        files: \.go$
        
      - id: go-mod-tidy
        name: go mod tidy
        entry: go mod tidy
        language: system
        files: go\.(mod|sum)$
        
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        files: \.go$
        pass_filenames: false
        
      - id: go-build
        name: go build
        entry: go build ./...
        language: system
        files: \.go$
        pass_filenames: false`

	outputPath := filepath.Join(outputDir, ".pre-commit-config.yaml")
	return g.templateEngine.RenderToFile(ctx, template, map[string]any{}, outputPath)
}
