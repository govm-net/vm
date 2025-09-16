package compiler

import (
	"fmt"
	"strings"

	"github.com/govm-net/vm/abi"
	"github.com/govm-net/vm/config"
	"github.com/govm-net/vm/core"
)

type Args struct {
	PackageName     string `json:"package_name,omitempty"`
	ContractAddress string
	Functions       []Function `json:"functions,omitempty"`
}

// Import represents an import statement in the contract
type Import struct {
	Path string `json:"path,omitempty"` // 导入路径
	Name string `json:"name,omitempty"` // 别名（如果有）
}

// Function represents a function in the contract
type Function struct {
	Name       string `json:"name,omitempty"`
	ArgsString string
	Inputs     []Parameter `json:"inputs,omitempty"`
	Outputs    []Parameter `json:"outputs,omitempty"`
}

// Parameter represents a function parameter or event field
type Parameter struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

func abi2args(address core.Address, abi *abi.ABI) Args {
	var out Args
	out.ContractAddress = address.String()
	out.PackageName = config.ContractPackageBase + "/" + address.String()
	for _, it := range abi.Functions {
		var f Function
		f.Name = it.Name
		for _, input := range it.Inputs {
			f.ArgsString += "args." + upperFirst(input.Name) + ","
			f.Inputs = append(f.Inputs, Parameter{
				Name: upperFirst(input.Name),
				Type: input.Type,
			})
		}
		for i, output := range it.Outputs {
			lo := Parameter{
				Name: output.Name,
				Type: output.Type,
			}
			if len(lo.Name) == 0 {
				lo.Name = fmt.Sprintf("rst%d", i)
			}
			f.Outputs = append(f.Outputs, lo)
		}
		f.ArgsString = strings.Trim(f.ArgsString, ",")
		out.Functions = append(out.Functions, f)
	}

	return out
}

func upperFirst(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
