package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSerialMCPBuilds(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build serial-mcp: %v", err)
	}
}

func TestSerialMCPStarts(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("serial-mcp failed to start: %v", err)
	}
	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, "serial-mcp") {
		t.Fatalf("expected serial-mcp in output, got %q", outputStr)
	}
	if !strings.Contains(outputStr, "0.2.0") {
		t.Fatalf("expected version 0.2.0, got %q", outputStr)
	}
}