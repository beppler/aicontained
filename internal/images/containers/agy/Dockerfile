FROM cli-tools-base

RUN curl -fsSL https://antigravity.google/cli/install.sh | bash -s -- --dir /usr/local/bin

ENV AGY_CLI_DISABLE_AUTO_UPDATE=true

ENTRYPOINT ["agy"]
