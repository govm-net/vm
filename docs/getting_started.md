# Getting Started with VM-CLI

VM-CLI is a command-line tool for deploying and executing smart contracts on the VM blockchain platform.

## Installation

```bash
go install github.com/govm-net/vm/cmd/vm-cli@latest
```

## Usage

### Deploying a Contract

To deploy a new contract:

```bash
vm-cli deploy -f <source_file> [-r <repo_dir>] [-w <wasm_dir>]
```

Parameters:
- `-f`: Source file of the contract (required)
- `-r`: Repository directory for storing contract files (default: "code")
- `-w`: WASM directory for storing compiled contracts (default: "wasm")

Example:
```bash
vm-cli deploy -f contract.go -r code -w wasm
```

### Executing a Contract

To execute a contract function:

```bash
vm-cli execute -c <contract_addr> -f <func_name> [-a <args_json>] -s <sender_addr> [-w <wasm_dir>]
```

Parameters:
- `-c`: Contract address (required)
- `-f`: Function name to execute (required)
- `-a`: Function arguments in JSON format (optional)
- `-s`: Transaction sender address (required)
- `-w`: WASM directory (default: "wasm")

Example:
```bash
vm-cli execute -c 0x1234... -f Transfer -a '[{"to":"0x1234...","amount":100}]' -s 0x9876... -w wasm
```

## Directory Structure

The CLI creates and manages the following directories:
- `code/`: Stores deployed contract files
- `wasm/`: Stores compiled WASM contracts

## Configuration

The VM engine is configured with the following default settings:
- Maximum contract size: 1MB
- Context type: DB (database-backed)
- Repository directory: "code"
- WASM contracts directory: "wasm"

## Error Handling

The CLI provides clear error messages for common issues:
- Missing required parameters
- Invalid file paths
- Contract deployment failures
- Contract execution errors

## Examples

1. Deploy a new contract:
```bash
vm-cli deploy -f mycontract.go
```

2. Execute a contract function:
```bash
vm-cli execute -c 0x1234... -f Transfer -a '[{"to":"0x1234...","amount":100}]' -s 0x9876...
```

## Notes

- All paths are relative to the current working directory unless specified as absolute paths
- The CLI automatically creates necessary directories if they don't exist
- Contract execution results are displayed in JSON format when available 