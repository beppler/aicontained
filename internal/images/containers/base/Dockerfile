FROM docker.io/library/debian:trixie-slim

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends bubblewrap ca-certificates curl procps tree libatomic1 openssh-client vim \
    && curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise MISE_INSTALL_HELP=0 sh \
    && rm -rf /var/lib/apt/lists/*

# Install the latest versions of some common tools system-wide, in mise's
# system dirs (config /etc/mise, installs /usr/local/share/mise), so users
# can override them with ~/.config/mise or a project mise.toml. The env is
# set for this step only, so at runtime mise uses its default layout. The
# system shims dir is on PATH so tools resolve without shell activation.
# Activate mise in interactive bash only.
RUN mkdir -p /etc/mise \
    && export MISE_DATA_DIR=/usr/local/share/mise MISE_GLOBAL_CONFIG_FILE=/etc/mise/config.toml \
    && mise use -g conda:git@latest conda:git-lfs@latest gh@latest go@latest jq@latest node@lts python@latest ripgrep@latest uv@latest \
    && mise cache clear \
    && chmod -R a+rX /usr/local/share/mise /etc/mise \
    && echo 'eval "$(mise activate bash)"' >> /etc/bash.bashrc

# Set up the default PATH and trusted config paths for all users
ENV PATH="/usr/local/share/mise/shims:$PATH" \
    MISE_TRUSTED_CONFIG_PATHS=/workspace
