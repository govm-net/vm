package abi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// HandlerGenerator generates handler functions from ABI
type HandlerGenerator struct {
	abi *ABI
}

var EnableFormatAfterGenerate = true

// NewHandlerGenerator creates a new handler generator
func NewHandlerGenerator(abi *ABI) *HandlerGenerator {
	return &HandlerGenerator{
		abi: abi,
	}
}

// Format formats the generated code using gofmt
func Format(code string) (string, error) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "vm-handler-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create temporary file
	tmpFile := filepath.Join(tmpDir, "handler.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Run gofmt
	cmd := exec.Command("gofmt", "-s", "-w", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gofmt failed: %s: %w", string(output), err)
	}

	// Read formatted code
	formattedCode, err := os.ReadFile(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to read formatted code: %w", err)
	}

	return string(formattedCode), nil
}

// GenerateHandlers generates handler functions for all exported functions
func (g *HandlerGenerator) GenerateHandlers() string {
	var sb strings.Builder

	// Add package name and imports
	sb.WriteString(fmt.Sprintf("package %s\n\n", g.abi.PackageName))
	sb.WriteString("import (\n")

	// Collect all required imports
	imports := make(map[string]string) // path -> alias
	// Add basic imports
	// imports["github.com/govm-net/vm/core"] = ""
	imports["fmt"] = ""
	imports["encoding/json"] = ""
	for _, fn := range g.abi.Functions {

		// Check input parameter types
		for _, input := range fn.Inputs {
			if imp := g.findImportForType(input.Type); imp != nil {
				if imp.Name != "" {
					imports[imp.Path] = imp.Name
				} else {
					imports[imp.Path] = ""
				}
			}
		}

		// Check output parameter types
		for _, output := range fn.Outputs {
			if imp := g.findImportForType(output.Type); imp != nil {
				if imp.Name != "" {
					imports[imp.Path] = imp.Name
				} else {
					imports[imp.Path] = ""
				}
			}
		}
	}

	// Add collected imports
	for path, alias := range imports {
		if alias != "" {
			sb.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, path))
		} else {
			sb.WriteString(fmt.Sprintf("\t\"%s\"\n", path))
		}
	}
	sb.WriteString(")\n\n")

	// Import json/fmt to avoid compilation failure in wasm
	sb.WriteString(`
func init(){
	if false{
		fmt.Println("init")
		json.Marshal(nil)
	}
}

`)

	// Generate parameter struct and handler for each exported function
	for _, fn := range g.abi.Functions {
		sb.WriteString(g.generateParamStruct(fn))
		sb.WriteString(g.generateHandler(fn))
	}

	return sb.String()
}

// findImportForType finds the corresponding import for a type
func (g *HandlerGenerator) findImportForType(typeStr string) *Import {
	// Handle pointer type
	typeStr = strings.TrimPrefix(typeStr, "*")

	// Handle array type
	typeStr = strings.TrimPrefix(typeStr, "[]")

	// Handle type with package name
	if strings.Contains(typeStr, ".") {
		parts := strings.Split(typeStr, ".")
		pkgName := parts[0]

		// Find matching package in imports list
		for _, imp := range g.abi.Imports {
			// If import has alias, check if alias matches
			if imp.Name == pkgName {
				return &imp
			}

			// If no alias, check if last part of path matches
			if imp.Name == "" {
				pathParts := strings.Split(imp.Path, "/")
				if pathParts[len(pathParts)-1] == pkgName {
					return &imp
				}
			}
		}
	}

	return nil
}

// generateParamStruct generates a parameter struct for a function
func (g *HandlerGenerator) generateParamStruct(fn Function) string {
	var sb strings.Builder

	// Generate struct definition
	sb.WriteString(fmt.Sprintf("type %sParams struct {\n", fn.Name))
	for _, input := range fn.Inputs {
		// Use json tag to match parameter name and add omitempty
		sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s,omitempty\"`\n",
			cases.Title(language.English).String(input.Name), input.Type, input.Name))
	}
	sb.WriteString("}\n\n")

	return sb.String()
}

// generateHandler generates a handler function for a given function
func (g *HandlerGenerator) generateHandler(fn Function) string {
	var sb strings.Builder

	// Generate function signature
	sb.WriteString(fmt.Sprintf("func handle%s(params []byte) (any, error) {\n", fn.Name))

	// Generate parameter parsing code
	if len(fn.Inputs) > 0 {
		sb.WriteString(fmt.Sprintf("\tvar args %sParams\n", fn.Name))
		sb.WriteString("\tif len(params) > 0 {\n")
		sb.WriteString("\t\tif err := json.Unmarshal(params, &args); err != nil {\n")
		sb.WriteString("\t\t\treturn nil, fmt.Errorf(\"failed to unmarshal params: %w\", err)\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t}\n\n")
	}

	// Check if return value variable is needed
	hasReturnValue := len(fn.Outputs) > 0
	if hasReturnValue {
		// Generate return value variable declaration
		sb.WriteString("\t")
		for i, output := range fn.Outputs {
			if i > 0 {
				sb.WriteString(", ")
			}
			if output.Name != "" {
				sb.WriteString(output.Name)
			} else {
				sb.WriteString(fmt.Sprintf("result%d", i))
			}
		}
		sb.WriteString(" := ")
	}

	// Generate function call
	sb.WriteString(fn.Name)
	sb.WriteString("(")

	// Add parameters
	firstParam := true
	for _, input := range fn.Inputs {
		if !firstParam {
			sb.WriteString(", ")
		}
		firstParam = false

		sb.WriteString(fmt.Sprintf("args.%s", cases.Title(language.English).String(input.Name)))
	}
	sb.WriteString(")\n\n")

	// Handle return values
	if hasReturnValue && len(fn.Outputs) > 1 {
		sb.WriteString("\tresults := make([]any, 0)\n")
		for i, output := range fn.Outputs {
			if output.Name != "" {
				sb.WriteString(fmt.Sprintf("\tresults = append(results, %s)\n", output.Name))
			} else {
				sb.WriteString(fmt.Sprintf("\tresults = append(results, result%d)\n", i))
			}
		}
		sb.WriteString("\treturn results, nil\n")
	} else if hasReturnValue && len(fn.Outputs) == 1 {
		sb.WriteString("\treturn ")
		if fn.Outputs[0].Name != "" {
			sb.WriteString(fmt.Sprintf("%s, nil\n", fn.Outputs[0].Name))
		} else {
			sb.WriteString("result0, nil\n")
		}
	} else {
		sb.WriteString("\treturn nil, nil\n")
	}
	sb.WriteString("}\n\n")
	return sb.String()
}

// GenerateHandlerFile generates a complete handler file
func GenerateHandlerFile(abi *ABI) (string, error) {
	generator := NewHandlerGenerator(abi)
	code := generator.GenerateHandlers()
	if !EnableFormatAfterGenerate {
		return code, nil
	}

	// Format code
	formattedCode, err := Format(code)
	if err != nil {
		return "", fmt.Errorf("failed to format code: %w", err)
	}

	return formattedCode, nil
}
