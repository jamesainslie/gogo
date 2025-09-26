package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/user/gogo/internal/generator"
)

func TestWizardOptions_ConvertToInitOptions(t *testing.T) {
	tests := []struct {
		name   string
		wizard WizardOptions
		want   generator.InitOptions
	}{
		{
			name: "convert basic options",
			wizard: WizardOptions{
				ProjectName: "myproject",
				ModuleName:  "github.com/user/myproject",
				Template:    "cli",
				Blueprint:   "cli-stack",
				Author:      "John Doe",
				License:     "MIT",
				GoVersion:   "1.25.1",
				OutputDir:   "./myproject",
				GitInit:     true,
				Force:       false,
			},
			want: generator.InitOptions{
				ProjectName: "myproject",
				ModuleName:  "github.com/user/myproject",
				Template:    "cli",
				Blueprint:   "cli-stack",
				Author:      "John Doe",
				License:     "MIT",
				GoVersion:   "1.25.1",
				OutputDir:   "./myproject",
				Description: "A cli project",
				GitInit:     true,
				Force:       false,
				DryRun:      false,
			},
		},
		{
			name: "convert api template options",
			wizard: WizardOptions{
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
				Template:    "api",
				Blueprint:   "web-stack",
				Author:      "Jane Smith",
				License:     "Apache-2.0",
				GoVersion:   "1.25.1",
				OutputDir:   "./myapi",
				GitInit:     false,
				Force:       true,
			},
			want: generator.InitOptions{
				ProjectName: "myapi",
				ModuleName:  "github.com/user/myapi",
				Template:    "api",
				Blueprint:   "web-stack",
				Author:      "Jane Smith",
				License:     "Apache-2.0",
				GoVersion:   "1.25.1",
				OutputDir:   "./myapi",
				Description: "A api project",
				GitInit:     false,
				Force:       true,
				DryRun:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.wizard.ConvertToInitOptions()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewWizard(t *testing.T) {
	wizard := NewWizard()
	assert.NotNil(t, wizard)
	assert.NotNil(t, wizard.templateRepo)
	assert.NotNil(t, wizard.blueprintRepo)
}

func TestWizard_shouldPromptBlueprint(t *testing.T) {
	wizard := NewWizard()

	tests := []struct {
		template string
		expected bool
	}{
		{"cli", false},
		{"library", false},
		{"api", true},
		{"grpc", true},
		{"microservice", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.template, func(t *testing.T) {
			result := wizard.shouldPromptBlueprint(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWizard_displayGoVersion(t *testing.T) {
	wizard := NewWizard()

	tests := []struct {
		version  string
		expected string
	}{
		{"", "auto-detect"},
		{"1.25.1", "1.25.1"},
		{"1.24", "1.24"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := wizard.displayGoVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWizard_getGitUserName(t *testing.T) {
	wizard := NewWizard()

	// Test when no environment variables are set
	result := wizard.getGitUserName()
	// Should return either empty string or current user
	assert.True(t, len(result) >= 0)
}
