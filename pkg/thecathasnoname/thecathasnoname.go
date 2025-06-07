package thecathasnoname

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/fatih/color"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/olekukonko/tablewriter"
)

// TheCatHasNoName is a demo/system instruction handler for system messages, tests, and orchestration demos.
type TheCatHasNoName struct {
	Logger *log.Logger
}

// New creates a new TheCatHasNoName handler.
func New(logger *log.Logger) *TheCatHasNoName {
	return &TheCatHasNoName{Logger: logger}
}

// SendInstruction sends a system instruction or test message.
func (c *TheCatHasNoName) SendInstruction(ctx context.Context, instruction string) {
	userID := ""
	if authCtx := contextx.Auth(ctx); authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
	}
	if userID != "" {
		c.Logger.Printf("[thecathasnoname] [user:%s] Instruction: %s", userID, instruction)
	} else {
		c.Logger.Printf("[thecathasnoname] Instruction: %s", instruction)
	}
}

// LogDemoEvent logs a demo event for testing/demo purposes.
func (c *TheCatHasNoName) LogDemoEvent(ctx context.Context, event string) {
	userID := ""
	if authCtx := contextx.Auth(ctx); authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
	}
	if userID != "" {
		c.Logger.Printf("[thecathasnoname] [user:%s] DemoEvent: %s", userID, event)
	} else {
		c.Logger.Printf("[thecathasnoname] DemoEvent: %s", event)
	}
}

// color codes.
const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorRed     = "\033[31m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
)

// Enhanced colorizeJSONIndented for better color and null support.
func colorizeJSONIndented(jsonStr string) string {
	var out strings.Builder
	inString := false
	for i := 0; i < len(jsonStr); i++ {
		c := jsonStr[i]
		switch {
		case c == '"':
			if inString {
				inString = false
			} else {
				inString = true
			}
			color.New(color.FgHiBlue, color.Bold).Fprint(&out, string(c))
		case inString:
			color.New(color.FgHiGreen, color.Bold).Fprint(&out, string(c))
		case c == ':' || c == '{' || c == '}' || c == '[' || c == ']' || c == ',':
			color.New(color.FgHiBlack).Fprint(&out, string(c))
		case c >= '0' && c <= '9':
			color.New(color.FgHiYellow, color.Bold).Fprint(&out, string(c))
		case c == 't' || c == 'f': // booleans
			color.New(color.FgHiMagenta, color.Bold).Fprint(&out, string(c))
		case strings.HasPrefix(jsonStr[i:], "null"):
			color.New(color.FgHiRed, color.Bold).Fprint(&out, "null")
			i += 3 // skip 'ull'
		default:
			color.New(color.FgHiBlack).Fprint(&out, string(c))
		}
	}
	return out.String()
}

// Helper: Render a metadata summary table with icons and types, optimized for human inspection.
func renderMetadataTable(meta interface{}) {
	table := tablewriter.NewWriter(os.Stdout)
	if err := table.Append([]string{"Field", "Type", "Value", "Icon"}); err != nil {
		fmt.Printf("Failed to append header row: %v\n", err)
		return
	}

	// Helper to determine depth from prefix
	depthOf := func(prefix string) int {
		if prefix == "" {
			return 0
		}
		return strings.Count(prefix, ".") + strings.Count(prefix, "[")
	}

	rowIcon := func(val interface{}) string {
		switch v := val.(type) {
		case string:
			return "🏷️"
		case float64, int, int64:
			return "🔢"
		case bool:
			if v {
				return "🟢"
			}
			return "⚪"
		case nil:
			return "🟥"
		case map[string]interface{}:
			return "📦"
		case []interface{}:
			return "📚"
		default:
			return "❓"
		}
	}
	rowType := func(val interface{}) string {
		if val == nil {
			return "null"
		}
		t := reflect.TypeOf(val)
		if t == nil {
			return "null"
		}
		return t.String()
	}

	// Collect all rows for sorting/highlighting
	type row struct {
		Field, Type, Value, Icon string
		IsCurrentState           bool
		Depth                    int
		IsKey                    bool // for hash, actor, etc.
	}
	var rows []row
	var keyRows []row

	var walk func(prefix string, v interface{})
	walk = func(prefix string, v interface{}) {
		switch val := v.(type) {
		case map[string]interface{}:
			for k, subv := range val {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				walk(key, subv)
			}
		case []interface{}:
			for i, subv := range val {
				key := fmt.Sprintf("%s[%d]", prefix, i)
				walk(key, subv)
			}
		default:
			isCurrent := strings.EqualFold(prefix, "current_state")
			isKey := prefix == "hash" || strings.HasSuffix(prefix, ".hash") || strings.HasSuffix(prefix, ".actor") || strings.HasSuffix(prefix, ".guest_actor") || strings.HasSuffix(prefix, ".versioning") || strings.HasSuffix(prefix, ".audit")
			r := row{
				Field:          prefix,
				Type:           rowType(val),
				Value:          fmt.Sprintf("%v", val),
				Icon:           rowIcon(val),
				IsCurrentState: isCurrent,
				Depth:          depthOf(prefix),
				IsKey:          isKey,
			}
			if isKey {
				keyRows = append(keyRows, r)
			} else {
				rows = append(rows, r)
			}
		}
	}

	// Convert meta to map[string]interface{} if needed
	var m map[string]interface{}
	switch v := meta.(type) {
	case map[string]interface{}:
		m = v
	case *map[string]interface{}:
		m = *v
	default:
		b, err := json.Marshal(v)
		if err == nil {
			if err := json.Unmarshal(b, &m); err != nil {
				fmt.Printf("Failed to unmarshal JSON: %v\n", err)
				return
			}
		}
	}
	if m == nil {
		return
	}
	walk("", m)

	// Sort: keyRows (hash, actor, etc.) first, then current_state, then others
	var currentRows, otherRows []row
	for _, r := range rows {
		if r.IsCurrentState {
			currentRows = append(currentRows, r)
		} else {
			otherRows = append(otherRows, r)
		}
	}

	// Add extra spacing before/after key sections
	if len(keyRows) > 0 {
		fmt.Println()
		color.New(color.FgHiMagenta, color.Bold, color.Underline).Println("Key Metadata Fields:")
		keyTable := tablewriter.NewWriter(os.Stdout)
		if err := keyTable.Append([]string{"Field", "Type", "Value", "Icon"}); err != nil {
			fmt.Printf("Failed to append header row: %v\n", err)
			return
		}
		for _, r := range keyRows {
			indent := strings.Repeat("  ", r.Depth)
			f := indent + r.Field
			field := color.New(color.FgMagenta, color.Bold).Sprint(f)
			typ := color.New(color.FgHiBlack).Sprint(r.Type)
			val := color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon := color.New(color.FgHiMagenta).Sprint(r.Icon)
			if err := keyTable.Append([]string{field, typ, val, icon}); err != nil {
				fmt.Printf("Failed to append row: %v\n", err)
				continue
			}
		}
		if err := keyTable.Render(); err != nil {
			fmt.Printf("Failed to render table: %v\n", err)
			return
		}
		fmt.Println()
		color.New(color.FgWhite, color.Bold).Println("───────────────────────────────────────────────────────────────")
	}

	if len(currentRows) > 0 {
		fmt.Println()
		color.New(color.FgHiYellow, color.Bold, color.Underline).Println("Current State Fields:")
		for _, r := range currentRows {
			indent := strings.Repeat("  ", r.Depth)
			f := indent + r.Field
			field := color.New(color.FgHiYellow, color.Bold).Sprint(f)
			typ := color.New(color.FgHiBlack).Sprint(r.Type)
			val := color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon := color.New(color.FgHiYellow).Sprint(r.Icon)
			if err := table.Append([]string{field, typ, val, icon}); err != nil {
				fmt.Printf("Failed to append row: %v\n", err)
				continue
			}
		}
		color.New(color.FgWhite, color.Bold).Println("───────────────────────────────────────────────────────────────")
	}

	for _, r := range otherRows {
		indent := strings.Repeat("  ", r.Depth)
		f := indent + r.Field
		var field, typ, val, icon string
		lower := strings.ToLower(r.Field)
		switch {
		case r.IsCurrentState:
			field = color.New(color.FgHiYellow, color.Bold, color.Underline).Sprint(f)
			typ = color.New(color.FgHiBlack).Sprint(r.Type)
			val = color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon = color.New(color.FgHiYellow).Sprint(r.Icon)
		case strings.Contains(lower, "actor"):
			field = color.New(color.FgHiMagenta, color.Bold).Sprint(f)
			typ = color.New(color.FgHiBlack).Sprint(r.Type)
			val = color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon = color.New(color.FgHiMagenta).Sprint(r.Icon)
		case strings.Contains(lower, "guest_actor"):
			field = color.New(color.FgHiYellow, color.Bold).Sprint(f)
			typ = color.New(color.FgHiBlack).Sprint(r.Type)
			val = color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon = color.New(color.FgHiYellow).Sprint(r.Icon)
		case strings.Contains(lower, "versioning"):
			field = color.New(color.FgHiGreen, color.Bold).Sprint(f)
			typ = color.New(color.FgHiBlack).Sprint(r.Type)
			val = color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon = color.New(color.FgHiGreen).Sprint(r.Icon)
		case strings.Contains(lower, "audit"):
			field = color.New(color.FgHiCyan, color.Bold).Sprint(f)
			typ = color.New(color.FgHiBlack).Sprint(r.Type)
			val = color.New(color.FgHiBlue, color.Bold).Sprint(r.Value)
			icon = color.New(color.FgHiCyan).Sprint(r.Icon)
		default:
			switch r.Type {
			case "string":
				field = color.New(color.FgHiBlue).Sprint(f)
				typ = color.New(color.FgHiBlack).Sprint(r.Type)
				val = color.New(color.FgHiGreen, color.Bold).Sprint(r.Value)
				icon = color.New(color.FgHiBlue).Sprint(r.Icon)
			case "float64", "int", "int64":
				field = color.New(color.FgHiYellow).Sprint(f)
				typ = color.New(color.FgHiBlack).Sprint(r.Type)
				val = color.New(color.FgHiMagenta, color.Bold).Sprint(r.Value)
				icon = color.New(color.FgHiYellow).Sprint(r.Icon)
			case "bool":
				field = color.New(color.FgHiMagenta).Sprint(f)
				typ = color.New(color.FgHiBlack).Sprint(r.Type)
				val = color.New(color.FgHiMagenta, color.Bold).Sprint(r.Value)
				icon = color.New(color.FgHiMagenta).Sprint(r.Icon)
			case "null":
				field = color.New(color.FgHiRed).Sprint(f)
				typ = color.New(color.FgHiBlack).Sprint(r.Type)
				val = color.New(color.FgHiRed, color.Bold).Sprint(r.Value)
				icon = color.New(color.FgHiRed).Sprint(r.Icon)
			default:
				field = color.New(color.FgHiCyan).Sprint(f)
				typ = color.New(color.FgHiBlack).Sprint(r.Type)
				val = color.New(color.FgHiCyan, color.Bold).Sprint(r.Value)
				icon = color.New(color.FgHiCyan).Sprint(r.Icon)
			}
		}
		if err := table.Append([]string{field, typ, val, icon}); err != nil {
			fmt.Printf("Failed to append row: %v\n", err)
			continue
		}
	}

	fmt.Println()
	if err := table.Render(); err != nil {
		fmt.Printf("Failed to render table: %v\n", err)
		return
	}
	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("───────────────────────────── Icon Legend ─────────────────────────────")
	color.New(color.FgHiBlue).Println("🏷️ string   ")
	color.New(color.FgHiYellow).Println("🔢 number   ")
	color.New(color.FgHiMagenta).Println("🟢 true   ⚪ false   ")
	color.New(color.FgHiRed).Println("🟥 null   ")
	color.New(color.FgHiCyan).Println("📦 object   📚 array   ❓ unknown")
	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("───────────────────────────────────────────────────────────────────────")
	fmt.Println()
}

// AnnounceSystemEvent logs a system-wide announcement with context, with color and structure, optimized for human review.
func (c *TheCatHasNoName) AnnounceSystemEvent(ctx context.Context, position, service, function string, metadata interface{}, extra ...interface{}) {
	fmt.Println()
	color.New(color.FgHiCyan, color.Bold).Println("\n==================== 🐾 SYSTEM ANNOUNCEMENT 🐾 ====================")
	color.New(color.FgHiMagenta, color.Bold).Printf("\n[%s] %s.%s\n", position, service, function)
	if metadata != nil {
		// Render summary table first
		color.New(color.FgHiYellow, color.Bold).Println("\nMetadata Summary:")
		renderMetadataTable(metadata)
		// Then pretty JSON
		b, err := json.MarshalIndent(metadata, "", "  ")
		if err == nil {
			color.New(color.FgHiBlack, color.Bold).Println("\n      ──────────────────────────────────────────────")
			for _, line := range strings.Split(string(b), "\n") {
				fmt.Print("      ")
				fmt.Println(colorizeJSONIndented(line))
			}
			color.New(color.FgHiBlack, color.Bold).Println("      ──────────────────────────────────────────────")
			fmt.Println()
		}
	}
	if len(extra) > 0 {
		color.New(color.FgHiYellow, color.Bold).Printf("Extra: %+v\n", extra...)
	}
	color.New(color.FgHiCyan, color.Bold).Println("\n===============================================================")
	fmt.Println()
	// Cat speaks after every system announcement
	color.New(color.FgHiMagenta, color.Bold, color.Underline).Println("The cat speaks!")
	if err := c.Speak(ctx, "The cat speaks!", toCommonpbMetadata(metadata)); err != nil {
		fmt.Printf("failed to speak: %v\n", err)
	}
}

// Helper to convert map[string]interface{} to *commonpb.Metadata (minimal, avoids import cycle).
func mapToProto(m map[string]interface{}) *commonpb.Metadata {
	b, err := json.Marshal(m)
	if err != nil {
		return &commonpb.Metadata{}
	}
	var meta commonpb.Metadata
	if err := json.Unmarshal(b, &meta); err != nil {
		return &commonpb.Metadata{}
	}
	return &meta
}

func toCommonpbMetadata(meta interface{}) *commonpb.Metadata {
	switch m := meta.(type) {
	case *commonpb.Metadata:
		return m
	case map[string]interface{}:
		return mapToProto(m)
	default:
		return nil
	}
}

// Example usage: if thecathasnoname.IsBenchmarkSummaryMap(m) { ... }.
func IsBenchmarkSummaryMap(m map[string]interface{}) bool {
	_, hasScenario := m["Scenario"]
	_, hasDuration := m["Duration"]
	return hasScenario && hasDuration
}

// Example usage: thecathasnoname.PrintBenchmarkTableFromMap(m).
func PrintBenchmarkTableFromMap(m map[string]interface{}) {
	table := tablewriter.NewWriter(os.Stdout)
	// tablewriter.Table does not have SetHeader; just print rows.
	for k, v := range m {
		if err := table.Append([]string{k, fmt.Sprintf("%v", v)}); err != nil {
			fmt.Printf("failed to append row: %v\n", err)
			continue
		}
	}
	if err := table.Render(); err != nil {
		fmt.Printf("failed to render table: %v\n", err)
		return
	}
}

// Example usage: if thecathasnoname.IsBenchmarkSummaryStruct(reflect.ValueOf(s)) { ... }.
func IsBenchmarkSummaryStruct(v reflect.Value) bool {
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == "Scenario" {
			return true
		}
	}
	return false
}

// Example usage: thecathasnoname.PrintBenchmarkTableFromStruct(reflect.ValueOf(s)).
func PrintBenchmarkTableFromStruct(v reflect.Value) {
	table := tablewriter.NewWriter(os.Stdout)
	// tablewriter.Table does not have SetHeader; just print rows.
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		val := v.Field(i).Interface()
		if err := table.Append([]string{name, fmt.Sprintf("%v", val)}); err != nil {
			fmt.Printf("failed to append row: %v\n", err)
			continue
		}
	}
	if err := table.Render(); err != nil {
		fmt.Printf("failed to render table: %v\n", err)
		return
	}
}

// Speak implements the tester.NexusSpeaker interface so the cat can act as a global speaker, with color and structure.
func (c *TheCatHasNoName) Speak(_ context.Context, msg string, meta *commonpb.Metadata) error {
	c.Logger.Printf("\n%s==================== CAT SPEAKS ====================%s\n", colorMagenta, colorReset)
	c.Logger.Printf("%sThe cat speaks!%s\n", colorYellow, colorReset)
	c.Logger.Printf("%sMessage:%s   %s\n", colorBlue, colorReset, msg)
	if meta != nil {
		b, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		c.Logger.Printf("%sMetadata:%s\n%s\n", colorGreen, colorReset, string(b))
	}
	c.Logger.Printf("%s===================================================%s\n\n", colorMagenta, colorReset)
	return nil
}

// Extend with more system/test/demo methods as needed.

func (c *TheCatHasNoName) SyntaxHighlightJSON(jsonStr string) string {
	var out strings.Builder
	inString := false
	for i := 0; i < len(jsonStr); i++ {
		c := jsonStr[i]
		switch {
		case c == '"':
			if inString {
				inString = false
			} else {
				inString = true
			}
			color.New(color.FgHiBlue, color.Bold).Fprint(&out, string(c))
		case inString:
			color.New(color.FgHiGreen, color.Bold).Fprint(&out, string(c))
		case c == ':' || c == '{' || c == '}' || c == '[' || c == ']' || c == ',':
			color.New(color.FgHiBlack).Fprint(&out, string(c))
		case c >= '0' && c <= '9':
			color.New(color.FgHiYellow, color.Bold).Fprint(&out, string(c))
		case c == 't' || c == 'f': // booleans
			color.New(color.FgHiMagenta, color.Bold).Fprint(&out, string(c))
		case strings.HasPrefix(jsonStr[i:], "null"):
			color.New(color.FgHiRed, color.Bold).Fprint(&out, "null")
			i += 3 // skip 'ull'
		default:
			color.New(color.FgHiBlack).Fprint(&out, string(c))
		}
	}
	return out.String()
}
