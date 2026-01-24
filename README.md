# WoL-NUT Dashboard

Lightweight Wake-on-LAN and NUT UPS monitoring dashboard for Raspberry Pi and Linux servers.

## Features

- **Wake-on-LAN** - Send magic packets to wake network devices
- **UPS Monitoring** - Real-time status from NUT (Network UPS Tools) servers
- **Battery Visualization** - Donut charts showing charge level, runtime, and load
- **Mobile Friendly** - Responsive design works on any device
- **Single Binary** - No dependencies, easy deployment
- **Auto-refresh** - UPS status updates every 30 seconds
- **Backup/Restore** - Export and import configuration

## Quick Install

### Bare Metal (Raspberry Pi / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/aloks98/wol-nut/main/scripts/install.sh | sudo bash
```

### Docker

```bash
docker run -d \
  --name wol-nut \
  --network host \
  -v wol-nut-data:/data \
  ghcr.io/aloks98/wol-nut:latest
```

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/aloks98/wol-nut/main/docker-compose.yml
docker compose up -d
```

## Configuration

Configuration via `/etc/wol-nut/config.yml` (bare metal) or environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `WOLNUT_SERVER_PORT` | 8080 | HTTP port |
| `WOLNUT_SERVER_HOST` | 0.0.0.0 | Bind address |
| `WOLNUT_DATA_PATH` | /var/lib/wol-nut | Data directory |
| `WOLNUT_LOG_LEVEL` | info | Log level (debug, info, warn, error) |

### Example config.yml

```yaml
server:
  port: 8080
  host: "0.0.0.0"

data:
  path: "/var/lib/wol-nut"

log:
  level: "info"
```

## Usage

### Adding Devices

1. Navigate to **Devices** page
2. Click **Add Device**
3. Enter name and MAC address (format: `AA:BB:CC:DD:EE:FF`)
4. Click **Add Device**

### Adding UPS Connections

1. Navigate to **UPS** page
2. Click **Add UPS**
3. Enter:
   - **Display Name**: Friendly name for the UPS
   - **NUT Server Host**: Address and port (e.g., `localhost:3493`)
   - **UPS Name**: UPS identifier as configured in NUT (usually `ups`)
4. Click **Add UPS**

### Waking Devices

From the Dashboard or Devices page, click the **Wake** button to send a magic packet.

## Management

```bash
# Service status
sudo systemctl status wol-nut

# View logs
sudo journalctl -u wol-nut -f

# Restart service
sudo systemctl restart wol-nut

# Update to latest version
curl -fsSL https://raw.githubusercontent.com/aloks98/wol-nut/main/scripts/update.sh | sudo bash

# Uninstall
curl -fsSL https://raw.githubusercontent.com/aloks98/wol-nut/main/scripts/uninstall.sh | sudo bash
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Dashboard |
| GET | `/devices` | Manage WoL devices |
| GET | `/ups` | Manage UPS connections |
| POST | `/wake/{id}` | Send WoL packet |
| GET | `/api/ups/status` | JSON UPS status |
| GET | `/config/export` | Download backup |
| POST | `/config/import` | Restore backup |
| GET | `/health` | Health check |

## Development

### Prerequisites

- Go 1.22+

### Build & Run

```bash
# Clone repository
git clone https://github.com/aloks98/wol-nut.git
cd wol-nut

# Run locally
go run .

# Build binary
go build -o wol-nut

# Run with config
./wol-nut --config config.example.yml
```

### Docker Development

```bash
docker compose -f docker-compose.dev.yml up --build
```

## Architecture

- **Backend**: Go with stdlib `net/http`
- **Templates**: `html/template` with embedded assets
- **Frontend**: HTMX + Alpine.js + Tailwind CSS + Chart.js
- **Data**: JSON file storage

## Requirements

- **Wake-on-LAN**: Requires network access to broadcast UDP packets
- **NUT Monitoring**: Requires TCP access to NUT server (port 3493)
- **Docker**: Use `network_mode: host` for full WoL functionality

## License

MIT
