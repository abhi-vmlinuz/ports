# ports

> **Know what's using your port. Without remembering `lsof`, `ss`, `ps`, and `/proc` incantations.**

Instead of:

```bash
lsof -i :3000
ps -p 18421 -o etime,cwd,args
kill -9 18421
```

just:

```bash
ports 3000
```

```text
● Port 3000 (TCP)
  Interface:  127.0.0.1
  Process:    node
  PID:        18421
  User:       abhi (current user)
  CWD:        ~/projects/api
  Command:    node server.js
  Uptime:     12m 31s
```

and when you want to terminate it safely:

```bash
ports kill 3000
```

---

## 1. Why `ports`?

When a port conflict happens during development or deployment, developers inevitably end up chaining commands together: `lsof -i :<port>`, copying the PID, running `ps` to check what it is, and running `kill -9`.

`ports` collapses this recurring terminal friction into a single, predictable, Unix-native binary.

- **Fast & Direct**: Directly parses `/proc/net/{tcp,tcp6,udp,udp6}` and `/proc/<pid>/fd/*`. Does not shell out to `lsof`, `ss`, or `netstat`.
- **Human First, Machine Compatible**: Beautiful tabular output on interactive terminals, clean uncolored output when piped, and strict JSON on demand.
- **Safe by Default**: Process kills default to `SIGTERM`, prompt for confirmation, and refuse to kill when multiple processes share a port.
- **Zero Configuration**: Single standalone binary with zero runtime dependencies.

---

## 2. Installation

### From Source (Go 1.22+)

```bash
git clone https://github.com/abhi/ports.git
cd ports
make install
```

Or install with `go install`:

```bash
go install ./cmd/ports
```

---

## 3. Usage

### List All Listening Ports

```bash
ports
```

```text
PORT    PROTO  ADDRESS      PROCESS   PID    USER
22      tcp    0.0.0.0      sshd      812    root
3000    tcp    127.0.0.1    node      18421  abhi
5432    tcp    127.0.0.1    postgres  1932   postgres
8080    tcp    0.0.0.0      java      9121   abhi

● 4 listening ports (2 user, 2 system)
```

- **Current User Highlighted**: Dev processes owned by your user account are highlighted so you spot them instantly.
- **System/Root Dimmed**: Background system services are visually muted.
- **Dual-Stack Accuracy**: Shows IPv4 and IPv6 bindings explicitly.

### Watch Mode (Interactive Split-Pane TUI)

Launch the interactive split-pane dashboard with live process inspection:

```bash
ports --watch
# or watch a specific port
ports 3000 -w
```

- **Split-Pane Layout**: Left panel lists all ports; right side panel displays full live details (Process, PID, User, CWD, Command, Uptime) for the selected item.
- **Navigation**: Use `↑` / `↓` arrow keys or Vim `j` / `k` to move selection.
- **Mouse Support**: Click any row with your mouse to inspect it immediately.
- **In-TUI Safe Kill**: Press `x` or `d` to terminate the selected process with in-TUI confirmation (`[y/N]`).
- **Refresh & Quit**: Press `r` to force refresh, `q` or `Esc` to exit cleanly.

### Inspect a Specific Port

Accepts standard port numbers as well as the `:port` leading-colon alias:

```bash
ports 3000
# or
ports :3000
```

```text
● Port 3000 (TCP)
  Interface:  127.0.0.1
  Process:    node
  PID:        18421
  User:       abhi (current user)
  CWD:        ~/projects/api
  Command:    node server.js
  Uptime:     12m 31s
```

### Safely Kill a Port's Process

```bash
ports kill 3000
```

```text
Port 3000 is used by:

  node
  PID:     18421
  User:    abhi
  CWD:     /home/abhi/projects/api
  Command: node server.js

Kill PID 18421? [y/N]: y
✓ sent SIGTERM to PID 18421
```

To skip interactive confirmation (e.g. in CI or trusted scripts):

```bash
ports kill 3000 --force
```

> [!NOTE]
> `ports kill` always sends `SIGTERM` first, allowing processes to shut down gracefully and flush connections or write locks.

---

## 4. Machine-Readable JSON

Pass `--json` to output strict JSON to `stdout`:

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
    "user": "abhi",
    "cwd": "/home/abhi/projects/api",
    "command": "node server.js",
    "uptime_seconds": 751
  }
]
```

Inspect a specific port as JSON:

```bash
ports 3000 --json
```

All diagnostic messages, hints, and logs are directed exclusively to `stderr`, keeping `stdout` pure and streamable into `jq`, `tq`, or shell pipelines.

---

## 5. Unix Composability

`ports` automatically detects when its output is redirected or piped and strips all ANSI colors:

```bash
# Filter processes
ports | grep node

# Save table to a file
ports > listening_ports.txt

# Extract PIDs
ports --json | jq '.[].pid'
```

Honors the standard `NO_COLOR` environment variable:

```bash
NO_COLOR=1 ports
```

---

## 6. Exit Codes

`ports` provides deterministic, scriptable exit codes:

| Exit Code | Meaning |
|---|---|
| `0` | Successful execution (ports listed or requested port found) |
| `1` | Operational error (process not found, kill failed, permission failure) |
| `2` | Invalid command line syntax or out-of-range port number |

---

## 7. Platform Requirement

`ports` is designed Linux-first and communicates directly with Linux `/proc` (`/proc/net/*`, `/proc/<pid>/fd/*`, `/proc/<pid>/{comm,cmdline,cwd,status,stat}`).

If executed on an unsupported operating system, `ports` exits cleanly with an informative message.

---

## 8. Security & Safety

- **OS-Level Signaling**: `ports kill` uses direct OS signal delivery (`syscall.Kill`) and never executes unvalidated shell commands like `kill -9 $(...)`.
- **Multi-Process Ambiguity Guard**: If multiple distinct processes share a port (e.g. separate listeners), `ports kill` refuses to guess and aborts with an informative error.
- **Graceful Unprivileged Mode**: When run without `sudo`, ports belonging to other system users still appear in discovery using the socket's kernel UID, marking inaccessible process metadata as `<permission denied>` without crashing.

---

## 9. Development & Installation

```bash
# Build
make build

# Run unit and integration tests
make test

# Static analysis
make lint

# Install binary
make install

# Optional: grant zero-sudo inspection capabilities
make setcap
```

---

## 10. Roadmap

- [x] **v0.1**: Core `/proc` engine, listening TCP & bound UDP discovery, single port inspection, interactive safe kill, JSON output, shell completion, TTY styling.
- [x] **v0.2 (In progress)**: Live `--watch` TUI mode (`ports --watch`), `make setcap` support.
- [ ] Docker container port correlation (`ports --docker`), availability checks (`ports check <port>`), process-name filtering (`ports --process <name>`).
