# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**hooker** is an internal tool that provides a Telegram bot interface for managing Docker containers in bulk or individually. Users can send commands via Telegram to start, stop, or otherwise control containers running on a host.

**Type:** Internal tool  
**Status:** Early development (tech stack TBD)  
**Primary Use Case:** Docker container lifecycle management via Telegram

## Architecture (Vision)

The hooker project will consist of two main components:

1. **Telegram Bot Handler** — Receives messages from Telegram API, parses commands, validates permissions, handles user interaction
2. **Docker Manager** — Executes container operations (start, stop, etc.) via Docker daemon/API
3. **Configuration** — Maps Telegram user IDs to containers/permissions, stores webhook secrets

### Technology Stack (TBD)

The language and framework are not yet decided. Consider:
- **Go** — native Docker SDK, lightweight for running as a daemon, easy deployment
- **Python** — simpler bot library ecosystem, faster iteration
- **Node.js** — cross-platform, good TypeScript support

Recommendation: Go (small binary, native Docker API, ideal for infrastructure tools).

## Development Commands

Once the tech stack is chosen, these commands will be defined:

```bash
# Build the binary
make build

# Run tests
make test

# Run a single test
make test TEST=TestFunctionName

# Lint code
make lint

# Run locally with test Telegram token
make run

# Check Docker connection
make docker-check
```

Update this section once the stack is finalized.

## Code Structure (Planned)

```
hooker/
├── cmd/
│   └── hooker/
│       └── main.go          # Entry point
├── internal/
│   ├── telegram/            # Telegram bot logic
│   │   ├── handler.go       # Message handling
│   │   └── commands.go      # Command parsing
│   ├── docker/              # Docker integration
│   │   ├── client.go        # Docker client wrapper
│   │   └── operations.go    # Container operations
│   └── config/              # Configuration
│       └── config.go        # Load config from env/file
├── Dockerfile               # Container image
├── Makefile
├── go.mod / go.sum          # Dependency management
└── README.md
```

(Adjust structure based on language choice)

## Key Decisions & Constraints

- **Security:** Telegram user ID whitelist required; no container access without explicit permission
- **State:** Stateless bot design (queries Docker API in real-time, no local persistence)
- **Deployment:** Should run as a single process/container, accessible from Telegram API (webhook or polling)
- **Error Handling:** Commands that fail should provide clear error messages back to user

## Common Development Tasks

### Adding a new Docker command
1. Define the command in `telegram/commands.go`
2. Add handler logic in `docker/operations.go`
3. Map command to user permissions in config
4. Test with `make test` and manual testing in Telegram
5. Update README with command documentation

### Testing locally
1. Create a test Telegram bot with BotFather
2. Set environment variables: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_WHITELIST_IDS`
3. Run `make run` to start the bot
4. Send test commands to the bot in Telegram

### Deploying
Exact process TBD, but likely:
- Build with `make build`
- Push binary/container image to deployment target
- Set environment variables on target
- Run the binary

## Debugging

**Telegram webhooks not receiving messages?**
- Verify bot token is correct
- Check firewall allows inbound on webhook port
- Verify public URL is accessible and using HTTPS

**Docker daemon connection fails?**
- Ensure Docker socket/API is accessible from where hooker runs
- If containerized, mount the Docker socket: `-v /var/run/docker.sock:/var/run/docker.sock`

## References & Documentation

- Telegram Bot API: https://core.telegram.org/bots/api
- Docker API: https://docs.docker.com/engine/api/
- Go Docker SDK: https://github.com/moby/moby/tree/master/client (if Go is chosen)
