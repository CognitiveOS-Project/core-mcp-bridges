package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/CognitiveOS-Project/core-mcp-bridges/internal/mcp"
)

func main() {
	s := mcp.New("package-mcp")

	s.Tools = []mcp.Tool{
		{
			Name:        "cognitiveos.package.search",
			Description: "Search the CognitiveOS package registry for available packages",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query for package name or description"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "cognitiveos.package.list",
			Description: "List all currently installed packages",
			InputSchema: map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "cognitiveos.package.install",
			Description: "Install a package from the CognitiveOS registry",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":    map[string]interface{}{"type": "string", "description": "Package name to install"},
					"version": map[string]interface{}{"type": "string", "description": "Optional version string (defaults to latest)"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "cognitiveos.package.remove",
			Description: "Uninstall an installed package",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Package name to remove"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "cognitiveos.package.info",
			Description: "Show detailed manifest information for a package",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Package name"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "cognitiveos.package.update",
			Description: "Update an installed package to the latest version",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Package name to update"},
				},
				"required": []string{"name"},
			},
		},
	}

	s.Handle("cognitiveos.package.search", func(args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: query is required")
		}
		return runCPM("search", query)
	})

	s.Handle("cognitiveos.package.list", func(args map[string]interface{}) (interface{}, error) {
		return runCPM("list")
	})

	s.Handle("cognitiveos.package.install", func(args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: name is required")
		}
		version, _ := args["version"].(string)
		cpmArgs := []string{"install", name}
		if version != "" {
			cpmArgs = append(cpmArgs, "--version", version)
		}
		return runCPM(cpmArgs...)
	})

	s.Handle("cognitiveos.package.remove", func(args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: name is required")
		}
		return runCPM("remove", name)
	})

	s.Handle("cognitiveos.package.info", func(args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: name is required")
		}
		return runCPM("info", name)
	})

	s.Handle("cognitiveos.package.update", func(args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: name is required")
		}
		return runCPM("update", name)
	})

	s.Log("package-mcp ready")
	if err := s.Run(); err != nil {
		s.Log("fatal: %v", err)
		os.Exit(1)
	}
}

type cpmResult struct {
	Status   string `json:"status"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

func runCPM(args ...string) (interface{}, error) {
	cpmPath := findCPM()
	if cpmPath == "" {
		return nil, fmt.Errorf("E_INTERNAL: cpm binary not found on PATH")
	}

	cmd := exec.Command(cpmPath, args...)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		return nil, fmt.Errorf("E_HARDWARE: cpm failed: %s", outputStr)
	}

	var structured map[string]interface{}
	if json.Unmarshal([]byte(outputStr), &structured) == nil {
		return structured, nil
	}

	return map[string]interface{}{
		"status": "ok",
		"output": outputStr,
	}, nil
}

func findCPM() string {
	paths := []string{
		"/cognitiveos/bin/cpm",
		"/usr/local/bin/cpm",
		"/usr/bin/cpm",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("cpm"); err == nil {
		return p
	}
	return ""
}
