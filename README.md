# ports

`ports` is a small Linux CLI written in Go that answers one question: **what is listening on this port, and what process owns it?**

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

`ports` directly parses the Linux kernel's socket tables in `/proc/net/` and maps socket inodes back to processes via `/proc/<pid>/fd/`. It doesn't shell out to external binaries, requires no background daemon or configuration, and handles unprivileged permissions cleanly by showing socket owners even when individual process details require elevation.

---

## Installation

### From source (Go 1.22+)

```bash
git clone https://github.com/abhi-vmlinuz/ports.git
cd ports
make install
```

This builds the binary to `bin/ports`, installs it to `/usr/bin`, and automatically registers shell completions for Fish and Zsh.

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

### List listening ports

Running `ports` with no arguments prints a sorted table of all active TCP listeners and locally bound UDP endpoints:

```bash
ports
```

```text
PORT    PROTO  ADDRESS      PROCESS   PID    USER
22      tcp    0.0.0.0      sshd      812    root
3000    tcp    127.0.0.1    node      18421  elish4h
5432    tcp    127.0.0.1    postgres  1932   postgres
8080    tcp    0.0.0.0      java      9121   elish4h

● 4 listening ports (2 user, 2 system)
```

- Processes belonging to your user account are highlighted so your own servers stand out immediately.
- System and root services are dimmed.
- Dual-stack services show IPv4 and IPv6 bindings as separate rows to reflect actual kernel socket state.

### Interactive split-pane watch mode

To monitor ports live as servers start and stop, use watch mode:

```bash
ports --watch
# or shorthand
ports -w
```

- **Split layout**: The left pane lists active ports; the right panel shows full metadata for the currently selected item.
- **Navigation**: Move selection with `↑` / `↓` or Vim `j` / `k`. Mouse wheel scrolling is supported.
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

You can also scope watch mode to a single port:

```bash
ports 3000 -w
```

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
  User:      elish4h (current user)
  CWD:       ~/projects/api
  Command:   node server.js
  Uptime:    14m 32s
```

If multiple processes or protocols bind the same port (for example, separate IPv4 and IPv6 listeners), each record is displayed.

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

Running `make install` installs Fish and Zsh completions automatically if their user directories exist.

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
