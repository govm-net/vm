package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/govm-net/vm/context"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

// Client implements the BlockchainContext interface by forwarding calls to a remote server
type Client struct {
	serverURL string
	client    *http.Client
	contract  string
	tx        string
}

// NewClient creates a new network client
func NewClient(serverURL string) *Client {
	return &Client{
		serverURL: serverURL,
		client:    &http.Client{},
	}
}

func init() {
	context.Register(context.NetContextType, NewNetworkContext)
}

// NewNetworkContext creates a new network-backed blockchain context
func NewNetworkContext(params map[string]any) types.BlockchainContext {
	serverURL := os.Getenv("VM_CONTEXT_SERVER_URL")
	if url, ok := params["server_url"].(string); ok && url != "" {
		serverURL = url
	}
	out := NewClient(serverURL)
	out.contract = os.Getenv("VM_CONTEXT_CONTRACT_ADDRESS")
	out.tx = os.Getenv("VM_CONTEXT_TRANSACTION")

	return out
}

// makeRequest sends an HTTP request to the server
func (c *Client) makeRequest(endpoint string, reqData interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.serverURL, endpoint)

	var reqBody []byte
	var err error

	if reqData != nil {
		reqBody, err = json.Marshal(reqData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %v", err)
		}
	}
	// add headers
	reqHeaders := http.Header{}
	reqHeaders.Set("X-Contract-Address", c.contract)
	reqHeaders.Set("X-Transaction", c.tx)
	reqHeaders.Set("Content-Type", "application/json")
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	request.Header = reqHeaders

	resp, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SetBlockInfo(height uint64, time int64, hash core.Hash) error {
	req := struct {
		Height uint64    `json:"height"`
		Time   int64     `json:"time"`
		Hash   core.Hash `json:"hash"`
	}{
		Height: height,
		Time:   time,
		Hash:   hash,
	}

	_, err := c.makeRequest("/set_block_info", req)
	return err
}

func (c *Client) SetTransactionInfo(hash core.Hash, from types.Address, to types.Address, value uint64) error {
	req := struct {
		Hash  core.Hash     `json:"hash"`
		From  types.Address `json:"from"`
		To    types.Address `json:"to"`
		Value uint64        `json:"value"`
	}{
		Hash:  hash,
		From:  from,
		To:    to,
		Value: value,
	}

	_, err := c.makeRequest("/set_transaction_info", req)
	return err
}

func (c *Client) BlockHeight() uint64 {
	data, err := c.makeRequest("/block_height", nil)
	if err != nil {
		// Return default value in case of error
		return 0
	}

	var result uint64
	err = json.Unmarshal(data, &result)
	if err != nil {
		return 0
	}

	return result
}

func (c *Client) BlockTime() int64 {
	data, err := c.makeRequest("/block_time", nil)
	if err != nil {
		// Return default value in case of error
		return 0
	}

	var result int64
	err = json.Unmarshal(data, &result)
	if err != nil {
		return 0
	}

	return result
}

func (c *Client) ContractAddress() types.Address {
	data, err := c.makeRequest("/contract_address", nil)
	if err != nil {
		// Return zero address in case of error
		return types.Address{}
	}

	var result types.Address
	err = json.Unmarshal(data, &result)
	if err != nil {
		return types.Address{}
	}

	return result
}

func (c *Client) TransactionHash() core.Hash {
	data, err := c.makeRequest("/transaction_hash", nil)
	if err != nil {
		// Return zero hash in case of error
		return core.Hash{}
	}

	var result core.Hash
	err = json.Unmarshal(data, &result)
	if err != nil {
		return core.Hash{}
	}

	return result
}

func (c *Client) SetGasLimit(limit int64) {
	req := struct {
		Limit int64 `json:"limit"`
	}{
		Limit: limit,
	}

	_, _ = c.makeRequest("/set_gas_limit", req)
}

func (c *Client) GetGas() int64 {
	data, err := c.makeRequest("/get_gas", nil)
	if err != nil {
		// Return default value in case of error
		return 0
	}

	var result int64
	err = json.Unmarshal(data, &result)
	if err != nil {
		return 0
	}

	return result
}

func (c *Client) Sender() types.Address {
	data, err := c.makeRequest("/sender", nil)
	if err != nil {
		// Return zero address in case of error
		return types.Address{}
	}

	var result types.Address
	err = json.Unmarshal(data, &result)
	if err != nil {
		return types.Address{}
	}

	return result
}

func (c *Client) Balance(addr types.Address) uint64 {
	req := struct {
		Address types.Address `json:"address"`
	}{
		Address: addr,
	}

	data, err := c.makeRequest("/balance", req)
	if err != nil {
		// Return default value in case of error
		return 0
	}

	var result uint64
	err = json.Unmarshal(data, &result)
	if err != nil {
		return 0
	}

	return result
}

func (c *Client) Transfer(contract, from, to types.Address, amount uint64) error {
	req := struct {
		Contract types.Address `json:"contract"`
		From     types.Address `json:"from"`
		To       types.Address `json:"to"`
		Amount   uint64        `json:"amount"`
	}{
		Contract: contract,
		From:     from,
		To:       to,
		Amount:   amount,
	}

	_, err := c.makeRequest("/transfer", req)
	return err
}

func (c *Client) CreateObject(contract types.Address) (types.VMObject, error) {
	req := struct {
		Contract types.Address `json:"contract"`
	}{
		Contract: contract,
	}

	data, err := c.makeRequest("/create_object", req)
	if err != nil {
		return nil, err
	}

	var result ObjectResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal object response: %v", err)
	}

	return &NetworkObject{
		client:   c,
		id:       result.ID,
		owner:    result.Owner,
		contract: result.Contract,
	}, nil
}

func (c *Client) CreateObjectWithID(contract types.Address, id core.ObjectID) (types.VMObject, error) {
	req := struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
	}{
		Contract: contract,
		ID:       id,
	}

	data, err := c.makeRequest("/create_object_with_id", req)
	if err != nil {
		return nil, err
	}

	var result ObjectResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal object response: %v", err)
	}

	return &NetworkObject{
		client:   c,
		id:       result.ID,
		owner:    result.Owner,
		contract: result.Contract,
	}, nil
}

func (c *Client) GetObject(contract types.Address, id core.ObjectID) (types.VMObject, error) {
	req := struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
	}{
		Contract: contract,
		ID:       id,
	}

	data, err := c.makeRequest("/get_object", req)
	if err != nil {
		return nil, err
	}

	var result ObjectResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal object response: %v", err)
	}

	return &NetworkObject{
		client:   c,
		id:       result.ID,
		owner:    result.Owner,
		contract: result.Contract,
	}, nil
}

func (c *Client) GetObjectWithOwner(contract types.Address, owner types.Address) (types.VMObject, error) {
	req := struct {
		Contract types.Address `json:"contract"`
		Owner    types.Address `json:"owner"`
	}{
		Contract: contract,
		Owner:    owner,
	}

	data, err := c.makeRequest("/get_object_with_owner", req)
	if err != nil {
		return nil, err
	}

	var result ObjectResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal object response: %v", err)
	}

	return &NetworkObject{
		client:   c,
		id:       result.ID,
		owner:    result.Owner,
		contract: result.Contract,
	}, nil
}

func (c *Client) DeleteObject(contract types.Address, id core.ObjectID) error {
	req := struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
	}{
		Contract: contract,
		ID:       id,
	}

	_, err := c.makeRequest("/delete_object", req)
	return err
}

func (c *Client) Call(caller types.Address, contract types.Address, function string, args ...any) ([]byte, error) {
	req := struct {
		Caller   types.Address `json:"caller"`
		Contract types.Address `json:"contract"`
		Function string        `json:"function"`
		Args     []any         `json:"args"`
	}{
		Caller:   caller,
		Contract: contract,
		Function: function,
		Args:     args,
	}

	data, err := c.makeRequest("/call", req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *Client) Log(contract types.Address, eventName string, keyValues ...any) {
	req := struct {
		Contract  types.Address `json:"contract"`
		EventName string        `json:"event_name"`
		KeyValues []any         `json:"key_values"`
	}{
		Contract:  contract,
		EventName: eventName,
		KeyValues: keyValues,
	}

	_, _ = c.makeRequest("/log", req)
}

// ObjectResponse represents the response structure for object-related operations
type ObjectResponse struct {
	ID       core.ObjectID `json:"id"`
	Owner    types.Address `json:"owner"`
	Contract types.Address `json:"contract"`
}

// NetworkObject implements the VMObject interface for network-based objects
type NetworkObject struct {
	client   *Client
	id       core.ObjectID
	owner    types.Address
	contract types.Address
}

func (o *NetworkObject) ID() core.ObjectID {
	return o.id
}

func (o *NetworkObject) Owner() types.Address {
	return o.owner
}

func (o *NetworkObject) Contract() types.Address {
	return o.contract
}

func (o *NetworkObject) SetOwner(contract, sender, addr types.Address) error {
	req := struct {
		Contract types.Address `json:"contract"`
		Sender   types.Address `json:"sender"`
		Addr     types.Address `json:"addr"`
		ID       core.ObjectID `json:"id"`
	}{
		Contract: contract,
		Sender:   sender,
		Addr:     addr,
		ID:       o.id,
	}

	_, err := o.client.makeRequest("/object/set_owner", req)
	if err != nil {
		return err
	}

	// Update local state
	o.owner = addr
	return nil
}

func (o *NetworkObject) Get(contract types.Address, field string) ([]byte, error) {
	req := struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
		Field    string        `json:"field"`
	}{
		Contract: contract,
		ID:       o.id,
		Field:    field,
	}

	data, err := o.client.makeRequest("/object/get", req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (o *NetworkObject) Set(contract, sender types.Address, field string, value []byte) error {
	req := struct {
		Contract types.Address `json:"contract"`
		Sender   types.Address `json:"sender"`
		ID       core.ObjectID `json:"id"`
		Field    string        `json:"field"`
		Value    []byte        `json:"value"`
	}{
		Contract: contract,
		Sender:   sender,
		ID:       o.id,
		Field:    field,
		Value:    value,
	}

	_, err := o.client.makeRequest("/object/set", req)
	return err
}
