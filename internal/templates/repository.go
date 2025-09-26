package templates

import (
	"context"
	"fmt"
)

// TemplateFile represents a file within a template
type TemplateFile struct {
	Name    string
	Content string
	Path    string // Relative path within the project
}

// Repository manages template storage and retrieval
type Repository struct {
	predefinedTemplates map[string]Template
	templateFiles       map[string][]TemplateFile
}

// NewRepository creates a new template repository
func NewRepository() *Repository {
	repo := &Repository{
		predefinedTemplates: make(map[string]Template),
		templateFiles:       make(map[string][]TemplateFile),
	}
	repo.initPredefinedTemplates()
	return repo
}

// GetPredefinedTemplate retrieves a predefined template by kind
func (r *Repository) GetPredefinedTemplate(ctx context.Context, kind string) (Template, error) {
	template, exists := r.predefinedTemplates[kind]
	if !exists {
		return Template{}, fmt.Errorf("template kind '%s' not found", kind)
	}
	return template, nil
}

// ListPredefinedTemplates returns all predefined templates
func (r *Repository) ListPredefinedTemplates(ctx context.Context) ([]Template, error) {
	templates := make([]Template, 0, len(r.predefinedTemplates))
	for _, template := range r.predefinedTemplates {
		templates = append(templates, template)
	}
	return templates, nil
}

// GetTemplateFiles returns all files for a template kind
func (r *Repository) GetTemplateFiles(ctx context.Context, kind string) ([]TemplateFile, error) {
	files, exists := r.templateFiles[kind]
	if !exists {
		return nil, fmt.Errorf("template files for kind '%s' not found", kind)
	}
	return files, nil
}

// initPredefinedTemplates initializes all predefined templates
func (r *Repository) initPredefinedTemplates() {
	// CLI template
	r.predefinedTemplates["cli"] = Template{
		Name: "CLI Application",
		Kind: "cli",
		Content: `A command-line application template with {{ ProjectName }}, module {{ ModuleName }}, by {{ Author }}`,
	}
	r.templateFiles["cli"] = []TemplateFile{
		{
			Name: "main.go",
			Path: "cmd/{{ ProjectName }}/main.go",
			Content: `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <command>\n", os.Args[0])
		os.Exit(1)
	}
	
	fmt.Printf("{{ ProjectName }} - A CLI application by {{ Author }}\n")
	fmt.Printf("Command: %s\n", os.Args[1])
}`,
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}`,
		},
		{
			Name: "README.md",
			Path: "README.md",
			Content: `# {{ ProjectName }}

{{ Description }}

## Installation

` + "```bash" + `
go install {{ ModuleName }}
` + "```" + `

## Usage

` + "```bash" + `
{{ ProjectName }} <command>
` + "```" + `

## Author

{{ Author }}`,
		},
		{
			Name: ".gitignore",
			Path: ".gitignore",
			Content: `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
{{ ProjectName }}

# Test binary
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
Thumbs.db`,
		},
		{
			Name: "Makefile",
			Path: "Makefile",
			Content: `.PHONY: build test clean run

BINARY_NAME={{ ProjectName }}
MAIN_PATH=./cmd/{{ ProjectName }}

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)`,
		},
	}

	// Library template
	r.predefinedTemplates["library"] = Template{
		Name: "Go Library",
		Kind: "library",
		Content: `A Go library template for {{ ProjectName }}, module {{ ModuleName }}, by {{ Author }}`,
	}
	r.templateFiles["library"] = []TemplateFile{
		{
			Name: "lib.go",
			Path: "{{ ProjectName }}.go",
			Content: `// Package {{ ProjectName }} {{ Description }}
package {{ ProjectName }}

// Version returns the library version
func Version() string {
	return "1.0.0"
}

// Hello returns a greeting message
func Hello(name string) string {
	return "Hello, " + name + "!"
}`,
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}`,
		},
		{
			Name: "README.md",
			Path: "README.md",
			Content: `# {{ ProjectName }}

{{ Description }}

## Installation

` + "```bash" + `
go get {{ ModuleName }}
` + "```" + `

## Usage

` + "```go" + `
package main

import (
	"fmt"
	"{{ ModuleName }}"
)

func main() {
	fmt.Println({{ ProjectName }}.Hello("World"))
}
` + "```" + `

## Author

{{ Author }}`,
		},
		{
			Name: ".gitignore",
			Path: ".gitignore",
			Content: `# Test binary
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
Thumbs.db`,
		},
	}

	// API template
	r.predefinedTemplates["api"] = Template{
		Name: "Web API",
		Kind: "api",
		Content: `A REST API template for {{ ProjectName }}, module {{ ModuleName }}, by {{ Author }}`,
	}
	r.templateFiles["api"] = []TemplateFile{
		{
			Name: "main.go",
			Path: "cmd/{{ ProjectName }}/main.go",
			Content: `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{{ ProjectName }} API - by {{ Author }}")
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `+"`"+`{"status":"ok"}`+"`"+`)
	})
	
	fmt.Println("Starting {{ ProjectName }} API on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}`,
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}`,
		},
		{
			Name: "README.md",
			Path: "README.md",
			Content: `# {{ ProjectName }} API

{{ Description }}

## Running

` + "```bash" + `
go run cmd/{{ ProjectName }}/main.go
` + "```" + `

## Endpoints

- ` + "`GET /`" + ` - Root endpoint
- ` + "`GET /health`" + ` - Health check

## Author

{{ Author }}`,
		},
		{
			Name: ".gitignore",
			Path: ".gitignore",
			Content: `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
{{ ProjectName }}

# Test binary
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
Thumbs.db`,
		},
		{
			Name: "Makefile",
			Path: "Makefile",
			Content: `.PHONY: build test clean run

BINARY_NAME={{ ProjectName }}
MAIN_PATH=./cmd/{{ ProjectName }}

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

dev:
	go run $(MAIN_PATH)`,
		},
	}

	// gRPC template
	r.predefinedTemplates["grpc"] = Template{
		Name: "gRPC Service",
		Kind: "grpc",
		Content: `A gRPC service template for {{ ProjectName }}, module {{ ModuleName }}, by {{ Author }}`,
	}
	r.templateFiles["grpc"] = []TemplateFile{
		{
			Name: "main.go",
			Path: "cmd/{{ ProjectName }}/main.go",
			Content: `package main

import (
	"fmt"
	"log"
	"net"
	
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	// Register your services here
	
	fmt.Println("{{ ProjectName }} gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}`,
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}

require (
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
)`,
		},
		{
			Name: "README.md",
			Path: "README.md",
			Content: `# {{ ProjectName }} gRPC Service

{{ Description }}

## Running

` + "```bash" + `
go run cmd/{{ ProjectName }}/main.go
` + "```" + `

## Author

{{ Author }}`,
		},
		{
			Name: ".gitignore",
			Path: ".gitignore",
			Content: `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
{{ ProjectName }}

# Test binary
*.test

# Output of the go coverage tool
*.out

# Generated protobuf files (uncomment if needed)
# *.pb.go

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
Thumbs.db`,
		},
	}

	// Microservice template
	r.predefinedTemplates["microservice"] = Template{
		Name: "Microservice",
		Kind: "microservice",
		Content: `A microservice template for {{ ProjectName }}, module {{ ModuleName }}, by {{ Author }}`,
	}
	r.templateFiles["microservice"] = []TemplateFile{
		{
			Name: "main.go",
			Path: "cmd/{{ ProjectName }}/main.go",
			Content: `package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `+"`"+`{"status":"ok","service":"{{ ProjectName }}"}`+"`"+`)
	})
	
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "# Metrics for {{ ProjectName }}")
	})
	
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	
	fmt.Println("{{ ProjectName }} microservice starting on :8080")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}`,
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}`,
		},
		{
			Name: "README.md",
			Path: "README.md",
			Content: `# {{ ProjectName }} Microservice

{{ Description }}

## Running

` + "```bash" + `
go run cmd/{{ ProjectName }}/main.go
` + "```" + `

## Endpoints

- ` + "`GET /health`" + ` - Health check
- ` + "`GET /metrics`" + ` - Metrics endpoint

## Docker

` + "```bash" + `
docker build -t {{ ProjectName }} .
docker run -p 8080:8080 {{ ProjectName }}
` + "```" + `

## Author

{{ Author }}`,
		},
		{
			Name: ".gitignore",
			Path: ".gitignore",
			Content: `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
{{ ProjectName }}

# Test binary
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
Thumbs.db`,
		},
	}
}
