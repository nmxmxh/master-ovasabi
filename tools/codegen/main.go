package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type ServiceConfig struct {
	PackageName string
	ServiceName string
	Methods     []MethodConfig
}

type MethodConfig struct {
	Name    string
	Request string
	Reply   string
}

func main() {
	var (
		packageName = flag.String("package", "", "Package name for generated code")
		serviceName = flag.String("service", "", "Service name")
		outputDir   = flag.String("output", ".", "Output directory")
	)
	flag.Parse()

	if *packageName == "" || *serviceName == "" {
		fmt.Println("Usage: codegen -package <package> -service <service> [-output <dir>]")
		os.Exit(1)
	}

	config := ServiceConfig{
		PackageName: *packageName,
		ServiceName: *serviceName,
		Methods: []MethodConfig{
			{Name: "Create", Request: "CreateRequest", Reply: "CreateResponse"},
			{Name: "Get", Request: "GetRequest", Reply: "GetResponse"},
			{Name: "List", Request: "ListRequest", Reply: "ListResponse"},
			{Name: "Update", Request: "UpdateRequest", Reply: "UpdateResponse"},
			{Name: "Delete", Request: "DeleteRequest", Reply: "DeleteResponse"},
		},
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate service file
	generateFile(filepath.Join(*outputDir, strings.ToLower(*serviceName)+"_service.go"), serviceTemplate, config)
	// Generate repository file
	generateFile(filepath.Join(*outputDir, strings.ToLower(*serviceName)+"_repository.go"), repositoryTemplate, config)
	// Generate handler file
	generateFile(filepath.Join(*outputDir, strings.ToLower(*serviceName)+"_handler.go"), handlerTemplate, config)
}

func generateFile(path string, tmpl string, config ServiceConfig) {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("Failed to close file: %v\n", err)
		}
	}()

	if err := t.Execute(f, config); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}
}

const serviceTemplate = `package {{.PackageName}}

import (
	"context"
	"errors"
)

// {{.ServiceName}}Service defines the interface for {{.ServiceName}} operations
type {{.ServiceName}}Service interface {
{{range .Methods}}
	{{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error)
{{end}}
}

// New{{.ServiceName}}Service creates a new {{.ServiceName}} service
func New{{.ServiceName}}Service(repo {{.ServiceName}}Repository) {{.ServiceName}}Service {
	return &{{.ServiceName}}ServiceImpl{
		repo: repo,
	}
}

// {{.ServiceName}}ServiceImpl implements {{.ServiceName}}Service
type {{.ServiceName}}ServiceImpl struct {
	repo {{.ServiceName}}Repository
}

{{range .Methods}}
func (s *{{$.ServiceName}}ServiceImpl) {{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error) {
	// TODO: Implement {{.Name}} method
	return nil, errors.New("not implemented")
}
{{end}}
`

const repositoryTemplate = `package {{.PackageName}}

import (
	"context"
	"database/sql"
)

// {{.ServiceName}}Repository defines the interface for {{.ServiceName}} data access
type {{.ServiceName}}Repository interface {
{{range .Methods}}
	{{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error)
{{end}}
}

// New{{.ServiceName}}Repository creates a new {{.ServiceName}} repository
func New{{.ServiceName}}Repository(db *sql.DB) {{.ServiceName}}Repository {
	return &{{.ServiceName}}RepositoryImpl{
		db: db,
	}
}

// {{.ServiceName}}RepositoryImpl implements {{.ServiceName}}Repository
type {{.ServiceName}}RepositoryImpl struct {
	db *sql.DB
}

{{range .Methods}}
func (r *{{$.ServiceName}}RepositoryImpl) {{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error) {
	// TODO: Implement {{.Name}} method
	return nil, nil
}
{{end}}
`

const handlerTemplate = `package {{.PackageName}}

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// {{.ServiceName}}Handler handles HTTP requests for {{.ServiceName}}
type {{.ServiceName}}Handler struct {
	service {{.ServiceName}}Service
}

// New{{.ServiceName}}Handler creates a new {{.ServiceName}} handler
func New{{.ServiceName}}Handler(service {{.ServiceName}}Service) *{{.ServiceName}}Handler {
	return &{{.ServiceName}}Handler{
		service: service,
	}
}

// RegisterRoutes registers the routes for {{.ServiceName}}
func (h *{{.ServiceName}}Handler) RegisterRoutes(r *gin.Engine) {
	group := r.Group("/api/{{.PackageName}}")
{{range .Methods}}
	group.{{.Name}}("/{{.Name | ToLower}}", h.handle{{.Name}})
{{end}}
}

{{range .Methods}}
func (h *{{$.ServiceName}}Handler) handle{{.Name}}(c *gin.Context) {
	var req {{.Request}}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.{{.Name}}(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
{{end}}
`
