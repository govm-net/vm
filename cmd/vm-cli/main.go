package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// 定义子命令
	deployCommand := flag.NewFlagSet("deploy", flag.ExitOnError)
	executeCommand := flag.NewFlagSet("execute", flag.ExitOnError)

	// deploy 命令的参数
	sourceFile := deployCommand.String("f", "", "Source file of the contract")
	repoDir := deployCommand.String("r", "code", "Repository directory")
	wasmDir := deployCommand.String("w", "wasm", "WASM directory")

	// execute 命令的参数
	contractAddr := executeCommand.String("c", "", "Contract address")
	funcName := executeCommand.String("f", "", "Function name to execute")
	argsJSON := executeCommand.String("a", "", "Function arguments in JSON format")
	sender := executeCommand.String("s", "", "Transaction sender address")
	wasmDir2 := executeCommand.String("w", "wasm", "WASM directory")

	// 检查参数
	if len(os.Args) < 2 {
		fmt.Println("expected 'deploy' or 'execute' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "deploy":
		deployCommand.Parse(os.Args[2:])
		if err := runDeploy(*sourceFile, *repoDir, *wasmDir); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "execute":
		executeCommand.Parse(os.Args[2:])
		if err := runExecute(*contractAddr, *funcName, *argsJSON, *sender, *wasmDir2); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		fmt.Println("expected 'deploy' or 'execute' subcommands")
		os.Exit(1)
	}
}
