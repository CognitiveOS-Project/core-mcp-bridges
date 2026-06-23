package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/CognitiveOS-Project/core-mcp-bridges/internal/mcp"
)

type serialSession struct {
	fd       int
	portPath string
	baud     int
}

var (
	sessions   = map[string]*serialSession{}
	sessionsMu sync.Mutex
	sessionID  int
)

var baudRates = map[int]uint32{
	1200:   syscall.B1200,
	2400:   syscall.B2400,
	4800:   syscall.B4800,
	9600:   syscall.B9600,
	19200:  syscall.B19200,
	38400:  syscall.B38400,
	57600:  syscall.B57600,
	115200: syscall.B115200,
	230400: 0x1003,
}

func main() {
	s := mcp.New("serial-mcp")

	s.Tools = []mcp.Tool{
		{
			Name:        "cognitiveos.serial.list_ports",
			Description: "List available serial ports on the system",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
		{
			Name:        "cognitiveos.serial.connect",
			Description: "Open a connection to a serial port",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"port":      map[string]interface{}{"type": "string", "description": "Device path (e.g. /dev/ttyUSB0)"},
					"baud_rate": map[string]interface{}{"type": "integer", "default": 9600, "enum": []int{1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400}, "description": "Baud rate"},
					"data_bits": map[string]interface{}{"type": "integer", "default": 8, "enum": []int{5, 6, 7, 8}},
				},
				"required": []string{"port"},
			},
		},
		{
			Name:        "cognitiveos.serial.send",
			Description: "Send data to a connected serial port",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string", "description": "Session ID from connect"},
					"data":       map[string]interface{}{"type": "string", "description": "Data to send (text)"},
				},
				"required": []string{"session_id", "data"},
			},
		},
		{
			Name:        "cognitiveos.serial.receive",
			Description: "Read data from a connected serial port",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string", "description": "Session ID from connect"},
					"bytes":      map[string]interface{}{"type": "integer", "default": 256, "description": "Number of bytes to read"},
				},
				"required": []string{"session_id"},
			},
		},
		{
			Name:        "cognitiveos.serial.disconnect",
			Description: "Close a serial port connection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{"type": "string", "description": "Session ID from connect"},
				},
				"required": []string{"session_id"},
			},
		},
	}

	s.Handle("cognitiveos.serial.list_ports", func(args map[string]interface{}) (interface{}, error) {
		var ports []string
		for _, pattern := range []string{"/dev/ttyUSB*", "/dev/ttyS*", "/dev/ttyAMA*", "/dev/ttyACM*"} {
			matches, _ := filepath.Glob(pattern)
			ports = append(ports, matches...)
		}
		if len(ports) == 0 {
			return nil, fmt.Errorf("E_PORT_NOT_FOUND: no serial ports found")
		}
		return strings.Join(ports, "\n"), nil
	})

	s.Handle("cognitiveos.serial.connect", func(args map[string]interface{}) (interface{}, error) {
		portPath, _ := args["port"].(string)
		if portPath == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: port is required")
		}

		baud := 9600
		if b, ok := args["baud_rate"].(float64); ok {
			baud = int(b)
		}

		rate, ok := baudRates[baud]
		if !ok {
			return nil, fmt.Errorf("E_INVALID_PARAM: unsupported baud rate %d", baud)
		}

		fd, err := syscall.Open(portPath, syscall.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			return nil, fmt.Errorf("E_PORT_BUSY: cannot open %s: %v", portPath, err)
		}

		// Set serial port attributes via ioctl
		var tios syscall.Termios
		tios.Cflag = syscall.CREAD | syscall.CLOCAL | rate | syscall.CS8
		tios.Iflag = syscall.IGNPAR
		tios.Oflag = 0
		tios.Lflag = 0
		tios.Cc[syscall.VMIN] = 1
		tios.Cc[syscall.VTIME] = 0

		if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&tios)), 0, 0, 0); errno != 0 {
			syscall.Close(fd)
			return nil, fmt.Errorf("E_HARDWARE: tcsetattr failed: %v", errno)
		}

		sessionsMu.Lock()
		sessionID++
		id := fmt.Sprintf("ser-%d", sessionID)
		sessions[id] = &serialSession{fd: fd, portPath: portPath, baud: baud}
		sessionsMu.Unlock()

		return map[string]interface{}{
			"session_id": id,
			"port":       portPath,
			"baud_rate":  baud,
		}, nil
	})

	s.Handle("cognitiveos.serial.send", func(args map[string]interface{}) (interface{}, error) {
		sid, _ := args["session_id"].(string)
		data, _ := args["data"].(string)
		if sid == "" || data == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: session_id and data are required")
		}

		sessionsMu.Lock()
		ses, ok := sessions[sid]
		sessionsMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("E_SESSION_NOT_FOUND: session %s not found", sid)
		}

		b := []byte(data)
		n, err := syscall.Write(ses.fd, b)
		if err != nil {
			return nil, fmt.Errorf("E_PORT_ERROR: write failed: %v", err)
		}
		return map[string]interface{}{"bytes_written": n}, nil
	})

	s.Handle("cognitiveos.serial.receive", func(args map[string]interface{}) (interface{}, error) {
		sid, _ := args["session_id"].(string)
		if sid == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: session_id is required")
		}

		sessionsMu.Lock()
		ses, ok := sessions[sid]
		sessionsMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("E_SESSION_NOT_FOUND: session %s not found", sid)
		}

		bytes := 256
		if b, ok := args["bytes"].(float64); ok {
			bytes = int(b)
		}

		buf := make([]byte, bytes)
		n, err := syscall.Read(ses.fd, buf)
		if err != nil {
			return nil, fmt.Errorf("E_TIMEOUT: read failed: %v", err)
		}
		return map[string]interface{}{"data": string(buf[:n]), "bytes_read": n}, nil
	})

	s.Handle("cognitiveos.serial.disconnect", func(args map[string]interface{}) (interface{}, error) {
		sid, _ := args["session_id"].(string)
		if sid == "" {
			return nil, fmt.Errorf("E_INVALID_PARAM: session_id is required")
		}

		sessionsMu.Lock()
		ses, ok := sessions[sid]
		delete(sessions, sid)
		sessionsMu.Unlock()
		if !ok {
			return nil, fmt.Errorf("E_SESSION_NOT_FOUND: session %s not found", sid)
		}

		syscall.Close(ses.fd)
		return map[string]interface{}{"status": "disconnected", "session_id": sid}, nil
	})

	s.Log("serial-mcp ready")
	if err := s.Run(); err != nil {
		s.Log("fatal: %v", err)
		os.Exit(1)
	}
}
