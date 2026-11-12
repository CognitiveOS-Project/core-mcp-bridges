package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGPIOMCPBuilds(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gpio-mcp: %v", err)
	}
}

func TestGPIOMCPStarts(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gpio-mcp failed to start: %v", err)
	}
	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, "gpio-mcp") {
		t.Fatalf("expected gpio-mcp in output, got %q", outputStr)
	}
	if !strings.Contains(outputStr, "0.2.0") {
		t.Fatalf("expected version 0.2.0, got %q", outputStr)
	}
}