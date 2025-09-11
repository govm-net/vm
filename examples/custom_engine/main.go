package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/govm-net/vm" // 修复导入路径
	"github.com/govm-net/vm/context/memory"
	"github.com/govm-net/vm/interfaces"
)

func main() {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "custom-engine-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建配置
	config := &interfaces.Config{
		MaxContractSize:  1024 * 1024, // 1MB
		WASIContractsDir: filepath.Join(tmpDir, "contracts"),
		CodeManagerDir:   filepath.Join(tmpDir, "code"),
		ContextType:      "memory",
	}

	// 创建自定义上下文
	customContext := memory.NewBlockchainContext(nil)

	// 使用自定义模块创建引擎
	engine, err := vm.NewEngine(config, vm.WithContext(customContext))
	if err != nil {
		panic(err)
	}
	defer engine.Close()

	fmt.Println("成功创建自定义引擎实例")
	fmt.Printf("使用的上下文类型: %T\n", engine.GetContext())
}
