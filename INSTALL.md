
# CheckCle Server Monitoring Agent - Installation Guide

This guide covers the installation of the CheckCle Server Monitoring Agent using the provided installation scripts.

## Prerequisites

- Linux system with systemd
- Root/sudo access
- Built package (run `make deb` first)

## Installation Methods

### Method 1: Interactive Installation (Recommended)

The interactive installer guides you through the configuration process:

```bash
sudo ./install.sh
```

This script will:
1. Install the .deb package
2. Auto-detect system information
3. Guide you through configuration
4. Start and enable the service
5. Test the installation

### Method 2: Quick Installation (Non-interactive)

For automated deployments, use the quick installer:

```bash
SERVER_TOKEN=your-token sudo ./quick-install.sh
```

Optional environment variables:
```bash
SERVER_TOKEN=your-token \
POCKETBASE_URL=http://your-pb-url:8090 \
SERVER_NAME=your-server-name \
sudo ./quick-install.sh
```

### Method 3: Manual Installation

1. **Install the package:**
   ```bash
   sudo dpkg -i dist/monitoring-agent.deb
   sudo apt-get install -f  # Fix dependencies if needed
   ```

2. **Configure the agent:**
   ```bash
   sudo nano /etc/monitoring-agent/monitoring-agent.env
   ```

3. **Start the service:**
   ```bash
   sudo systemctl enable monitoring-agent
   sudo systemctl start monitoring-agent
   ```

## Configuration

The agent requires the following configuration in `/etc/monitoring-agent/monitoring-agent.env`:

### Required Settings
```bash
SERVER_TOKEN=your-unique-server-token
POCKETBASE_URL=http://your-pocketbase-server:8090
```

### Auto-detected Settings (can be overridden)
```bash
SERVER_NAME=your-server-name
HOSTNAME=your-hostname
IP_ADDRESS=your-server-ip
OS_TYPE=linux
```

### Optional Settings
```bash
AGENT_ID=monitoring-agent-001
CHECK_INTERVAL=30s
HEALTH_CHECK_PORT=8081
REMOTE_CONTROL_ENABLED=true
COMMAND_CHECK_INTERVAL=10s
REPORT_INTERVAL=5m
MAX_RETRIES=3
REQUEST_TIMEOUT=10s
```

## Post-Installation

### Verify Installation
```bash
# Check service status
sudo systemctl status monitoring-agent

# View logs
sudo journalctl -u monitoring-agent -f

# Test health endpoint
curl http://localhost:8081/health
```

### Service Management
```bash
# Start service
sudo systemctl start monitoring-agent

# Stop service
sudo systemctl stop monitoring-agent

# Restart service
sudo systemctl restart monitoring-agent

# Enable auto-start
sudo systemctl enable monitoring-agent

# Disable auto-start
sudo systemctl disable monitoring-agent
```

## Troubleshooting

### Common Issues

1. **Service fails to start:**
   - Check configuration: `sudo nano /etc/monitoring-agent/monitoring-agent.env`
   - Verify PocketBase URL is accessible
   - Check logs: `sudo journalctl -u monitoring-agent -f`

2. **Permission errors:**
   - Ensure config file ownership: `sudo chown root:monitoring-agent /etc/monitoring-agent/monitoring-agent.env`
   - Set proper permissions: `sudo chmod 640 /etc/monitoring-agent/monitoring-agent.env`

3. **Network connectivity:**
   - Test PocketBase connection: `curl http://your-pocketbase-url/api/health`
   - Check firewall settings

### Log Analysis
```bash
# View recent logs
sudo journalctl -u monitoring-agent -n 50

# Follow logs in real-time
sudo journalctl -u monitoring-agent -f

# View logs with timestamps
sudo journalctl -u monitoring-agent -o short-iso
```

## Uninstallation

### Using the installer script
```bash
sudo ./install.sh --uninstall
```

### Manual uninstallation
```bash
# Stop and disable service
sudo systemctl stop monitoring-agent
sudo systemctl disable monitoring-agent

# Remove package
sudo dpkg -r monitoring-agent

# Complete removal (including config)
sudo dpkg --purge monitoring-agent
```

## File Locations

- **Binary**: `/usr/bin/monitoring-agent`
- **Configuration**: `/etc/monitoring-agent/monitoring-agent.env`
- **Service file**: `/etc/systemd/system/monitoring-agent.service`
- **Working directory**: `/var/lib/monitoring-agent`
- **Log directory**: `/var/log/monitoring-agent`

## Support

For issues and support:
1. Check the logs: `sudo journalctl -u monitoring-agent -f`
2. Verify configuration is complete
3. Test network connectivity to PocketBase
4. Review this troubleshooting guide

## Example Complete Installation

```bash
# 1. Build the package
make clean && make deb

# 2. Install interactively
sudo ./install.sh

# 3. Or quick install
SERVER_TOKEN=my-secret-token sudo ./quick-install.sh

# 4. Verify installation
sudo systemctl status monitoring-agent
curl http://localhost:8081/health
```