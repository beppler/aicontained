FROM cli-tools-base

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
