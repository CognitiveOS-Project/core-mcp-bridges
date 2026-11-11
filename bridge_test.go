package bridges

import (
	"os/exec"
	"testing"
)

var bridgeBinaries = []struct {
	name  string
	dir   string
	flags []string
}{
	{name: "display-mcp", dir: "./display", flags: []string{"--help"}},
	{name: "audio-mcp", dir: "./audio", flags: []string{"--help"}},
	{name: "network-mcp", dir: "./network", flags: []string{"--help"}},
	{name: "gpio-mcp", dir: "./gpio", flags: []string{"--help"}},
	{name: "serial-mcp", dir: "./serial", flags: []string{"--help"}},
	{name: "package-mcp", dir: "./package", flags: []string{"--help"}},
}

func TestBridgeBinariesBuild(t *testing.T) {
	for _, b := range bridgeBinaries {
		t.Run(b.name, func(t *testing.T) {
			args := append([]string{"build", "-o", "/dev/null"}, b.dir)
			cmd := exec.Command("go", args...)
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to build %s: %v", b.name, err)
			}
		})
	}
}
