# Quick Start Guide

## 1. Setup Your Environment

```bash
cp .env.example .env
```

Edit `.env` and add your credentials:
```
TELEGRAM_BOT_TOKEN=8035118924:AAF8htPim7MYIEpjvgw7zguwDYjAF_uSX6Q
TELEGRAM_ADMIN_ID=<your-user-id>
```

Find your user ID: send `/start` to [@userinfobot](https://t.me/userinfobot) and it will reply with your ID.

## 2. Run Unit Tests (no Docker needed)

```bash
go mod tidy
go test ./...
```

This verifies command parsing and bot logic without connecting to Telegram or Docker.

## 3. Test Locally with Your Bot

### Option A: Native binary

```bash
source .env
go run ./cmd/hooker
```

In Telegram, send your bot:
- `/list` — should show your Docker containers
- `/stop nginx` — stop a specific container (replace `nginx` with a real container name)
- `/start nginx` — start it again
- `/start group <tag>` — if you have labeled containers

Watch the terminal for any errors.

### Option B: Docker

```bash
source .env
make docker-build
docker run -it \
  -e TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN \
  -e TELEGRAM_ADMIN_ID=$TELEGRAM_ADMIN_ID \
  -v /var/run/docker.sock:/var/run/docker.sock \
  hooker:latest
```

Same Telegram test commands as above.

## 4. Test with a Non-Admin User

From a **different Telegram account** (different user ID), send any command to your bot. The bot should **not reply** — this confirms auth is working.

## 5. Test Graceful Shutdown

While the bot is running, press `Ctrl+C` in the terminal. It should exit cleanly with no errors.

## 6. Troubleshooting

### Bot doesn't receive messages
- Check that your bot token is correct
- Verify your TELEGRAM_ADMIN_ID matches your user ID
- Ensure your bot is active (send `/start` to BotFather to enable it)

### Docker operations fail
- Verify Docker socket is accessible: `ls -la /var/run/docker.sock`
- Test Docker directly: `docker ps`
- If running in a container, ensure socket is mounted with `-v /var/run/docker.sock:/var/run/docker.sock`

### Permission denied errors
- If running natively, ensure your user is in the `docker` group: `groups` and `id`
- Add yourself: `sudo usermod -aG docker $USER` (then log out and back in)

## 7. Production Deployment

Once tested locally, deploy as a container:

```bash
docker run -d \
  --name hooker \
  --restart unless-stopped \
  -e TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN \
  -e TELEGRAM_ADMIN_ID=$TELEGRAM_ADMIN_ID \
  -v /var/run/docker.sock:/var/run/docker.sock \
  hooker:latest

# Check logs
docker logs -f hooker

# Stop gracefully
docker stop hooker
```

## Security Notes

- **Keep your token secret** — anyone with it can control your bot
- Store `.env` safely, never commit it
- The bot only accepts commands from your Telegram user ID; all others are silently ignored
- The bot runs with access to Docker, so only let trusted users run containers

See `README.md` for more information.
