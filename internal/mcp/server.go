package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type Handler func(map[string]interface{}) (interface{}, error)

type Server struct {
	Name    string
	Tools   []Tool
	handlers map[string]Handler
	started time.Time
	logger  *log.Logger
	mu      sync.Mutex
}

func New(name string) *Server {
	s := &Server{
		Name:     name,
		handlers: make(map[string]Handler),
		started:  time.Now(),
		logger:   log.New(os.Stderr, name+": ", log.LstdFlags),
	}
	return s
}

func (s *Server) Handle(tool string, fn Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[tool] = fn
}

func (s *Server) Run() error {
	s.logger.Printf("starting, pid=%d", os.Getpid())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		s.logger.Printf("shutting down")
		os.Exit(0)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		s.handleMessage(line)
	}
	return scanner.Err()
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (s *Server) handleMessage(line string) {
	var req jsonRPCRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return
	}

	if req.ID == nil {
		s.handleNotification(req.Method, req.Params)
		return
	}

	resp := s.dispatch(&req)
	reply, _ := json.Marshal(resp)
	fmt.Println(string(reply))
}

func (s *Server) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "healthcheck":
		s.sendHealthcheckResponse()
	}
}

func (s *Server) dispatch(req *jsonRPCRequest) *jsonRPCResponse {
	switch req.Method {
	case "mcp.list_tools":
		return s.handleListTools(req.ID)
	default:
		return s.handleToolCall(req.ID, req.Method, req.Params)
	}
}

func (s *Server) handleListTools(id *json.RawMessage) *jsonRPCResponse {
	s.mu.Lock()
	tools := make([]Tool, len(s.Tools))
	copy(tools, s.Tools)
	s.mu.Unlock()

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolCall(id *json.RawMessage, method string, params json.RawMessage) *jsonRPCResponse {
	s.mu.Lock()
	handler, ok := s.handlers[method]
	s.mu.Unlock()

	if !ok {
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &rpcError{
				Code:    -32601,
				Message: fmt.Sprintf("Tool not found: %s", method),
			},
		}
	}

	var args map[string]interface{}
	if len(params) > 0 {
		var callParams struct {
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(params, &callParams); err == nil {
			args = callParams.Arguments
		}
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	result, err := handler(args)
	if err != nil {
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32000, Message: err.Error()},
		}
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("%v", result)},
			},
		},
	}
}

func (s *Server) sendHealthcheckResponse() {
	resp := map[string]interface{}{
		"type":           "healthcheck_ok",
		"uptime_seconds": int(time.Since(s.started).Seconds()),
		"tools_healthy":  true,
	}
	reply, _ := json.Marshal(resp)
	s.logger.Printf("healthcheck: uptime=%ds", int(time.Since(s.started).Seconds()))
	fmt.Println(string(reply))
}

func (s *Server) Log(format string, args ...interface{}) {
	s.logger.Printf(format, args...)
}
