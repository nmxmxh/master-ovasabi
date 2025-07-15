package registration

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// PayloadValidator provides centralized payload validation and cleaning
// based on service registration schemas and protobuf field definitions
type PayloadValidator struct {
	generator     *DynamicServiceRegistrationGenerator
	messageFields map[string][]string // Maps message type to its field names
	logger        *zap.Logger
}

// NewPayloadValidator creates a new payload validator using existing registration infrastructure
func NewPayloadValidator(logger *zap.Logger, protoPath string) (*PayloadValidator, error) {
	generator := NewDynamicServiceRegistrationGenerator(logger, protoPath, "")

	pv := &PayloadValidator{
		generator:     generator,
		messageFields: make(map[string][]string),
		logger:        logger,
	}

	// Pre-load message field definitions from proto files
	if err := pv.loadMessageFields(); err != nil {
		return nil, fmt.Errorf("failed to load message fields: %w", err)
	}

	return pv, nil
}

// loadMessageFields loads all message field definitions from proto files
func (pv *PayloadValidator) loadMessageFields() error {
	protoServices, err := pv.generator.discoverProtoServices()
	if err != nil {
		return fmt.Errorf("failed to discover proto services: %w", err)
	}

	// Extract field definitions for each message type
	for _, service := range protoServices {
		// For each method, parse the input message to get its fields
		for _, method := range service.Methods {
			if fields, err := pv.parseMessageFields(method.InputType); err == nil {
				pv.messageFields[method.InputType] = fields
			}
		}
	}

	return nil
}

// parseMessageFields parses a protobuf message definition to extract field names
func (pv *PayloadValidator) parseMessageFields(messageType string) ([]string, error) {
	// Find the proto file that contains this message
	protoFile, err := pv.findProtoFileForMessage(messageType)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(protoFile)
	if err != nil {
		return nil, err
	}

	return pv.extractFieldsFromMessage(string(content), messageType)
}

// findProtoFileForMessage finds the proto file that contains a specific message type
func (pv *PayloadValidator) findProtoFileForMessage(messageType string) (string, error) {
	var foundFile string

	err := pv.generator.walkProtoFiles(func(path string, content []byte) error {
		if strings.Contains(string(content), "message "+messageType+" {") {
			foundFile = path
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if foundFile == "" {
		return "", fmt.Errorf("message type %s not found in proto files", messageType)
	}

	return foundFile, nil
}

// extractFieldsFromMessage extracts field names from a message definition
func (pv *PayloadValidator) extractFieldsFromMessage(content, messageType string) ([]string, error) {
	var fields []string

	// Find the message definition
	messageRegex := regexp.MustCompile(`message\s+` + regexp.QuoteMeta(messageType) + `\s*\{([^}]+)\}`)
	matches := messageRegex.FindStringSubmatch(content)

	if len(matches) < 2 {
		return nil, fmt.Errorf("message %s not found or malformed", messageType)
	}

	messageBody := matches[1]
	lines := strings.Split(messageBody, "\n")

	// Parse field definitions
	// Handle simple types: "string field_name = 1;"
	// Handle repeated fields: "repeated string field_name = 2;"
	// Handle complex types: "common.Metadata metadata = 5;"
	// Handle generic types: "map<string, string> labels = 6;"
	fieldRegex := regexp.MustCompile(`^\s*(?:repeated\s+)?(\w+(?:\.\w+)*(?:<[^>]+>)?)\s+(\w+)\s*=\s*\d+;`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		fieldMatches := fieldRegex.FindStringSubmatch(line)
		if len(fieldMatches) >= 3 {
			fieldName := fieldMatches[2] // field name is always the last word before =
			fieldType := fieldMatches[1] // field type for debugging
			fields = append(fields, fieldName)

			// Debug log the field extraction
			pv.logger.Debug("Extracted protobuf field",
				zap.String("message_type", messageType),
				zap.String("field_name", fieldName),
				zap.String("field_type", fieldType),
				zap.String("line", line))
		} else {
			// Log lines that don't match for debugging
			pv.logger.Debug("Failed to parse protobuf field line",
				zap.String("message_type", messageType),
				zap.String("line", line))
		}
	}

	return fields, nil
}

// ValidateAndCleanPayload validates and cleans a payload based on the target service and action
func (pv *PayloadValidator) ValidateAndCleanPayload(eventType string, payload *structpb.Struct) (*structpb.Struct, error) {
	if payload == nil || payload.Fields == nil {
		return payload, nil
	}

	// Parse event type to extract service and action
	service, action, err := pv.parseEventType(eventType)
	if err != nil {
		pv.logger.Warn("Could not parse event type for payload validation",
			zap.String("event_type", eventType),
			zap.Error(err))
		return payload, nil // Return original payload if we can't parse
	}

	// Get the request model for this service/action
	requestModel, err := pv.getRequestModel(service, action)
	if err != nil {
		pv.logger.Warn("Could not find request model for payload validation",
			zap.String("service", service),
			zap.String("action", action),
			zap.Error(err))
		return payload, nil // Return original payload if we can't find the model
	}

	// Get the valid fields for this message type
	validFields, exists := pv.messageFields[requestModel]
	if !exists {
		pv.logger.Warn("Message fields not found for request model",
			zap.String("request_model", requestModel))
		return payload, nil // Return original payload if fields not found
	}

	// Clean the payload to only include valid protobuf fields
	cleanedFields := make(map[string]*structpb.Value)

	for _, fieldName := range validFields {
		if value, exists := payload.Fields[fieldName]; exists {
			cleanedFields[fieldName] = value
		}
	}

	pv.logger.Debug("Cleaned payload for service",
		zap.String("service", service),
		zap.String("action", action),
		zap.String("request_model", requestModel),
		zap.Int("original_fields", len(payload.Fields)),
		zap.Int("cleaned_fields", len(cleanedFields)),
		zap.Strings("valid_fields", validFields))

	return &structpb.Struct{Fields: cleanedFields}, nil
}

// parseEventType extracts service and action from canonical event type
// Format: {service}:{action}:v{version}:{state}
func (pv *PayloadValidator) parseEventType(eventType string) (service, action string, err error) {
	parts := strings.Split(eventType, ":")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid event type format: %s", eventType)
	}

	return parts[0], parts[1], nil
}

// getRequestModel gets the request model for a service/action combination using standard naming conventions
func (pv *PayloadValidator) getRequestModel(service, action string) (string, error) {
	// Convert to PascalCase for protobuf naming convention
	servicePascal := pv.toPascalCase(service)
	actionPascal := pv.toPascalCase(action)

	// Try different naming patterns in order of preference
	patterns := []string{
		// Pattern 1: {Action}Request (for cases like "SearchRequest", "SuggestRequest")
		actionPascal + "Request",
		// Pattern 2: {Action}{Service}Request (for cases like "CreateUserRequest", "UpdateUserRequest")
		actionPascal + servicePascal + "Request",
		// Pattern 3: {Service}{Action}Request (alternative pattern)
		servicePascal + actionPascal + "Request",
	}

	// Check each pattern to see if we have field definitions for it
	for _, requestModel := range patterns {
		if _, exists := pv.messageFields[requestModel]; exists {
			pv.logger.Debug("Found request model using naming pattern",
				zap.String("service", service),
				zap.String("action", action),
				zap.String("request_model", requestModel))
			return requestModel, nil
		}
	}

	// Log available message types for debugging
	var availableTypes []string
	for msgType := range pv.messageFields {
		if strings.HasSuffix(msgType, "Request") {
			availableTypes = append(availableTypes, msgType)
		}
	}

	pv.logger.Debug("Request model not found, tried patterns",
		zap.String("service", service),
		zap.String("action", action),
		zap.Strings("tried_patterns", patterns),
		zap.Strings("available_request_types", availableTypes))

	return "", fmt.Errorf("request model not found for %s.%s (tried patterns: %v)", service, action, patterns)
}

// toPascalCase converts a string to PascalCase (first letter uppercase, rest lowercase for each word)
func (pv *PayloadValidator) toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Handle simple case: single word
	words := strings.Fields(strings.ToLower(s))
	if len(words) == 0 {
		return s
	}

	var result strings.Builder
	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(word[:1]) + word[1:])
		}
	}

	return result.String()
}

// walkProtoFiles is a helper method to walk through proto files
func (g *DynamicServiceRegistrationGenerator) walkProtoFiles(callback func(path string, content []byte) error) error {
	return g.walkProtoDir(g.protoPath, callback)
}

// walkProtoDir recursively walks through a directory looking for proto files
func (g *DynamicServiceRegistrationGenerator) walkProtoDir(dir string, callback func(path string, content []byte) error) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := dir + "/" + entry.Name()

		if entry.IsDir() {
			if err := g.walkProtoDir(path, callback); err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".proto") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if err := callback(path, content); err != nil {
				return err
			}
		}
	}

	return nil
}
