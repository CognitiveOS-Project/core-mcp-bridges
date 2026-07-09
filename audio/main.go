package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/CognitiveOS-Project/core-mcp-bridges/internal/mcp"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Println("audio-mcp 0.2.0")
			return
		}
	}

	s := mcp.New("audio-mcp")
	s.SetVersion("0.2.0")

	s.Tools = []mcp.Tool{
		{
			Name:        "cognitiveos.audio.play",
			Description: "Play an audio file through the default ALSA output device",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path":   map[string]interface{}{"type": "string", "description": "Absolute path to the audio file"},
					"volume": map[string]interface{}{"type": "number", "minimum": 0, "maximum": 100, "default": 100, "description": "Volume percentage"},
					"block":  map[string]interface{}{"type": "boolean", "default": true, "description": "Wait for playback to finish before returning"},
				},
				"required": []string{"path"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "enum": []string{"playing", "played"}},
					"path":   map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "cognitiveos.audio.capture",
			Description: "Capture audio from the default microphone input",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"duration_seconds": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 60, "default": 5, "description": "Capture duration in seconds"},
					"output_path":      map[string]interface{}{"type": "string", "description": "Path to save the captured audio"},
				},
				"required": []string{"output_path"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status":           map[string]interface{}{"type": "string", "enum": []string{"captured"}},
					"path":             map[string]interface{}{"type": "string"},
					"duration_seconds": map[string]interface{}{"type": "integer"},
				},
			},
		},
		{
			Name:        "cognitiveos.audio.tts",
			Description: "Generate and play text-to-speech audio",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text":  map[string]interface{}{"type": "string", "description": "Text to speak"},
					"voice": map[string]interface{}{"type": "string", "default": "default", "description": "Voice model identifier"},
					"speed": map[string]interface{}{"type": "number", "minimum": 0.5, "maximum": 2.0, "default": 1.0, "description": "Speech speed multiplier"},
				},
				"required": []string{"text"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "enum": []string{"spoken"}},
					"text":   map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "cognitiveos.audio.set_volume",
			Description: "Set the system audio volume level",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"volume": map[string]interface{}{"type": "number", "minimum": 0, "maximum": 100, "description": "Volume percentage"},
					"device": map[string]interface{}{"type": "string", "default": "default", "description": "ALSA device name"},
				},
				"required": []string{"volume"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string", "enum": []string{"volume_set"}},
					"volume": map[string]interface{}{"type": "number"},
					"device": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "cognitiveos.audio.mute",
			Description: "Mute or unmute the system audio",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"muted":  map[string]interface{}{"type": "boolean", "description": "True to mute, false to unmute"},
					"device": map[string]interface{}{"type": "string", "default": "default", "description": "ALSA device name"},
				},
				"required": []string{"muted"},
			},
			OutputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{"type": "string"},
					"muted":  map[string]interface{}{"type": "boolean"},
				},
			},
		},
		{
			Name:        "cognitiveos.audio.list_devices",
			Description: "List available audio playback and capture devices",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			OutputSchema: map[string]interface{}{
				"type": "string",
			},
		},
	}

	s.Handle("cognitiveos.audio.play", func(args map[string]interface{}) (interface{}, error) {
		path, _ := args["path"].(string)
		if path == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: path is required")
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("E_NOT_FOUND: audio file not found: %s", path)
		}

		ext := strings.ToLower(path)
		var cmd *exec.Cmd
		switch {
		case strings.HasSuffix(ext, ".wav"):
			cmd = exec.Command("aplay", path)
		case strings.HasSuffix(ext, ".mp3"):
			cmd = exec.Command("mpg123", "-q", path)
		default:
			return nil, fmt.Errorf("E_UNSUPPORTED_FORMAT: unsupported audio format: %s", ext)
		}

		block, _ := args["block"].(bool)
		if !block {
			_ = cmd.Start()
			return map[string]interface{}{"status": "playing", "path": path}, nil
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: playback failed: %s", strings.TrimSpace(string(output)))
		}
		return map[string]interface{}{"status": "played", "path": path}, nil
	})

	s.Handle("cognitiveos.audio.capture", func(args map[string]interface{}) (interface{}, error) {
		outputPath, _ := args["output_path"].(string)
		if outputPath == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: output_path is required")
		}

		duration := 5
		if d, ok := args["duration_seconds"].(float64); ok {
			duration = int(d)
		}

		cmd := exec.Command("arecord", "-d", strconv.Itoa(duration), "-f", "cd", outputPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: arecord failed: %s", strings.TrimSpace(string(output)))
		}
		return map[string]interface{}{"status": "captured", "path": outputPath, "duration_seconds": duration}, nil
	})

	s.Handle("cognitiveos.audio.tts", func(args map[string]interface{}) (interface{}, error) {
		text, _ := args["text"].(string)
		if text == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: text is required")
		}

		tmpFile := "/tmp/cognitiveos-tts.wav"
		cmd := exec.Command("espeak", "-w", tmpFile, text)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: TTS failed (espeak not installed?): %v", err)
		}
		playCmd := exec.Command("aplay", tmpFile)
		_ = playCmd.Run()
		return map[string]interface{}{"status": "spoken", "text": text}, nil
	})

	s.Handle("cognitiveos.audio.set_volume", func(args map[string]interface{}) (interface{}, error) {
		vol, ok := args["volume"].(float64)
		if !ok {
			return nil, fmt.Errorf("E_INVALID_PARAM: volume is required")
		}
		device, _ := args["device"].(string)
		if device == "" {
			device = "default"
		}

		cmd := exec.Command("amixer", "set", "Master", strconv.Itoa(int(vol))+"%")
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: amixer failed: %s", strings.TrimSpace(string(output)))
		}
		return map[string]interface{}{"status": "volume_set", "volume": vol, "device": device}, nil
	})

	s.Handle("cognitiveos.audio.mute", func(args map[string]interface{}) (interface{}, error) {
		muted, ok := args["muted"].(bool)
		if !ok {
			return nil, fmt.Errorf("E_INVALID_PARAM: muted is required")
		}

		state := "mute"
		if !muted {
			state = "unmute"
		}
		cmd := exec.Command("amixer", "set", "Master", state)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: amixer failed: %s", strings.TrimSpace(string(output)))
		}
		return map[string]interface{}{"status": state, "muted": muted}, nil
	})

	s.Handle("cognitiveos.audio.list_devices", func(args map[string]interface{}) (interface{}, error) {
		cmd := exec.Command("aplay", "-l")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("E_HARDWARE: aplay -l failed: %v", err)
		}
		return strings.TrimSpace(string(output)), nil
	})

	s.Log("audio-mcp ready")
	if err := s.Run(); err != nil {
		s.Log("fatal: %v", err)
		os.Exit(1)
	}
}
