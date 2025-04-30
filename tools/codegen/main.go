package main

import (
	"flag"
	"fmt"
	"log"
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

// CodeGenerator handles code generation for services
type CodeGenerator struct {
	outputDir string
	templates map[string]string
}

// NewCodeGenerator creates a new code generator instance
func NewCodeGenerator(outputDir string) *CodeGenerator {
	return &CodeGenerator{
		outputDir: outputDir,
		templates: map[string]string{
			"service":    serviceTemplate,
			"repository": repositoryTemplate,
			"handler":    handlerTemplate,
		},
	}
}

// GenerateService generates all the files for a service
func (g *CodeGenerator) GenerateService(config ServiceConfig) error {
	// Create service directory
	serviceDir := filepath.Join(g.outputDir, strings.ToLower(config.ServiceName))
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Generate each component
	components := map[string]string{
		"service.go":    "service",
		"repository.go": "repository",
		"handler.go":    "handler",
	}

	for filename, templateName := range components {
		if err := g.generateFile(
			filepath.Join(serviceDir, filename),
			g.templates[templateName],
			config,
		); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filename, err)
		}
	}

	return nil
}

// generateFile generates a single file using the provided template and config
func (g *CodeGenerator) generateFile(path, tmpl string, config ServiceConfig) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Printf("error closing file: %v\n", cerr)
		}
	}()

	if err := t.Execute(f, config); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return nil
}

func main() {
	// Parse command line flags
	outputDir := flag.String("output", "internal/service", "Output directory for generated code")
	serviceName := flag.String("service", "", "Name of the service to generate")
	packageName := flag.String("package", "", "Package name for the generated code")
	flag.Parse()

	if *serviceName == "" {
		log.Fatal("Service name is required")
	}

	if *packageName == "" {
		*packageName = strings.ToLower(*serviceName)
	}

	// Create absolute path for output directory
	outAbs, err := filepath.Abs(*outputDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for output directory: %v", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outAbs, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Example method configurations
	methods := []MethodConfig{
		{
			Name:    "Create",
			Request: "CreateRequest",
			Reply:   "CreateResponse",
		},
		{
			Name:    "Get",
			Request: "GetRequest",
			Reply:   "GetResponse",
		},
		{
			Name:    "Update",
			Request: "UpdateRequest",
			Reply:   "UpdateResponse",
		},
		{
			Name:    "Delete",
			Request: "DeleteRequest",
			Reply:   "DeleteResponse",
		},
		{
			Name:    "List",
			Request: "ListRequest",
			Reply:   "ListResponse",
		},
	}

	// Create service configuration
	config := ServiceConfig{
		PackageName: *packageName,
		ServiceName: *serviceName,
		Methods:     methods,
	}

	// Create code generator
	generator := NewCodeGenerator(outAbs)

	// Generate service code
	if err := generator.GenerateService(config); err != nil {
		log.Fatalf("Failed to generate service code: %v", err)
	}

	log.Printf("Service code generated successfully in %s", outAbs)
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

// {{.ServiceName}}ServiceImpl implements {{.ServiceName}}Service
type {{.ServiceName}}ServiceImpl struct {
	repo {{.ServiceName}}Repository
}

// New{{.ServiceName}}Service creates a new {{.ServiceName}} service
func New{{.ServiceName}}Service(repo {{.ServiceName}}Repository) {{.ServiceName}}Service {
	return &{{.ServiceName}}ServiceImpl{
		repo: repo,
	}
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
	"errors"
)

// {{.ServiceName}}Repository defines the interface for {{.ServiceName}} data operations
type {{.ServiceName}}Repository interface {
{{range .Methods}}
	{{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error)
{{end}}
}

// {{.ServiceName}}RepositoryImpl implements {{.ServiceName}}Repository
type {{.ServiceName}}RepositoryImpl struct {
	// Add database connection or other dependencies here
}

// New{{.ServiceName}}Repository creates a new {{.ServiceName}} repository
func New{{.ServiceName}}Repository() {{.ServiceName}}Repository {
	return &{{.ServiceName}}RepositoryImpl{}
}

{{range .Methods}}
func (r *{{$.ServiceName}}RepositoryImpl) {{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error) {
	// TODO: Implement {{.Name}} method
	return nil, errors.New("not implemented")
}
{{end}}
`

const handlerTemplate = `package {{.PackageName}}

import (
	"context"
	"net/http"
)

// {{.ServiceName}}Handler handles HTTP requests for {{.ServiceName}} operations
type {{.ServiceName}}Handler struct {
	service {{.ServiceName}}Service
}

// New{{.ServiceName}}Handler creates a new {{.ServiceName}} handler
func New{{.ServiceName}}Handler(service {{.ServiceName}}Service) *{{.ServiceName}}Handler {
	return &{{.ServiceName}}Handler{
		service: service,
	}
}

{{range .Methods}}
func (h *{{$.ServiceName}}Handler) {{.Name}}(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &{{.Request}}{}
	
	// TODO: Parse request body into req
	
	resp, err := h.service.{{.Name}}(ctx, req)
	if err != nil {
		// TODO: Handle error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// TODO: Write response
}
{{end}}
`
