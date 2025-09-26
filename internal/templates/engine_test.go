package templates

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateEngine_RenderString(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()

	tests := []struct {
		name      string
		template  string
		variables map[string]any
		expected  string
		wantErr   bool
	}{
		{
			name:      "simple variable substitution",
			template:  "Hello {{ name }}!",
			variables: map[string]any{"name": "World"},
			expected:  "Hello World!",
			wantErr:   false,
		},
		{
			name:      "multiple variables",
			template:  "package {{ package }}\n\nfunc {{ function }}() {}",
			variables: map[string]any{"package": "main", "function": "Hello"},
			expected:  "package main\n\nfunc Hello() {}",
			wantErr:   false,
		},
		{
			name:      "conditional rendering",
			template:  "{% if hasMain %}func main() {}{% endif %}",
			variables: map[string]any{"hasMain": true},
			expected:  "func main() {}",
			wantErr:   false,
		},
		{
			name:      "loop rendering",
			template:  "{% for dep in dependencies %}{{ dep }}\n{% endfor %}",
			variables: map[string]any{"dependencies": []string{"fmt", "os"}},
			expected:  "fmt\nos\n",
			wantErr:   false,
		},
		{
			name:      "invalid template syntax",
			template:  "{{ unclosed",
			variables: map[string]any{},
			expected:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.RenderString(ctx, tt.template, tt.variables)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTemplateEngine_RenderToFile(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()
	tempDir := t.TempDir()

	template := "package {{ package }}\n\nfunc main() {\n\tprintln(\"{{ message }}\")\n}"
	variables := map[string]any{
		"package": "main",
		"message": "Hello, World!",
	}
	outputPath := filepath.Join(tempDir, "main.go")

	err := engine.RenderToFile(ctx, template, variables, outputPath)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	expected := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
	assert.Equal(t, expected, string(content))
}

func TestTemplateEngine_RenderTemplate(t *testing.T) {
	engine := NewEngine()
	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a template
	template := Template{
		Name:    "cli-main",
		Content: "package main\n\nfunc main() {\n\tprintln(\"{{ AppName }}\")\n}",
	}

	variables := map[string]any{
		"AppName": "My CLI App",
	}

	outputPath := filepath.Join(tempDir, "main.go")

	err := engine.RenderTemplate(ctx, template, variables, outputPath)
	require.NoError(t, err)

	// Verify file contents
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	expected := "package main\n\nfunc main() {\n\tprintln(\"My CLI App\")\n}"
	assert.Equal(t, expected, string(content))
}
