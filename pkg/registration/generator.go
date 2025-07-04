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

	"github.com/nmxmxh/master-ovasabi/config/registry"
	"go.uber.org/zap"
)

// DynamicServiceRegistrationGenerator generates service registration configs
// by analyzing proto files, Go code, and service interfaces through reflection
type DynamicServiceRegistrationGenerator struct {
	logger    *zap.Logger
	protoPath string
	srcPath   string
}

// NewDynamicServiceRegistrationGenerator creates a new generator instance
func NewDynamicServiceRegistrationGenerator(logger *zap.Logger, protoPath, srcPath string) *DynamicServiceRegistrationGenerator {
	return &DynamicServiceRegistrationGenerator{
		logger:    logger,
		protoPath: protoPath,
		srcPath:   srcPath,
	}
}

// ServiceRegistrationConfig represents the structure of a service registration
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

// SchemaConfig represents the schema configuration for a service
type SchemaConfig struct {
	ProtoPath string   `json:"proto_path,omitempty"`
	Methods   []string `json:"methods,omitempty"`
}

// EndpointConfig represents an endpoint configuration
type EndpointConfig struct {
	Path        string   `json:"path"`
	Method      string   `json:"method"`
	Actions     []string `json:"actions"`
	Description string   `json:"description"`
}

// ActionConfig represents an action configuration
type ActionConfig struct {
	ProtoMethod        string   `json:"proto_method"`
	RequestModel       string   `json:"request_model"`
	ResponseModel      string   `json:"response_model"`
	RestRequiredFields []string `json:"rest_required_fields,omitempty"`
}

// ProtoServiceInfo contains information extracted from proto files
type ProtoServiceInfo struct {
	ServiceName string
	Methods     []ProtoMethodInfo
	Messages    []string
}

// ProtoMethodInfo contains information about a proto method
type ProtoMethodInfo struct {
	Name        string
	InputType   string
	OutputType  string
	Description string
}

// GenerateServiceRegistrations generates service registration configs for all services
func (g *DynamicServiceRegistrationGenerator) GenerateServiceRegistrations(ctx context.Context) ([]ServiceRegistrationConfig, error) {
	var configs []ServiceRegistrationConfig

	// Walk through proto files to discover services
	protoServices, err := g.discoverProtoServices()
	if err != nil {
		g.logger.Error("Failed to discover proto services", zap.Error(err))
		return nil, err
	}

	for _, protoService := range protoServices {
		config, err := g.generateServiceConfig(ctx, protoService)
		if err != nil {
			g.logger.Error("Failed to generate service config",
				zap.String("service", protoService.ServiceName),
				zap.Error(err))
			continue
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// discoverProtoServices discovers all services from proto files
func (g *DynamicServiceRegistrationGenerator) discoverProtoServices() ([]ProtoServiceInfo, error) {
	var services []ProtoServiceInfo

	err := filepath.WalkDir(g.protoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".proto") {
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

// parseProtoFile parses a proto file to extract service information
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

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Extract service name
		if strings.HasPrefix(line, "service ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentService = strings.TrimSuffix(parts[1], " {")
				serviceInfo.ServiceName = currentService
				inService = true
			}
		}

		// Extract methods within service
		if inService && strings.HasPrefix(line, "rpc ") {
			method := g.parseRPCMethod(line)
			if method != nil {
				serviceInfo.Methods = append(serviceInfo.Methods, *method)
			}
		}

		// Extract messages
		if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				messageName := strings.TrimSuffix(parts[1], " {")
				serviceInfo.Messages = append(serviceInfo.Messages, messageName)
			}
		}

		// End of service
		if inService && line == "}" {
			inService = false
		}
	}

	if serviceInfo.ServiceName == "" {
		return nil, nil
	}

	return serviceInfo, nil
}

// parseRPCMethod parses an RPC method line from proto file
func (g *DynamicServiceRegistrationGenerator) parseRPCMethod(line string) *ProtoMethodInfo {
	// Example: rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
	rpcRegex := regexp.MustCompile(`rpc\s+(\w+)\s*\(([^)]+)\)\s*returns\s*\(([^)]+)\)`)
	matches := rpcRegex.FindStringSubmatch(line)

	if len(matches) != 4 {
		return nil
	}

	return &ProtoMethodInfo{
		Name:       matches[1],
		InputType:  matches[2],
		OutputType: matches[3],
	}
}

// generateServiceConfig generates a service configuration
func (g *DynamicServiceRegistrationGenerator) generateServiceConfig(ctx context.Context, protoService ProtoServiceInfo) (ServiceRegistrationConfig, error) {
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

	return config, nil
}

// normalizeServiceName normalizes service names from proto to match convention
func (g *DynamicServiceRegistrationGenerator) normalizeServiceName(name string) string {
	// Convert from "UserService" to "user"
	name = strings.TrimSuffix(name, "Service")
	return strings.ToLower(name)
}

// inferCapabilities infers service capabilities from methods and context
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

// inferDependencies infers service dependencies from method signatures and imports
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

// generateSchemaConfig generates schema configuration
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

// inferProtoPath infers the proto path for a service
func (g *DynamicServiceRegistrationGenerator) inferProtoPath(serviceName string) string {
	normalizedName := g.normalizeServiceName(serviceName)
	return fmt.Sprintf("api/protos/%s/v1/%s.proto", normalizedName, normalizedName)
}

// generateEndpointConfigs generates endpoint configurations
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

// methodToAction converts a method name to an action name
func (g *DynamicServiceRegistrationGenerator) methodToAction(methodName string) string {
	// Convert CamelCase to snake_case
	re := regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := re.ReplaceAllString(methodName, "${1}_${2}")
	return strings.ToLower(snake)
}

// generateActionMap generates action mapping configuration
func (g *DynamicServiceRegistrationGenerator) generateActionMap(service ProtoServiceInfo) map[string]ActionConfig {
	actionMap := make(map[string]ActionConfig)

	for _, method := range service.Methods {
		actionName := g.methodToAction(method.Name)
		actionMap[actionName] = ActionConfig{
			ProtoMethod:        method.Name,
			RequestModel:       method.InputType,
			ResponseModel:      method.OutputType,
			RestRequiredFields: g.inferRequiredFields(method),
		}
	}

	return actionMap
}

// inferRequiredFields infers required fields for REST endpoints
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

// enhanceWithCodeAnalysis enhances config with code analysis
func (g *DynamicServiceRegistrationGenerator) enhanceWithCodeAnalysis(ctx context.Context, config *ServiceRegistrationConfig) {
	// Analyze Go source files for additional context
	servicePath := filepath.Join(g.srcPath, "internal", "service", config.Name)

	if _, err := os.Stat(servicePath); os.IsNotExist(err) {
		return
	}

	// Parse Go files for additional metadata
	filepath.WalkDir(servicePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") {
			return nil
		}

		g.analyzeGoFile(path, config)
		return nil
	})
}

// analyzeGoFile analyzes a Go file for service metadata
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

// analyzeServiceStruct analyzes service struct for metadata
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

// analyzeFunctionForCapabilities analyzes function for capabilities
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

// removeDuplicates removes duplicate strings from slice
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

// GenerateAndSaveConfig generates and saves service registration configuration
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
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return err
	}

	g.logger.Info("Generated service registration configuration",
		zap.String("output", outputPath),
		zap.Int("services", len(configs)))

	return nil
}

// IntrospectService uses reflection to analyze a service interface
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

// extractServiceNameFromInterface extracts service name from interface type
func (g *DynamicServiceRegistrationGenerator) extractServiceNameFromInterface(t reflect.Type) string {
	name := t.Name()
	name = strings.TrimSuffix(name, "Server")
	name = strings.TrimSuffix(name, "Service")
	return strings.ToLower(name)
}

// analyzeMethod analyzes a method using reflection
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

// RegisterServiceDynamically registers a service with dynamic configuration generation
func (g *DynamicServiceRegistrationGenerator) RegisterServiceDynamically(
	ctx context.Context,
	service interface{},
	registryInstance *registry.ServiceRegistration,
) error {
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
