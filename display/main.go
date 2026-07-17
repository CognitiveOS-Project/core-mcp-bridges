package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/CognitiveOS-Project/core-mcp-bridges/internal/mcp"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Println("display-mcp 0.2.0")
			return
		}
	}

	s := mcp.New("display-mcp")
	s.SetVersion("0.2.0")

	s.Tools = []mcp.Tool{
		{
			Name:        "cognitiveos.display.render_image",
			Description: "Render an image file to the primary display framebuffer",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string", "description": "Absolute path to the image file"},
					"fit":  map[string]interface{}{"type": "string", "enum": []string{"fill", "fit", "stretch"}, "default": "fit", "description": "How to fit the image to the screen"},
				},
				"required": []string{"path"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "enum": []string{"rendered"}},
					"path":   map[string]interface{}{"type": "string"},
					"fit":    map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "cognitiveos.display.render_video",
			Description: "Play a video file on the primary display using DRM direct rendering",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":   map[string]interface{}{"type": "string", "description": "Absolute path to the video file"},
					"loop":   map[string]interface{}{"type": "boolean", "default": false, "description": "Loop the video"},
					"volume": map[string]interface{}{"type": "number", "minimum": 0, "maximum": 100, "default": 100, "description": "Audio volume percentage"},
				},
				"required": []string{"path"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "enum": []string{"played"}},
					"path":   map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "cognitiveos.display.screenshot",
			Description: "Capture the current framebuffer contents to a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"output_path": map[string]interface{}{"type": "string", "description": "Path to save the screenshot"},
				},
				"required": []string{"output_path"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":       map[string]interface{}{"type": "string"},
					"size_bytes": map[string]interface{}{"type": "integer"},
				},
			},
		},
		{
			Name:        "cognitiveos.display.clear",
			Description: "Clear the framebuffer and return display to idle state",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "enum": []string{"cleared"}},
				},
			},
		},
	}

	s.Handle("cognitiveos.display.render_image", func(args map[string]interface{}) (interface{}, error) {
		path, _ := args["path"].(string)
		if path == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: path is required")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("E_NOT_FOUND: image not found: %s", path)
		}

		fit, _ := args["fit"].(string)
		if fit == "" {
			fit = "fit"
		}

		mpvArgs := []string{"--vo=drm", "--no-terminal", "--keep-open=no"}
		switch fit {
		case "fill":
			mpvArgs = append(mpvArgs, "--panscan=1.0", path)
		case "stretch":
			mpvArgs = append(mpvArgs, "--panscan=1.0", "--geometry=100%", path)
		default:
			mpvArgs = append(mpvArgs, "--autofit=fit", path)
		}
		cmd := exec.Command("mpv", mpvArgs...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: mpv failed: %s", strings.TrimSpace(string(output)))
		}
		return map[string]interface{}{"status": "rendered", "path": path, "fit": fit}, nil
	})

	s.Handle("cognitiveos.display.render_video", func(args map[string]interface{}) (interface{}, error) {
		path, _ := args["path"].(string)
		if path == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: path is required")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("E_NOT_FOUND: video not found: %s", path)
		}

		mpvArgs := []string{"--vo=drm", "--no-terminal", "--keep-open=no"}
		loop, _ := args["loop"].(bool)
		if loop {
			mpvArgs = append(mpvArgs, "--loop=inf")
		}
		if vol, ok := args["volume"].(float64); ok {
			mpvArgs = append(mpvArgs, fmt.Sprintf("--volume=%.0f", vol))
		}
		mpvArgs = append(mpvArgs, path)

		cmd := exec.Command("mpv", mpvArgs...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: mpv failed: %s", strings.TrimSpace(string(output)))
		}
		return map[string]interface{}{"status": "played", "path": path}, nil
	})

	s.Handle("cognitiveos.display.screenshot", func(args map[string]interface{}) (interface{}, error) {
		outputPath, _ := args["output_path"].(string)
		if outputPath == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: output_path is required")
		}

		fb, err := os.Open("/dev/fb0")
		if err != nil {
			return nil, fmt.Errorf("E_NO_DEVICE: cannot open framebuffer: %v", err)
		}
		defer fb.Close()

		out, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("E_INVALID_PARAM: cannot create output: %v", err)
		}
		defer out.Close()

		buf := make([]byte, 1024*1024)
		var total int64
		for {
			n, err := fb.Read(buf)
			if n > 0 {
				if _, werr := out.Write(buf[:n]); werr != nil {
					return nil, fmt.Errorf("E_HARDWARE: write error: %v", werr)
				}
				total += int64(n)
			}
			if err != nil {
				break
			}
		}
		return map[string]interface{}{"path": outputPath, "size_bytes": total}, nil
	})

	s.Handle("cognitiveos.display.clear", func(args map[string]interface{}) (interface{}, error) {
		fb, err := os.OpenFile("/dev/fb0", os.O_WRONLY, 0)
		if err != nil {
			return nil, fmt.Errorf("E_NO_DEVICE: cannot open framebuffer: %v", err)
		}
		defer fb.Close()

		stat, err := fb.Stat()
		if err != nil {
			return map[string]interface{}{"status": "cleared"}, nil
		}
		size := stat.Size()
		if size <= 0 {
			size = 1024 * 768 * 4
		}
		zeros := make([]byte, size)
		if _, err := fb.Write(zeros); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: clear failed: %v", err)
		}
		return map[string]interface{}{"status": "cleared"}, nil
	})

	s.Log("display-mcp ready")
	if err := s.Run(); err != nil {
		s.Log("fatal: %v", err)
		os.Exit(1)
	}
}
