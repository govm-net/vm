package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vm-cli",
	Short: "VM management command line tool",
	Long: `VM management command line tool for deploying and executing WebAssembly smart contracts.
Complete documentation is available at https://github.com/govm-net/vm`,
}

func init() {
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(executeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
