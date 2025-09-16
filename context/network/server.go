package network

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/govm-net/vm/context"
	"github.com/govm-net/vm/core"
	"github.com/govm-net/vm/types"
)

// Server implements an HTTP server that forwards requests to a db.Context
type Server struct {
	ctx    types.BlockchainContext
	server *http.Server
}

// NewServer creates a new network server
func NewServer(addr string, ctxType context.ContextType, params map[string]any) *Server {
	// Create a db context for the server to use
	dbCtx, err := context.Get(ctxType, params)
	if err != nil {
		panic(err)
	}

	server := &Server{
		ctx: dbCtx,
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server
}

// setupRoutes registers all HTTP endpoints
func (s *Server) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/set_block_info", s.handleSetBlockInfo)
	mux.HandleFunc("/set_transaction_info", s.handleSetTransactionInfo)
	mux.HandleFunc("/block_height", s.handleBlockHeight)
	mux.HandleFunc("/block_time", s.handleBlockTime)
	mux.HandleFunc("/contract_address", s.handleContractAddress)
	mux.HandleFunc("/transaction_hash", s.handleTransactionHash)
	mux.HandleFunc("/set_gas_limit", s.handleSetGasLimit)
	mux.HandleFunc("/get_gas", s.handleGetGas)
	mux.HandleFunc("/sender", s.handleSender)
	mux.HandleFunc("/balance", s.handleBalance)
	mux.HandleFunc("/transfer", s.handleTransfer)
	mux.HandleFunc("/create_object", s.handleCreateObject)
	mux.HandleFunc("/create_object_with_id", s.handleCreateObjectWithID)
	mux.HandleFunc("/get_object", s.handleGetObject)
	mux.HandleFunc("/get_object_with_owner", s.handleGetObjectWithOwner)
	mux.HandleFunc("/delete_object", s.handleDeleteObject)
	mux.HandleFunc("/call", s.handleCall)
	mux.HandleFunc("/log", s.handleLog)
	mux.HandleFunc("/object/set_owner", s.handleObjectSetOwner)
	mux.HandleFunc("/object/get", s.handleObjectGet)
	mux.HandleFunc("/object/set", s.handleObjectSet)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	return s.server.Close()
}

// Helper function to send JSON response
func (s *Server) sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// Helper function to read JSON request
func (s *Server) readJSONRequest(r *http.Request, data interface{}) error {
	return json.NewDecoder(r.Body).Decode(data)
}

func (s *Server) handleSetBlockInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Height uint64    `json:"height"`
		Time   int64     `json:"time"`
		Hash   core.Hash `json:"hash"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	err := s.ctx.SetBlockInfo(req.Height, req.Time, req.Hash)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to set block info: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, true)
}

func (s *Server) handleSetTransactionInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hash  core.Hash     `json:"hash"`
		From  types.Address `json:"from"`
		To    types.Address `json:"to"`
		Value uint64        `json:"value"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	err := s.ctx.SetTransactionInfo(req.Hash, req.From, req.To, req.Value)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to set transaction info: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, true)
}

func (s *Server) handleBlockHeight(w http.ResponseWriter, r *http.Request) {
	height := s.ctx.BlockHeight()
	s.sendJSONResponse(w, height)
}

func (s *Server) handleBlockTime(w http.ResponseWriter, r *http.Request) {
	time := s.ctx.BlockTime()
	s.sendJSONResponse(w, time)
}

func (s *Server) handleContractAddress(w http.ResponseWriter, r *http.Request) {
	addr := s.ctx.ContractAddress()
	s.sendJSONResponse(w, addr)
}

func (s *Server) handleTransactionHash(w http.ResponseWriter, r *http.Request) {
	hash := s.ctx.TransactionHash()
	s.sendJSONResponse(w, hash)
}

func (s *Server) handleSetGasLimit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Limit int64 `json:"limit"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	s.ctx.SetGasLimit(req.Limit)
	s.sendJSONResponse(w, true)
}

func (s *Server) handleGetGas(w http.ResponseWriter, r *http.Request) {
	gas := s.ctx.GetGas()
	s.sendJSONResponse(w, gas)
}

func (s *Server) handleSender(w http.ResponseWriter, r *http.Request) {
	sender := s.ctx.Sender()
	s.sendJSONResponse(w, sender)
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address types.Address `json:"address"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	balance := s.ctx.Balance(req.Address)
	s.sendJSONResponse(w, balance)
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		From     types.Address `json:"from"`
		To       types.Address `json:"to"`
		Amount   uint64        `json:"amount"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	err := s.ctx.Transfer(req.Contract, req.From, req.To, req.Amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to transfer: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, true)
}

func (s *Server) handleCreateObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	obj, err := s.ctx.CreateObject(req.Contract)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create object: %v", err), http.StatusInternalServerError)
		return
	}

	response := ObjectResponse{
		ID:       obj.ID(),
		Owner:    obj.Owner(),
		Contract: obj.Contract(),
	}

	s.sendJSONResponse(w, response)
}

func (s *Server) handleCreateObjectWithID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	obj, err := s.ctx.CreateObjectWithID(req.Contract, req.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create object: %v", err), http.StatusInternalServerError)
		return
	}

	response := ObjectResponse{
		ID:       obj.ID(),
		Owner:    obj.Owner(),
		Contract: obj.Contract(),
	}

	s.sendJSONResponse(w, response)
}

func (s *Server) handleGetObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	obj, err := s.ctx.GetObject(req.Contract, req.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get object: %v", err), http.StatusInternalServerError)
		return
	}

	response := ObjectResponse{
		ID:       obj.ID(),
		Owner:    obj.Owner(),
		Contract: obj.Contract(),
	}

	s.sendJSONResponse(w, response)
}

func (s *Server) handleGetObjectWithOwner(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		Owner    types.Address `json:"owner"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	obj, err := s.ctx.GetObjectWithOwner(req.Contract, req.Owner)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get object: %v", err), http.StatusInternalServerError)
		return
	}

	response := ObjectResponse{
		ID:       obj.ID(),
		Owner:    obj.Owner(),
		Contract: obj.Contract(),
	}

	s.sendJSONResponse(w, response)
}

func (s *Server) handleDeleteObject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	err := s.ctx.DeleteObject(req.Contract, req.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete object: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, true)
}

func (s *Server) handleCall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Caller   types.Address `json:"caller"`
		Contract types.Address `json:"contract"`
		Function string        `json:"function"`
		Args     []any         `json:"args"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	result, err := s.ctx.Call(req.Caller, req.Contract, req.Function, req.Args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to call: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, result)
}

func (s *Server) handleLog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract  types.Address `json:"contract"`
		EventName string        `json:"event_name"`
		KeyValues []any         `json:"key_values"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	s.ctx.Log(req.Contract, req.EventName, req.KeyValues...)
	s.sendJSONResponse(w, true)
}

func (s *Server) handleObjectSetOwner(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		Sender   types.Address `json:"sender"`
		Addr     types.Address `json:"addr"`
		ID       core.ObjectID `json:"id"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	// For simplicity, we're assuming the object exists and can be retrieved
	// In a real implementation, you might want to handle this differently
	obj, err := s.ctx.GetObject(req.Contract, req.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get object: %v", err), http.StatusInternalServerError)
		return
	}

	err = obj.SetOwner(req.Contract, req.Sender, req.Addr)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to set owner: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, true)
}

func (s *Server) handleObjectGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		ID       core.ObjectID `json:"id"`
		Field    string        `json:"field"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	// For simplicity, we're assuming the object exists and can be retrieved
	// In a real implementation, you might want to handle this differently
	obj, err := s.ctx.GetObject(req.Contract, req.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get object: %v", err), http.StatusInternalServerError)
		return
	}

	value, err := obj.Get(req.Contract, req.Field)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get field: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, value)
}

func (s *Server) handleObjectSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Contract types.Address `json:"contract"`
		Sender   types.Address `json:"sender"`
		ID       core.ObjectID `json:"id"`
		Field    string        `json:"field"`
		Value    []byte        `json:"value"`
	}

	if err := s.readJSONRequest(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	// For simplicity, we're assuming the object exists and can be retrieved
	// In a real implementation, you might want to handle this differently
	obj, err := s.ctx.GetObject(req.Contract, req.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get object: %v", err), http.StatusInternalServerError)
		return
	}

	err = obj.Set(req.Contract, req.Sender, req.Field, req.Value)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to set field: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendJSONResponse(w, true)
}
