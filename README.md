# Containerized AI Coding CLIs

Run **Claude Code**, **Codex**, **GitHub Copilot**, **Antigravity**, and **OpenCode** in isolated containers — same commands, every OS, no Node.js or Python sprawl on your host. Each tool ships as its own image with its own runtime and its own persistent home directory, and it only ever sees the project directory you launch it from.

| Tool | Command | Entrypoint notes |
| ---- | ------- | --------------- |
| Claude Code | `claude` | `IS_SANDBOX=1` so `--dangerously-skip-permissions` works as root |
| Codex | `codex` | full-access sandbox mode inside the container |
| GitHub Copilot | `copilot` | |
| Antigravity | `agy` | |
| OpenCode | `opencode` | provider-agnostic: ANTHROPIC / OPENAI / GEMINI keys all supported |

## Container engines

Auto-detected per OS, no configuration needed:

| OS | Engine |
| -- | ------ |
| macOS | Apple `container` (Apple Silicon only, macOS 15+) |
| Linux | Podman (rootless) |
| Windows | WSL Containers (`wslc.exe`) — native, ships with WSL |

Images are stored **per engine and per machine** — you must build on each machine you use; there's no shared cache between `container`, Podman, and `wslc`.

## Setup

### Install the binary

`cli-tools` is a single Go binary; all the per-tool commands (`claude`, `codex`, ...) are links to it:

```bash
go install github.com/beppler/cli-tools/cmd/cli-tools@latest
```

Or, from a checkout of this repo:

```bash
go build -o "$HOME/.local/bin/cli-tools" ./cmd/cli-tools
```

To set up a machine without Go, cross-compile everywhere first with `scripts/cross-build.sh` (output lands in `dist/`, one binary per platform), copy the matching `dist/cli-tools-<os>-<arch>` to the machine, and continue below.

### macOS / Linux

```bash
cli-tools install           # install the binary and create all five tool links
cli-tools install claude agy # only some
cli-tools build             # build all images (no install)
cli-tools build claude agy  # only some
cli-tools build --prune-images    # also prune dangling build images
cli-tools build --no-cache        # skip the build cache and refetch everything
```

Build and install are separate: `cli-tools build` only builds the container images; `cli-tools install` copies the binary to `~/.local/bin/` and creates one link per tool there — make sure `~/.local/bin` is on your `PATH`. On a fresh machine run `install` once, then `build` (in either order).

`cli-tools build` uses the engine's build cache — layers only rebuild when their inputs change. Tool installs and base-image OS updates (`apt-get upgrade`) fetch from the network, so the cache won't pick up tool or security-patch updates on its own; use `--no-cache` to rebuild from scratch and get the latest tools, or run it periodically to pick up OS security patches.

`cli-tools install` copies the binary to `~/.local/bin/` and creates one link per tool there — make sure `~/.local/bin` is on your `PATH`.

On macOS, `container` requires Apple Silicon and macOS 15+ (full support on macOS 26). If the daemon isn't running yet, `cli-tools build` starts it for you; on some setups you may need to run `container system start` again after a reboot.

### Windows (WSL Containers preview)

One-time setup in an elevated PowerShell:

```powershell
wsl --update --pre-release
wsl --shutdown
```

Reopen your terminal, then confirm it's there:

```powershell
wslc --version   # expect 2.9.3+
```

Then the same commands as on macOS/Linux (`go install ...` first):

```powershell
cli-tools install           # all tools
cli-tools install claude agy # only some
cli-tools build
cli-tools build claude agy
cli-tools build --prune-images
```

`cli-tools install` installs to `%LOCALAPPDATA%\Programs\bin`. Add that to your PATH, then `claude`, `codex`, `copilot`, `agy`, `opencode` work from PowerShell or cmd.exe.

`wslc.exe` ships as part of WSL itself and runs each container in its own lightweight Hyper-V VM. Symlink creation on Windows needs Developer Mode (or elevation); without it, `cli-tools install` makes hardlink copies instead and warns.

### WSL (Podman networking)

To make Podman network work on WSL, install the `slirp4netns` package:

```bash
sudo apt update && sudo apt install -y slirp4netns
```

And add the following lines to `/etc/containers/containers.conf`:

```toml
[network]
default_rootless_network_cmd = "slirp4netns"
```

## Usage

Same on every OS — from any project directory:

```bash
cd ~/my-project
claude    # or codex / copilot / agy / opencode
```

**How mounting works:** the current directory is bind-mounted at `/workspace/<hash-of-host-path>/<folder-name>`. The folder keeps its own name inside the container. The hash is deliberate — these tools keep per-project state (sessions, memory, folder trust) keyed by path, so every project must map to a unique in-container path. Side effect: renaming or moving a project on the host gives it a fresh hash and a fresh session history.

**Auth persistence:** each tool keeps its credentials in a dedicated host directory, bind-mounted as `/root` in the container — log in once, stay logged in:

```text
~/.cli-tools/claude-home          (macOS/Linux)
~/.cli-tools/codex-home
~/.cli-tools/copilot-home
~/.cli-tools/antigravity-home
~/.cli-tools/opencode-home
```

On Windows the same layout lives under `%USERPROFILE%\.cli-tools\`.

## Logging in

| Tool | Method | Notes |
| ---- | ------ | ----- |
| Claude, Antigravity | Browser OAuth | prints a URL/code to paste into your host browser — like SSH login |
| Codex, Copilot | Device code | browser OAuth can't work: the callback server binds `127.0.0.1` inside the container, reachable from nowhere — no port-forwarding fix is possible |
| Antigravity (personal accounts) | API key | OAuth unavailable; set `GEMINI_API_KEY` in a `.env` in `antigravity-home` — get one at [Google AI Studio](https://aistudio.google.com/app/apikey) |
| OpenCode | Any | reads whichever of `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GEMINI_API_KEY` (or `GOOGLE_API_KEY`) is set, or `opencode auth login` in-container |

API keys from your shell are passed through automatically (including PowerShell on Windows — the binary is a native Windows process, so it sees your environment variables directly; no WSL-side configuration needed).

Note that OpenAI device-code login is **disabled by default** and must be turned on before it will work:

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

No privilege escalation via setuid; the dropped capabilities include network admin, raw sockets, kill, chroot, and capability editing. The retained ones are what file handling and `apt-get install` inside the container need. Podman must run rootless — with `sudo podman`, container root is host root.

**Claude skip-permissions:** the image sets `IS_SANDBOX=1`, which lets `claude --dangerously-skip-permissions` run as root (Claude Code otherwise refuses it when running as root). It's still opt-in — plain `claude` asks for permissions as usual.

## Troubleshooting: leftover opencode state folders

opencode keeps a service registration in its state directory, which can't be shared between running instances. Each `opencode` run therefore mounts its own temporary state folder over `/root/.local/state/opencode`, named after the launcher's PID and removed when the run ends:

```text
~/.cli-tools/opencode-home/.local/state/opencode-<pid>
```

A crash or `kill -9` can leave one behind. A folder is an orphan when no process with that PID is running. To list them (macOS/Linux):

```bash
for d in ~/.cli-tools/opencode-home/.local/state/opencode-*; do
  [ -d "$d" ] || continue
  kill -0 "${d##*-}" 2>/dev/null || echo "orphan: $d"
done
```

On Windows (PowerShell):

```powershell
Get-ChildItem "$env:USERPROFILE\.cli-tools\opencode-home\.local\state" -Directory -Filter 'opencode-*' |
  Where-Object { -not (Get-Process -Id ($_.Name -replace '^opencode-','') -ErrorAction SilentlyContinue) } |
  ForEach-Object { "orphan: $($_.FullName)" }
```

Orphans are safe to delete. A reused PID can make an orphan look alive, so a folder that is never listed isn't proof that it's in use.

**What persists between runs:** model selection (`model.json`), prompt history (`prompt-history.jsonl`), and the release-highlights "seen" marker (`latest/tui/highlights.json` — without it the "What's new" dialog would open on every run) are seeded from, and saved back to, the shared state folder on every run:

```text
~/.cli-tools/opencode-home/.local/state/opencode
```

Only `service.json` (the per-run server registration) and the remaining `latest/` and `locks/` content stay ephemeral — they can't be shared between running instances. When two instances run at once, the one that exits last wins for `model.json` and the highlights marker. Prompt history is the exception: `prompt-history.jsonl` is a JSONL log, so each run appends only the entries that are not already in the shared file, and the histories of concurrent instances merge. If a run crashes before the save-back, the previous saved copy is used.

## Troubleshooting: Windows line endings (CRLF)

A repo cloned on Windows has CRLF files on disk and LF in the index, which the host's global `core.autocrlf=true` hides. The container doesn't see your Windows gitconfig, so without help `git status` would show every file as modified. The launcher therefore sets these settings for Git inside the container on Windows runs:

```text
core.autocrlf=true
core.filemode=false
```

They are set through `GIT_CONFIG_*` environment variables, so only Windows runs are affected and the images stay portable.

Two things to know:

- A committed `.gitattributes` still takes precedence over these settings.
- The environment settings override the repo's own `.git/config`. If you deliberately keep a repo LF-only on Windows (`core.autocrlf=false`), a `git checkout` or `git stash pop` inside the container can rewrite files to CRLF.

To check what Git sees inside the container:

```bash
git config --show-origin --get-all core.autocrlf   # which setting applies, and from where
git ls-files --eol                                 # index (i/) vs disk (w/) line endings
git diff --ignore-cr-at-eol --stat                 # is anything really changed?
```

If you still see phantom diffs, set the same values in the repo itself (stored in `.git/config`, so nothing is committed and collaborators are unaffected):

```bash
git config core.autocrlf true
git config core.filemode false
```

It also helps to add this to the project's `CLAUDE.md` / `AGENTS.md`: "This working tree uses Windows line endings. Never convert or 'fix' line endings. Verify real changes with `git diff --ignore-cr-at-eol`." Don't run `dos2unix` in the container: it only moves the phantom changes to the Windows side and breaks `.bat`/`.cmd` files.
