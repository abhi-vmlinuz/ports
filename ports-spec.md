# ports — A Human-Friendly Port & Process Inspector

> **Know what's using your port. Without remembering `lsof`, `ss`, `ps`, and `/proc` incantations.**

`ports` is a small, fast, Linux-first CLI written in **Go** that answers a simple question:

```text
"What is using this port?"
```

It is designed for developers, DevOps engineers, platform engineers, and anyone who spends a lot of time in a terminal.

The philosophy is the same as tools such as `fzf`, `zoxide`, `ripgrep`, `eza`, and `tq`: **take a recurring source of terminal friction and collapse it into a tiny, composable command.**

---

## 1. Project Goal

Build a Unix-native CLI that makes local port/process discovery dramatically easier.

The common workflow today looks like:

```bash
lsof -i :3000
ps -p <PID> -o etime,cwd,args
kill -9 <PID>
```

`ports` should turn that into:

```bash
ports 3000
```

and, when explicitly requested:

```bash
ports kill 3000
```

The tool should be:

- fast
- predictable
- readable
- scriptable
- safe around destructive operations
- dependency-light
- useful without configuration
- composable through stdout/stdin
- Linux-first
- implemented in Go using the standard library wherever practical

---

# 2. Scope

## v0.1 — MVP

The first release MUST focus on:

1. Discovering listening TCP/UDP sockets.
2. Mapping listening sockets to owning processes.
3. Displaying a clean human-readable table.
4. Inspecting a specific port.
5. Showing useful process metadata.
6. Killing the process associated with a port through an explicit command.
7. Providing machine-readable JSON output.
8. Providing meaningful exit codes.
9. Providing shell completion through Cobra.
10. Handling permissions and inaccessible `/proc` entries gracefully.

## v0.2 — Planned

After the MVP is stable:

- `ports --watch`
- process-name filtering
- `ports check <port>`
- Docker port/container correlation
- richer interactive output
- IPv6 improvements if required
- additional protocol/state filtering

## Future / Explicitly Out of v0.1

Do NOT implement these in v0.1:

- remote host inspection
- Kubernetes integration
- packet capture
- network traffic analysis
- firewall management
- port scanning
- service management
- automatic killing without explicit user intent
- configuration files
- background daemons
- AI features
- a giant TUI
- replacing `ss`, `lsof`, or `netstat` feature-for-feature

The MVP must remain small.

---

# 3. Target Platform

## Primary target

Linux.

The first implementation should use Linux `/proc` interfaces directly.

The code should isolate Linux-specific discovery logic so that future platforms can be added without contaminating the core model/rendering layers.

A reasonable future structure is:

```text
internal/
    proc/
    platform/
        linux/
```

Do not pretend that macOS/BSD/Windows are supported in v0.1.

If the binary is executed on an unsupported OS, fail clearly rather than silently behaving incorrectly.

---

# 4. Technology

## Language

**Go**

Target a modern Go version available at implementation time.

Use the standard library wherever possible.

Expected standard-library packages include:

- `os`
- `io`
- `bufio`
- `path/filepath`
- `strconv`
- `strings`
- `fmt`
- `time`
- `syscall` / appropriate OS APIs where required
- `encoding/json`
- `sort`
- `regexp` if filtering requires it

## CLI framework

Use **Cobra**.

The command structure should be conventional and shell-completion friendly.

Avoid introducing large dependencies for functionality that can reasonably be implemented with the Go standard library.

A terminal rendering dependency may be introduced later if the table renderer genuinely benefits from it, but v0.1 should remain dependency-light.

---

# 5. Command Surface

The intended v0.1 interface is:

```bash
ports
ports <port>
ports kill <port>
ports --json
ports <port> --json
ports kill <port> --force
ports --version
ports --help
```

## Default command

```bash
ports
```

Lists listening ports.

Default behavior:

- TCP and UDP
- listening/bound sockets only
- local machine only
- human-readable table
- sorted numerically by port
- no configuration required

Example:

```text
PORT   PROCESS   PID    USER   PROTO
22     sshd      812    root   tcp
3000   node      18421  abhi   tcp
5432   postgres  1932   abhi   tcp
8080   java      9121   abhi   tcp
```

Do not print excessive explanatory text by default.

The default output should feel like a Unix utility.

---

# 6. Inspecting a Specific Port

Command:

```bash
ports 3000
```

This should locate the process owning the listening socket on port `3000`.

Example:

```text
Port:       3000
Protocol:   TCP
Process:    node
PID:        18421
User:       abhi
CWD:        /home/abhi/projects/api
Command:    node server.js
Uptime:     12m 31s
```

If multiple processes/sockets match the port, display all relevant records rather than arbitrarily selecting one.

For example:

```text
Port: 3000

TCP
  Process: node
  PID:     18421
  User:    abhi
  CWD:     /home/abhi/projects/api
  Command: node server.js

TCP6
  Process: node
  PID:     18421
  User:    abhi
  CWD:     /home/abhi/projects/api
  Command: node server.js
```

Avoid falsely reporting duplicate processes merely because the same process owns both IPv4 and IPv6 sockets. The internal model should distinguish socket records from process records.

---

# 7. Process Metadata

When a port is mapped to a process, attempt to obtain:

- PID
- process name
- username
- UID
- command line
- current working directory
- process start time
- calculated uptime

Possible `/proc` sources:

```text
/proc/<pid>/comm
/proc/<pid>/cmdline
/proc/<pid>/status
/proc/<pid>/stat
/proc/<pid>/cwd
```

## Command line

`/proc/<pid>/cmdline` is NUL-separated.

Convert it into a normal shell-like readable representation.

Do not blindly assume spaces in arguments are safe to reconstruct perfectly as a shell command. The display is informational, not something that should be copy-pasted directly into a shell.

## CWD

Resolve:

```text
/proc/<pid>/cwd
```

through the symlink.

If permission is denied or the process disappears while reading it, display:

```text
<unavailable>
```

rather than failing the entire command.

---

# 8. Port Discovery Architecture

The implementation should NOT simply shell out to:

```bash
lsof
ss
netstat
```

The point of this tool is to understand the system directly.

For Linux, the intended conceptual pipeline is:

```text
/proc/net/tcp
/proc/net/tcp6
/proc/net/udp
/proc/net/udp6
        |
        v
socket information
        |
        v
socket inode
        |
        v
/proc/<pid>/fd/*
        |
        v
PID
        |
        v
process metadata
        |
        v
PortRecord
        |
        +----> table renderer
        |
        +----> JSON renderer
```

The implementation may improve upon this design if testing reveals a better approach, but the resulting behavior must remain equivalent.

---

# 9. `/proc/net/*` Parsing

The Linux files:

```text
/proc/net/tcp
/proc/net/tcp6
/proc/net/udp
/proc/net/udp6
```

contain socket information.

The implementation must parse at least:

- local address
- local port
- connection state
- socket inode
- protocol/family information

For TCP, only sockets in the appropriate listening state should be shown by default.

For UDP, Linux does not have TCP-style `LISTEN` semantics. Treat locally bound UDP sockets as listening/bound endpoints for the purpose of this tool.

Do not mislabel UDP as TCP LISTEN.

---

# 10. Hexadecimal Parsing

Linux `/proc/net/tcp*` and related files expose addresses/ports in hexadecimal.

The implementation must convert them correctly.

Example:

```text
1F90
```

represents:

```text
8080
```

Use safe parsing with `strconv.ParseUint`.

Do not manually perform fragile string arithmetic.

---

# 11. Socket → Process Mapping

The critical systems problem is mapping a socket inode back to its owning PID.

A process's file descriptors are available under:

```text
/proc/<pid>/fd/
```

Socket descriptors appear as symlinks similar to:

```text
socket:[123456]
```

The implementation should:

1. Discover socket inodes.
2. Iterate candidate `/proc/<pid>/fd` entries.
3. Identify socket symlinks.
4. Extract the inode.
5. Map inode → PID.
6. Enrich the PID with process metadata.

Do not assume that every socket can be mapped successfully.

Processes may:

- exit during inspection
- be hidden by permissions
- have inaccessible `/proc` entries
- change state between reads

The implementation must tolerate races.

---

# 12. Process Lifetime Races

This is important.

The system can change while `ports` is running.

For example:

```text
ports discovers PID 18421
        ↓
process exits
        ↓
ports reads /proc/18421/cmdline
        ↓
ENOENT
```

This is normal.

Do not crash.

Use graceful fallbacks.

Possible behavior:

```text
Process: node
PID:     18421
Command: <exited during inspection>
```

or omit the vanished record if it is no longer valid.

The tool should favor correct results over pretending the system is static.

---

# 13. User / UID Resolution

Determine the process UID from `/proc/<pid>/status`.

Resolve the UID to a username.

The default output should show the username.

If the username cannot be resolved:

```text
UID:1001
```

or an equivalent fallback is acceptable.

Do not fail because a username cannot be resolved.

---

# 14. Ownership Visualisation

The default table should make it immediately obvious whether the process belongs to the current user.

Conceptually:

```text
PORT   PROCESS   PID    USER   PROTO
22     sshd      812    root   tcp
3000   node      18421  abhi   tcp
5432   postgres  1932   abhi   tcp
```

If terminal styling is used:

- current-user processes may receive a subtle positive/highlight treatment
- root/system processes should visually differ
- color must NEVER be the only way information is communicated

For non-TTY output, automatically disable decorative color.

Avoid excessive colors.

The output must remain readable in:

```bash
ports > output.txt
ports | less
ports | tq
```

---

# 15. Sorting

Default sorting:

1. numeric port ascending
2. protocol
3. PID

Example:

```text
22
80
443
3000
5432
8080
```

Do not sort lexicographically:

```text
22
3000
443
5432
80
```

---

# 16. Filtering by Port

Numeric positional argument:

```bash
ports 3000
```

The argument must be a valid port number:

```text
1–65535
```

Reject:

```bash
ports abc
ports -1
ports 0
ports 65536
```

with a concise error.

Example:

```text
error: invalid port "70000": must be between 1 and 65535
```

Exit non-zero.

---

# 17. Kill Command

Use an explicit subcommand:

```bash
ports kill 3000
```

This is preferred over making destructive behavior a casual flag.

The command should:

1. resolve port → process
2. show exactly what will be killed
3. require explicit confirmation unless `--force` is provided
4. send a normal termination signal first
5. report the result

Example:

```text
Port 3000 is used by:

  node
  PID:     18421
  User:    abhi
  CWD:     /home/abhi/projects/api
  Command: node server.js

Kill PID 18421? [y/N]
```

After confirmation:

```text
✓ sent SIGTERM to PID 18421
```

Do NOT default to `SIGKILL`.

---

# 18. Kill Safety

This command is destructive.

Safety requirements:

- never silently kill a process
- never select an arbitrary process when multiple owners exist
- always display PID and process information
- default to SIGTERM
- require confirmation
- `--force` bypasses confirmation only
- `--force` does NOT automatically mean SIGKILL

If SIGTERM is insufficient, a future feature may support an explicit escalation option such as:

```bash
ports kill 3000 --signal KILL
```

but this is NOT required for v0.1.

Do not implement:

```bash
ports kill 3000
```

as an equivalent of:

```bash
kill -9 $(lsof -t -i:3000)
```

That would destroy the primary safety advantage of the tool.

---

# 19. Multiple Owners

If a port has multiple relevant socket/process records:

```text
Multiple processes are associated with port 3000:

PID     PROCESS
18421   node
19231   node
```

Do not silently choose one.

Require a future explicit selection mechanism or reject the kill request with a useful message.

For v0.1, safest behavior:

```text
error: multiple processes are associated with port 3000; refusing to kill automatically
```

---

# 20. Permission Errors

Running as a normal user may prevent access to some processes.

The tool must still show information it can access.

Example:

```text
PORT   PROCESS   PID    USER
3000   node      18421  abhi
5432   postgres  1932   postgres
```

If process metadata cannot be read:

```text
PROCESS   <permission denied>
```

Do not turn one inaccessible process into total command failure.

However, if fundamental socket discovery itself cannot be performed, return a clear fatal error.

---

# 21. JSON Output

Command:

```bash
ports --json
```

should produce valid JSON and NOTHING ELSE on stdout.

No decorative messages.

Example:

```json
[
  {
    "port": 3000,
    "protocol": "tcp",
    "address": "0.0.0.0",
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

The exact schema should be treated as an API once released.

Use stable, snake_case JSON field names.

For unavailable values, prefer:

```json
null
```

rather than fake strings such as:

```json
"unknown"
```

when the field is genuinely unavailable.

---

# 22. JSON Contract

Define a Go model approximately like:

```go
type PortRecord struct {
    Port          uint16 `json:"port"`
    Protocol      string `json:"protocol"`
    Address       string `json:"address"`
    PID           int    `json:"pid"`
    Process       string `json:"process"`
    UID           *int   `json:"uid"`
    User          *string `json:"user"`
    CWD           *string `json:"cwd"`
    Command       *string `json:"command"`
    UptimeSeconds *int64 `json:"uptime_seconds"`
}
```

The implementation may use a different internal representation.

The public JSON output should remain stable.

---

# 23. Exit Codes

The tool MUST be scriptable.

Suggested semantics:

```text
0 = successful execution
1 = runtime/operational error
2 = invalid command/argument
```

For:

```bash
ports check 3000
```

planned for v0.2:

```text
0 = port is free
1 = port is occupied
2 = invalid port
```

Do not implement `check` in v0.1 unless it can be done cleanly.

---

# 24. stdout vs stderr

Human-readable successful output:

```text
stdout
```

Errors:

```text
stderr
```

JSON successful output:

```text
stdout
```

Never pollute JSON stdout with logs.

This is critical for:

```bash
ports --json | jq .
```

and:

```bash
ports --json > ports.json
```

---

# 25. Watch Mode — v0.2

Planned command:

```bash
ports --watch
```

Expected behavior:

```text
PORT   PROCESS   PID    USER   PROTO
3000   node      18421  abhi   tcp
5432   postgres  1932   abhi   tcp
```

Refresh periodically.

The UX should eventually allow a user to watch:

```text
:3000
```

disappear when the old process exits and reappear when the new server starts.

Use Bubble Tea only if/when interactive TUI behavior is actually needed.

Do not pull Bubble Tea into the MVP simply because it is available.

---

# 26. Docker Integration — v0.2

Planned command:

```bash
ports --docker
```

The goal is to correlate host ports with Docker containers.

Example:

```text
PORT   PROCESS   PID    CONTAINER
3000   docker    ...    nexus-api
5432   docker    ...    postgres
8080   java      ...    -
```

Possible data source:

```bash
docker ps
```

or Docker's API.

Do not make Docker a mandatory dependency.

If Docker is unavailable:

```text
ports
```

must work normally.

---

# 27. Architecture

Keep responsibilities separated.

Suggested structure:

```text
ports/
├── cmd/
│   └── ports/
│       └── main.go
│
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── list.go
│   │   └── kill.go
│   │
│   ├── model/
│   │   └── port.go
│   │
│   ├── proc/
│   │   ├── sockets.go
│   │   ├── processes.go
│   │   ├── parser.go
│   │   └── mapper.go
│   │
│   ├── renderer/
│   │   ├── table.go
│   │   └── json.go
│   │
│   └── platform/
│       └── linux.go
│
├── tests/
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

This is a starting point, not a mandate.

Do not create packages merely to make the tree look sophisticated.

Prefer simple, cohesive packages.

---

# 28. Internal Data Model

Separate raw Linux parsing from the user-facing model.

For example:

```text
Linux socket record
        ↓
normalized socket
        ↓
PID association
        ↓
process metadata
        ↓
PortRecord
        ↓
renderer
```

This makes testing substantially easier.

The renderer must NOT know how `/proc/net/tcp` works.

The `/proc` parser must NOT know how tables are rendered.

---

# 29. Rendering

Human output should be deliberately boring.

That is a compliment.

Example:

```text
PORT   PROCESS   PID    USER   PROTO
22     sshd      812    root   tcp
3000   node      18421  abhi   tcp
5432   postgres  1932   abhi   tcp
8080   java      9121   abhi   tcp
```

Avoid:

- giant banners
- ASCII logos
- unnecessary emojis
- verbose explanations
- decorative borders
- animations in normal mode

The user wants information, not a dashboard.

---

# 30. Terminal Width

The renderer should handle narrow terminals sensibly.

If the terminal is too narrow:

- truncate long command lines
- preserve important columns
- do not wrap the entire table into unreadability

For example:

```text
COMMAND
node server.js --config /home/abhi/projects/api/...
```

Long command strings should not break the table.

---

# 31. TTY Detection

Decorative formatting should be enabled only when stdout is a TTY.

When output is piped:

```bash
ports | grep node
```

or redirected:

```bash
ports > ports.txt
```

produce plain output.

JSON mode is always plain machine-readable output.

---

# 32. Error Handling Philosophy

Errors should be:

- concise
- actionable
- written to stderr
- non-panicking
- useful to humans

Bad:

```text
panic: runtime error
```

Good:

```text
error: unable to inspect PID 18421: permission denied
```

Better when appropriate:

```text
error: port 3000 is associated with a process you do not have permission to inspect
hint: try running ports with elevated privileges
```

Do not recommend `sudo` for every problem.

Only recommend it when necessary.

---

# 33. Testing Strategy

This project MUST have tests.

Do not rely exclusively on manually running:

```bash
ports
```

## Unit tests

Test:

- hexadecimal port parsing
- TCP state parsing
- IPv4 parsing
- IPv6 parsing
- `/proc/net/*` line parsing
- socket inode extraction
- command-line parsing
- port validation
- sorting
- JSON serialization
- process metadata parsing

Use fixture strings/files instead of depending on the developer's live `/proc`.

---

# 34. Integration Tests

Where practical, create real local sockets during tests.

Example concept:

```text
test starts TCP listener on ephemeral port
        ↓
run discovery
        ↓
assert listener port is detected
        ↓
assert PID/process association is correct
```

Avoid tests that depend on fixed ports such as `3000`, because another application may already occupy them.

Use ephemeral ports.

---

# 35. Kill Testing

Be extremely careful.

Never write tests that can accidentally kill arbitrary system processes.

The kill tests should:

1. spawn a child process created specifically for the test
2. have the child bind an ephemeral port
3. invoke the kill logic
4. verify the child terminates
5. clean up even when assertions fail

Never use:

```text
PID 1
sshd
systemd
docker
postgres
```

as test targets.

---

# 36. Security Considerations

`ports kill` is a process-control operation.

Requirements:

- never execute a shell command to kill a PID
- use OS-level process signaling
- validate the PID obtained from discovery
- never accept arbitrary shell expressions as process selectors
- never construct shell commands from user input
- default to SIGTERM
- require confirmation
- clearly display target process information

Do NOT implement:

```go
exec.Command("sh", "-c", "kill -9 "+pid)
```

Use the appropriate Go process/signal APIs.

---

# 37. Performance

The tool should feel instantaneous on a normal developer workstation.

The expensive part is likely scanning:

```text
/proc/<pid>/fd/
```

across many processes.

Design the discovery layer sensibly.

Potential optimisations:

- parse socket tables first
- only inspect PIDs relevant to discovered socket inodes where possible
- avoid reading the same process metadata repeatedly
- cache metadata during one invocation
- avoid unnecessary recursive filesystem traversal

Do NOT prematurely build a persistent daemon or complicated cache.

A single invocation should be enough.

---

# 38. Process Metadata Caching

During one invocation:

```text
PID 18421
```

may own multiple sockets.

Do not repeatedly read:

```text
/proc/18421/status
/proc/18421/cmdline
/proc/18421/cwd
```

for every socket.

Cache process metadata per PID for the duration of the command.

---

# 39. Duplicate Handling

A process may own:

```text
0.0.0.0:3000
[::]:3000
```

and therefore appear through multiple socket entries.

The internal representation should preserve protocol/family/socket information but the default display should avoid confusing duplicate process rows where appropriate.

Define a deterministic deduplication policy.

Do not simply deduplicate based on PID alone, because one process may genuinely own multiple different ports.

---

# 40. IPv4 / IPv6

Support:

```text
/proc/net/tcp
/proc/net/tcp6
/proc/net/udp
/proc/net/udp6
```

Represent addresses consistently.

Examples:

```text
0.0.0.0
127.0.0.1
192.168.1.10
::
::1
```

Do not display raw hexadecimal IPv6 data to users.

---

# 41. Address Semantics

The user should be able to understand whether a service is:

```text
localhost-only
```

or:

```text
all interfaces
```

or:

```text
specific interface
```

For example:

```text
ADDRESS       PORT   PROCESS
127.0.0.1     3000   node
0.0.0.0       8080   java
```

The initial default table may omit ADDRESS to stay compact, but the detailed port view should expose it.

---

# 42. CLI Help

Help should be concise and useful.

Example:

```text
$ ports --help

Find processes listening on local ports.

Usage:
  ports [port]
  ports kill <port>

Examples:
  ports
  ports 3000
  ports kill 3000
  ports --json

Commands:
  kill        terminate the process using a port

Flags:
  --json      output machine-readable JSON
  --force     skip confirmation for destructive operations
  -h, --help  help for ports
  -v, --version
```

Do not dump a huge manual into `--help`.

Put detailed documentation in README/man pages.

---

# 43. Shell Completion

Cobra should provide completion.

At minimum support:

```bash
ports <TAB>
ports kill <TAB>
```

For port numbers, dynamic completion is optional and NOT required for v0.1.

Generate completion for:

- bash
- zsh
- fish
- PowerShell if Cobra setup makes it trivial

Do not spend significant development time custom-building completion logic.

---

# 44. Version Information

Support:

```bash
ports --version
```

Version should be injected at build time.

Example:

```text
ports version 0.1.0
```

Use linker flags rather than hard-coding a release version in multiple files.

---

# 45. Build

The project should support:

```bash
go build ./...
go test ./...
go vet ./...
```

A Makefile may provide:

```bash
make build
make test
make lint
make install
```

The exact Makefile targets are up to implementation.

The project should produce a standalone Linux binary.

---

# 46. Cross Compilation

The project should remain compatible with standard Go cross-compilation mechanisms.

Example:

```bash
GOOS=linux GOARCH=amd64 go build
GOOS=linux GOARCH=arm64 go build
```

Because `/proc` discovery is Linux-specific, do not claim cross-platform runtime support merely because Go can cross-compile.

---

# 47. Documentation

README must explain:

1. What `ports` solves.
2. Why it exists.
3. Installation.
4. Basic usage.
5. Detailed port inspection.
6. Kill workflow.
7. JSON usage.
8. Exit codes.
9. Linux requirement.
10. Security considerations.
11. Development.
12. Roadmap.

Lead with examples.

Do not lead with architecture.

---

# 48. Example README Opening

The eventual README should communicate the value immediately:

```text
# ports

See what's using your ports.

Instead of:

lsof -i :3000
ps -p 18421 -o etime,cwd,args

just:

ports 3000

Port:       3000
Protocol:   TCP
Process:    node
PID:        18421
User:       abhi
CWD:        /home/abhi/projects/api
Command:    node server.js
Uptime:     12m 31s
```

The tool should feel like a tiny Unix primitive, not an enterprise product.

---

# 49. Design Principles

These are non-negotiable.

## 49.1 Do one thing well

`ports` is not a network diagnostic suite.

Its primary job is:

> **port → process**

Everything else must support that.

## 49.2 Human first, machine compatible

Default output should be beautiful and readable.

`--json` should be stable and scriptable.

## 49.3 No configuration

Install it.

Run:

```bash
ports
```

It should work.

## 49.4 Safe destructive operations

Never surprise the user with process termination.

## 49.5 Unix composability

This must work:

```bash
ports | grep node
```

```bash
ports --json | tq
```

```bash
ports --json > ports.json
```

## 49.6 Don't hide complexity behind magic

The tool should make Linux easier to understand, not obscure what is happening.

## 49.7 Small binary, small codebase

Prefer 1,000 lines of clear code over 5,000 lines of abstractions.

---

# 50. Non-Goals

`ports` is NOT:

- a port scanner
- a vulnerability scanner
- a firewall
- a packet analyzer
- a replacement for Wireshark
- a replacement for `nmap`
- a replacement for `lsof`
- a replacement for `ss`
- a process manager
- a service manager
- a Kubernetes networking tool
- a Docker management CLI

It is a **developer-oriented local port/process inspector**.

---

# 51. Definition of Done — v0.1

v0.1 is complete when all of the following work:

### Discovery

```bash
ports
```

correctly lists listening TCP and bound UDP ports.

### Inspection

```bash
ports 3000
```

shows:

- port
- address
- protocol
- process
- PID
- user
- CWD where available
- command where available
- uptime where available

### JSON

```bash
ports --json
ports 3000 --json
```

produces valid JSON with no stdout contamination.

### Kill

```bash
ports kill 3000
```

shows the target and asks for confirmation.

```bash
ports kill 3000 --force
```

skips confirmation.

Default signal is SIGTERM.

### Errors

Invalid ports, missing ports, permission issues, and disappearing processes are handled cleanly.

### Tests

```bash
go test ./...
```

passes.

### Static checks

```bash
go vet ./...
```

passes.

### Build

```bash
go build ./...
```

passes.

### CLI

Cobra help, version, and completion work.

### Documentation

README explains the tool and its safety model.

---

# 52. Suggested Implementation Order

Do not implement everything simultaneously.

Build in this order:

```text
1. Initialize Go module
        ↓
2. Cobra root command
        ↓
3. PortRecord model
        ↓
4. Parse /proc/net/tcp
        ↓
5. Parse /proc/net/tcp6
        ↓
6. Parse /proc/net/udp
        ↓
7. Parse /proc/net/udp6
        ↓
8. Socket inode discovery
        ↓
9. PID mapping
        ↓
10. Process metadata
        ↓
11. Human table renderer
        ↓
12. `ports <port>`
        ↓
13. JSON renderer
        ↓
14. Error handling
        ↓
15. Kill command
        ↓
16. Tests
        ↓
17. README
        ↓
18. Release v0.1.0
```

Do NOT start with Docker, Bubble Tea, or watch mode.

---

# 53. Agent Instructions

The coding agent should follow these rules:

1. Read this specification before changing architecture.
2. Implement v0.1 only.
3. Do not silently expand scope.
4. Do not add dependencies without justification.
5. Do not shell out to `lsof`, `ss`, `netstat`, or similar tools for core discovery.
6. Use Linux `/proc` interfaces for the core implementation.
7. Keep Linux-specific code isolated.
8. Write tests alongside implementation.
9. Run `go test ./...` frequently.
10. Run `go vet ./...`.
11. Never weaken kill safety to make implementation easier.
12. Do not introduce a TUI in v0.1.
13. Do not implement Docker integration in v0.1.
14. Do not create configuration files.
15. Do not add AI functionality.
16. Prefer straightforward Go over clever abstractions.
17. Preserve clean stdout/stderr semantics.
18. Treat JSON output as an API.
19. Handle `/proc` races gracefully.
20. Before declaring completion, verify every item in the Definition of Done.

If a design decision is not specified, choose the smallest, most Unix-like solution that preserves composability and safety.

---

# 54. Product Philosophy

`ports` belongs to a larger family of terminal tools built around one principle:

> **The computer already knows. Don't make the human interrogate it manually.**

Examples:

```text
tq
→ Make structured JSON output understandable.

ports
→ Make port ownership understandable.

peek
→ Make files and directories understandable.

recent
→ Make recent work discoverable.

waitfor
→ Make asynchronous state waitable.

retry
→ Make transient failures recoverable.
```

Each tool should solve a narrow problem exceptionally well.

The objective is not to create another giant CLI.

The objective is to create **tiny lifesavers** that become part of a developer's muscle memory.

---

# 55. Final Product Standard

Before accepting a release, ask:

```text
Would I install this on a fresh Linux machine?

Would I remember the command without checking the README?

Does it save me from a command chain I currently type?

Is the default output immediately understandable?

Can I pipe it into another Unix tool?

Does the destructive operation make me feel safe?

Is the implementation smaller than the problem deserves?

Does it solve a real annoyance rather than demonstrate technology?
```

If the answer to these is not consistently "yes", the feature is not ready.

**Build the smallest useful `ports` first.**
