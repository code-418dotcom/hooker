# hooker

Telegram bot to start, stop, and restart Docker containers individually, in bulk, or by group label.

## Features

- **Single admin auth** — Only one Telegram user can issue commands
- **Bulk operations** — Start/stop/restart all containers at once
- **Group operations** — Control containers by Docker label (`hooker.group=<tag>`)
- **Status listing** — See all containers and their state
- **Long polling** — No public HTTPS endpoint required
- **Graceful shutdown** — Clean exit on SIGTERM

## Getting Started

### Prerequisites

- Go 1.22+
- Docker (local or remote via `DOCKER_HOST`)
- Telegram bot token from [@BotFather](https://t.me/BotFather)
- Your Telegram user ID (get it from [@userinfobot](https://t.me/userinfobot))

### Setup

1. Clone the repo and copy the environment template:
```bash
git clone https://github.com/code-418dotcom/hooker.git
cd hooker
cp .env.example .env
```

2. Edit `.env` and add:
   - Your bot token (from BotFather)
   - Your Telegram user ID (from userinfobot)

3. Load environment and build:
```bash
source .env
make build
make run
```

### Supported Commands

Send these commands to your bot on Telegram:

| Command | Action |
|---------|--------|
| `/list` or `/status` | List all containers with state |
| `/start <name>` | Start a container |
| `/stop <name>` | Stop a container |
| `/restart <name>` | Restart a container |
| `/start all` | Start all stopped containers |
| `/stop all` | Stop all running containers |
| `/restart all` | Restart all containers |
| `/start group <tag>` | Start containers with label `hooker.group=<tag>` |
| `/stop group <tag>` | Stop containers with label `hooker.group=<tag>` |

### Tag containers for group operations

```bash
# Run a container with a group label
docker run -l hooker.group=media --name plex ...

# Or add label to existing container
docker update -l hooker.group=media <container>
```

## Development

### Run tests
```bash
make test
```

### Build binary
```bash
make build
```

### Build Docker image
```bash
make docker-build
```

### Deploy to Docker

```bash
docker run -d \
  --name hooker \
  --restart unless-stopped \
  -e TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN \
  -e TELEGRAM_ADMIN_ID=$TELEGRAM_ADMIN_ID \
  -v /var/run/docker.sock:/var/run/docker.sock \
  hooker:latest
```

## Architecture

- **`cmd/hooker/main.go`** — Entry point, wires dependencies
- **`internal/config/`** — Load environment variables
- **`internal/docker/`** — Docker SDK wrapper, container operations
- **`internal/telegram/`** — Telegram bot, command parsing, message handling

See `CLAUDE.md` for detailed architecture notes.
