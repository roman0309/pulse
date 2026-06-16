# Convenience targets for building and publishing the Pulse host agent.

# Override to publish under your own registry/owner, e.g.:
#   make agent-publish AGENT_IMAGE=ghcr.io/myorg/pulse-agent:v0.1.0
AGENT_IMAGE ?= pulse-agent:local

.PHONY: agent-image agent-publish agent-binary up down

## Build the agent image locally (usable immediately on this host)
agent-image:
	docker build -t $(AGENT_IMAGE) -f backend/Dockerfile.agent backend

## Build + push the agent image to a registry
agent-publish:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(AGENT_IMAGE) -f backend/Dockerfile.agent backend --push

## Build a standalone agent binary for the current OS/arch -> ./pulse-agent
agent-binary:
	cd backend && CGO_ENABLED=0 go build -trimpath -o ../pulse-agent ./cmd/agent

## Bring the whole stack up / down
up:
	docker compose up -d --build

down:
	docker compose down
