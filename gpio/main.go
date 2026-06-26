package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/CognitiveOS-Project/core-mcp-bridges/internal/mcp"
)

var gpioBase = "/sys/class/gpio"

func main() {
	s := mcp.New("gpio-mcp")

	s.Tools = []mcp.Tool{
		{
			Name:        "cognitiveos.gpio.pin_read",
			Description: "Read the digital value of a GPIO pin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pin":  map[string]interface{}{"type": "integer", "description": "GPIO pin number (board or chip-relative)"},
					"chip": map[string]interface{}{"type": "integer", "default": 0, "description": "GPIO chip number"},
				},
				"required": []string{"pin"},
			},
		},
		{
			Name:        "cognitiveos.gpio.pin_write",
			Description: "Set the digital value of a GPIO pin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pin":   map[string]interface{}{"type": "integer", "description": "GPIO pin number"},
					"value": map[string]interface{}{"type": "integer", "enum": []int{0, 1}, "description": "0 = LOW, 1 = HIGH"},
					"chip":  map[string]interface{}{"type": "integer", "default": 0, "description": "GPIO chip number"},
				},
				"required": []string{"pin", "value"},
			},
		},
		{
			Name:        "cognitiveos.gpio.pwm",
			Description: "Set PWM duty cycle on a PWM-capable pin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pin":              map[string]interface{}{"type": "integer", "description": "PWM pin number"},
					"duty_cycle_percent": map[string]interface{}{"type": "number", "minimum": 0, "maximum": 100, "description": "Duty cycle 0-100%"},
					"frequency_hz":     map[string]interface{}{"type": "integer", "default": 1000, "description": "PWM frequency in Hz"},
					"chip":             map[string]interface{}{"type": "integer", "default": 0, "description": "PWM chip number"},
				},
				"required": []string{"pin", "duty_cycle_percent"},
			},
		},
		{
			Name:        "cognitiveos.gpio.mode",
			Description: "Set the direction mode of a GPIO pin",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pin":  map[string]interface{}{"type": "integer", "description": "GPIO pin number"},
					"mode": map[string]interface{}{"type": "string", "enum": []string{"input", "output", "input_pullup", "input_pulldown"}, "description": "Pin direction mode"},
					"chip": map[string]interface{}{"type": "integer", "default": 0, "description": "GPIO chip number"},
				},
				"required": []string{"pin", "mode"},
			},
		},
		{
			Name:        "cognitiveos.gpio.list_pins",
			Description: "List all available GPIO pins and their current state",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chip": map[string]interface{}{"type": "integer", "default": 0, "description": "GPIO chip number"},
				},
			},
		},
	}

	s.Handle("cognitiveos.gpio.pin_read", func(args map[string]interface{}) (interface{}, error) {
		pin := intFromArgs(args, "pin")
		if pin < 0 {
			return nil, fmt.Errorf("E_INVALID_PARAM: pin is required")
		}
		if err := gpioExport(pin); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: export failed: %v", err)
		}
		val, err := os.ReadFile(filepath.Join(gpioBase, fmt.Sprintf("gpio%d", pin), "value"))
		if err != nil {
			return nil, fmt.Errorf("E_INVALID_PIN: %v", err)
		}
		return map[string]interface{}{"pin": pin, "value": int(strings.TrimSpace(string(val))[0] - '0')}, nil
	})

	s.Handle("cognitiveos.gpio.pin_write", func(args map[string]interface{}) (interface{}, error) {
		pin := intFromArgs(args, "pin")
		val := intFromArgs(args, "value")
		if pin < 0 {
			return nil, fmt.Errorf("E_INVALID_PARAM: pin is required")
		}
		if val < 0 || val > 1 {
			return nil, fmt.Errorf("E_INVALID_PARAM: value must be 0 or 1")
		}

		if err := gpioExport(pin); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: export failed: %v", err)
		}
		if err := gpioSetDir(pin, "out"); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: set direction failed: %v", err)
		}

		valPath := filepath.Join(gpioBase, fmt.Sprintf("gpio%d", pin), "value")
		if err := os.WriteFile(valPath, []byte(strconv.Itoa(val)), 0644); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: write failed: %v", err)
		}
		return map[string]interface{}{"pin": pin, "value": val}, nil
	})

	s.Handle("cognitiveos.gpio.mode", func(args map[string]interface{}) (interface{}, error) {
		pin := intFromArgs(args, "pin")
		mode, _ := args["mode"].(string)
		if pin < 0 {
			return nil, fmt.Errorf("E_INVALID_PARAM: pin is required")
		}

		dir := "in"
		if mode == "output" {
			dir = "out"
		}
		if err := gpioExport(pin); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: export failed: %v", err)
		}
		if err := gpioSetDir(pin, dir); err != nil {
			return nil, fmt.Errorf("E_MODE_NOT_SUPPORTED: %v", err)
		}
		return map[string]interface{}{"pin": pin, "mode": mode}, nil
	})

	s.Handle("cognitiveos.gpio.pwm", func(args map[string]interface{}) (interface{}, error) {
		pin := intFromArgs(args, "pin")
		duty := floatFromArgs(args, "duty_cycle_percent")
		freq := intFromArgs(args, "frequency_hz")
		if freq <= 0 {
			freq = 1000
		}

		pwmPath := fmt.Sprintf("/sys/class/pwm/pwmchip0/pwm%d", pin)
		if err := os.MkdirAll(pwmPath, 0755); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: mkdir pwm: %v", err)
		}

		period := 1000000000 / freq
		dutyNs := int(float64(period) * duty / 100.0)

		if err := os.WriteFile(filepath.Join(pwmPath, "period"), []byte(strconv.Itoa(period)), 0644); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: write period: %v", err)
		}
		if err := os.WriteFile(filepath.Join(pwmPath, "duty_cycle"), []byte(strconv.Itoa(dutyNs)), 0644); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: write duty_cycle: %v", err)
		}
		if err := os.WriteFile(filepath.Join(pwmPath, "enable"), []byte("1"), 0644); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: write enable: %v", err)
		}

		return map[string]interface{}{"pin": pin, "duty_cycle_percent": duty, "frequency_hz": freq}, nil
	})

	s.Handle("cognitiveos.gpio.list_pins", func(args map[string]interface{}) (interface{}, error) {
		entries, err := os.ReadDir(gpioBase)
		if err != nil {
			return nil, fmt.Errorf("E_CHIP_NOT_FOUND: %v", err)
		}
		var pins []string
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "gpio") && e.IsDir() {
				pins = append(pins, e.Name())
			}
		}
		return strings.Join(pins, "\n"), nil
	})

	s.Log("gpio-mcp ready")
	if err := s.Run(); err != nil {
		s.Log("fatal: %v", err)
		os.Exit(1)
	}
}

func gpioExport(pin int) error {
	path := filepath.Join(gpioBase, fmt.Sprintf("gpio%d", pin))
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(filepath.Join(gpioBase, "export"), []byte(strconv.Itoa(pin)), 0644)
}

func gpioSetDir(pin int, dir string) error {
	return os.WriteFile(filepath.Join(gpioBase, fmt.Sprintf("gpio%d", pin), "direction"), []byte(dir), 0644)
}

func intFromArgs(args map[string]interface{}, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return -1
}

func floatFromArgs(args map[string]interface{}, key string) float64 {
	switch v := args[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	}
	return 0
}
