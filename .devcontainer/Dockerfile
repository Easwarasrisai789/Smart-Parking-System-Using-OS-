FROM mcr.microsoft.com/devcontainers/go:1-bookworm

# Install common tooling for Go development
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
        curl \
        git \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set up workspace defaults
WORKDIR /workspaces

# Keep the container lean and ready for Go projects
RUN go version
