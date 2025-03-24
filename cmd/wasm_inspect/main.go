package main

import (
	"fmt"
	"os"

	"github.com/wasmerio/wasmer-go/wasmer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run cmd/wasm_inspect/main.go <wasm文件路径>")
		os.Exit(1)
	}

	wasmPath := os.Args[1]
	fmt.Printf("检查WASM文件: %s\n", wasmPath)

	// 读取WASM文件
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		fmt.Printf("无法读取WASM文件: %v\n", err)
		os.Exit(1)
	}

	// 创建Wasmer实例
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// 编译模块
	module, err := wasmer.NewModule(store, wasmBytes)
	if err != nil {
		fmt.Printf("无法编译模块: %v\n", err)
		os.Exit(1)
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
			fmt.Printf("  - %s: %s\n", export.Name(), kindStr)

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
