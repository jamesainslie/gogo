package components

// ComponentTemplate represents a template for generating components
type ComponentTemplate struct {
	Name    string
	Path    string
	Content string
}

// getComponentTemplates returns all component templates organized by type
func getComponentTemplates() map[string][]ComponentTemplate {
	templates := make(map[string][]ComponentTemplate)

	// Handler templates
	templates["handler"] = []ComponentTemplate{
		{
			Name: "handler",
			Path: "internal/handlers/{{ SnakeName }}_handler.go",
			Content: `package handlers

import (
	"net/http"
{% if IsGin %}
	"github.com/gin-gonic/gin"
{% elif IsEcho %}
	"github.com/labstack/echo/v4"
{% elif IsChi %}
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
{% endif %}
{% if ModuleName %}
	"{{ ModuleName }}/internal/models"
	"{{ ModuleName }}/internal/services"
{% endif %}
)

// {{ TitleName }}Handler handles {{ TitleName }} related requests
type {{ TitleName }}Handler struct {
	service services.{{ TitleName }}Service
}

// New{{ TitleName }}Handler creates a new {{ TitleName }} handler
func New{{ TitleName }}Handler(service services.{{ TitleName }}Service) *{{ TitleName }}Handler {
	return &{{ TitleName }}Handler{
		service: service,
	}
}

{% if IsGin %}
// Get{{ TitleName }}s handles GET /{{ KebabName }}s
func (h *{{ TitleName }}Handler) Get{{ TitleName }}s(c *gin.Context) {
	{{ CamelName }}s, err := h.service.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": {{ CamelName }}s})
}

// Get{{ TitleName }} handles GET /{{ KebabName }}s/:id
func (h *{{ TitleName }}Handler) Get{{ TitleName }}(c *gin.Context) {
	id := c.Param("id")
	
	{{ CamelName }}, err := h.service.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "{{ TitleName }} not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": {{ CamelName }}})
}

// Create{{ TitleName }} handles POST /{{ KebabName }}s
func (h *{{ TitleName }}Handler) Create{{ TitleName }}(c *gin.Context) {
	var req models.Create{{ TitleName }}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	{{ CamelName }}, err := h.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"data": {{ CamelName }}})
}

// Update{{ TitleName }} handles PUT /{{ KebabName }}s/:id
func (h *{{ TitleName }}Handler) Update{{ TitleName }}(c *gin.Context) {
	id := c.Param("id")
	
	var req models.Update{{ TitleName }}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	{{ CamelName }}, err := h.service.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": {{ CamelName }}})
}

// Delete{{ TitleName }} handles DELETE /{{ KebabName }}s/:id
func (h *{{ TitleName }}Handler) Delete{{ TitleName }}(c *gin.Context) {
	id := c.Param("id")
	
	if err := h.service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusNoContent, nil)
}

// RegisterRoutes registers all {{ TitleName }} routes
func (h *{{ TitleName }}Handler) RegisterRoutes(r *gin.Engine) {
	{{ CamelName }}Group := r.Group("/api/v1/{{ KebabName }}s")
	{
		{{ CamelName }}Group.GET("", h.Get{{ TitleName }}s)
		{{ CamelName }}Group.GET("/:id", h.Get{{ TitleName }})
		{{ CamelName }}Group.POST("", h.Create{{ TitleName }})
		{{ CamelName }}Group.PUT("/:id", h.Update{{ TitleName }})
		{{ CamelName }}Group.DELETE("/:id", h.Delete{{ TitleName }})
	}
}
{% endif %}`,
		},
		{
			Name: "handler_test",
			Path: "internal/handlers/{{ SnakeName }}_handler_test.go",
			Content: `package handlers

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
{% if ModuleName %}
	"{{ ModuleName }}/internal/services"
{% endif %}
)

// Mock{{ TitleName }}Service is a mock implementation of {{ TitleName }}Service
type Mock{{ TitleName }}Service struct {
	mock.Mock
}

func (m *Mock{{ TitleName }}Service) GetAll() ([]interface{}, error) {
	args := m.Called()
	return args.Get(0).([]interface{}), args.Error(1)
}

func (m *Mock{{ TitleName }}Service) GetByID(id string) (interface{}, error) {
	args := m.Called(id)
	return args.Get(0), args.Error(1)
}

func (m *Mock{{ TitleName }}Service) Create(req interface{}) (interface{}, error) {
	args := m.Called(req)
	return args.Get(0), args.Error(1)
}

func (m *Mock{{ TitleName }}Service) Update(id string, req interface{}) (interface{}, error) {
	args := m.Called(id, req)
	return args.Get(0), args.Error(1)
}

func (m *Mock{{ TitleName }}Service) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestNew{{ TitleName }}Handler(t *testing.T) {
	mockService := &Mock{{ TitleName }}Service{}
	handler := New{{ TitleName }}Handler(mockService)
	
	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

// TODO: Add more comprehensive tests for each handler method`,
		},
	}

	// Model templates
	templates["model"] = []ComponentTemplate{
		{
			Name: "model",
			Path: "internal/models/{{ SnakeName }}.go",
			Content: `package models

import (
	"time"
{% if IsGorm %}
	"gorm.io/gorm"
{% endif %}
)

// {{ TitleName }} represents a {{ TitleName }} entity
type {{ TitleName }} struct {
{% if IsGorm %}
	ID        uint           ` + "`gorm:\"primarykey\" json:\"id\"`" + `
	CreatedAt time.Time      ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time      ` + "`json:\"updated_at\"`" + `
	DeletedAt gorm.DeletedAt ` + "`gorm:\"index\" json:\"deleted_at,omitempty\"`" + `
{% else %}
	ID        string    ` + "`json:\"id\" db:\"id\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\" db:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at\" db:\"updated_at\"`" + `
{% endif %}
	
	// Add your {{ TitleName }} fields here
	Name        string ` + "`json:\"name\"{% if IsGorm %} gorm:\"not null\"{% else %} db:\"name\"{% endif %}`" + `
	Description string ` + "`json:\"description\"{% if IsGorm %}{% else %} db:\"description\"{% endif %}`" + `
}

// Create{{ TitleName }}Request represents a request to create a {{ TitleName }}
type Create{{ TitleName }}Request struct {
	Name        string ` + "`json:\"name\" binding:\"required\"`" + `
	Description string ` + "`json:\"description\"`" + `
}

// Update{{ TitleName }}Request represents a request to update a {{ TitleName }}
type Update{{ TitleName }}Request struct {
	Name        string ` + "`json:\"name\"`" + `
	Description string ` + "`json:\"description\"`" + `
}

{% if IsGorm %}
// TableName returns the table name for {{ TitleName }}
func ({{ TitleName }}) TableName() string {
	return "{{ SnakeName }}s"
}
{% endif %}`,
		},
		{
			Name: "model_test",
			Path: "internal/models/{{ SnakeName }}_test.go",
			Content: `package models

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func Test{{ TitleName }}_TableName(t *testing.T) {
{% if IsGorm %}
	{{ CamelName }} := {{ TitleName }}{}
	assert.Equal(t, "{{ SnakeName }}s", {{ CamelName }}.TableName())
{% else %}
	// Add tests for your {{ TitleName }} model
	assert.True(t, true) // Placeholder test
{% endif %}
}

func TestCreate{{ TitleName }}Request_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request Create{{ TitleName }}Request
		valid   bool
	}{
		{
			name: "valid request",
			request: Create{{ TitleName }}Request{
				Name:        "Test {{ TitleName }}",
				Description: "Test description",
			},
			valid: true,
		},
		{
			name: "empty name",
			request: Create{{ TitleName }}Request{
				Name:        "",
				Description: "Test description",
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add validation logic here
			assert.NotEmpty(t, tt.request.Name == "" && !tt.valid)
		})
	}
}`,
		},
	}

	// Service templates
	templates["service"] = []ComponentTemplate{
		{
			Name: "service",
			Path: "internal/services/{{ SnakeName }}_service.go",
			Content: `package services

import (
	"fmt"
{% if ModuleName %}
	"{{ ModuleName }}/internal/models"
{% endif %}
)

// {{ TitleName }}Service defines the interface for {{ TitleName }} operations
type {{ TitleName }}Service interface {
	GetAll() ([]*models.{{ TitleName }}, error)
	GetByID(id string) (*models.{{ TitleName }}, error)
	Create(req *models.Create{{ TitleName }}Request) (*models.{{ TitleName }}, error)
	Update(id string, req *models.Update{{ TitleName }}Request) (*models.{{ TitleName }}, error)
	Delete(id string) error
}

// {{ CamelName }}Service implements {{ TitleName }}Service
type {{ CamelName }}Service struct {
	// Add your dependencies here (repository, database, etc.)
}

// New{{ TitleName }}Service creates a new {{ TitleName }} service
func New{{ TitleName }}Service() {{ TitleName }}Service {
	return &{{ CamelName }}Service{}
}

// GetAll retrieves all {{ TitleName }}s
func (s *{{ CamelName }}Service) GetAll() ([]*models.{{ TitleName }}, error) {
	// TODO: Implement GetAll logic
	return nil, fmt.Errorf("not implemented")
}

// GetByID retrieves a {{ TitleName }} by ID
func (s *{{ CamelName }}Service) GetByID(id string) (*models.{{ TitleName }}, error) {
	// TODO: Implement GetByID logic
	return nil, fmt.Errorf("not implemented")
}

// Create creates a new {{ TitleName }}
func (s *{{ CamelName }}Service) Create(req *models.Create{{ TitleName }}Request) (*models.{{ TitleName }}, error) {
	// TODO: Implement Create logic
	return nil, fmt.Errorf("not implemented")
}

// Update updates an existing {{ TitleName }}
func (s *{{ CamelName }}Service) Update(id string, req *models.Update{{ TitleName }}Request) (*models.{{ TitleName }}, error) {
	// TODO: Implement Update logic
	return nil, fmt.Errorf("not implemented")
}

// Delete deletes a {{ TitleName }}
func (s *{{ CamelName }}Service) Delete(id string) error {
	// TODO: Implement Delete logic
	return fmt.Errorf("not implemented")
}`,
		},
		{
			Name: "service_test",
			Path: "internal/services/{{ SnakeName }}_service_test.go",
			Content: `package services

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
{% if ModuleName %}
	"{{ ModuleName }}/internal/models"
{% endif %}
)

func TestNew{{ TitleName }}Service(t *testing.T) {
	service := New{{ TitleName }}Service()
	assert.NotNil(t, service)
}

func Test{{ TitleName }}Service_GetAll(t *testing.T) {
	service := New{{ TitleName }}Service()
	
	{{ CamelName }}s, err := service.GetAll()
	
	// Since this is not implemented yet, we expect an error
	assert.Error(t, err)
	assert.Nil(t, {{ CamelName }}s)
	assert.Contains(t, err.Error(), "not implemented")
}

func Test{{ TitleName }}Service_GetByID(t *testing.T) {
	service := New{{ TitleName }}Service()
	
	{{ CamelName }}, err := service.GetByID("test-id")
	
	// Since this is not implemented yet, we expect an error
	assert.Error(t, err)
	assert.Nil(t, {{ CamelName }})
	assert.Contains(t, err.Error(), "not implemented")
}

func Test{{ TitleName }}Service_Create(t *testing.T) {
	service := New{{ TitleName }}Service()
	req := &models.Create{{ TitleName }}Request{
		Name:        "Test {{ TitleName }}",
		Description: "Test description",
	}
	
	{{ CamelName }}, err := service.Create(req)
	
	// Since this is not implemented yet, we expect an error
	assert.Error(t, err)
	assert.Nil(t, {{ CamelName }})
	assert.Contains(t, err.Error(), "not implemented")
}`,
		},
	}

	// Migration templates
	templates["migration"] = []ComponentTemplate{
		{
			Name: "migration",
			Path: "migrations/001_{{ SnakeName }}.sql",
			Content: `-- Migration: {{ Name }}
-- Created: {{ Year }}

-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS {{ SnakeName }}s (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_{{ SnakeName }}s_name ON {{ SnakeName }}s(name);
CREATE INDEX IF NOT EXISTS idx_{{ SnakeName }}s_deleted_at ON {{ SnakeName }}s(deleted_at);

-- +goose Down  
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS {{ SnakeName }}s;`,
		},
	}

	// Middleware templates
	templates["middleware"] = []ComponentTemplate{
		{
			Name: "middleware",
			Path: "internal/middleware/{{ SnakeName }}_middleware.go",
			Content: `package middleware

import (
{% if IsGin %}
	"github.com/gin-gonic/gin"
{% elif IsEcho %}
	"github.com/labstack/echo/v4"
{% elif IsChi %}
	"net/http"
{% endif %}
)

{% if IsGin %}
// {{ TitleName }}Middleware creates a new {{ TitleName }} middleware
func {{ TitleName }}Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement {{ TitleName }} middleware logic
		
		c.Next()
	}
}
{% elif IsEcho %}
// {{ TitleName }}Middleware creates a new {{ TitleName }} middleware
func {{ TitleName }}Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Implement {{ TitleName }} middleware logic
			
			return next(c)
		}
	}
}
{% elif IsChi %}
// {{ TitleName }}Middleware creates a new {{ TitleName }} middleware
func {{ TitleName }}Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement {{ TitleName }} middleware logic
		
		next.ServeHTTP(w, r)
	})
}
{% endif %}`,
		},
	}

	// Test templates
	templates["test"] = []ComponentTemplate{
		{
			Name: "test",
			Path: "internal/{{ SnakeName }}/{{ SnakeName }}_test.go",
			Content: `package {{ SnakeName }}

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test{{ TitleName }}(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "valid case",
			input:    "test input",
			expected: "test output",
			wantErr:  false,
		},
		{
			name:     "error case",
			input:    nil,
			expected: nil,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Implement your test logic here
			
			if tt.wantErr {
				assert.Error(t, nil) // Replace with actual error check
			} else {
				assert.NoError(t, nil) // Replace with actual success check
				assert.Equal(t, tt.expected, tt.input) // Replace with actual assertion
			}
		})
	}
}

func TestIntegration{{ TitleName }}(t *testing.T) {
	// Integration tests
	t.Skip("Integration test not implemented")
}

func BenchmarkTest{{ TitleName }}(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// TODO: Add benchmark logic
	}
}`,
		},
	}

	return templates
}
