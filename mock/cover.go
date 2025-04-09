package mock

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	GasImportPath      = "github.com/govm-net/vm/mock"
	GasPackageNickName = ""
	GasPackageName     = "mock"
	GasConsumeGasFunc  = "ConsumeGas"
)

// AddMockEnterExit adds mock.Enter/Exit to exported functions
func AddMockEnterExit(packageName string, code []byte) ([]byte, error) {
	// Parse the code
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", string(code), parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code: %w", err)
	}

	// Create a new file with the same package and imports
	newFile := &ast.File{
		Package: f.Package,
		Name:    f.Name,
		Imports: f.Imports,
	}

	// Add mock package import if not exists
	hasMockImport := false
	for _, imp := range f.Imports {
		if imp.Path.Value == `"`+GasImportPath+`"` {
			hasMockImport = true
			break
		}
	}
	if !hasMockImport {
		newFile.Imports = append(newFile.Imports, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: `"` + GasImportPath + `"`,
			},
		})
	}

	// Process each declaration
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Only process exported functions
			if d.Recv == nil && ast.IsExported(d.Name.Name) {
				// Create Enter/Exit statements
				enterStmt := &ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent(GasPackageName),
							Sel: ast.NewIdent("Enter"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: `core.AddressFromString("` + packageName + `")`,
							},
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: `"` + d.Name.Name + `"`,
							},
						},
					},
				}
				exitStmt := &ast.DeferStmt{
					Call: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent(GasPackageName),
							Sel: ast.NewIdent("Exit"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: `core.AddressFromString("` + packageName + `")`,
							},
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: `"` + d.Name.Name + `"`,
							},
						},
					},
				}

				// Add Enter and defer Exit at the beginning of the function
				d.Body.List = append([]ast.Stmt{enterStmt, exitStmt}, d.Body.List...)
			}
		}
		newFile.Decls = append(newFile.Decls, decl)
	}

	// Convert back to source code
	var buf strings.Builder
	if err := format.Node(&buf, fset, newFile); err != nil {
		return nil, fmt.Errorf("failed to format code: %w", err)
	}

	return []byte(buf.String()), nil
}

// AddGasConsumption adds gas consumption tracking to the code
func AddGasConsumption(packageName string, code []byte) ([]byte, error) {
	// Create temporary directory for cover files
	tmpDir, err := os.MkdirTemp("", "cover-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write source code to temp file
	srcFile := filepath.Join(tmpDir, "source.go")
	if err := os.WriteFile(srcFile, code, 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// Generate coverage code using go tool cover
	coverFile := filepath.Join(tmpDir, "source_cover.go")
	cmd := exec.Command("go", "tool", "cover", "-mode=atomic", "-var=vm_cover_atomic_", "-o", coverFile, srcFile)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate cover code: %w", err)
	}

	// Read the generated cover code
	coverCode, err := os.ReadFile(coverFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cover code: %w", err)
	}

	// Add mock package import
	importStmt := fmt.Sprintf("\nimport %q\n", GasImportPath)
	if GasPackageNickName != "" {
		importStmt = fmt.Sprintf("\nimport %s %q\n", GasPackageNickName, GasImportPath)
		GasPackageName = GasPackageNickName
	}

	// Replace coverage statements with gas consumption using regex
	codeStr := string(coverCode)
	re := regexp.MustCompile(`_cover_atomic_\.AddUint32\(&vm_cover_atomic_\.Count\[(\d+)\],\s*1\)`)
	codeStr = re.ReplaceAllString(codeStr, fmt.Sprintf("%s.%s(int64(vm_cover_atomic_.NumStmt[$1]))", GasPackageName, GasConsumeGasFunc))

	codeStr = strings.ReplaceAll(codeStr, "import _cover_atomic_ \"sync/atomic\"", importStmt)
	codeStr = strings.ReplaceAll(codeStr, "var _ = _cover_atomic_.LoadUint32", "")

	// Add mock.Enter/Exit to exported functions
	codeBytes, err := AddMockEnterExit(packageName, []byte(codeStr))
	if err != nil {
		return nil, fmt.Errorf("failed to add mock.Enter/Exit: %w", err)
	}

	return codeBytes, nil
}
