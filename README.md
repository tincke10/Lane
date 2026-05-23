# Lane

> Port-collision-free project switcher for parallel development.

Lane is a small CLI that lets you run multiple development projects in
parallel without their ports stepping on each other. It does **one thing**:
it remembers which ports each of your projects has reserved, and exports
those ports as environment variables into your current shell — the same
pattern used by [direnv](https://direnv.net/) and
[asdf](https://asdf-vm.com/).

Lane never edits your project files. It does not start, stop, or proxy
anything. It is a thin coordination layer over the convention your tools
already understand:

```yaml
# docker-compose.yml (the convention)
services:
  app:
    ports:
      - "${APP_PORT:-80}:80"   # APP_PORT supplied by Lane
```

```bash
$ eval "$(lane use my-laravel-app)"
$ docker compose up           # uses Lane's reserved APP_PORT
```

---

## Why Lane?

If you have ever tried to run a Laravel/Sail project, a Vite dev server,
and a Postgres container in parallel — across two or three different
projects — you have seen this dance:

- Project A and project B both want `:80`, `:3306`, `:5173`.
- You hand-edit `docker-compose.yml` or `.env` in one of them.
- You forget. Two weeks later you boot both, ports collide, things break.

Lane fixes the **port assignment** part of that problem. It does not try
to be a process manager, a service mesh, or an orchestrator. It does not
isolate filesystems or contexts — your OS and your editor already do
that. It just makes sure that when you say "I am working on project A",
your shell exports the ports project A is allowed to use, and nothing
else.

---

## How it works

Lane stores one TOML file (`~/.config/lane/projects.toml` by default,
honoring `$XDG_CONFIG_HOME`) describing each registered project, the
tech stack it uses, and the ports it has reserved.

When you `lane use <project>`, Lane prints a sequence of `export`
statements to stdout. You feed those into your shell with `eval`:

```bash
eval "$(lane use my-project)"
```

Your shell now has `APP_PORT`, `VITE_PORT`, `FORWARD_DB_PORT`, etc.
exported. Any `docker compose up`, `php artisan serve`, or `npm run
dev` you run after that picks them up automatically — as long as your
project's config follows the standard Docker/Sail convention of
`${APP_PORT:-80}:80`.

To clear them when you switch projects:

```bash
eval "$(lane unuse)"
```

That is the whole model.

---

## Installation

### From source

Requires Go 1.25 or newer.

```bash
git clone git@github.com:tincke10/Lane.git
cd Lane
make install
```

That installs `lane` into `$(go env GOPATH)/bin`. Make sure that
directory is on your `PATH`.

Or build the binary locally:

```bash
make build
./bin/lane help
```

---

## Quick start

```bash
# 1. Register a project. Lane detects the stack and allocates free ports.
$ cd ~/code/my-laravel-app
$ lane init
registered "my-laravel-app" at /Users/me/code/my-laravel-app
  stack: [docker laravel mysql node php redis vite]
  APP_PORT=8080
  FORWARD_DB_PORT=33060
  FORWARD_REDIS_PORT=63790
  VITE_PORT=5173

# 2. Switch to it in any shell.
$ eval "$(lane use my-laravel-app)"
$ echo $APP_PORT
8080

# 3. List everything Lane knows about.
$ lane list
my-laravel-app
  path:  /Users/me/code/my-laravel-app
  stack: docker, laravel, mysql, node, php, redis, vite
  APP_PORT=8080
  FORWARD_DB_PORT=33060
  FORWARD_REDIS_PORT=63790
  VITE_PORT=5173

# 4. Clean up when you are done.
$ eval "$(lane unuse)"
```

---

## Commands

| Command | What it does |
|---|---|
| `lane init [--name N] [--path P]` | Register the project in `P` (defaults to cwd). Detects the stack, allocates free ports, persists to the registry. |
| `lane use <name>` | Print `export` statements for the project. Aborts (with empty stdout and a stderr message) if any reserved port is currently in use. |
| `lane unuse` | Print `unset` statements for the project named in `LANE_ACTIVE_PROJECT`. Safe to run when nothing is active. |
| `lane list` (alias `ls`) | List all registered projects with their paths, stacks, and ports. |
| `lane rm <name>` (alias `remove`) | Remove a project from the registry. |
| `lane doctor` | Diagnose registry health: missing paths, cross-project port collisions, currently bound ports, stack drift, orphaned active project. Exits non-zero only on errors; warnings are informational. |
| `lane hook <bash\|zsh>` | Print the shell hook code for auto-activation on `cd`. Run once during shell setup. |
| `lane export` | Hook-driven activation diff. Called by the installed hook on every prompt; not typically invoked directly. |
| `lane help` | Show usage. |
| `lane version` | Print the version. |

### Flags

- `lane init --name <name>` — Override the project name. Defaults to the
  basename of `--path`.
- `lane init --path <path>` — Project path. Defaults to the current
  working directory. The path is stored absolute.

---

## Stack detection

Lane inspects the **top level** of the project directory (no recursion)
and looks for known marker files. Detection is best-effort and
intentionally conservative.

| Marker file(s) | Base tag | Deeper tag |
|---|---|---|
| `composer.json` | `php` | `laravel` (if `require."laravel/framework"` is present) |
| `package.json` | `node` | `vite` (if `vite` is in `dependencies` or `devDependencies`) |
| `pyproject.toml` or `requirements.txt` | `python` | — |
| `docker-compose.yml` / `docker-compose.yaml` / `compose.yml` / `compose.yaml` | `docker` | `mysql`, `postgres`, `redis` (substring match) |

A malformed `composer.json` or `package.json` still yields the base tag
(`php`, `node`) and silently skips the deeper detection — Lane will not
fail because of a transient broken JSON file.

---

## Port allocation policy

The first time you `lane init`, Lane allocates the lowest free port at
or above the base for each env var your stack needs. Reserved ports
from other registered projects are skipped, so two projects never get
the same port.

| Stack marker | Env var | Base port |
|---|---|---|
| `laravel` | `APP_PORT` | `8080` |
| `vite` | `VITE_PORT` | `5173` |
| `mysql` | `FORWARD_DB_PORT` | `33060` |
| `postgres` | `FORWARD_DB_PORT` | `54320` |
| `redis` | `FORWARD_REDIS_PORT` | `63790` |

Lane scans up to 1000 consecutive ports above the base before giving
up. The check uses a TCP listen on `127.0.0.1`, which is inherently
racy — another process can grab the port between the check and your
actual use. `lane use` re-checks at activation time and refuses to
export if any port is busy.

---

## Activation pattern

Lane follows the direnv / asdf playbook: it prints, your shell evals.
Nothing magic, no daemon, no hooks installed unless you add them
yourself.

### One-shot per shell

```bash
eval "$(lane use my-project)"
# ... work ...
eval "$(lane unuse)"
```

### Persistent shell aliases (bash / zsh)

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
lane-use()   { eval "$(command lane use   "$@")"; }
lane-unuse() { eval "$(command lane unuse "$@")"; }
```

Then:

```bash
lane-use my-project
lane-unuse
```

### Auto-activation on `cd` (recommended)

Install the hook once and Lane keeps your env in sync automatically as
you `cd` around. Each prompt, Lane walks up from the current directory,
finds the nearest registered project, and emits the diff against what
your shell currently has activated.

Add to `~/.zshrc`:

```bash
eval "$(lane hook zsh)"
```

Or to `~/.bashrc`:

```bash
eval "$(lane hook bash)"
```

What you get:

- `cd ~/code/my-laravel-app` → `APP_PORT`, `VITE_PORT`, etc. exported.
- `cd ~/code/other-project` → previous project's vars unset, new project's exported.
- `cd ~` (no project) → everything unset, shell back to clean state.
- `cd deep/inside/my-laravel-app/src/components` → still activates
  `my-laravel-app`. Lane walks up the tree until it finds a match.

The hook is **silent and fast**: it does not check port availability on
every prompt (`lane doctor` is the place for that), it does not call out
to the network, and it stays quiet on transient errors so a broken
registry never breaks your prompt.

### tmux / zellij / terminal tabs

Because activation lives in environment variables, each tab/pane/window
keeps its own active project. The auto-activation hook works
independently in each one — open a new tab, `cd` into a different
project, and that tab activates that project without touching the
others.

### Collision safety

If a reserved port is already bound by some other process when you run
`lane use`, Lane prints nothing to stdout and writes the conflict to
stderr with a non-zero exit code. That way `eval "$(lane use ...)"`
never silently activates a broken environment.

```bash
$ eval "$(lane use my-project)"
lane: port collision detected for "my-project" — not activating:
  APP_PORT=8080 is in use
$ echo $?
1
```

---

## Configuration

Lane stores its registry as TOML at:

```
$XDG_CONFIG_HOME/lane/projects.toml      # if $XDG_CONFIG_HOME is set
$HOME/.config/lane/projects.toml         # fallback
```

The file is written atomically (`*.tmp` + rename) so an interrupted
write cannot corrupt your registry. It is plain text — feel free to
edit it by hand if you need to bulk-rename or relocate things.

### Environment variables Lane reads

- `XDG_CONFIG_HOME` — overrides the default registry location.
- `LANE_ACTIVE_PROJECT` — set by `lane use`, read by `lane unuse` to
  know what to unset.

### Environment variables Lane sets (via `lane use`)

- `LANE_ACTIVE_PROJECT` — name of the currently active project.
- One `export` per reserved port (e.g., `APP_PORT`, `VITE_PORT`,
  `FORWARD_DB_PORT`, `FORWARD_REDIS_PORT`).

---

## Development

```bash
make build         # compile to ./bin/lane
make install       # install to GOPATH/bin
make test          # run tests with race detector
make test-cover    # run tests + open HTML coverage report
make lint          # golangci-lint
make fmt           # gofmt + go vet
make tidy          # go mod tidy
make clean         # remove build artifacts
```

### Project layout

```
cmd/lane/             Entry point: parses --version, delegates to internal/cmd.
internal/registry/    TOML-backed registry of projects and reserved ports.
internal/ports/       Free-port detection and collision-aware allocator.
internal/stack/       Filesystem-based stack detection (composer/package.json/etc).
internal/activator/   Generates POSIX `export` / `unset` statements.
internal/doctor/      Diagnostic checks for registry health (used by `lane doctor`).
internal/cmd/         CLI dispatcher and subcommand handlers.
```

Each internal package is independently testable. `ports` and `stack`
have no dependencies on the registry; `cmd` is the only layer that
wires them together.

---

## Scope and non-goals

**Lane does:**

- Track which projects have which ports reserved.
- Detect a project's stack from filesystem markers.
- Export shell variables that downstream tools (Docker, Vite, Laravel,
  etc.) read by convention.
- Refuse to activate when a reserved port is already in use.

**Lane does not:**

- Start, stop, or restart your services.
- Edit `docker-compose.yml`, `.env`, or any other file in your project.
- Proxy or rewrite network traffic.
- Manage Docker volumes, images, or networks.

If you need any of those, Lane will compose cleanly with the right
dedicated tool — it stays out of the way on purpose.

---

## Why "Lane"?

Each project gets its own lane on the road. They run in parallel, they
do not cross each other, and you can switch lanes whenever you want.
The name avoids the saturated nautical metaphors (`harbor`, `berth`,
`pier`, `regatta`) that already exist in this space.

---

## License

MIT. See [LICENSE](LICENSE).
