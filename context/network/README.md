# Network Context Module

The network context module provides a way to interact with blockchain context through HTTP requests. It consists of two main components:

1. **Client**: Implements the [BlockchainContext](file:///Volumes/ssd/myproject/gvm/vm/types/contract_functions.go#L124-L142) interface by forwarding calls to a remote server via HTTP
2. **Server**: Receives HTTP requests and forwards them to a database-backed context implementation

## Architecture

```
+----------------+         HTTP         +----------------+         +----------------+
|   Client App   | -------------------> | Network Server | ------> |  DB Context    |
+----------------+                      +----------------+         +----------------+
                                          |                              |
                                          |                              |
                                          v                              v
                                   +----------------+         +----------------+
                                   |  HTTP Handler  | ------> | DB Operations  |
                                   +----------------+         +----------------+
```

## Usage

### Server Side

To start a network context server:

```go
server := network.NewServer("localhost:8080")
err := server.Start()
if err != nil {
    log.Fatal(err)
}
```

The server will listen on the specified address and forward all requests to an internal database context.

### Client Side

To use the network context client:

```go
// Create a network context client
client := network.NewClient("http://localhost:8080")

// Use it like any other BlockchainContext
height := client.BlockHeight()
balance := client.Balance(someAddress)
// ... etc
```

You can also register it with the context registry:

```go
params := map[string]any{
    "server_url": "http://localhost:8080",
}
ctx, err := context.Get(context.NetContextType, params)
```

## Adapter for Context Interface

The module also provides an adapter that allows a [BlockchainContext](file:///Volumes/ssd/myproject/gvm/vm/types/contract_functions.go#L124-L142) to be used as a [Context](file:///Volumes/ssd/myproject/gvm/vm/types/contract_functions.go#L69-L82) interface implementation. This is useful when you need to use the simpler [Context](file:///Volumes/ssd/myproject/gvm/vm/types/contract_functions.go#L69-L82) interface but have a [BlockchainContext](file:///Volumes/ssd/myproject/gvm/vm/types/contract_functions.go#L124-L142) implementation.

```go
// Create a BlockchainContext (e.g., network client)
blockchainCtx := network.NewClient("http://localhost:8080")

// Create a contract address
contractAddr := core.AddressFromString("0x1234567890abcdef1234567890abcdef12345678")

// Create the adapter
adapter := network.NewBlockchainContextAdapter(blockchainCtx, contractAddr)

// Now you can use it as a Context
height := adapter.BlockHeight()
obj := adapter.CreateObject()
// ... etc
```

## API Endpoints

The server exposes the following HTTP endpoints that correspond to the [BlockchainContext](file:///Volumes/ssd/myproject/gvm/vm/types/contract_functions.go#L124-L142) interface methods:

- `POST /set_block_info`
- `POST /set_transaction_info`
- `GET /block_height`
- `GET /block_time`
- `GET /contract_address`
- `GET /transaction_hash`
- `POST /set_gas_limit`
- `GET /get_gas`
- `GET /sender`
- `POST /balance`
- `POST /transfer`
- `POST /create_object`
- `POST /create_object_with_id`
- `POST /get_object`
- `POST /get_object_with_owner`
- `POST /delete_object`
- `POST /call`
- `POST /log`
- `POST /object/set_owner`
- `POST /object/get`
- `POST /object/set`

## Implementation Details

The client serializes method calls and their parameters into JSON and sends them via HTTP POST requests to the corresponding endpoints on the server. The server deserializes these requests, calls the appropriate methods on its internal database context, and returns the results back to the client.