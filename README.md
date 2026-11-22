# Containerized AI coding harnesses

Run **Claude Code**, **Codex**, **GitHub Copilot**, **Antigravity**, **OpenCode**, and **Pi** in isolated containers — same commands, every OS, no Node.js or Python sprawl on your host. Each tool ships as its own image with its own runtime and its own persistent home directory, and it only ever sees the project directory you launch it from.

| Tool | Command | Entrypoint notes |
| ---- | ------- | --------------- |
| Claude Code | `claude` | `IS_SANDBOX=1` so `--dangerously-skip-permissions` works as root |
| Codex | `codex` | full-access sandbox mode inside the container |
| GitHub Copilot | `copilot` | |
| Antigravity | `agy` | |
| OpenCode | `opencode` | provider-agnostic: ANTHROPIC / OPENAI / GEMINI keys all supported |
| Pi | `pi` | provider-agnostic like OpenCode |

## Container engines

Auto-detected per OS, no configuration needed:

| OS | Engine |
| -- | ------ |
| macOS | Apple `container` (Apple Silicon only, macOS 15+) |
| Linux | Podman (rootless) |
| Windows | WSL Containers (`wslc.exe`) — native, ships with WSL |

Images are stored **per engine and per machine** — you must build on each machine you use; there's no shared cache between `container`, Podman, and `wslc`.

## Setup

### 1. Get the binary

Single Go binary; all the tool commands (`claude`, `codex`, ...) are links to it:

```bash
go run github.com/beppler/aicontained/cmd/aicontained@latest install
```

Or from a checkout of this repo: `go run ./cmd/aicontained install`.

No Go on the target machine? Run `make` from a checkout anywhere (GNU make; on Windows install it with e.g. `winget install GnuWin32.Make` — writes one binary per platform to `dist/`), copy the matching `dist/aicontained-<os>-<arch>` over, and use it as `aicontained`.

### 2. Create the tool links and build the images

Same on every OS:

```bash
aicontained install           # copy binary + create all six tool links
aicontained install claude agy  # only some tools
aicontained build             # build all images (no install)
aicontained build claude agy  # only some images
aicontained build --no-cache  # rebuild from scratch: picks up new tools + OS security patches the build cache can't see
aicontained build --prune-images  # also prune dangling build images
```

On a fresh machine run `install` once, then `build`. `install` puts the binary in `~/.local/bin/` (Windows: `%LOCALAPPDATA%\Programs\bin`) and makes one link per tool — add that directory to your `PATH`.

### Platform notes

**Windows (WSL Containers preview).** One-time setup in an elevated PowerShell:

```powershell
wsl --update --pre-release
wsl --shutdown
```

Reopen the terminal, then confirm `wslc --version` prints 2.9.3+. After `aicontained install`, `claude`, `codex`, `copilot`, `agy`, `opencode`, `pi` work from PowerShell or cmd.exe. Symlink creation there needs Developer Mode (or elevation); without it, `install` fails with a message saying exactly that. `wslc.exe` ships as part of WSL itself and runs each container in its own lightweight Hyper-V VM.

**macOS.** Apple `container` requires Apple Silicon and macOS 15+ (full support on macOS 26). `aicontained build` starts the daemon for you; on some setups you may need `container system start` again after a reboot.

**WSL (Podman networking).** Podman networking needs `slirp4netns`:

```bash
sudo apt update && sudo apt install -y slirp4netns
```

And in `/etc/containers/containers.conf`:

```toml
[network]
default_rootless_network_cmd = "slirp4netns"
```

## Usage

Same on every OS — from any project directory:

```bash
cd ~/my-project
claude    # or codex / copilot / agy / opencode / pi
```

**How mounting works:** the current directory is bind-mounted at `/workspace/<hash-of-host-path>/<folder-name>`. The hash is deliberate — these tools key per-project state (sessions, memory, folder trust) by path, so every project must map to a unique in-container path. Side effect: renaming or moving a project on the host gives it a fresh hash and a fresh session history.

**Auth persistence:** each tool keeps its credentials in a dedicated host folder per tool under `~/.aicontained/` (macOS/Linux) or `%USERPROFILE%\.aicontained\` (Windows), bind-mounted as `/root` in the container — log in once, stay logged in.

## Logging in

| Tool | Method | Notes |
| ---- | ------ | ----- |
| Claude, Antigravity | Browser OAuth | prints a URL/code to paste into your host browser — like SSH login |
| Codex, Copilot | Device code | browser OAuth can't work: the callback server binds `127.0.0.1` inside the container, reachable from nowhere — no port-forwarding fix is possible |
| Antigravity (personal accounts) | API key | OAuth unavailable; set `GEMINI_API_KEY` in a `.env` in `antigravity-home` — get one at [Google AI Studio](https://aistudio.google.com/app/apikey) |
| OpenCode | Any | reads whichever of `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) is set, or `opencode auth login` in-container |
| Pi | Any | reads whichever of `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` is set, or `/login` in-container |

API keys from your shell are passed through automatically (the binary is a native Windows process, so PowerShell env vars work directly — no WSL-side configuration).

OpenAI device-code login is **disabled by default** and must be enabled first:

- Personal account: ChatGPT → Settings → Security → enable device code login
- Team/Enterprise/Edu: a workspace admin must allow it under Workspace Settings → Permissions → "Allow device code login" — it's greyed out for regular members otherwise

Or skip login entirely:

```bash
export ANTHROPIC_API_KEY=...   # or OPENAI_API_KEY / GEMINI_API_KEY
# PowerShell: $env:ANTHROPIC_API_KEY = "..."
```

## Sandboxing

Tools run as root **inside** their containers — that root is not your host root:

- rootless Podman maps container root to your own user
- Apple `container` and `wslc` run each container in a lightweight per-container VM

Verify Podman is running rootless:

```bash
podman info --format '{{.Host.Security.Rootless}}'    # expect true
```

**Linux hardening** (applied automatically by the launcher on Podman):

```text
--security-opt no-new-privileges
--cap-drop=ALL --cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID
```

The retained capabilities are what file handling and `apt-get install` inside the container need; everything else is dropped (including network admin, raw sockets, kill, chroot, and capability editing), so no privilege escalation via setuid. Podman must run rootless — with `sudo podman`, container root is host root.

**Claude skip-permissions:** the image sets `IS_SANDBOX=1`, which lets `claude --dangerously-skip-permissions` run as root (Claude Code otherwise refuses it when running as root). It's still opt-in — plain `claude` asks for permissions as usual.

## Troubleshooting: leftover opencode state folders

opencode keeps a service registration in its state directory, which can't be shared between running instances. Each `opencode` run therefore mounts its own temporary state folder over `/root/.local/state/opencode`, named after the launcher's PID and removed when the run ends:

```text
~/.aicontained/opencode-home/.local/state/opencode-<pid>
```

A crash or `kill -9` can leave one behind. To list orphans — folders whose PID has no running process (macOS/Linux):

```bash
for d in ~/.aicontained/opencode-home/.local/state/opencode-*; do
  [ -d "$d" ] || continue
  kill -0 "${d##*-}" 2>/dev/null || echo "orphan: $d"
done
```

On Windows (PowerShell):

```powershell
Get-ChildItem "$env:USERPROFILE\.aicontained\opencode-home\.local\state" -Directory -Filter 'opencode-*' |
  Where-Object { -not (Get-Process -Id ($_.Name -replace '^opencode-','') -ErrorAction SilentlyContinue) } |
  ForEach-Object { "orphan: $($_.FullName)" }
```

Orphans are safe to delete. A reused PID can make an orphan look alive, so a folder that is never listed isn't proof that it's in use.

**What persists between runs:** model selection (`model.json`), prompt history (`prompt-history.jsonl`), and the release-highlights "seen" marker (`latest/tui/highlights.json` — without it the "What's new" dialog would open on every run) are seeded from, and saved back to, the shared state folder (`~/.aicontained/opencode-home/.local/state/opencode`) on every run. Only `service.json` (the per-run server registration) and the rest of `latest/` and `locks/` stay ephemeral. With concurrent instances, the last to exit wins for `model.json` and the marker; prompt history merges instead, because each run appends only entries not already in the shared JSONL file. If a run crashes before the save-back, the previous saved copy is used.

## Troubleshooting: Windows line endings (CRLF)

The launcher sets `core.autocrlf=true` and `core.filemode=false` for Git inside the container on Windows runs — via `GIT_CONFIG_*` env vars, so only Windows runs are affected and the images stay portable. That keeps repos cloned on Windows from showing every file as modified.

Two caveats:

- A committed `.gitattributes` still takes precedence.
- These settings override the repo's `.git/config` — a deliberately LF-only repo (`core.autocrlf=false`) can get files rewritten to CRLF by a `git checkout` or `git stash pop` inside the container.

**Go tools (`gofmt -l` lists every file):** `gofmt` treats CRLF as unformatted, and `core.autocrlf` can't help because the files really are CRLF on disk. Commit a `.gitattributes` with `*.go text eol=lf` (this repo does), then refresh the working tree: `git rm --cached -r -q . && git reset --hard` (commit or stash changes first).

Still seeing phantom diffs? Check with `git diff --ignore-cr-at-eol --stat`; if needed, set the same two values in the repo's `.git/config` and add to the project's `CLAUDE.md`/`AGENTS.md`: "This working tree uses Windows line endings. Never convert or 'fix' line endings." Don't run `dos2unix` — it only moves the phantom changes to the Windows side and breaks `.bat`/`.cmd` files.

## Upgrading from cli-tools

The project was renamed to `aicontained`. Do these steps **before** running any tool, so your logins carry over:

```bash
mv ~/.cli-tools ~/.aicontained     # Windows: rename %USERPROFILE%\.cli-tools to .aicontained
aicontained install                # re-point the tool links at the new binary
aicontained build                  # images are renamed too, so rebuild them (required)
rm ~/.local/bin/cli-tools          # Windows: delete cli-tools.exe from %LOCALAPPDATA%\Programs\bin
podman images --format '{{.Repository}}' | grep '^cli-tools-' | xargs -r podman rmi   # old images
```

For `container` or `wslc`, remove the old `cli-tools-*` images with that engine's own `rmi`/`image delete`.
If you already ran a tool after upgrading, `~/.aicontained` exists and `mv` would nest the old folder inside it; merge the folders by hand instead.

## License

MIT — see [LICENSE](LICENSE).
