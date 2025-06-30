
# CheckCle Server Monitoring Agent

A Go-based monitoring agent with gRPC support, PocketBase integration, and remote control capabilities.

## Features

- System metrics collection (CPU, Memory, Disk, Network, Docker Container)
- Multiple communication protocols:
  - gRPC for efficient communication
  - PocketBase for database storage
  - HTTP REST API as fallback
- Remote control commands (start/stop monitoring)
- Health check endpoints
- Configurable via environment variables
- **Linux .deb package for easy installation**

## Quick Install (Debian/Ubuntu)

### Method 1: Install from .deb package (Recommended)

1. Build the .deb package:
   ```bash
   cd monitoring-agent
   chmod +x build.sh
   ./build.sh
   ```

2. Install the package:
   ```bash
   sudo dpkg -i build/monitoring-agent_1.0.0_amd64.deb
   sudo apt-get install -f  # Install any missing dependencies
   ```

3. Configure the agent:
   ```bash
   sudo nano /etc/monitoring-agent/monitoring-agent.env
   # Set your API_KEY, server URLs, and other configuration
   ```

4. Start the service:
   ```bash
   sudo systemctl enable monitoring-agent
   sudo systemctl start monitoring-agent
   sudo systemctl status monitoring-agent
   ```

### Method 2: Manual Installation

#### Prerequisites

1. Install Go 1.21 or later
2. Install Protocol Buffers compiler (protoc) - optional:
   ```bash
   sudo apt-get update
   sudo apt-get install protobuf-compiler
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

#### Build and Install

```bash
cd monitoring-agent

# Generate Protobuf Files (optional)
protoc --go_out=. --go-grpc_out=. proto/monitoring.proto

# Install Dependencies
go mod tidy

# Build and install manually
make install
```

## Configuration

### Package Installation Configuration

When installed via .deb package, edit `/etc/monitoring-agent/monitoring-agent.env`:

```bash
# Required for HTTP mode
API_KEY=your-secret-api-key

# Server endpoints
SERVER_URL=https://your-server.com
GRPC_SERVER_ADDR=your-server.com:50051
POCKETBASE_URL=https://your-pocketbase.com

# Optional: Custom agent ID
AGENT_ID=monitoring-agent-custom-001
SERVER_TOKEN=xxxxx
```

### Manual Installation Configuration

Set environment variables directly:

#### Basic Configuration
- `AGENT_ID`: Unique identifier for the agent (default: "monitoring-agent-001")
- `CHECK_INTERVAL`: Metrics collection interval (default: "30s")
- `HEALTH_CHECK_PORT`: Health check server port (default: 9091)

#### HTTP REST API (fallback)
- `SERVER_URL`: Server URL for HTTP API (default: "http://localhost:8080")
- `API_KEY`: API key for authentication

#### gRPC Configuration
- `GRPC_ENABLED`: Enable gRPC communication (default: true)
- `GRPC_SERVER_ADDR`: gRPC server address (default: "localhost:50051")

#### PocketBase Configuration
- `POCKETBASE_ENABLED`: Enable PocketBase integration (default: false)
- `POCKETBASE_URL`: PocketBase server URL (default: "http://localhost:8090")

#### Remote Control
- `REMOTE_CONTROL_ENABLED`: Enable remote control (default: true)
- `COMMAND_CHECK_INTERVAL`: Command check interval (default: "10s")

## Usage

### Package Installation

```bash
# Check status
sudo systemctl status monitoring-agent

# View logs
sudo journalctl -u monitoring-agent -f

# Restart service
sudo systemctl restart monitoring-agent

# Stop service
sudo systemctl stop monitoring-agent

# Disable service
sudo systemctl disable monitoring-agent
```

### Manual Installation

```bash
go run main.go
```

### Health Check Endpoints

- `GET /health` - Agent health status
- `GET /status` - Current system metrics
- `POST /control/start` - Start monitoring
- `POST /control/stop` - Stop monitoring

Default health check URL: `http://localhost:9091/health`

### Remote Commands

The agent supports the following remote commands via gRPC or PocketBase:
- `start` - Start monitoring
- `stop` - Stop monitoring
- `restart` - Restart monitoring
- `config_update` - Update configuration

## Building .deb Package

### Requirements

- Go 1.21+
- `dpkg-dev` package: `sudo apt-get install dpkg-dev`
- `protoc` (optional): `sudo apt-get install protobuf-compiler`

### Build Process

```bash
# Quick build
./build.sh

# Or step by step
make clean
make deb

# The package will be created at: build/monitoring-agent_1.0.0_amd64.deb
```

### Package Management

```bash
# Install
sudo dpkg -i build/monitoring-agent_1.0.0_amd64.deb

# Remove (keep configuration)
sudo dpkg -r monitoring-agent

# Purge (remove everything)
sudo dpkg -P monitoring-agent

# Check if installed
dpkg -l | grep monitoring-agent
```

## PocketBase Collections

If using PocketBase, create the following collections:

### metrics
```javascript
    {
      "collectionId": "pbc_1998570700",
      "collectionName": "server_metrics",
      "id": "test",
      "server_id": "test",
      "timestamp": "2022-01-01 10:00:00.123Z",
      "ram_total": "test",
      "ram_used": "test",
      "ram_free": "test",
      "cpu_cores": "test",
      "cpu_usage": "test",
      "cpu_free": "test",
      "disk_total": "test",
      "disk_used": "test",
      "disk_free": "test",
      "status": "test",
      "network_rx_bytes": 123,
      "network_tx_bytes": 123,
      "network_rx_speed": 123,
      "network_tx_speed": 123,
      "agent_status": "test",
      "created": "2022-01-01 10:00:00.123Z",
      "updated": "2022-01-01 10:00:00.123Z"
    },
```

### agent_status
```javascript
{
  "agent_id": "text",
  "status": "text",
  "last_seen": "date",
  "version": "text",
  "message": "text"
}
```

### agent_commands
```javascript
{
  "agent_id": "text",
  "command": "text",
  "parameters": "json",
  "executed": "bool",
  "created": "date"
}
```

## Example Configurations

### Production with gRPC and PocketBase
```bash
# /etc/monitoring-agent/monitoring-agent.env
GRPC_ENABLED=true
GRPC_SERVER_ADDR=monitoring.company.com:50051
POCKETBASE_ENABLED=true
POCKETBASE_URL=https://pocketbase.company.com
AGENT_ID=server-001-monitoring
SERVER_TOKEN=xxxxx
```

### Simple HTTP-only mode
```bash
# /etc/monitoring-agent/monitoring-agent.env
GRPC_ENABLED=false
POCKETBASE_ENABLED=false
API_KEY=your-secret-api-key
SERVER_URL=https://monitoring-api.company.com
AGENT_ID=server-001-http
SERVER_TOKEN=xxxxx
```

## Troubleshooting

### Check service status
```bash
sudo systemctl status monitoring-agent
sudo journalctl -u monitoring-agent -f
```

### Check health endpoint
```bash
curl http://localhost:8081/health
```

### Permission issues
```bash
# Fix ownership
sudo chown -R monitoring-agent:monitoring-agent /var/lib/monitoring-agent
sudo chown -R monitoring-agent:monitoring-agent /var/log/monitoring-agent
```

### Uninstall and reinstall
```bash
sudo dpkg -P monitoring-agent
sudo dpkg -i build/monitoring-agent_1.0.0_amd64.deb
```

## Development

### Manual build
```bash
go build -o monitoring-agent main.go
```

### Testing locally
```bash
make install
sudo systemctl start monitoring-agent
```

### Clean up
```bash
make clean
make uninstall
```
