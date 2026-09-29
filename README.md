# ports

<p align="center">
  <strong>Fast Linux CLI and live TUI that answers: what is listening on this port, and what process owns it?</strong>
</p>

<p align="center">
  <a href="https://github.com/abhi-vmlinuz/ports/releases"><img src="https://img.shields.io/github/v/release/abhi-vmlinuz/ports?style=flat-square&color=blue" alt="Release"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
</p>

---

Instead of remembering and chaining tools:

```bash
lsof -i :3000
ps -p 18421 -o etime,cwd,args
kill -9 18421
```

you run:

```bash
ports 3000
```

```text
● Port 3000  [TCP]
  Interface: 127.0.0.1 (localhost)
  Process:   node
  PID:       18421
  Origin:    interactive
  User:      elish4h (current user)
  CWD:       ~/projects/api
  Command:   node server.js
  Uptime:    14m 32s
```

and when you need to stop it:

```bash
ports kill 3000
```

---

## Why it exists

When a local port conflict happens during development, the typical fix involves running `lsof` or `ss`, copying a PID, inspecting `ps` to make sure it's the right process, and then killing it. If you aren't running as root, `lsof` often prints nothing at all without explaining why.

`ports` directly parses the Linux kernel's socket tables in `/proc/net/` and maps socket inodes back to processes via `/proc/<pid>/fd/`. It inspects `/proc/<pid>/cgroup` to pinpoint the process origin (Docker container, systemd service, or interactive shell), doesn't shell out to external binaries, requires no background daemon or configuration, and handles unprivileged permissions cleanly by showing socket owners even when individual process details require elevation.

---

## Installation

### From source (Go 1.21+)

```bash
git clone https://github.com/abhi-vmlinuz/ports.git
cd ports
make install
```

This builds the binary to `bin/ports`, installs it to `/usr/bin`, and automatically registers shell completions for Bash, Fish, and Zsh.

You can also install directly with the Go toolchain:

```bash
go install ./cmd/ports
```

### Running without sudo (Linux Capabilities)

On Linux, `/proc/<pid>/fd/` has `0700` permissions owned by the process's user. Reading another user's file descriptors (such as system daemons or containers running as root) requires elevation.

Rather than running the binary under `sudo` or using dangerous `setuid root` permissions, you can grant `ports` the single capability it needs to read file descriptors:

```bash
make setcap
# runs: sudo setcap cap_sys_ptrace+ep /usr/bin/ports
```

`CAP_SYS_PTRACE` allows reading `/proc/<pid>/fd/` symlinks across all users without granting network administration or write permissions.

---

## Usage

### Interactive split-pane dashboard (default)

Running `ports` (or `sudo ports`) in an interactive terminal opens the live split-pane TUI dashboard:

```bash
ports
```

- **Split layout**: The left pane lists active ports with protocol, bind address, process name, and PID. The right pane shows comprehensive metadata (origin controller, user, CWD, full command line, and uptime) for the selected process.
- **Theme support (`t`)**: Press `t` to cycle through 16 built-in TrueColor themes in real time. Each styled theme paints the entire terminal window with solid 24-bit background colors matching `rinode` (Cyberpunk, Catppuccin, Gruvbox, etc.), while `default` preserves transparent terminal wallpapers. Your choice is automatically persisted to `~/.config/ports/config.json` (and cleanly saved to `$SUDO_USER`'s home directory when running under `sudo`).
- **Live search bar (`/`)**: Press `/` or click the search box to search ports and processes. Filters in real time on **every keystroke** (like `fzf`):
  - Numeric queries match port numbers (`80`, `:3000`) and process PIDs (`1435`, `pid:1435`).
  - Text queries match process names with substring and fuzzy subsequence matching (`brave`, `brv`, `node`).
  - Multi-token queries supported (`node 8080`, `tcp 53`, `docker 3000`).
  - Matched characters in Port, Process, and PID columns are highlighted directly in the table.
  - Press `Enter` or `↓` to focus the filtered table; press `Esc` to clear the filter.
- **Navigation & Scrolling**: Move selection with `↑` / `↓` or Vim `j` / `k`. Scroll smoothly through long lists with mouse wheel, `PageUp` / `PageDown` (`Ctrl+D` / `Ctrl+U`), and jump directly to top or bottom with `Home` / `End` (`g` / `G`). The TUI dynamically handles window resize (`SIGWINCH`), seamlessly adapting column widths, toggling split layout on narrow windows, and clamping scroll offsets to prevent blank spaces.
- **Action popup modal**: Press `Enter`, `Space`, `m`, or click any port row to open the interactive action menu:
  1. Kill process (`SIGTERM`)
  2. Force kill (`SIGKILL`)
  3. Copy JSON to clipboard
  4. Copy PID
  5. Copy Command line
  6. Copy Address (`IP:Port`)
- **Clipboard export**: Uses terminal OSC 52 sequences (works over SSH and local terminal emulators) with native `wl-copy`, `xclip`, and `xsel` fallbacks.
- **Safe termination**: Quick kill with `x` or through the action menu, with confirmation prompt (`[y/N]`).
- **Refresh / Quit**: Press `r` to trigger a manual scan, or `q` / `Esc` to exit.

You can also scope the dashboard to a single port:

```bash
ports 3000 -w
```

### Static snapshot table

To print a one-off snapshot table directly to stdout without launching the interactive TUI, use `-s` or `--snapshot`:

```bash
ports -s
# or
ports --snapshot
```

```text
PORT    PROTO  ADDRESS      PROCESS   PID    USER
22      tcp    0.0.0.0      sshd      812    root
3000    tcp    127.0.0.1    node      18421  elish4h
5432    tcp    127.0.0.1    postgres  1932   postgres
8080    tcp    0.0.0.0      java      9121   elish4h

● 4 listening ports (2 user, 2 system)
```

- When stdout is piped or redirected (`ports | grep node`, `ports > listening.txt`), `ports` automatically outputs the snapshot table without needing `-s`.
- Processes belonging to your user account are highlighted so your own servers stand out immediately.
- System and root services are dimmed.

### Themes

`ports` includes 16 TrueColor palettes:

```bash
# List available themes with color previews
ports --theme list

# Run with a specific theme
ports --theme catppuccin
ports -t dracula -s
```

Available palettes: `default`, `catppuccin`, `nord`, `dracula`, `gruvbox`, `tokyo_night`, `rose_pine`, `one_dark`, `monokai`, `kanagawa`, `cyberpunk`, `everforest`, `ayu`, `synthwave`, `solarized`, and `matrix`.

Inside the TUI, press `t` to cycle through them. Your selected theme is remembered across sessions.

### Inspect a specific port

Pass the port number as an argument. A leading colon is accepted as an alias:

```bash
ports 3000
# or
ports :3000
```

```text
● Port 3000  [TCP]
  Interface: 127.0.0.1 (localhost)
  Process:   node
  PID:       18421
  Origin:    interactive
  User:      elish4h (current user)
  CWD:       ~/projects/api
  Command:   node server.js
  Uptime:    14m 32s
```

If multiple processes or protocols bind the same port (for example, separate IPv4 and IPv6 listeners), each record is displayed.

### Scripting: Print PID only (`--pid`, `-p`)

When writing shell scripts, aliases, or CI pipelines, pass `--pid` (or `-p`) to output only the PID of the process listening on a port:

```bash
# Terminate process on port 3000
kill $(ports 3000 -p)

# Force-kill whatever is holding port 8080
kill -9 $(ports 8080 --pid)

# Check if port 5432 has an active listener
if pid=$(ports 5432 -p 2>/dev/null); then
  echo "Database running with PID $pid"
fi
```

- Outputs only the numeric PID to stdout with zero formatting or header text.
- If multiple processes are bound to the port, distinct PIDs are printed on separate lines.
- Exits with return code `0` if a PID was found, or `1` if the port is unoccupied or the PID cannot be resolved.

### Kill a process on a port

To terminate whatever is holding a port:

```bash
ports kill 3000
```

`ports kill` resolves the owning PID, displays the command and working directory, and prompts for confirmation:

```text
Port 3000 is used by:

  node
  PID:     18421
  User:    elish4h
  CWD:     /home/elish4h/projects/api
  Command: node server.js

Kill PID 18421? [y/N]: y
✓ sent SIGTERM to PID 18421
```

- Signals are sent directly via `syscall.Kill`. It never invokes shell commands like `kill -9 $(...)`.
- The default signal is `SIGTERM` to allow clean process shutdown.
- If multiple distinct processes own sockets on the port, `ports kill` refuses to guess and prints an error listing the PIDs.
- To bypass interactive confirmation in scripts or CI:

```bash
ports kill 3000 --force
```

### JSON output

Add `--json` to output machine-readable JSON to stdout:

```bash
ports --json | jq .
```

```json
[
  {
    "port": 3000,
    "protocol": "tcp",
    "address": "127.0.0.1",
    "pid": 18421,
    "process": "node",
    "origin": "interactive",
    "uid": 1000,
    "user": "elish4h",
    "cwd": "/home/elish4h/projects/api",
    "command": "node server.js",
    "uptime_seconds": 872
  }
]
```

Inspect a single port as JSON:

```bash
ports 3000 --json
```

All warnings, hints, and errors go strictly to stderr, keeping stdout clean for piping into tools like `jq` or `tq`.

---

## Unix composability

When stdout is redirected or piped, all ANSI color codes and decorative characters are stripped automatically:

```bash
# Filter plain output
ports | grep node

# Save table to a file
ports > active_ports.txt

# Extract listening port numbers with jq
ports --json | jq '.[].port'
```

`ports` also respects the `NO_COLOR` environment variable:

```bash
NO_COLOR=1 ports
```

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success (listening ports found and listed, or requested port inspected) |
| `1` | Operational error (port not in use, process not found, kill failed) |
| `2` | Syntax error (invalid arguments or out-of-range port number) |

---

## Shell completion

Completion scripts can be generated directly:

```bash
# Bash
ports completion bash | sudo tee /etc/bash_completion.d/ports

# Zsh
ports completion zsh > ~/.zsh/completion/_ports

# Fish
ports completion fish > ~/.config/fish/completions/ports.fish
```

Running `make install` or `sudo make install` installs system-wide completions to `/usr/share/bash-completion/completions/ports`, `/usr/share/zsh/site-functions/_ports`, and `/usr/share/fish/vendor_completions.d/ports.fish` automatically.

---

## Development

```bash
# Build binary
make build

# Run unit and integration tests
make test

# Static analysis
make lint
```

Integration tests spawn ephemeral TCP/UDP listeners on `127.0.0.1:0` to verify live `/proc` discovery, and spawn isolated child processes to test SIGTERM termination safety without affecting host services.

---

## License

[MIT](LICENSE)
