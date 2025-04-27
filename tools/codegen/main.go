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

func generateFile(path, tmpl string, config ServiceConfig) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer f.Close()

	if err := t.Execute(f, config); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return nil
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
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate files
	files := []struct {
		path string
		tmpl string
	}{
		{filepath.Join(*outputDir, strings.ToLower(*serviceName)+"_service.go"), serviceTemplate},
		{filepath.Join(*outputDir, strings.ToLower(*serviceName)+"_repository.go"), repositoryTemplate},
		{filepath.Join(*outputDir, strings.ToLower(*serviceName)+"_handler.go"), handlerTemplate},
	}

	for _, file := range files {
		if err := generateFile(file.path, file.tmpl, config); err != nil {
			fmt.Printf("Error generating %s: %v\n", file.path, err)
			os.Exit(1)
		}
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
