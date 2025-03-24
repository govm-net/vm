package vm

import (
	"fmt"
	"io/ioutil"

	"github.com/wasmerio/wasmer-go/wasmer"
)

// InspectWasm 检查WASM文件并打印其导出和导入
func InspectWasm(wasmPath string) error {
	fmt.Printf("检查WASM文件: %s\n", wasmPath)

	// 读取WASM文件
	wasmBytes, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("无法读取WASM文件: %v", err)
	}

	// 创建Wasmer实例
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// 编译模块
	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		return fmt.Errorf("无法编译模块: %v", err)
	}

	// 获取导出内容
	exports := module.Exports()

	fmt.Println("\n导出的函数:")
	for _, export := range exports {
		exportType := export.Type()
		kind := exportType.Kind()
		kindStr := "未知"

		switch kind {
		case wasmer.FUNCTION:
			kindStr = "函数"
			// 直接获取函数类型，不解包为两个变量
			funcType := wasmer.NewFunctionType([]*wasmer.ValueType{}, []*wasmer.ValueType{})
			params := funcType.Params()
			results := funcType.Results()

			// 显示函数签名
			fmt.Printf("  - %s: %s (", export.Name(), kindStr)

			for i, param := range params {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(getTypeString(param.Kind()))
			}

			fmt.Print(") -> (")

			for i, result := range results {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(getTypeString(result.Kind()))
			}

			fmt.Println(")")

		case wasmer.GLOBAL:
			kindStr = "全局变量"
			fmt.Printf("  - %s: %s\n", export.Name(), kindStr)
		case wasmer.MEMORY:
			kindStr = "内存"
			fmt.Printf("  - %s: %s\n", export.Name(), kindStr)
		case wasmer.TABLE:
			kindStr = "表"
			fmt.Printf("  - %s: %s\n", export.Name(), kindStr)
		}
	}

	fmt.Println("\n导入需求:")
	imports := module.Imports()

	for _, imp := range imports {
		fmt.Printf("  - 模块: '%s', 名称: '%s', 类型: %s\n",
			imp.Module(), imp.Name(), getImportKindString(imp.Type().Kind()))
	}

	return nil
}

func getTypeString(kind wasmer.ValueKind) string {
	switch kind {
	case wasmer.I32:
		return "i32"
	case wasmer.I64:
		return "i64"
	case wasmer.F32:
		return "f32"
	case wasmer.F64:
		return "f64"
	default:
		return "unknown"
	}
}

func getImportKindString(kind wasmer.ExternKind) string {
	switch kind {
	case wasmer.FUNCTION:
		return "函数"
	case wasmer.GLOBAL:
		return "全局变量"
	case wasmer.MEMORY:
		return "内存"
	case wasmer.TABLE:
		return "表"
	default:
		return "未知"
	}
}
