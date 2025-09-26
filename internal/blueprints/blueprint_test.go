package blueprints

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlueprint_Resolve(t *testing.T) {
	tests := []struct {
		name      string
		blueprint Blueprint
		inputs    map[string]any
		expected  map[string]any
		wantErr   bool
	}{
		{
			name: "web stack blueprint",
			blueprint: Blueprint{
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
				},
			},
			inputs: map[string]any{
				"ProjectName": "myapi",
				"ModuleName":  "github.com/user/myapi",
			},
			expected: map[string]any{
				"ProjectName": "myapi",
				"ModuleName":  "github.com/user/myapi",
				"Components":  []string{"gin", "gorm", "viper"},
				"HasDatabase": true,
				"DatabaseType": "postgres",
				"HasMigrations": true,
				"MigrationType": "goose",
				"HasPrometheus": true,
				"LoggingType": "slog",
				"TestFramework": "testify",
				"CoverageMin": 0.80,
			},
			wantErr: false,
		},
		{
			name: "cli stack blueprint",
			blueprint: Blueprint{
				Name:  "cli-stack",
				Stack: "cli",
				Config: BlueprintConfig{
					Components: []string{"cobra", "viper"},
					Testing: map[string]any{
						"framework": "testify",
					},
				},
			},
			inputs: map[string]any{
				"ProjectName": "mycli",
				"ModuleName":  "github.com/user/mycli",
			},
			expected: map[string]any{
				"ProjectName": "mycli",
				"ModuleName":  "github.com/user/mycli",
				"Components":  []string{"cobra", "viper"},
				"HasDatabase": false,
				"TestFramework": "testify",
			},
			wantErr: false,
		},
		{
			name: "grpc stack blueprint",
			blueprint: Blueprint{
				Name:  "grpc-stack",
				Stack: "grpc",
				Config: BlueprintConfig{
					Components: []string{"grpc", "protobuf"},
					Observability: map[string]any{
						"tracing": "jaeger",
					},
				},
			},
			inputs: map[string]any{
				"ProjectName": "mygrpc",
				"ModuleName":  "github.com/user/mygrpc",
			},
			expected: map[string]any{
				"ProjectName": "mygrpc",
				"ModuleName":  "github.com/user/mygrpc",
				"Components":  []string{"grpc", "protobuf"},
				"HasTracing": true,
				"TracingType": "jaeger",
				"HasDatabase": false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewResolver()
			result, err := resolver.Resolve(context.Background(), tt.blueprint, tt.inputs)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			
			// Check that all expected keys are present with correct values
			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				assert.True(t, exists, "key %s should exist in result", key)
				assert.Equal(t, expectedValue, actualValue, "value for key %s should match", key)
			}
		})
	}
}

func TestBlueprintRepository_GetBlueprint(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	tests := []struct {
		name        string
		blueprintID string
		expectFound bool
		expectStack string
	}{
		{
			name:        "web stack blueprint",
			blueprintID: "web-stack",
			expectFound: true,
			expectStack: "web",
		},
		{
			name:        "cli stack blueprint",
			blueprintID: "cli-stack",
			expectFound: true,
			expectStack: "cli",
		},
		{
			name:        "grpc stack blueprint",
			blueprintID: "grpc-stack",
			expectFound: true,
			expectStack: "grpc",
		},
		{
			name:        "microservice stack blueprint",
			blueprintID: "microservice-stack",
			expectFound: true,
			expectStack: "microservice",
		},
		{
			name:        "non-existent blueprint",
			blueprintID: "nonexistent",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprint, err := repo.GetBlueprint(ctx, tt.blueprintID)
			
			if tt.expectFound {
				require.NoError(t, err)
				assert.Equal(t, tt.blueprintID, blueprint.Name)
				assert.Equal(t, tt.expectStack, blueprint.Stack)
				assert.NotEmpty(t, blueprint.Config.Components)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestBlueprintRepository_ListBlueprints(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	blueprints, err := repo.ListBlueprints(ctx)
	require.NoError(t, err)

	// Should have at least 4 predefined blueprints
	assert.GreaterOrEqual(t, len(blueprints), 4)

	// Verify expected blueprints are present
	expectedBlueprints := []string{"web-stack", "cli-stack", "grpc-stack", "microservice-stack"}
	actualNames := make([]string, len(blueprints))
	for i, bp := range blueprints {
		actualNames[i] = bp.Name
	}

	for _, expected := range expectedBlueprints {
		assert.Contains(t, actualNames, expected)
	}
}

func TestBlueprintRepository_GetBlueprintsByStack(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	tests := []struct {
		name         string
		stack        string
		expectCount  int
		expectNames  []string
	}{
		{
			name:        "web stack blueprints",
			stack:       "web",
			expectCount: 1,
			expectNames: []string{"web-stack"},
		},
		{
			name:        "cli stack blueprints",
			stack:       "cli",
			expectCount: 1,
			expectNames: []string{"cli-stack"},
		},
		{
			name:        "grpc stack blueprints",
			stack:       "grpc",
			expectCount: 1,
			expectNames: []string{"grpc-stack"},
		},
		{
			name:        "nonexistent stack",
			stack:       "nonexistent",
			expectCount: 0,
			expectNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blueprints, err := repo.GetBlueprintsByStack(ctx, tt.stack)
			require.NoError(t, err)
			
			assert.Len(t, blueprints, tt.expectCount)
			
			if tt.expectCount > 0 {
				actualNames := make([]string, len(blueprints))
				for i, bp := range blueprints {
					actualNames[i] = bp.Name
				}
				
				for _, expectedName := range tt.expectNames {
					assert.Contains(t, actualNames, expectedName)
				}
			}
		})
	}
}
