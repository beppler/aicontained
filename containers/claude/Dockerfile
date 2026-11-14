FROM docker.io/library/debian:trixie-slim

RUN apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends ca-certificates curl git tree \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise MISE_INSTALL_HELP=0 sh

# `mise use -g` both installs the tools and registers them in the global
# config file, so they are active everywhere without `mise activate`.
ENV MISE_DATA_DIR=/usr/local/share/mise \
    MISE_GLOBAL_CONFIG_FILE=/usr/local/share/mise/config.toml \
    MISE_TRUSTED_CONFIG_PATHS=/workspace \
    PATH="/usr/local/share/mise/shims:$PATH"

RUN mise use -g python@latest uv@latest gh@latest jq@latest ripgrep@latest

# Install with a temporary HOME so nothing lands in /root (the launcher
# bind-mounts a host dir over it), then copy the self-contained binary
# to /usr/local/bin.
RUN export HOME=/tmp/claude-home \
    && export PATH="$HOME/.local/bin:$PATH" \
    && mkdir -p "$HOME" \
    && curl -fsSL https://claude.ai/install.sh | bash \
    && install -m 0755 "$(readlink -f "$HOME/.local/bin/claude")" /usr/local/bin/claude \
    && rm -rf "$HOME"

ENV DISABLE_AUTOUPDATER=1
ENV IS_SANDBOX=1

ENTRYPOINT ["claude"]
