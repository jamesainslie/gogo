package blueprints

import (
	"context"
	"fmt"
)

// BlueprintConfig represents the configuration for a blueprint
type BlueprintConfig struct {
	Components    []string               `json:"components"`
	Database      map[string]any         `json:"database,omitempty"`
	Observability map[string]any         `json:"observability,omitempty"`
	Testing       map[string]any         `json:"testing,omitempty"`
	CI            map[string]any         `json:"ci,omitempty"`
	Docker        map[string]any         `json:"docker,omitempty"`
	Extra         map[string]any         `json:"extra,omitempty"`
}

// Blueprint represents a stack blueprint
type Blueprint struct {
	ID     int             `json:"id"`
	Name   string          `json:"name"`
	Stack  string          `json:"stack"`
	Config BlueprintConfig `json:"config"`
}

// BlueprintResolver interface for resolving blueprint variables
type BlueprintResolver interface {
	Resolve(ctx context.Context, blueprint Blueprint, inputs map[string]any) (map[string]any, error)
}

// Resolver implements BlueprintResolver
type Resolver struct{}

// NewResolver creates a new blueprint resolver
func NewResolver() *Resolver {
	return &Resolver{}
}

// Resolve resolves a blueprint with input variables to produce template variables
func (r *Resolver) Resolve(ctx context.Context, blueprint Blueprint, inputs map[string]any) (map[string]any, error) {
	// Start with input variables
	result := make(map[string]any)
	for k, v := range inputs {
		result[k] = v
	}

	// Add components
	if len(blueprint.Config.Components) > 0 {
		result["Components"] = blueprint.Config.Components
	}

	// Process database configuration
	if len(blueprint.Config.Database) > 0 {
		result["HasDatabase"] = true
		if dbType, ok := blueprint.Config.Database["type"]; ok {
			result["DatabaseType"] = dbType
		}
		if migrations, ok := blueprint.Config.Database["migrations"]; ok {
			result["HasMigrations"] = true
			result["MigrationType"] = migrations
		}
	} else {
		result["HasDatabase"] = false
	}

	// Process observability configuration
	if len(blueprint.Config.Observability) > 0 {
		if prometheus, ok := blueprint.Config.Observability["prometheus"]; ok && prometheus == true {
			result["HasPrometheus"] = true
		}
		if logging, ok := blueprint.Config.Observability["logging"]; ok {
			result["LoggingType"] = logging
		}
		if tracing, ok := blueprint.Config.Observability["tracing"]; ok {
			result["HasTracing"] = true
			result["TracingType"] = tracing
		}
	}

	// Process testing configuration
	if len(blueprint.Config.Testing) > 0 {
		if framework, ok := blueprint.Config.Testing["framework"]; ok {
			result["TestFramework"] = framework
		}
	}

	// Process CI configuration
	if len(blueprint.Config.CI) > 0 {
		if coverageMin, ok := blueprint.Config.CI["coverage_min"]; ok {
			result["CoverageMin"] = coverageMin
		}
	}

	// Process Docker configuration
	if len(blueprint.Config.Docker) > 0 {
		result["HasDocker"] = true
		if baseImage, ok := blueprint.Config.Docker["base_image"]; ok {
			result["DockerBaseImage"] = baseImage
		}
	}

	// Add any extra configuration
	if len(blueprint.Config.Extra) > 0 {
		for k, v := range blueprint.Config.Extra {
			result[k] = v
		}
	}

	return result, nil
}

// Repository manages blueprint storage and retrieval
type Repository struct {
	blueprints map[string]Blueprint
}

// NewRepository creates a new blueprint repository
func NewRepository() *Repository {
	repo := &Repository{
		blueprints: make(map[string]Blueprint),
	}
	repo.initPredefinedBlueprints()
	return repo
}

// GetBlueprint retrieves a blueprint by name or ID
func (r *Repository) GetBlueprint(ctx context.Context, nameOrID string) (Blueprint, error) {
	blueprint, exists := r.blueprints[nameOrID]
	if !exists {
		return Blueprint{}, fmt.Errorf("blueprint '%s' not found", nameOrID)
	}
	return blueprint, nil
}

// ListBlueprints returns all blueprints
func (r *Repository) ListBlueprints(ctx context.Context) ([]Blueprint, error) {
	blueprints := make([]Blueprint, 0, len(r.blueprints))
	for _, blueprint := range r.blueprints {
		blueprints = append(blueprints, blueprint)
	}
	return blueprints, nil
}

// GetBlueprintsByStack returns blueprints for a specific stack
func (r *Repository) GetBlueprintsByStack(ctx context.Context, stack string) ([]Blueprint, error) {
	var blueprints []Blueprint
	for _, blueprint := range r.blueprints {
		if blueprint.Stack == stack {
			blueprints = append(blueprints, blueprint)
		}
	}
	return blueprints, nil
}

// initPredefinedBlueprints initializes predefined blueprints
func (r *Repository) initPredefinedBlueprints() {
	// Web stack blueprint
	r.blueprints["web-stack"] = Blueprint{
		ID:    1,
		Name:  "web-stack",
		Stack: "web",
		Config: BlueprintConfig{
			Components: []string{"gin", "gorm", "viper"},
			Database: map[string]any{
				"type":       "postgres",
				"migrations": "goose",
			},
			Observability: map[string]any{
				"prometheus": true,
				"logging":    "slog",
			},
			Testing: map[string]any{
				"framework": "testify",
			},
			CI: map[string]any{
				"coverage_min": 0.80,
			},
			Docker: map[string]any{
				"base_image": "golang:1.25.1",
				"expose":     8080,
			},
		},
	}

	// CLI stack blueprint
	r.blueprints["cli-stack"] = Blueprint{
		ID:    2,
		Name:  "cli-stack",
		Stack: "cli",
		Config: BlueprintConfig{
			Components: []string{"cobra", "viper"},
			Testing: map[string]any{
				"framework": "testify",
			},
			CI: map[string]any{
				"coverage_min": 0.75,
			},
		},
	}

	// gRPC stack blueprint
	r.blueprints["grpc-stack"] = Blueprint{
		ID:    3,
		Name:  "grpc-stack",
		Stack: "grpc",
		Config: BlueprintConfig{
			Components: []string{"grpc", "protobuf"},
			Observability: map[string]any{
				"tracing": "jaeger",
				"logging": "slog",
			},
			Testing: map[string]any{
				"framework": "testify",
			},
			CI: map[string]any{
				"coverage_min": 0.80,
			},
			Docker: map[string]any{
				"base_image": "golang:1.25.1",
				"expose":     50051,
			},
		},
	}

	// Microservice stack blueprint
	r.blueprints["microservice-stack"] = Blueprint{
		ID:    4,
		Name:  "microservice-stack",
		Stack: "microservice",
		Config: BlueprintConfig{
			Components: []string{"gin", "prometheus", "jaeger"},
			Database: map[string]any{
				"type":       "postgres",
				"migrations": "goose",
			},
			Observability: map[string]any{
				"prometheus": true,
				"tracing":    "jaeger",
				"logging":    "slog",
				"health":     true,
			},
			Testing: map[string]any{
				"framework": "testify",
			},
			CI: map[string]any{
				"coverage_min": 0.85,
			},
			Docker: map[string]any{
				"base_image":     "golang:1.25.1",
				"expose":         8080,
				"health_check":   true,
				"multi_stage":    true,
			},
		},
	}
}
