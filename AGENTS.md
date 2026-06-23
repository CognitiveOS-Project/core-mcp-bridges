# Core MCP Bridges

Hardware abstraction layer for CognitiveOS — lightweight MCP servers that expose device capabilities to the AI.

## Servers

| Server | Exposes | Backend |
|--------|---------|---------|
| `display-mcp` | render_image, render_video, screenshot | fbv, mpv --vo=drm, /dev/fb0 |
| `audio-mcp` | play_audio, capture_mic, tts | ALSA (aplay, arecord), mpg123 |
| `network-mcp` | scan_wifi, connect, status | iwconfig, wpa_supplicant, ping |
| `gpio-mcp` | pin_read, pin_write, pwm | /sys/class/gpio, libgpiod |
| `serial-mcp` | port_list, connect, send, receive | /dev/tty* |

## Build

```bash
go build -o bin/display-mcp ./display/
go build -o bin/audio-mcp ./audio/
go build -o bin/network-mcp ./network/
go build -o bin/gpio-mcp ./gpio/
go build -o bin/serial-mcp ./serial/
```

Each server is a standalone binary implementing the MCP JSON-RPC protocol over stdio.

## Protocol

All servers follow the MCP standard (see product-specs for CognitiveOS conventions). Tools are stateless where possible; state lives in cognitiveosd.
