
# Server Monitoring Agent - Packaging Guide

This directory contains the packaging infrastructure for building the Server Monitoring Agent as both .deb and .rpm packages for Linux distributions.

## Quick Start

```bash
# Build all available packages (DEB and RPM)
./build.sh

# Build specific package type
./build.sh deb #Build DEB packages for both architectures
./build.sh rpm 
./build.sh rpm arm64 #Build RPM package for ARM64 only
./build.sh all amd64 #- Build all packages for AMD64 only
```

## Package Structure

```
packaging/
├── control                    # Debian package metadata
├── monitoring-agent.spec      # RPM package specification
├── postinst                  # Debian post-installation script
├── prerm                     # Debian pre-removal script
├── postrm                    # Debian post-removal script
├── monitoring-agent.service  # Systemd service file
└── monitoring-agent.conf     # Default configuration template
```

## Build System

### Prerequisites

**For DEB packages:**
```bash
sudo apt-get install dpkg-dev
```

**For RPM packages:**
```bash
# Ubuntu/Debian
sudo apt-get install rpm

# CentOS/RHEL/Fedora
sudo yum install rpm-build
# or
sudo dnf install rpm-build
```

### Make Targets

- `make build` - Build the Go binary
- `make deb` - Create .deb package
- `make rpm` - Create .rpm package
- `make package-all` - Create both .deb and .rpm packages
- `make install-deb` - Install the .deb package
- `make install-rpm` - Install the .rpm package
- `make clean` - Clean build artifacts
- `make help` - Show all available targets

### Build Script

The `build.sh` script provides a convenient wrapper:

```bash
./build.sh           # Build all available packages
./build.sh deb       # Build only DEB package
./build.sh rpm       # Build only RPM package
```

## Installation

### System Requirements

- Linux distribution with systemd
- libc6 (>= 2.17) for DEB packages
- glibc for RPM packages

### DEB Installation (Ubuntu/Debian)

```bash
# Install the package
sudo dpkg -i dist/monitoring-agent.deb

# Install dependencies if needed
sudo apt-get install -f
```

### RPM Installation (CentOS/RHEL/Fedora)

```bash
# Install the package
sudo rpm -ivh dist/monitoring-agent-1.0.0-1.x86_64.rpm

# Alternative using package manager
sudo yum localinstall dist/monitoring-agent-1.0.0-1.x86_64.rpm
# or
sudo dnf localinstall dist/monitoring-agent-1.0.0-1.x86_64.rpm
```

### Configuration (Required for both packages)

1. **Configure the agent:**
   ```bash
   sudo nano /etc/monitoring-agent/monitoring-agent.env
   ```

   Set these required variables:
   ```bash
   SERVER_NAME=your-server-name
   HOSTNAME=your-hostname
   IP_ADDRESS=your-server-ip
   OS_TYPE=your-os-type
   SERVER_TOKEN=your-unique-token
   POCKETBASE_URL=http://your-pocketbase-server:8090
   ```

2. **Enable and start the service:**
   ```bash
   sudo systemctl enable monitoring-agent
   sudo systemctl start monitoring-agent
   ```

3. **Verify the installation:**
   ```bash
   sudo systemctl status monitoring-agent
   curl http://localhost:8081/health
   ```

## Configuration

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `SERVER_NAME` | Name of the server being monitored | Yes | - |
| `HOSTNAME` | Hostname of the server | Yes | - |
| `IP_ADDRESS` | IP address of the server | Yes | - |
| `OS_TYPE` | Operating system type | Yes | - |
| `SERVER_TOKEN` | Unique token for server identification | Yes | - |
| `POCKETBASE_URL` | PocketBase server URL | Yes | `http://localhost:8090` |
| `HEALTH_CHECK_PORT` | Port for health check endpoint | No | `8081` |
| `CHECK_INTERVAL` | Monitoring check interval | No | `30s` |

### File Locations

- **Service config**: `/etc/monitoring-agent/monitoring-agent.env`
- **Systemd service**: `/etc/systemd/system/monitoring-agent.service`
- **Working directory**: `/var/lib/monitoring-agent`
- **Log directory**: `/var/log/monitoring-agent`
- **Binary location**: `/usr/bin/monitoring-agent`

## Service Management

### Common Commands

```bash
# Start the service
sudo systemctl start monitoring-agent

# Stop the service
sudo systemctl stop monitoring-agent

# Restart the service
sudo systemctl restart monitoring-agent

# Check service status
sudo systemctl status monitoring-agent

# View logs
sudo journalctl -u monitoring-agent -f

# Enable auto-start
sudo systemctl enable monitoring-agent

# Disable auto-start
sudo systemctl disable monitoring-agent
```

### Health Check

The agent provides a health check endpoint:

```bash
curl http://localhost:8081/health
```

## Uninstallation

### DEB Packages
```bash
# Remove package but keep configuration
sudo dpkg -r monitoring-agent

# Remove package and all configuration
sudo dpkg --purge monitoring-agent
```

### RPM Packages
```bash
# Remove package
sudo rpm -e monitoring-agent

# Remove package (alternative)
sudo yum remove monitoring-agent
# or
sudo dnf remove monitoring-agent
```

## Development

### Building from Source

```bash
# Clone and build
git clone <repository>
cd server-monitoring-agent
./build.sh
```

### Testing Packages

```bash
# Test DEB installation
make install-deb

# Test RPM installation
make install-rpm

# Run tests
make test
```

## Package Information

- **Package name**: `monitoring-agent`
- **Version**: `1.0.0`
- **Architecture**: `amd64`/`x86_64`
- **Service name**: `monitoring-agent`
- **Binary location**: `/usr/bin/monitoring-agent`

## Troubleshooting

### Common Issues

1. **Service won't start**
   - Check configuration: `sudo systemctl status monitoring-agent`
   - Verify environment variables are set correctly
   - Check logs: `sudo journalctl -u monitoring-agent`

2. **Permission denied errors**
   - Ensure proper file ownership
   - Check systemd service permissions

3. **Network connectivity issues**
   - Verify PocketBase URL is accessible
   - Check firewall settings
   - Test with curl: `curl http://your-pocketbase-url/api/health`

### Log Files

- **Systemd logs**: `sudo journalctl -u monitoring-agent`
- **Application logs**: `/var/log/monitoring-agent/` (if file logging is enabled)