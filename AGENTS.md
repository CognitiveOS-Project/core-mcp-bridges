# Core MCP Bridges

Hardware abstraction layer for CognitiveOS — 6 lightweight MCP servers that expose device capabilities to the AI.

| Server | Exposes | Backend |
|--------|---------|---------|
| `display-mcp` | render_image, render_video, screenshot, clear | fbv, mpv --vo=drm, /dev/fb0 |
| `audio-mcp` | play, capture, tts, set_volume, mute, list_devices | ALSA (aplay, arecord, amixer), espeak |
| `network-mcp` | scan, connect, disconnect, status, list_interfaces | iw, wpa_supplicant, dhcpcd, ip |
| `gpio-mcp` | pin_read, pin_write, pwm, mode, list_pins | /sys/class/gpio, /sys/class/pwm |
| `serial-mcp` | list_ports, connect, send, receive, disconnect | /dev/tty* raw syscalls |
| `package-mcp` | search, list, install, remove, info, update | cpm CLI |

## Build

```bash
make build    # compile all bridges to build/bin/
make test     # run tests
make lint     # go vet
make clean    # remove build artifacts
```

Each binary is standalone, implements MCP JSON-RPC over stdio.

## Protocol

All servers implement the MCP JSON-RPC 2.0 protocol over stdin/stdout:

- `mcp.list_tools` — Returns available tool metadata
- `<tool_name>` — Calls a tool with arguments
- `healthcheck` — Notification; responds with `healthcheck_ok`

See `product-specs/specs/mcp-conventions.md` for full protocol details.

## Dependencies

Zero external Go dependencies for gpio, display, audio, network. Serial uses raw syscalls (`syscall` package). Only `github.com/spf13/cobra` is not used — MCP is implemented directly.

## Cloning Convention
- Use SSH () for development.
- Use HTTPS () for build scripts that clone public dependencies.
## Cloning Convention
- Use SSH (git@github.com:) for development.
- Use HTTPS (https://github.com/) for build scripts that clone public dependencies.
