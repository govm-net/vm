package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/govm-net/vm/vm"
	"github.com/spf13/cobra"
)

var (
	sourceFile string
	repoDir    string
	wasmDir    string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a smart contract",
	Long: `Deploy a smart contract to the VM system.
Example: vm-cli deploy -f contract.go -r /path/to/repo -w /path/to/wasm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 读取源代码文件
		code, err := os.ReadFile(sourceFile)
		if err != nil {
			return fmt.Errorf("failed to read source file: %w", err)
		}

		// api.DefaultContractConfig = func() api.ContractConfig {
		// 	return api.ContractConfig{
		// 		MaxGas:       1000000,
		// 		MaxCallDepth: 8,
		// 		MaxCodeSize:  1024 * 1024, // 1MB
		// 		AllowedImports: map[string]string{
		// 			"github.com/govm-net/vm": "v1.0.0",
		// 			// Additional allowed imports would be listed here
		// 		},
		// 		// Replaces: map[string]string{
		// 		// 	"github.com/govm-net/vm": "./../../../",
		// 		// },
		// 	}
		// }

		// 创建VM引擎配置
		config := &vm.Config{
			MaxContractSize:  1024 * 1024, // 1MB
			CodeManagerDir:   repoDir,
			WASIContractsDir: wasmDir,
		}

		slog.Info("deploying contract", "config", config)

		// 创建VM引擎
		engine, err := vm.NewEngine(config)
		if err != nil {
			return fmt.Errorf("failed to create VM engine: %w", err)
		}
		defer engine.Close()

		// 部署合约
		address, err := engine.DeployContract(code)
		if err != nil {
			return fmt.Errorf("failed to deploy contract: %w", err)
		}

		fmt.Printf("Contract deployed successfully!\n")
		fmt.Printf("Contract address: %s\n", address)
		fmt.Printf("Contract files are stored in: %s\n", filepath.Join(repoDir, address.String()))

		return nil
	},
}

func init() {
	deployCmd.Flags().StringVarP(&sourceFile, "file", "f", "", "Source file of the contract (required)")
	deployCmd.Flags().StringVarP(&repoDir, "repo", "r", "", "Repository directory")
	deployCmd.Flags().StringVarP(&wasmDir, "wasm", "w", "", "WASM directory")
	deployCmd.MarkFlagRequired("file")
	deployCmd.MarkFlagRequired("repo")
	deployCmd.MarkFlagRequired("wasm")
}
