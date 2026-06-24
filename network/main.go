package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/core-mcp-bridges/internal/mcp"
)

func main() {
	s := mcp.New("network-mcp")

	s.Tools = []mcp.Tool{
		{
			Name:        "cognitiveos.network.scan",
			Description: "Scan for available Wi-Fi networks",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"interface": map[string]interface{}{"type": "string", "default": "wlan0", "description": "Wireless interface name"},
				},
			},
		},
		{
			Name:        "cognitiveos.network.connect",
			Description: "Connect to a Wi-Fi network",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ssid":           map[string]interface{}{"type": "string", "description": "Network SSID"},
					"password":       map[string]interface{}{"type": "string", "description": "Network password (if encrypted)"},
					"interface":      map[string]interface{}{"type": "string", "default": "wlan0", "description": "Wireless interface name"},
					"timeout_seconds": map[string]interface{}{"type": "integer", "default": 15, "description": "Connection timeout"},
				},
				"required": []string{"ssid"},
			},
		},
		{
			Name:        "cognitiveos.network.disconnect",
			Description: "Disconnect from the current network",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"interface": map[string]interface{}{"type": "string", "default": "wlan0", "description": "Interface to disconnect"},
				},
			},
		},
		{
			Name:        "cognitiveos.network.status",
			Description: "Get current network connectivity status",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"interface": map[string]interface{}{"type": "string", "default": "all", "description": "Interface name or 'all'"},
				},
			},
		},
		{
			Name:        "cognitiveos.network.list_interfaces",
			Description: "List available network interfaces",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	s.Handle("cognitiveos.network.scan", func(args map[string]interface{}) (interface{}, error) {
		iface, _ := args["interface"].(string)
		if iface == "" {
			iface = "wlan0"
		}

		cmd := exec.Command("iw", "dev", iface, "scan")
		output, err := cmd.Output()
		if err != nil {
			// fallback to iwlist
			cmd2 := exec.Command("iwlist", iface, "scan")
			if output2, err2 := cmd2.Output(); err2 == nil {
				return strings.TrimSpace(string(output2)), nil
			}
			return nil, fmt.Errorf("E_SCAN_FAILED: scan failed: %v", err)
		}
		return strings.TrimSpace(string(output)), nil
	})

	s.Handle("cognitiveos.network.connect", func(args map[string]interface{}) (interface{}, error) {
		ssid, _ := args["ssid"].(string)
		if ssid == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: ssid is required")
		}
		iface, _ := args["interface"].(string)
		if iface == "" {
			iface = "wlan0"
		}
		password, _ := args["password"].(string)

		// Write wpa_supplicant config
		confDir := "/cognitiveos/run/network"
		os.MkdirAll(confDir, 0755)
		confPath := filepath.Join(confDir, "wpa_"+ssid+".conf")

		conf := fmt.Sprintf(`network={
	ssid="%s"
	key_mgmt=%s
	psk="%s"
}
`, ssid, map[bool]string{true: "WPA-PSK", false: "NONE"}[password != ""], password)

		if err := os.WriteFile(confPath, []byte(conf), 0600); err != nil {
			return nil, fmt.Errorf("E_HARDWARE: cannot write config: %v", err)
		}

		cmd := exec.Command("wpa_supplicant", "-B", "-i", iface, "-c", confPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("E_CONNECTION_FAILED: wpa_supplicant: %s", strings.TrimSpace(string(output)))
		}

		dhcp := exec.Command("dhcpcd", "-n", iface)
		dhcp.Run()

		return map[string]interface{}{"status": "connecting", "ssid": ssid, "interface": iface}, nil
	})

	s.Handle("cognitiveos.network.disconnect", func(args map[string]interface{}) (interface{}, error) {
		iface, _ := args["interface"].(string)
		if iface == "" {
			iface = "wlan0"
		}

		exec.Command("wpa_cli", "-i", iface, "terminate").Run()
		exec.Command("dhcpcd", "-k", iface).Run()
		return map[string]interface{}{"status": "disconnected", "interface": iface}, nil
	})

	s.Handle("cognitiveos.network.status", func(args map[string]interface{}) (interface{}, error) {
		iface, _ := args["interface"].(string)
		if iface == "" || iface == "all" {
			cmd := exec.Command("ip", "addr")
			output, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("E_HARDWARE: ip addr failed: %v", err)
			}
			return strings.TrimSpace(string(output)), nil
		}

		cmd := exec.Command("ip", "addr", "show", iface)
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("E_NO_INTERFACE: interface %s not found", iface)
		}
		return strings.TrimSpace(string(output)), nil
	})

	s.Handle("cognitiveos.network.list_interfaces", func(args map[string]interface{}) (interface{}, error) {
		entries, err := os.ReadDir("/sys/class/net")
		if err != nil {
			return nil, fmt.Errorf("E_HARDWARE: cannot list interfaces: %v", err)
		}
		var ifaces []string
		for _, e := range entries {
			ifaces = append(ifaces, e.Name())
		}
		return strings.Join(ifaces, "\n"), nil
	})

	s.Log("network-mcp ready")
	if err := s.Run(); err != nil {
		s.Log("fatal: %v", err)
		os.Exit(1)
	}
}
