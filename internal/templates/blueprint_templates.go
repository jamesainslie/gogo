package templates

// BlueprintTemplateFile represents a template file that uses blueprint variables
type BlueprintTemplateFile struct {
	Name     string
	Path     string
	Content  string
	Requires []string // Required blueprint features/components
}

// GetBlueprintTemplates returns blueprint-aware template files for different stacks
func GetBlueprintTemplates() map[string][]BlueprintTemplateFile {
	templates := make(map[string][]BlueprintTemplateFile)

	// Web stack templates
	templates["web"] = []BlueprintTemplateFile{
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
{% if HasDatabase %}
	"database/sql"
	_ "github.com/lib/pq"
{% endif %}
{% if "gin" in Components %}
	"github.com/gin-gonic/gin"
{% endif %}
{% if "viper" in Components %}
	"github.com/spf13/viper"
{% endif %}
{% if HasPrometheus %}
	"github.com/prometheus/client_golang/prometheus/promhttp"
{% endif %}
)

func main() {
{% if "viper" in Components %}
	// Load configuration
	viper.SetDefault("port", "8080")
	viper.SetDefault("host", "0.0.0.0")
	viper.AutomaticEnv()
{% endif %}

{% if HasDatabase %}
	// Database connection
	dbURL := viper.GetString("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost/{{ ProjectName }}?sslmode=disable"
	}
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
{% endif %}

{% if "gin" in Components %}
	// Setup Gin router
	r := gin.Default()
	
	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "{{ ProjectName }}"})
	})
	
{% if HasPrometheus %}
	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
{% endif %}

	// API routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	// Server configuration
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", viper.GetString("host"), viper.GetString("port")),
		Handler: r,
	}
{% else %}
	// Basic HTTP server
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, ` + "`" + `{"status":"ok","service":"{{ ProjectName }}"}` + "`" + `)
	})
	
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
{% endif %}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()
	
	fmt.Printf("Starting {{ ProjectName }} on %s\n", srv.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("Server failed:", err)
	}
}`,
			Requires: []string{},
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}

require (
{% if "gin" in Components %}
	github.com/gin-gonic/gin v1.9.1
{% endif %}
{% if "viper" in Components %}
	github.com/spf13/viper v1.16.0
{% endif %}
{% if HasDatabase %}
	github.com/lib/pq v1.10.9
{% endif %}
{% if "gorm" in Components %}
	gorm.io/gorm v1.25.4
	gorm.io/driver/postgres v1.5.2
{% endif %}
{% if HasPrometheus %}
	github.com/prometheus/client_golang v1.16.0
{% endif %}
)`,
			Requires: []string{},
		},
		{
			Name: "Dockerfile",
			Path: "Dockerfile",
			Content: `# Build stage
FROM {{ DockerBaseImage }}-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o {{ ProjectName }} ./cmd/{{ ProjectName }}

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/{{ ProjectName }} .

{% if HasDocker and "expose" in DockerBaseImage %}
EXPOSE {{ DockerBaseImage.expose }}
{% else %}
EXPOSE 8080
{% endif %}

{% if HasDocker and "health_check" in DockerBaseImage %}
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1
{% endif %}

CMD ["./{{ ProjectName }}"]`,
			Requires: []string{"HasDocker"},
		},
		{
			Name: "docker-compose.yml",
			Path: "docker-compose.yml",
			Content: `version: '3.8'

services:
  {{ ProjectName }}:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
{% if HasDatabase %}
      - DATABASE_URL=postgres://postgres:password@db:5432/{{ ProjectName }}?sslmode=disable
    depends_on:
      - db
{% endif %}
{% if HasTracing %}
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    depends_on:
      - jaeger
{% endif %}

{% if HasDatabase %}
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: {{ ProjectName }}
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
{% endif %}

{% if HasTracing %}
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"
      - "14268:14268"
{% endif %}

{% if HasDatabase %}
volumes:
  postgres_data:
{% endif %}`,
			Requires: []string{},
		},
	}

	// CLI stack templates
	templates["cli"] = []BlueprintTemplateFile{
		{
			Name: "main.go",
			Path: "cmd/{{ ProjectName }}/main.go",
			Content: `package main

import (
	"os"
	
{% if "cobra" in Components %}
	"{{ ModuleName }}/internal/cmd"
{% else %}
	"fmt"
{% endif %}
)

func main() {
{% if "cobra" in Components %}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
{% else %}
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <command>\n", os.Args[0])
		os.Exit(1)
	}
	
	fmt.Printf("{{ ProjectName }} - A CLI application by {{ Author }}\n")
	fmt.Printf("Command: %s\n", os.Args[1])
{% endif %}
}`,
			Requires: []string{},
		},
		{
			Name: "root.go",
			Path: "internal/cmd/root.go",
			Content: `package cmd

import (
	"fmt"
	"os"

{% if "cobra" in Components %}
	"github.com/spf13/cobra"
{% endif %}
{% if "viper" in Components %}
	"github.com/spf13/viper"
{% endif %}
)

{% if "cobra" in Components %}
var rootCmd = &cobra.Command{
	Use:   "{{ ProjectName }}",
	Short: "{{ Description }}",
	Long:  "{{ Description }}",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
{% if "viper" in Components %}
	cobra.OnInitialize(initConfig)
	
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.{{ ProjectName }}.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
{% endif %}
}

{% if "viper" in Components %}
var cfgFile string

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".{{ ProjectName }}")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
{% endif %}
{% endif %}`,
			Requires: []string{"cobra"},
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}

require (
{% if "cobra" in Components %}
	github.com/spf13/cobra v1.7.0
{% endif %}
{% if "viper" in Components %}
	github.com/spf13/viper v1.16.0
{% endif %}
)`,
			Requires: []string{},
		},
	}

	// gRPC stack templates
	templates["grpc"] = []BlueprintTemplateFile{
		{
			Name: "main.go",
			Path: "cmd/{{ ProjectName }}/main.go",
			Content: `package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
{% if HasTracing %}
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
{% endif %}
	
	"{{ ModuleName }}/internal/server"
)

func main() {
{% if HasTracing %}
	// Initialize Jaeger tracer
	cfg, err := config.FromEnv()
	if err != nil {
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
	}
	
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
{% endif %}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	
	// Register services
	server.RegisterServices(s)
	
	// Enable reflection for grpcurl
	reflection.Register(s)
	
	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Shutting down gRPC server...")
		s.GracefulStop()
	}()
	
	fmt.Println("{{ ProjectName }} gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}`,
			Requires: []string{},
		},
		{
			Name: "server.go",
			Path: "internal/server/server.go",
			Content: `package server

import (
	"google.golang.org/grpc"
	
	// Import your generated protobuf packages here
	// pb "{{ ModuleName }}/proto/{{ ProjectName }}"
)

// RegisterServices registers all gRPC services
func RegisterServices(s *grpc.Server) {
	// Register your services here
	// pb.Register{{ ProjectName }}ServiceServer(s, &{{ ProjectName }}Server{})
}

// {{ ProjectName }}Server implements the gRPC service
type {{ ProjectName }}Server struct {
	// Add your dependencies here
}

// Implement your gRPC methods here`,
			Requires: []string{},
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}

require (
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
{% if HasTracing %}
	github.com/opentracing/opentracing-go v1.2.0
	github.com/uber/jaeger-client-go v2.30.0+incompatible
{% endif %}
)`,
			Requires: []string{},
		},
	}

	// Microservice stack templates
	templates["microservice"] = []BlueprintTemplateFile{
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
	
{% if "gin" in Components %}
	"github.com/gin-gonic/gin"
{% endif %}
{% if HasPrometheus %}
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
{% endif %}
{% if HasTracing %}
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
{% endif %}
)

{% if HasPrometheus %}
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "{{ ProjectName }}_requests_total",
			Help: "Total number of requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "{{ ProjectName }}_request_duration_seconds",
			Help: "Request duration in seconds",
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
}
{% endif %}

func main() {
{% if HasTracing %}
	// Initialize Jaeger tracer
	cfg, err := config.FromEnv()
	if err != nil {
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
	}
	
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
	}
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)
{% endif %}

{% if "gin" in Components %}
	r := gin.Default()
	
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "{{ ProjectName }}",
			"version": "1.0.0",
		})
	})
	
	// Readiness check
	r.GET("/ready", func(c *gin.Context) {
		// Add readiness checks here (database, dependencies, etc.)
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	
{% if HasPrometheus %}
	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
{% endif %}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
{% else %}
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, ` + "`" + `{"status":"ok","service":"{{ ProjectName }}","version":"1.0.0"}` + "`" + `)
	})
	
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
{% endif %}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()
	
	fmt.Printf("{{ ProjectName }} microservice starting on :8080\n")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("Server failed:", err)
	}
}`,
			Requires: []string{},
		},
		{
			Name: "go.mod",
			Path: "go.mod",
			Content: `module {{ ModuleName }}

go {{ GoVersion }}

require (
{% if "gin" in Components %}
	github.com/gin-gonic/gin v1.9.1
{% endif %}
{% if HasPrometheus %}
	github.com/prometheus/client_golang v1.16.0
{% endif %}
{% if HasTracing %}
	github.com/opentracing/opentracing-go v1.2.0
	github.com/uber/jaeger-client-go v2.30.0+incompatible
{% endif %}
)`,
			Requires: []string{},
		},
	}

	return templates
}
