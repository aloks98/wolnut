# WoL-NUT Dashboard

Lightweight Wake-on-LAN and NUT UPS monitoring dashboard for Raspberry Pi and Linux servers.

## Features

- **Wake-on-LAN** - Send magic packets to wake network devices
- **UPS Monitoring** - Real-time status from NUT (Network UPS Tools) servers
- **Battery Visualization** - Donut charts showing charge level and load percentage
- **Detailed UPS Stats** - View power draw, voltage, runtime, temperature, and more
- **Mobile Friendly** - Responsive design works on any device
- **Single Binary** - No dependencies, embedded frontend
- **Auto-refresh** - UPS status updates every 15 seconds
- **Backup/Restore** - Export and import configuration

## Screenshots

The dashboard displays:
- Headline UPS status ("On grid power" / "On battery" / "Battery low") with at-a-glance color and icon
- Battery charge, load, and runtime
- Power draw (current and nominal wattage)
- Input/output voltage and frequency
- Raw NUT status codes

## Quick Install

### Bare Metal (Raspberry Pi / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/install.sh | sudo bash
```

### Docker

```bash
docker run -d \
  --name wol-nut \
  --network host \
  --cap-add NET_RAW \
  -v wol-nut-data:/data \
  ghcr.io/aloks98/wolnut:latest
```

> `--cap-add NET_RAW` lets the device online-status check use ICMP ping. It's
> optional — without it the check falls back to probing common TCP ports, so a
> host that's up but exposes none of them would show as offline.

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/aloks98/wolnut/master/docker-compose.yml
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

### Finding Your UPS Name

If you're unsure of your UPS name in NUT:

```bash
# List all UPS devices on the server
upsc -l localhost
```

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
curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/update.sh | sudo bash

# Uninstall
curl -fsSL https://raw.githubusercontent.com/aloks98/wolnut/master/scripts/uninstall.sh | sudo bash
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/devices` | List all devices |
| POST | `/api/devices` | Add device |
| DELETE | `/api/devices/{id}` | Delete device |
| POST | `/api/devices/{id}/wake` | Wake device |
| GET | `/api/ups` | List all UPS entries |
| POST | `/api/ups` | Add UPS |
| DELETE | `/api/ups/{id}` | Delete UPS |
| GET | `/api/ups/status` | Get all UPS statuses with detailed info |
| GET | `/api/config/export` | Export data.json backup |
| POST | `/api/config/import` | Import data.json backup |
| GET | `/api/health` | Health check |

### UPS Status Response

The `/api/ups/status` endpoint returns detailed UPS information:

```json
{
  "id": "abc123",
  "name": "Main UPS",
  "host": "localhost:3493",
  "online": true,
  "status": "OL CHRG",
  "status_label": "Online, Charging",
  "is_online": true,
  "is_on_battery": false,
  "is_low_battery": false,
  "is_charging": true,
  "battery_charge": 100,
  "battery_voltage": 27.2,
  "battery_runtime": 3600,
  "load": 15,
  "power": 150,
  "nominal": 1000,
  "current": 0.7,
  "input_voltage": 230,
  "input_frequency": 50,
  "output_voltage": 230,
  "output_frequency": 50,
  "model": "Smart-UPS 1000",
  "manufacturer": "APC",
  "serial": "ABC123",
  "firmware": "1.2.3",
  "temperature": 25,
  "estimated_wattage": 150
}
```

## Development

### Prerequisites

- Go 1.22+
- Node.js 22+ with pnpm

### Build & Run

```bash
# Clone repository
git clone https://github.com/aloks98/wolnut.git
cd wolnut

# Install frontend dependencies
cd web && pnpm install && cd ..

# Development mode (frontend + backend separately)
# Terminal 1: Frontend dev server
cd web && pnpm dev

# Terminal 2: Backend (reads from web/build)
go run .

# Production build
cd web && pnpm build && cd ..
go build -o wol-nut

# Run with config
./wol-nut --config config.example.yml
```

### Docker Development

```bash
# Build with docker compose
docker compose build

# Run
docker compose up
```

## Architecture

| Layer | Technology |
|-------|------------|
| Frontend Framework | SvelteKit (static) |
| UI Components | shadcn-svelte |
| Styling | Tailwind CSS v4 |
| State Management | Svelte 5 runes |
| Backend | Go (net/http) |
| Data Storage | JSON file |
| Embedding | Go embed |

The frontend is built as a static SPA and embedded into the Go binary at compile time. In production, a single binary serves both the API and the frontend.

## Requirements

- **Wake-on-LAN**: Requires network access to broadcast UDP packets
- **NUT Monitoring**: Requires TCP access to NUT server (port 3493)
- **Docker**: Use `network_mode: host` for full WoL functionality

## License

MIT
