package registration

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/nmxmxh/master-ovasabi/amadeus/pkg/kg"
	"github.com/nmxmxh/master-ovasabi/config/registry"
	"go.uber.org/zap"
)

// DynamicServiceRegistrationGenerator generates service registration configs
// by analyzing proto files, Go code, and service interfaces through reflection.
type DynamicServiceRegistrationGenerator struct {
	logger    *zap.Logger
	protoPath string
	srcPath   string
}

// NewDynamicServiceRegistrationGenerator creates a new generator instance.
func NewDynamicServiceRegistrationGenerator(logger *zap.Logger, protoPath, srcPath string) *DynamicServiceRegistrationGenerator {
	return &DynamicServiceRegistrationGenerator{
		logger:    logger,
		protoPath: protoPath,
		srcPath:   srcPath,
	}
}

// ServiceRegistrationConfig represents the structure of a service registration.
type ServiceRegistrationConfig struct {
	Name               string                  `json:"name"`
	Version            string                  `json:"version"`
	Capabilities       []string                `json:"capabilities"`
	Dependencies       []string                `json:"dependencies"`
	Schema             SchemaConfig            `json:"schema"`
	Endpoints          []EndpointConfig        `json:"endpoints"`
	Models             []string                `json:"models"`
	HealthCheck        string                  `json:"health_check"`
	Metrics            string                  `json:"metrics"`
	MetadataEnrichment bool                    `json:"metadata_enrichment"`
	ActionMap          map[string]ActionConfig `json:"action_map,omitempty"`
}

// SchemaConfig represents the schema configuration for a service.
type SchemaConfig struct {
	ProtoPath string   `json:"proto_path,omitempty"`
	Methods   []string `json:"methods,omitempty"`
}

// EndpointConfig represents an endpoint configuration.
type EndpointConfig struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Actions     []string `json:"actions"`
	Description string   `json:"description"`
}

// ActionConfig represents an action configuration.
type ActionConfig struct {
	ProtoMethod        string                 `json:"proto_method"`
	RequestModel       string                 `json:"request_model"`
	ResponseModel      string                 `json:"response_model"`
	RestRequiredFields []string               `json:"rest_required_fields,omitempty"`
	Fields             map[string]FieldConfig `json:"fields,omitempty"`
}

type FieldConfig struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// ProtoServiceInfo contains information extracted from proto files.
type ProtoServiceInfo struct {
	ServiceName string
	Methods     []ProtoMethodInfo
	Messages    []string
}

// ProtoMethodInfo contains information about a proto method.
type ProtoMethodInfo struct {
	Name        string
	InputType   string
	OutputType  string
	Description string
}

// GenerateServiceRegistrations generates service registration configs for all services.
func (g *DynamicServiceRegistrationGenerator) GenerateServiceRegistrations(ctx context.Context) ([]ServiceRegistrationConfig, error) {
	// Walk through proto files to discover services
	protoServices, err := g.discoverProtoServices()
	if err != nil {
		g.logger.Error("Failed to discover proto services", zap.Error(err))
		return nil, err
	}

	configs := make([]ServiceRegistrationConfig, 0, len(protoServices))
	for _, protoService := range protoServices {
		config := g.generateServiceConfig(ctx, protoService)
		configs = append(configs, config)
	}

	return configs, nil
}

// discoverProtoServices discovers all services from proto files.
func (g *DynamicServiceRegistrationGenerator) discoverProtoServices() ([]ProtoServiceInfo, error) {
	var services []ProtoServiceInfo

	err := filepath.WalkDir(g.protoPath, func(path string, d fs.DirEntry, err error) error {
		_ = d // Use d to avoid revive unused-parameter warning
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".proto") {
			return nil
		}

		// Skip proto files in the 'common' directory as they do not contain service definitions
		if strings.Contains(path, "api/protos/common/") {
			g.logger.Debug("Skipping proto file in common directory", zap.String("path", path))
			return nil
		}

		service, err := g.parseProtoFile(path)
		if err != nil {
			g.logger.Warn("Failed to parse proto file", zap.String("path", path), zap.Error(err))
			return nil
		}

		if service != nil {
			services = append(services, *service)
		}

		return nil
	})

	return services, err
}

// parseProtoFile parses a proto file to extract service information.
func (g *DynamicServiceRegistrationGenerator) parseProtoFile(path string) (*ProtoServiceInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	serviceInfo := &ProtoServiceInfo{
		Methods:  []ProtoMethodInfo{},
		Messages: []string{},
	}

	lines := strings.Split(string(content), "\n")
	var currentService string
	var inService bool

	var rpcBuffer string
	var rpcActive bool
	for idx := 0; idx < len(lines); idx++ {
		line := strings.TrimSpace(lines[idx])
		// Ignore comments and blank lines
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Extract service name
		if strings.HasPrefix(line, "service ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentService = strings.TrimSuffix(parts[1], " {")
				serviceInfo.ServiceName = currentService
				inService = true
				g.logger.Debug("Entered service block", zap.String("service", currentService), zap.Int("line", idx))
			}
			continue
		}

		// Extract methods within service block
		if inService {
			if strings.HasPrefix(line, "rpc ") {
				// Start collecting rpc definition
				rpcBuffer = line
				rpcActive = true
				// If 'returns' is on the same line, parse immediately
				if strings.Contains(line, "returns") {
					g.logger.Debug("Found rpc line", zap.String("service", currentService), zap.String("line", rpcBuffer), zap.Int("idx", idx))
					method := g.parseRPCMethod(rpcBuffer)
					if method != nil {
						serviceInfo.Methods = append(serviceInfo.Methods, *method)
						g.logger.Debug("Extracted rpc method", zap.String("service", currentService), zap.String("method", method.Name))
					}
					rpcBuffer = ""
					rpcActive = false
				}
				continue
			}
			if rpcActive {
				// Continue collecting rpc definition until ';' or '{' or '}'
				rpcBuffer += " " + line
				if strings.Contains(line, "returns") || strings.HasSuffix(line, ";") || strings.HasSuffix(line, "{") || strings.HasSuffix(line, "}") {
					g.logger.Debug("Found rpc line (multiline)", zap.String("service", currentService), zap.String("line", rpcBuffer), zap.Int("idx", idx))
					method := g.parseRPCMethod(rpcBuffer)
					if method != nil {
						serviceInfo.Methods = append(serviceInfo.Methods, *method)
						g.logger.Debug("Extracted rpc method", zap.String("service", currentService), zap.String("method", method.Name))
					}
					rpcBuffer = ""
					rpcActive = false
				}
				continue
			}
			// Only exit service block on a line with only '}'
			if line == "}" {
				inService = false
				g.logger.Debug("Exited service block", zap.String("service", currentService), zap.Int("line", idx))
			}
			continue
		}

		// Extract messages
		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				messageName := strings.TrimSuffix(parts[1], " {")
				serviceInfo.Messages = append(serviceInfo.Messages, messageName)
			}
		}
	}

	// Ensure that if we found a service, but no methods, we still return the serviceInfo
	// This is to prevent missing service blocks in output config
	if serviceInfo.ServiceName != "" && len(serviceInfo.Methods) == 0 {
		g.logger.Warn("Service block found but no methods extracted", zap.String("service", serviceInfo.ServiceName), zap.String("path", path))
	}

	if serviceInfo.ServiceName == "" {
		return nil, fmt.Errorf("no service block found in proto file: %s", path)
	}

	return serviceInfo, nil
}

// parseRPCMethod parses an RPC method line from proto file.
func (g *DynamicServiceRegistrationGenerator) parseRPCMethod(line string) *ProtoMethodInfo {
	// Support multi-line rpc definitions
	// Example: rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
	rpcRegex := regexp.MustCompile(`rpc\s+(\w+)\s*\(([^)]+)\)\s*returns\s*\(([^)]+)\)`)
	matches := rpcRegex.FindStringSubmatch(line)

	if len(matches) == 4 {
		return &ProtoMethodInfo{
			Name:       matches[1],
			InputType:  matches[2],
			OutputType: matches[3],
		}
	}

	// Fallback: try to parse rpc without returns (for incomplete/malformed lines)
	rpcRegexNoReturns := regexp.MustCompile(`rpc\s+(\w+)\s*\(([^)]+)\)`)
	matches = rpcRegexNoReturns.FindStringSubmatch(line)
	if len(matches) == 3 {
		return &ProtoMethodInfo{
			Name:       matches[1],
			InputType:  matches[2],
			OutputType: "", // Unknown output type
		}
	}
	return nil
}

// generateServiceConfig generates a service configuration.
func (g *DynamicServiceRegistrationGenerator) generateServiceConfig(ctx context.Context, protoService ProtoServiceInfo) ServiceRegistrationConfig {
	serviceName := g.normalizeServiceName(protoService.ServiceName)

	config := ServiceRegistrationConfig{
		Name:               serviceName,
		Version:            "v1",
		Capabilities:       g.inferCapabilities(protoService),
		Dependencies:       g.inferDependencies(protoService),
		Schema:             g.generateSchemaConfig(protoService),
		Endpoints:          g.generateEndpointConfigs(protoService),
		Models:             protoService.Messages,
		HealthCheck:        fmt.Sprintf("/health/%s", serviceName),
		Metrics:            fmt.Sprintf("/metrics/%s", serviceName),
		MetadataEnrichment: true,
		ActionMap:          g.generateActionMap(protoService),
	}

	// Try to enhance with code analysis
	g.enhanceWithCodeAnalysis(ctx, &config)

	return config
}

// normalizeServiceName normalizes service names from proto to match convention.
func (g *DynamicServiceRegistrationGenerator) normalizeServiceName(name string) string {
	// Convert from "UserService" to "user"
	name = strings.TrimSuffix(name, "Service")
	return strings.ToLower(name)
}

// inferCapabilities infers service capabilities from methods and context.
func (g *DynamicServiceRegistrationGenerator) inferCapabilities(service ProtoServiceInfo) []string {
	capabilities := []string{}
	methodNames := make(map[string]bool)

	for _, method := range service.Methods {
		methodName := strings.ToLower(method.Name)
		methodNames[methodName] = true
	}

	// Standard capability inference rules
	capabilityRules := map[string][]string{
		"user_mgmt":           {"createuser", "getuser", "updateuser", "deleteuser", "listuser"},
		"authentication":      {"authenticate", "login", "logout", "createsession", "revokesession"},
		"authorization":       {"authorize", "checkpermission", "assignrole", "revokerole"},
		"notification":        {"sendnotification", "sendemail", "sendsms", "sendpush"},
		"content":             {"createcontent", "getcontent", "updatecontent", "deletecontent"},
		"commerce":            {"createorder", "getorder", "createquote", "initiatepayment"},
		"analytics":           {"trackevent", "captureevent", "getreport", "listreports"},
		"moderation":          {"submitcontentformoderation", "approvecontent", "rejectcontent"},
		"search":              {"search", "suggest", "autocomplete"},
		"scheduler":           {"createjob", "runjob", "listjobs", "deletejob"},
		"media":               {"uploadmedia", "getmedia", "deletemedia", "streammedia"},
		"messaging":           {"sendmessage", "getmessage", "creatchat", "listmessages"},
		"referral":            {"createreferral", "getreferral", "getreferralstats"},
		"orchestration":       {"registerpattern", "orchestrate", "tracepattern"},
		"admin":               {"createuser", "updaterole", "getauditlogs", "getsettings"},
		"security":            {"validatetoken", "encrypt", "decrypt", "audit"},
		"metadata_enrichment": {"enrichmetadata", "updatemetadata", "normalizedata"},
	}

	for capability, keywords := range capabilityRules {
		for _, keyword := range keywords {
			if methodNames[keyword] {
				capabilities = append(capabilities, capability)
				break
			}
		}
	}

	// Always add metadata enrichment for services with metadata
	if len(capabilities) > 0 {
		capabilities = append(capabilities, "metadata_enrichment")
	}

	return capabilities
}

// inferDependencies infers service dependencies from method signatures and imports.
func (g *DynamicServiceRegistrationGenerator) inferDependencies(service ProtoServiceInfo) []string {
	dependencies := []string{}

	// Common dependency patterns
	dependencyPatterns := map[string][]string{
		"user":              {"user", "profile", "session", "auth"},
		"notification":      {"notification", "email", "sms", "push"},
		"security":          {"security", "auth", "permission", "role"},
		"content":           {"content", "article", "post", "comment"},
		"commerce":          {"order", "payment", "quote", "billing"},
		"analytics":         {"event", "track", "report", "analytics"},
		"search":            {"search", "index", "query", "suggest"},
		"localization":      {"locale", "translation", "i18n"},
		"contentmoderation": {"moderation", "approve", "reject", "flag"},
		"nexus":             {"orchestrate", "pattern", "workflow"},
	}

	serviceNameLower := strings.ToLower(service.ServiceName)
	for _, method := range service.Methods {
		methodLower := strings.ToLower(method.Name)
		inputLower := strings.ToLower(method.InputType)
		outputLower := strings.ToLower(method.OutputType)

		for dep, patterns := range dependencyPatterns {
			// Skip self-dependency
			if strings.Contains(serviceNameLower, dep) {
				continue
			}

			for _, pattern := range patterns {
				if strings.Contains(methodLower, pattern) ||
					strings.Contains(inputLower, pattern) ||
					strings.Contains(outputLower, pattern) {
					dependencies = append(dependencies, dep)
					break
				}
			}
		}
	}

	return g.removeDuplicates(dependencies)
}

// generateSchemaConfig generates schema configuration.
func (g *DynamicServiceRegistrationGenerator) generateSchemaConfig(service ProtoServiceInfo) SchemaConfig {
	methods := make([]string, len(service.Methods))
	for i, method := range service.Methods {
		methods[i] = method.Name
	}

	return SchemaConfig{
		ProtoPath: g.inferProtoPath(service.ServiceName),
		Methods:   methods,
	}
}

// inferProtoPath infers the proto path for a service.
func (g *DynamicServiceRegistrationGenerator) inferProtoPath(serviceName string) string {
	normalizedName := g.normalizeServiceName(serviceName)
	return fmt.Sprintf("api/protos/%s/v1/%s.proto", normalizedName, normalizedName)
}

// generateEndpointConfigs generates endpoint configurations.
func (g *DynamicServiceRegistrationGenerator) generateEndpointConfigs(service ProtoServiceInfo) []EndpointConfig {
	serviceName := g.normalizeServiceName(service.ServiceName)

	actions := make([]string, len(service.Methods))
	for i, method := range service.Methods {
		actions[i] = g.methodToAction(method.Name)
	}

	return []EndpointConfig{
		{
			Path:        fmt.Sprintf("/api/%s_ops", serviceName),
			Method:      "POST",
			Actions:     actions,
			Description: fmt.Sprintf("Composable %s operations endpoint. Each action maps to a gRPC/proto method and supports metadata enrichment.", serviceName),
		},
	}
}

// methodToAction converts a method name to an action name.
func (g *DynamicServiceRegistrationGenerator) methodToAction(methodName string) string {
	// Convert CamelCase to snake_case
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(methodName, "${1}_${2}")
	return strings.ToLower(snake)
}

// generateActionMap generates action mapping configuration.
func (g *DynamicServiceRegistrationGenerator) generateActionMap(service ProtoServiceInfo) map[string]ActionConfig {
	actionMap := make(map[string]ActionConfig)

	// Build a map of message name to fields for proto messages
	messageFields := g.extractProtoMessageFields(service)

	for _, method := range service.Methods {
		actionName := g.methodToAction(method.Name)
		requiredFields := g.inferRequiredFields(method)
		fields := make(map[string]FieldConfig)
		// Use extracted proto message fields if available
		if msgFields, ok := messageFields[method.InputType]; ok {
			for fname, ftype := range msgFields {
				fields[fname] = FieldConfig{
					Type:     ftype,
					Required: contains(requiredFields, fname),
				}
			}
		} else {
			// Fallback: just mark required fields as string type
			for _, fname := range requiredFields {
				fields[fname] = FieldConfig{Type: "string", Required: true}
			}
		}
		actionMap[actionName] = ActionConfig{
			ProtoMethod:        method.Name,
			RequestModel:       method.InputType,
			ResponseModel:      method.OutputType,
			RestRequiredFields: requiredFields,
			Fields:             fields,
		}
	}

	return actionMap
}

// extractProtoMessageFields parses proto messages and returns a map of message name to field name/type.
func (g *DynamicServiceRegistrationGenerator) extractProtoMessageFields(service ProtoServiceInfo) map[string]map[string]string {
	// This is a simple parser for proto message fields
	// Only supports basic types and ignores nested/complex fields for now
	messageFields := make(map[string]map[string]string)
	protoPath := g.inferProtoPath(service.ServiceName)
	content, err := os.ReadFile(protoPath)
	if err != nil {
		return messageFields
	}
	lines := strings.Split(string(content), "\n")
	var currentMsg string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentMsg = parts[1]
				messageFields[currentMsg] = make(map[string]string)
			}
			continue
		}
		if currentMsg != "" && line == "}" {
			currentMsg = ""
			continue
		}
		if currentMsg != "" && line != "" && !strings.HasPrefix(line, "//") {
			// Example: string name = 1;
			fieldParts := strings.Fields(line)
			if len(fieldParts) >= 3 {
				ftype := fieldParts[0]
				fname := fieldParts[1]
				// Remove trailing semicolon and '='
				fname = strings.Split(fname, "=")[0]
				fname = strings.TrimSpace(strings.TrimSuffix(fname, ";"))
				messageFields[currentMsg][fname] = ftype
			}
		}
	}
	return messageFields
}

// contains checks if a slice contains a string.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// inferRequiredFields infers required fields for REST endpoints.
func (g *DynamicServiceRegistrationGenerator) inferRequiredFields(method ProtoMethodInfo) []string {
	fields := []string{}

	// Common required field patterns
	methodLower := strings.ToLower(method.Name)
	inputLower := strings.ToLower(method.InputType)

	// Always require metadata for enrichment
	fields = append(fields, "metadata")

	// Method-specific requirements
	if strings.Contains(methodLower, "create") {
		if strings.Contains(inputLower, "user") {
			fields = append(fields, "name", "email")
		}
		if strings.Contains(inputLower, "content") {
			fields = append(fields, "title", "content", "user_id")
		}
	}

	if strings.Contains(methodLower, "get") || strings.Contains(methodLower, "update") || strings.Contains(methodLower, "delete") {
		if strings.Contains(inputLower, "user") {
			fields = append(fields, "user_id")
		}
		if strings.Contains(inputLower, "content") {
			fields = append(fields, "content_id")
		}
	}

	if strings.Contains(methodLower, "list") {
		fields = append(fields, "page", "page_size")
	}

	return g.removeDuplicates(fields)
}

// enhanceWithCodeAnalysis enhances config with code analysis.
func (g *DynamicServiceRegistrationGenerator) enhanceWithCodeAnalysis(ctx context.Context, config *ServiceRegistrationConfig) {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		g.logger.Warn("enhanceWithCodeAnalysis cancelled by context", zap.Error(ctx.Err()))
		return
	}
	// Analyze Go source files for additional context
	servicePath := filepath.Join(g.srcPath, "internal", "service", config.Name)

	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return
	} else if err != nil {
		g.logger.Warn("os.Stat error", zap.Error(err))
		return
	}

	// Parse Go files for additional metadata
	if err := filepath.WalkDir(servicePath, func(path string, d fs.DirEntry, err error) error {
		_ = d // Use d to avoid revive unused-parameter warning
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		g.analyzeGoFile(path, config)
		return nil
	}); err != nil {
		g.logger.Warn("filepath.WalkDir error", zap.Error(err))
		return
	}
}

// analyzeGoFile analyzes a Go file for service metadata.
func (g *DynamicServiceRegistrationGenerator) analyzeGoFile(path string, config *ServiceRegistrationConfig) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
	}

	// Look for service struct and its methods
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			if x.Name.Name == "Service" {
				// Found service struct, analyze its methods
				g.analyzeServiceStruct(x, config)
			}
		case *ast.FuncDecl:
			// Analyze function for capabilities
			g.analyzeFunctionForCapabilities(x, config)
		}
		return true
	})
}

// analyzeServiceStruct analyzes service struct for metadata.
func (g *DynamicServiceRegistrationGenerator) analyzeServiceStruct(spec *ast.TypeSpec, config *ServiceRegistrationConfig) {
	// Extract capabilities from struct fields and comments
	if structType, ok := spec.Type.(*ast.StructType); ok {
		for _, field := range structType.Fields.List {
			if field.Doc != nil {
				for _, comment := range field.Doc.List {
					if strings.Contains(comment.Text, "capability:") {
						capability := strings.TrimSpace(strings.Split(comment.Text, "capability:")[1])
						config.Capabilities = append(config.Capabilities, capability)
					}
				}
			}
		}
	}
}

// analyzeFunctionForCapabilities analyzes function for capabilities.
func (g *DynamicServiceRegistrationGenerator) analyzeFunctionForCapabilities(fn *ast.FuncDecl, config *ServiceRegistrationConfig) {
	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			if strings.Contains(comment.Text, "capability:") {
				capability := strings.TrimSpace(strings.Split(comment.Text, "capability:")[1])
				config.Capabilities = append(config.Capabilities, capability)
			}
		}
	}
}

// removeDuplicates removes duplicate strings from slice.
func (g *DynamicServiceRegistrationGenerator) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// GenerateAndSaveConfig generates and saves service registration configuration.
func (g *DynamicServiceRegistrationGenerator) GenerateAndSaveConfig(ctx context.Context, outputPath string) error {
	configs, err := g.GenerateServiceRegistrations(ctx)
	if err != nil {
		return err
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}

	// Save to file
	if err := os.WriteFile(outputPath, jsonData, 0o600); err != nil {
		return err
	}

	g.logger.Info("Generated service registration configuration",
		zap.String("output", outputPath),
		zap.Int("services", len(configs)))

	return nil
}

// IntrospectService uses reflection to analyze a service interface.
func (g *DynamicServiceRegistrationGenerator) IntrospectService(service interface{}) (*ServiceRegistrationConfig, error) {
	serviceType := reflect.TypeOf(service)
	if serviceType == nil {
		return nil, fmt.Errorf("service is nil")
	}

	// Handle interface types
	if serviceType.Kind() == reflect.Interface {
		serviceName := g.extractServiceNameFromInterface(serviceType)

		config := &ServiceRegistrationConfig{
			Name:               serviceName,
			Version:            "v1",
			Capabilities:       []string{},
			Dependencies:       []string{},
			Models:             []string{},
			MetadataEnrichment: true,
			ActionMap:          make(map[string]ActionConfig),
		}

		// Analyze methods
		for i := 0; i < serviceType.NumMethod(); i++ {
			method := serviceType.Method(i)
			g.analyzeMethod(method, config)
		}

		return config, nil
	}

	return nil, fmt.Errorf("unsupported service type: %s", serviceType.Kind())
}

// extractServiceNameFromInterface extracts service name from interface type.
func (g *DynamicServiceRegistrationGenerator) extractServiceNameFromInterface(t reflect.Type) string {
	name := t.Name()
	name = strings.TrimSuffix(name, "Server")
	name = strings.TrimSuffix(name, "Service")
	return strings.ToLower(name)
}

// analyzeMethod analyzes a method using reflection.
func (g *DynamicServiceRegistrationGenerator) analyzeMethod(method reflect.Method, config *ServiceRegistrationConfig) {
	methodName := method.Name

	// Skip unexported methods
	if !method.IsExported() {
		return
	}

	// Add to capabilities based on method name
	capabilities := g.inferCapabilities(ProtoServiceInfo{
		ServiceName: config.Name,
		Methods: []ProtoMethodInfo{
			{Name: methodName},
		},
	})

	for _, cap := range capabilities {
		found := false
		for _, existing := range config.Capabilities {
			if existing == cap {
				found = true
				break
			}
		}
		if !found {
			config.Capabilities = append(config.Capabilities, cap)
		}
	}

	// Add to action map
	actionName := g.methodToAction(methodName)
	config.ActionMap[actionName] = ActionConfig{
		ProtoMethod:   methodName,
		RequestModel:  fmt.Sprintf("%sRequest", methodName),
		ResponseModel: fmt.Sprintf("%sResponse", methodName),
	}
}

// RegisterServiceDynamically registers a service with dynamic configuration generation.
func (g *DynamicServiceRegistrationGenerator) RegisterServiceDynamically(
	ctx context.Context,
	service interface{},
	registryInstance *registry.ServiceRegistration,
) error {
	// Use context for diagnostics/cancellation (lint fix)
	if ctx != nil && ctx.Err() != nil {
		g.logger.Warn("RegisterServiceDynamically cancelled by context", zap.Error(ctx.Err()))
		return ctx.Err()
	}
	_ = registryInstance // Use registryInstance to avoid revive unused-parameter warning
	// Generate config through introspection
	config, err := g.IntrospectService(service)
	if err != nil {
		return err
	}

	// Convert to registry format
	methods := make([]registry.ServiceMethod, 0, len(config.ActionMap))
	for _, action := range config.ActionMap {
		methods = append(methods, registry.ServiceMethod{
			Name:        action.ProtoMethod,
			Parameters:  action.RestRequiredFields,
			Description: fmt.Sprintf("Auto-generated method for %s", action.ProtoMethod),
		})
	}

	serviceReg := registry.ServiceRegistration{
		ServiceName:  config.Name,
		Methods:      methods,
		RegisteredAt: time.Now(),
		Description:  fmt.Sprintf("Auto-generated registration for %s service", config.Name),
		Version:      config.Version,
		External:     false,
	}

	// Register with the registry
	registry.RegisterService(serviceReg)

	g.logger.Info("Dynamically registered service",
		zap.String("service", config.Name),
		zap.Int("methods", len(methods)))

	return nil
}

// UpdateKnowledgeGraph manually updates the knowledge graph with current service registrations.
func (g *DynamicServiceRegistrationGenerator) UpdateKnowledgeGraph(ctx context.Context) error {
	logger := g.logger.With(zap.String("operation", "update_knowledge_graph"))
	logger.Info("Updating knowledge graph with service registration data...")

	// Get the default knowledge graph instance
	knowledgeGraph := kg.DefaultKnowledgeGraph()

	// Generate the current service configurations
	services, err := g.GenerateServiceRegistrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate service registrations: %w", err)
	}

	// Update the knowledge graph with service information
	successCount := 0
	for _, service := range services {
		serviceInfo := map[string]interface{}{
			"metadata": map[string]interface{}{
				"version":      service.Version,
				"capabilities": service.Capabilities,
				"endpoints":    service.Endpoints,
				"models":       service.Models,
				"schema":       service.Schema,
				"health_check": service.HealthCheck,
				"metrics":      service.Metrics,
				"updated_at":   time.Now().UTC(),
			},
			"dependencies": service.Dependencies,
		}

		// Add or update the service in the knowledge graph
		if err := knowledgeGraph.AddService("dynamic_services", service.Name, serviceInfo); err != nil {
			logger.Warn("Failed to add service to knowledge graph",
				zap.String("service", service.Name),
				zap.Error(err))
			continue
		}

		successCount++
		logger.Debug("Updated service in knowledge graph",
			zap.String("service", service.Name),
			zap.String("version", service.Version))
	}

	// Update the amadeus_integration section with registration metadata
	integrationInfo := map[string]interface{}{
		"service_registration": map[string]interface{}{
			"last_update":    time.Now().UTC(),
			"services_count": len(services),
			"success_count":  successCount,
			"auto_discovery": true,
			"generator_config": map[string]interface{}{
				"proto_path": g.protoPath,
				"src_path":   g.srcPath,
			},
		},
	}

	if err := knowledgeGraph.UpdateNode("amadeus_integration", integrationInfo); err != nil {
		return fmt.Errorf("failed to update amadeus integration info: %w", err)
	}

	logger.Info("Successfully updated knowledge graph",
		zap.Int("total_services", len(services)),
		zap.Int("updated_services", successCount))

	return nil
}

// EventState is the canonical set of event states for all services.
var EventStates = []string{"requested", "started", "success", "failed", "completed"}

// GenerateEventTypes generates all canonical event types and Redis key patterns for all services and methods.
func (g *DynamicServiceRegistrationGenerator) GenerateEventTypesWithVersioning(ctx context.Context, version string) (map[string][]string, error) {
	configs, err := g.GenerateServiceRegistrations(ctx)
	if err != nil {
		return nil, err
	}
	eventTypes := make(map[string][]string)
	for _, cfg := range configs {
		service := cfg.Name
		ver := version
		if cfg.Version != "" {
			ver = cfg.Version
		}
		for _, method := range cfg.Schema.Methods {
			action := g.methodToAction(method)
			for _, state := range EventStates {
				eventType := service + ":" + action + ":v" + ver + ":" + state
				eventTypes[service] = append(eventTypes[service], eventType)
			}
		}
	}
	return eventTypes, nil
}

// WriteEventTypesGo writes event types as Go constants for use in code.
func WriteEventTypesGo(eventTypes map[string][]string, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("// Code generated by generator.go. DO NOT EDIT.\n\npackage events\n\n"); err != nil {
		return err
	}
	for _, types := range eventTypes {
		for _, t := range types {
			constName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(t, ":", "_"), ".", "_"))
			if _, err := fmt.Fprintf(f, "const %s = \"%s\"\n", constName, t); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteEventTypesJSON writes event types as a JSON file for docs or validation.
func WriteEventTypesJSON(eventTypes map[string][]string, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(eventTypes)
}
