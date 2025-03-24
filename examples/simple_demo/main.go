// 演示如何使用MiniVM部署和执行合约
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/govm-net/vm/vm" // 引入vm包
)

func main() {
	// 创建临时目录用于存储合约
	tempDir, err := ioutil.TempDir("", "govm-demo")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir) // 程序结束时清理

	// 初始化虚拟机
	miniVM, err := vm.NewMiniVM(tempDir)
	if err != nil {
		fmt.Printf("初始化虚拟机失败: %v\n", err)
		os.Exit(1)
	}

	// 设置区块信息
	miniVM.SetBlockInfo(100, 1624553600) // 区块高度100，时间戳

	// 读取合约WASM文件
	// 注意：需要先使用tinygo编译Go源代码为WASM
	// tinygo build -o contract.wasm -target wasm contract.go
	wasmPath := filepath.Join("examples", "simple_contract", "contract.wasm")
	wasmCode, err := ioutil.ReadFile(wasmPath)
	if err != nil {
		fmt.Printf("读取WASM文件失败: %v\n", err)
		fmt.Println("请先使用TinyGo编译合约:")
		fmt.Println("tinygo build -o contract.wasm -target wasm contract.go")
		os.Exit(1)
	}

	// 部署合约
	contractAddr, err := miniVM.DeployContract(wasmCode)
	if err != nil {
		fmt.Printf("部署合约失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("合约已部署，地址: %s\n", contractAddr)

	// 初始化合约
	_, err = miniVM.ExecuteContract(contractAddr, "user1", "Initialize")
	if err != nil {
		fmt.Printf("初始化合约失败: %v\n", err)
		os.Exit(1)
	}

	// 增加计数器
	result, err := miniVM.ExecuteContract(contractAddr, "user1", "Increment", uint64(5))
	if err != nil {
		fmt.Printf("调用Increment失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("调用Increment结果: %v\n", result)

	// 获取计数器值
	result, err = miniVM.ExecuteContract(contractAddr, "user1", "GetCounter")
	if err != nil {
		fmt.Printf("调用GetCounter失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("当前计数器值: %v\n", result)

	// 再次增加计数器
	result, err = miniVM.ExecuteContract(contractAddr, "user1", "Increment", uint64(10))
	if err != nil {
		fmt.Printf("调用Increment失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("调用Increment结果: %v\n", result)

	// 获取计数器值
	result, err = miniVM.ExecuteContract(contractAddr, "user1", "GetCounter")
	if err != nil {
		fmt.Printf("调用GetCounter失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("当前计数器值: %v\n", result)

	// 重置计数器
	_, err = miniVM.ExecuteContract(contractAddr, "user1", "Reset")
	if err != nil {
		fmt.Printf("调用Reset失败: %v\n", err)
		os.Exit(1)
	}

	// 获取计数器值
	result, err = miniVM.ExecuteContract(contractAddr, "user1", "GetCounter")
	if err != nil {
		fmt.Printf("调用GetCounter失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("重置后计数器值: %v\n", result)

	fmt.Println("演示完成!")
}
