FROM docker.io/library/debian:trixie-slim

RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ca-certificates curl git tree \
    && rm -rf /var/lib/apt/lists/*

# Install mise with the official installer outside /root (the launcher
# bind-mounts a host dir over it). Keep tool installs in mise's system data
# directory too, so they don't land in that per-harness mount, and put the
# system shim directory on PATH so tools resolve without shell activation.
# Trust the mounted project tree so project mise.toml files load
# without an interactive trust prompt.
RUN curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise MISE_INSTALL_HELP=0 sh

# `mise use -g` both installs the tools and registers them in the global
# config file, so they are active everywhere without `mise activate`.
ENV MISE_DATA_DIR=/usr/local/share/mise \
    MISE_GLOBAL_CONFIG_FILE=/usr/local/share/mise/config.toml \
    MISE_TRUSTED_CONFIG_PATHS=/workspace \
    PATH="/usr/local/share/mise/shims:$PATH"

RUN mise use -g python@latest uv@latest gh@latest jq@latest ripgrep@latest

RUN curl -fsSL https://gh.io/copilot-install | bash

ENTRYPOINT ["copilot", "--no-auto-update"]
