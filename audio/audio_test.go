package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestAudioMCPBuilds(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "/dev/null", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build audio-mcp: %v", err)
	}
}

func TestAudioMCPStarts(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("audio-mcp failed to start: %v", err)
	}
	outputStr := strings.TrimSpace(string(output))
	if !strings.Contains(outputStr, "audio-mcp") {
		t.Fatalf("expected audio-mcp in output, got %q", outputStr)
	}
	if !strings.Contains(outputStr, "0.2.0") {
		t.Fatalf("expected version 0.2.0, got %q", outputStr)
	}
}