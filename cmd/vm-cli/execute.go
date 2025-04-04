package main

import (
	"encoding/json"
	"fmt"

	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/vm"
	"github.com/govm-net/vm/wasi"
	"github.com/spf13/cobra"
)

var (
	contractAddr string
	funcName     string
	argsJSON     string
	sender       string
)

var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute a smart contract function",
	Long: `Execute a function in a deployed smart contract.
Example: vm-cli execute -c 1234567890abcdef1234567890abcdef12345678 -f Transfer -a '[{"to":"0x1234...","amount":100}]' -s 0x9876...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 解析合约地址
		address := core.AddressFromString(contractAddr)

		// 创建VM引擎配置
		config := &vm.Config{
			MaxContractSize:  1024 * 1024, // 1MB
			WASIContractsDir: repoDir,
		}

		// 创建VM引擎
		engine, err := vm.NewEngine(config)
		if err != nil {
			return fmt.Errorf("failed to create VM engine: %w", err)
		}
		defer engine.Close()

		// 创建区块链上下文
		ctx := wasi.NewDefaultBlockchainContext()
		engine = engine.WithContext(ctx)

		// 设置执行上下文
		senderAddr := core.AddressFromString(sender)
		ctx.SetExecutionContext(address, senderAddr)

		// 解析参数
		var params []byte
		if argsJSON != "" {
			params = []byte(argsJSON)
		}

		// 执行合约函数
		result, err := engine.Execute(address, funcName, params)
		if err != nil {
			return fmt.Errorf("failed to execute contract: %w", err)
		}

		// 打印执行结果
		if result != nil {
			resultJSON, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal result: %w", err)
			}
			fmt.Printf("Execution result:\n%s\n", string(resultJSON))
		} else {
			fmt.Printf("Function executed successfully with no return value\n")
		}

		return nil
	},
}

func init() {
	executeCmd.Flags().StringVarP(&contractAddr, "contract", "c", "", "Contract address (required)")
	executeCmd.Flags().StringVarP(&funcName, "function", "f", "", "Function name to execute (required)")
	executeCmd.Flags().StringVarP(&argsJSON, "args", "a", "", "Function arguments in JSON format")
	executeCmd.Flags().StringVarP(&sender, "sender", "s", "", "Transaction sender address (required)")
	executeCmd.Flags().StringVarP(&repoDir, "repo", "r", "", "Repository directory (required)")

	executeCmd.MarkFlagRequired("contract")
	executeCmd.MarkFlagRequired("function")
	executeCmd.MarkFlagRequired("sender")
	executeCmd.MarkFlagRequired("repo")
}
