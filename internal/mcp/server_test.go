package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stdout = stdout
	return buf.String()
}

func TestListTools(t *testing.T) {
	s := New("test-server")
	s.Tools = []Tool{
		{Name: "test.tool", Description: "A test tool", InputSchema: map[string]interface{}{"type": "object"}},
	}

	output := captureStdout(func() {
		s.handleMessage(`{"jsonrpc":"2.0","id":1,"method":"mcp.list_tools"}`)
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &resp); err != nil {
		t.Fatalf("unmarshal: %v", output)
	}
	if resp["id"] != float64(1) {
		t.Fatalf("expected id 1, got %v", resp["id"])
	}
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result object")
	}
	tools, ok := result["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Fatal("expected 1 tool")
	}
}

func TestToolCall(t *testing.T) {
	s := New("test-server")
	s.Handle("test.hello", func(args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		return "hello " + name, nil
	})

	output := captureStdout(func() {
		s.handleMessage(`{"jsonrpc":"2.0","id":2,"method":"test.hello","params":{"arguments":{"name":"world"}}}`)
	})

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result, _ := resp["result"].(map[string]interface{})
	content, _ := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("expected content")
	}
}

func TestUnknownTool(t *testing.T) {
	s := New("test-server")

	output := captureStdout(func() {
		s.handleMessage(`{"jsonrpc":"2.0","id":3,"method":"unknown.tool"}`)
	})

	var resp map[string]interface{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(output)), &resp)

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result object")
	}
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Fatal("expected isError=true for unknown tool")
	}
	content, _ := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("expected error content")
	}
}

func TestHealthcheck(t *testing.T) {
	s := New("test-server")

	output := captureStdout(func() {
		s.handleMessage(`{"jsonrpc":"2.0","method":"healthcheck"}`)
	})

	if !strings.Contains(output, "healthcheck_ok") {
		t.Fatalf("expected healthcheck_ok, got: %s", output)
	}
}
