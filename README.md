# core-mcp-bridges

Hardware abstraction layer for CognitiveOS — lightweight MCP servers that expose device capabilities to the AI over stdio JSON-RPC 2.0.

## Bridges

| Server | Exposes | Backend |
|--------|---------|---------|
| `display-mcp` | render_image, render_video, screenshot, clear | fbv, mpv --vo=drm, /dev/fb0 |
| `audio-mcp` | play, capture, tts, set_volume, mute, list_devices | ALSA (aplay, arecord, amixer), espeak |
| `network-mcp` | scan, connect, disconnect, status, list_interfaces | iw, wpa_supplicant, dhcpcd, ip |
| `gpio-mcp` | pin_read, pin_write, pwm, mode, list_pins | /sys/class/gpio, /sys/class/pwm |
| `serial-mcp` | list_ports, connect, send, receive, disconnect | /dev/tty\* raw syscalls |

## Build

```bash
go build -o bin/display-mcp ./display
go build -o bin/audio-mcp ./audio
go build -o bin/network-mcp ./network
go build -o bin/gpio-mcp ./gpio
go build -o bin/serial-mcp ./serial
```

Each binary is standalone and implements the MCP protocol over stdin/stdout.

## Protocol

All servers implement MCP JSON-RPC 2.0 over stdio:

- `mcp.list_tools` — Returns available tool metadata
- `<tool_name>` — Calls a tool with arguments
- `healthcheck` — Notification; responds with `healthcheck_ok`

See `product-specs/specs/mcp-conventions.md` for the full protocol spec.

## Dependencies

Zero external Go dependencies for gpio, display, audio, network. Serial uses raw syscalls (`syscall` package).

## Related

- [CognitiveOS](https://github.com/CognitiveOS-Project/cognitiveos) — main project repository
- [cognitive-os.org](https://cognitive-os.org) — project website
- [cognitiveosd](https://github.com/CognitiveOS-Project/cognitiveosd) — daemon that spawns these MCP servers
- [Product Specs](https://github.com/CognitiveOS-Project/product-specs) — MCP protocol conventions
- [CognitiveOS Project](https://github.com/CognitiveOS-Project) — GitHub organization

## Contributing

1. Branch from `development`, not `main`
2. Use topic branches: `feature/<name>`, `fix/<name>`, `bugfix/<name>`
3. Open a PR to `development` with a clear title and description
4. Merge via squash after review
5. Changes flow to `main` via a release PR

See the [SDLC repo](https://github.com/CognitiveOS-Project/sdlc) for the full contribution guide, code review standards, and testing strategy.

## Author

**Jean Machuca** — [GitHub](https://github.com/jeanmachuca) · [Sponsor](https://github.com/sponsors/jeanmachuca)

## License

MIT
