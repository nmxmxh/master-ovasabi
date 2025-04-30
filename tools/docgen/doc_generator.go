package docgen

import (
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// DocGenerator generates markdown documentation from code comments
type DocGenerator struct {
	sourceDir string
	outputDir string
}

// NewDocGenerator creates a new documentation generator
func NewDocGenerator(sourceDir, outputDir string) *DocGenerator {
	return &DocGenerator{
		sourceDir: sourceDir,
		outputDir: outputDir,
	}
}

// Generate generates documentation for all packages in the source directory
func (g *DocGenerator) Generate() error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk through source directory
	return filepath.Walk(g.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-directories
		if !info.IsDir() {
			return nil
		}

		// Skip vendor and hidden directories
		if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Generate documentation for package
		if err := g.generatePackageDoc(path); err != nil {
			return fmt.Errorf("failed to generate documentation for %s: %w", path, err)
		}

		return nil
	})
}

// generatePackageDoc generates documentation for a single package
func (g *DocGenerator) generatePackageDoc(pkgPath string) error {
	// Create file set
	fset := token.NewFileSet()

	// Parse package
	pkgs, err := parser.ParseDir(fset, pkgPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	for pkgName, pkg := range pkgs {
		// Skip test packages
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		// Create package documentation
		docPkg := doc.New(pkg, pkgPath, 0)

		// Generate markdown
		content := g.generateMarkdown(docPkg)

		// Create output file
		relPath, err := filepath.Rel(g.sourceDir, pkgPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		outputPath := filepath.Join(g.outputDir, relPath, "README.md")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write documentation: %w", err)
		}
	}

	return nil
}

// generateMarkdown generates markdown documentation for a package
func (g *DocGenerator) generateMarkdown(pkg *doc.Package) string {
	var sb strings.Builder

	// Package documentation
	sb.WriteString(fmt.Sprintf("# Package %s\n\n", pkg.Name))
	if pkg.Doc != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", pkg.Doc))
	}

	// Constants
	if len(pkg.Consts) > 0 {
		sb.WriteString("## Constants\n\n")
		for _, c := range pkg.Consts {
			sb.WriteString(fmt.Sprintf("### %s\n\n", c.Names[0]))
			if c.Doc != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", c.Doc))
			}
		}
	}

	// Variables
	if len(pkg.Vars) > 0 {
		sb.WriteString("## Variables\n\n")
		for _, v := range pkg.Vars {
			sb.WriteString(fmt.Sprintf("### %s\n\n", v.Names[0]))
			if v.Doc != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", v.Doc))
			}
		}
	}

	// Types
	if len(pkg.Types) > 0 {
		sb.WriteString("## Types\n\n")
		for _, t := range pkg.Types {
			sb.WriteString(fmt.Sprintf("### %s\n\n", t.Name))
			if t.Doc != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", t.Doc))
			}

			// Type methods
			if len(t.Methods) > 0 {
				sb.WriteString("#### Methods\n\n")
				for _, m := range t.Methods {
					sb.WriteString(fmt.Sprintf("##### %s\n\n", m.Name))
					if m.Doc != "" {
						sb.WriteString(fmt.Sprintf("%s\n\n", m.Doc))
					}
				}
			}
		}
	}

	// Functions
	if len(pkg.Funcs) > 0 {
		sb.WriteString("## Functions\n\n")
		for _, f := range pkg.Funcs {
			sb.WriteString(fmt.Sprintf("### %s\n\n", f.Name))
			if f.Doc != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", f.Doc))
			}
		}
	}

	return sb.String()
}
