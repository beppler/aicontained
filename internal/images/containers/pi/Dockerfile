FROM cli-tools-base

# Install with a temporary HOME so the npm cache doesn't land in /root (the
# launcher bind-mounts a host dir over it), then link the launcher script to
# /usr/local/bin. The npm bin is a stub that loads its bundle from beside
# itself inside the package, so it must stay linked, not copied out; npm's
# global install lives in mise's data dir, inside the image, and npm
# prefix -g is its bin root (not on PATH at build time).
RUN export HOME=/tmp/pi-home \
    && mkdir -p "$HOME" \
    && npm install -g --ignore-scripts @earendil-works/pi-coding-agent \
    && ln -s "$(readlink -f "$(npm prefix -g)/bin/pi")" /usr/local/bin/pi \
    && rm -rf "$HOME"

ENV PI_SKIP_VERSION_CHECK=1

ENTRYPOINT ["pi"]
