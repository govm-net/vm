package mock

import (
	"fmt"
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
	codeStr = re.ReplaceAllString(codeStr, fmt.Sprintf("%s.%s(vm_cover_atomic_.NumStmt[$1])", GasPackageName, GasConsumeGasFunc))

	codeStr = strings.ReplaceAll(codeStr, "; import _cover_atomic_ \"sync/atomic\"", importStmt)
	codeStr = strings.ReplaceAll(codeStr, "var _ = _cover_atomic_.LoadUint32", "")

	// Add import statement after package declaration
	// packageEnd := strings.Index(codeStr, "\n") + 1
	// codeStr = codeStr[:packageEnd] + importStmt + codeStr[packageEnd:]

	return []byte(codeStr), nil
}
