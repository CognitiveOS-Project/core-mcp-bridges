package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDisplayMCPBuilds(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build display-mcp: %v", err)
	}
}

func TestDisplayMCPStarts(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("display-mcp failed to start: %v", err)
	}
	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, "display-mcp") {
		t.Fatalf("expected display-mcp in output, got %q", outputStr)
	}
	if !strings.Contains(outputStr, "0.2.0") {
		t.Fatalf("expected version 0.2.0, got %q", outputStr)
	}
}