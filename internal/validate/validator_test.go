package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateModuleName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid simple module name",
			input:   "mymodule",
			wantErr: false,
		},
		{
			name:    "valid github module name",
			input:   "github.com/user/repo",
			wantErr: false,
		},
		{
			name:    "valid module with subdirs",
			input:   "github.com/user/repo/subdir",
			wantErr: false,
		},
		{
			name:    "empty module name",
			input:   "",
			wantErr: true,
		},
		{
			name:    "module with consecutive dots",
			input:   "github.com/user..repo",
			wantErr: true,
		},
		{
			name:    "module starting with dot",
			input:   ".github.com/user/repo",
			wantErr: true,
		},
		{
			name:    "module ending with dot",
			input:   "github.com/user/repo.",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModuleName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid simple project name",
			input:   "myproject",
			wantErr: false,
		},
		{
			name:    "valid project name with hyphens",
			input:   "my-project",
			wantErr: false,
		},
		{
			name:    "valid project name with underscores",
			input:   "my_project",
			wantErr: false,
		},
		{
			name:    "valid project name with numbers",
			input:   "project123",
			wantErr: false,
		},
		{
			name:    "empty project name",
			input:   "",
			wantErr: true,
		},
		{
			name:    "project name starting with number",
			input:   "123project",
			wantErr: true,
		},
		{
			name:    "project name starting with hyphen",
			input:   "-project",
			wantErr: true,
		},
		{
			name:    "project name with spaces",
			input:   "my project",
			wantErr: true,
		},
		{
			name:    "project name with special chars",
			input:   "my@project",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGoVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty version (auto-detect)",
			input:   "",
			wantErr: false,
		},
		{
			name:    "valid major.minor version",
			input:   "1.21",
			wantErr: false,
		},
		{
			name:    "valid major.minor.patch version",
			input:   "1.21.5",
			wantErr: false,
		},
		{
			name:    "valid version 1.25.1",
			input:   "1.25.1",
			wantErr: false,
		},
		{
			name:    "invalid version without major",
			input:   "21",
			wantErr: true,
		},
		{
			name:    "invalid version format",
			input:   "v1.21",
			wantErr: true,
		},
		{
			name:    "invalid version with letters",
			input:   "1.21a",
			wantErr: true,
		},
		{
			name:    "invalid version with too many parts",
			input:   "1.21.5.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGoVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
