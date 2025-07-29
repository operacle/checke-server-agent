#!/bin/bash

# CheckCle Server Monitoring Agent - One-Click Installation Script
# This script provides fully automated installation using environment variables

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="/etc/monitoring-agent/monitoring-agent.env"

# GitHub release base URL
GITHUB_BASE_URL="https://github.com/operacle/checke-server-agent/releases/download/v1.0.0"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect system architecture and package format with fallback
detect_system() {
    log_info "Detecting system architecture and package format..."
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            PACKAGE_ARCH="amd64"
            ;;
        aarch64|arm64)
            PACKAGE_ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            log_info "Supported architectures: x86_64 (amd64), aarch64/arm64"
            exit 1
            ;;
    esac
    
    # Detect OS and preferred package format
    PREFERRED_FORMAT=""
    PREFERRED_MANAGER=""
    
    if [[ -f /etc/os-release ]]; then
        OS_ID=$(grep "^ID=" /etc/os-release | cut -d'=' -f2 | tr -d '"')
        OS_LIKE=$(grep "^ID_LIKE=" /etc/os-release 2>/dev/null | cut -d'=' -f2 | tr -d '"' || echo "")
        
        log_info "Detected OS: $OS_ID"
        [[ -n "$OS_LIKE" ]] && log_info "OS family: $OS_LIKE"
        
        # Determine preferred package format based on OS
        case "$OS_ID" in
            debian|ubuntu|linuxmint|elementary|pop)
                PREFERRED_FORMAT="deb"
                PREFERRED_MANAGER="dpkg"
                ;;
            rhel|centos|fedora|rocky|almalinux|oracle|amazonlinux)
                PREFERRED_FORMAT="rpm"
                if command -v dnf >/dev/null 2>&1; then
                    PREFERRED_MANAGER="dnf"
                elif command -v yum >/dev/null 2>&1; then
                    PREFERRED_MANAGER="yum"
                else
                    PREFERRED_MANAGER="rpm"
                fi
                ;;
            arch|manjaro|artix|endeavouros)
                # Arch-based systems - use tar.gz fallback
                log_info "Arch-based system detected, using tar.gz format"
                PREFERRED_FORMAT="tar.gz"
                PREFERRED_MANAGER="tar"
                ;;
            *)
                # Check OS_LIKE for family detection
                if [[ "$OS_LIKE" == *"debian"* ]] || [[ "$OS_LIKE" == *"ubuntu"* ]]; then
                    PREFERRED_FORMAT="deb"
                    PREFERRED_MANAGER="dpkg"
                elif [[ "$OS_LIKE" == *"rhel"* ]] || [[ "$OS_LIKE" == *"fedora"* ]]; then
                    PREFERRED_FORMAT="rpm"
                    if command -v dnf >/dev/null 2>&1; then
                        PREFERRED_MANAGER="dnf"
                    elif command -v yum >/dev/null 2>&1; then
                        PREFERRED_MANAGER="yum"
                    else
                        PREFERRED_MANAGER="rpm"
                    fi
                elif [[ "$OS_LIKE" == *"arch"* ]]; then
                    PREFERRED_FORMAT="tar.gz"
                    PREFERRED_MANAGER="tar"
                fi
                ;;
        esac
    fi
    
    # If no preferred format detected, check available package managers
    if [[ -z "$PREFERRED_FORMAT" ]]; then
        if command -v dpkg >/dev/null 2>&1; then
            PREFERRED_FORMAT="deb"
            PREFERRED_MANAGER="dpkg"
        elif command -v rpm >/dev/null 2>&1; then
            PREFERRED_FORMAT="rpm"
            if command -v dnf >/dev/null 2>&1; then
                PREFERRED_MANAGER="dnf"
            elif command -v yum >/dev/null 2>&1; then
                PREFERRED_MANAGER="yum"
            else
                PREFERRED_MANAGER="rpm"
            fi
        else
            # Fallback to tar.gz for unsupported systems
            log_info "No native package manager found, using tar.gz fallback"
            PREFERRED_FORMAT="tar.gz"
            PREFERRED_MANAGER="tar"
        fi
    fi
    
    # Set initial package format and manager
    PACKAGE_FORMAT="$PREFERRED_FORMAT"
    PACKAGE_MANAGER="$PREFERRED_MANAGER"
    
    # Construct initial package filename and URL
    construct_package_url
    
    log_success "System detection complete:"
    log_info "  Architecture: $ARCH -> $PACKAGE_ARCH"
    log_info "  Package format: $PACKAGE_FORMAT"
    log_info "  Package manager: $PACKAGE_MANAGER"
    log_info "  Package URL: $PACKAGE_URL"
}

# Construct package URL based on current format
construct_package_url() {
    if [[ "$PACKAGE_FORMAT" == "tar.gz" ]]; then
        PACKAGE_FILENAME="monitoring-agent_1.0.0_${PACKAGE_ARCH}.tar.gz"
    else
        PACKAGE_FILENAME="monitoring-agent_1.0.0_${PACKAGE_ARCH}.${PACKAGE_FORMAT}"
    fi
    PACKAGE_URL="${GITHUB_BASE_URL}/${PACKAGE_FILENAME}"
}

# Validate required environment variables - check both current and sudo environments
validate_environment() {
    # Check if SERVER_TOKEN is available in current environment or passed as argument
    if [[ -z "$SERVER_TOKEN" ]]; then
        log_error "SERVER_TOKEN environment variable is required"
        log_info "Usage examples:"
        log_info "  SERVER_TOKEN=your-token sudo -E bash $0"
        log_info "  sudo SERVER_TOKEN=your-token bash $0" 
        log_info "  curl -L script-url | SERVER_TOKEN=your-token sudo bash"
        log_info ""
        log_info "Required: SERVER_TOKEN"
        log_info "Optional: POCKETBASE_URL, SERVER_NAME, AGENT_ID, HEALTH_CHECK_PORT"
        exit 1
    fi
    
    log_success "Environment validation passed"
    log_info "SERVER_TOKEN: ${SERVER_TOKEN:0:8}..."
    [[ -n "$POCKETBASE_URL" ]] && log_info "POCKETBASE_URL: $POCKETBASE_URL"
    [[ -n "$SERVER_NAME" ]] && log_info "SERVER_NAME: $SERVER_NAME"
    [[ -n "$AGENT_ID" ]] && log_info "AGENT_ID: $AGENT_ID"
}

# Download package with fallback to tar.gz
download_package() {
    local temp_dir="/tmp/monitoring-agent-install"
    mkdir -p "$temp_dir"
    
    log_info "Downloading monitoring agent package..."
    log_info "URL: $PACKAGE_URL"
    
    # Try to download the preferred package format first
    if curl -L -f -s -o "$temp_dir/$PACKAGE_FILENAME" "$PACKAGE_URL" 2>/dev/null; then
        # Verify the file was actually downloaded and has content
        if [[ -s "$temp_dir/$PACKAGE_FILENAME" ]]; then
            DOWNLOADED_PACKAGE="$temp_dir/$PACKAGE_FILENAME"
            log_success "Package downloaded successfully: $PACKAGE_FILENAME"
            return 0
        else
            log_warning "Downloaded file is empty, trying fallback..."
            rm -f "$temp_dir/$PACKAGE_FILENAME"
        fi
    else
        log_warning "Failed to download preferred package format: $PACKAGE_FORMAT"
    fi
    
    # If preferred format failed and it's not tar.gz, try tar.gz fallback
    if [[ "$PACKAGE_FORMAT" != "tar.gz" ]]; then
        log_warning "Native package ($PACKAGE_FORMAT) not available, trying tar.gz fallback..."
        
        # Switch to tar.gz format
        PACKAGE_FORMAT="tar.gz"
        PACKAGE_MANAGER="tar"
        construct_package_url
        
        log_info "Fallback URL: $PACKAGE_URL"
        
        if curl -L -f -s -o "$temp_dir/$PACKAGE_FILENAME" "$PACKAGE_URL" 2>/dev/null; then
            # Verify the file was actually downloaded and has content
            if [[ -s "$temp_dir/$PACKAGE_FILENAME" ]]; then
                DOWNLOADED_PACKAGE="$temp_dir/$PACKAGE_FILENAME"
                log_success "Fallback package downloaded successfully: $PACKAGE_FILENAME"
                return 0
            else
                log_error "Downloaded fallback file is empty"
                rm -f "$temp_dir/$PACKAGE_FILENAME"
            fi
        else
            log_error "Failed to download tar.gz fallback"
        fi
    fi
    
    # Both attempts failed
    log_error "Failed to download package from: $PACKAGE_URL"
    log_info "Please check:"
    log_info "  1. Internet connectivity"
    log_info "  2. Package availability for your architecture ($PACKAGE_ARCH)"
    log_info "  3. GitHub repository access"
    exit 1
}

# Install tar.gz package
install_tar_package() {
    log_info "Installing monitoring agent from tar.gz package..."
    
    local temp_extract_dir="/tmp/monitoring-agent-extract"
    mkdir -p "$temp_extract_dir"
    
    # Extract tar.gz package
    if tar -xzf "$DOWNLOADED_PACKAGE" -C "$temp_extract_dir"; then
        log_success "Package extracted successfully"
    else
        log_error "Failed to extract tar.gz package"
        exit 1
    fi
    
    # Install files to their proper locations
    log_info "Installing files to system directories..."
    
    # Create necessary directories
    mkdir -p /usr/bin
    mkdir -p /etc/monitoring-agent
    mkdir -p /var/lib/monitoring-agent
    mkdir -p /var/log/monitoring-agent
    
    # Detect systemd service directory based on OS
    if [[ -d /lib/systemd/system ]]; then
        SYSTEMD_DIR="/lib/systemd/system"
    elif [[ -d /usr/lib/systemd/system ]]; then
        SYSTEMD_DIR="/usr/lib/systemd/system"
    else
        SYSTEMD_DIR="/etc/systemd/system"
    fi
    mkdir -p "$SYSTEMD_DIR"
    
    # Copy files from extracted package
    if [[ -f "$temp_extract_dir/usr/bin/monitoring-agent" ]]; then
        cp "$temp_extract_dir/usr/bin/monitoring-agent" /usr/bin/
        chmod +x /usr/bin/monitoring-agent
        log_success "Binary installed to /usr/bin/monitoring-agent"
    else
        log_error "Binary not found in package"
        exit 1
    fi
    
    if [[ -f "$temp_extract_dir/etc/monitoring-agent/monitoring-agent.env" ]]; then
        cp "$temp_extract_dir/etc/monitoring-agent/monitoring-agent.env" /etc/monitoring-agent/
        log_success "Configuration template installed"
    fi
    
    # Create monitoring-agent user and group if they don't exist
    if ! getent group monitoring-agent >/dev/null 2>&1; then
        groupadd --system monitoring-agent
        log_success "Created monitoring-agent group"
    else
        log_info "monitoring-agent group already exists"
    fi
    
    if ! getent passwd monitoring-agent >/dev/null 2>&1; then
        useradd --system --gid monitoring-agent --home-dir /var/lib/monitoring-agent --shell /bin/false --comment "Monitoring Agent" monitoring-agent
        log_success "Created monitoring-agent user"
    else
        log_info "monitoring-agent user already exists"
    fi
    
    # Set proper ownership and permissions
    chown -R monitoring-agent:monitoring-agent /var/lib/monitoring-agent
    chown -R monitoring-agent:monitoring-agent /var/log/monitoring-agent
    chown -R monitoring-agent:monitoring-agent /etc/monitoring-agent
    chmod 755 /var/lib/monitoring-agent
    chmod 755 /var/log/monitoring-agent
    chmod 755 /etc/monitoring-agent
    
    # Check if Docker is installed and configure service file accordingly
    DOCKER_AVAILABLE=false
    if command -v docker >/dev/null 2>&1 && getent group docker >/dev/null 2>&1; then
        DOCKER_AVAILABLE=true
        # Add monitoring-agent user to docker group if Docker is available
        usermod -a -G docker monitoring-agent
        log_info "Docker detected, added monitoring-agent user to docker group"
    else
        log_info "Docker not detected, skipping Docker group configuration"
    fi
    
    # Create the systemd service file with proper Docker configuration
    create_systemd_service_file "$SYSTEMD_DIR" "$DOCKER_AVAILABLE"
    
    # Clean up extraction directory
    rm -rf "$temp_extract_dir"
    
    log_success "Tar.gz package installation completed"
}

# Create systemd service file with Docker configuration
create_systemd_service_file() {
    local systemd_dir="$1"
    local docker_available="$2"
    
    log_info "Creating systemd service file at $systemd_dir/monitoring-agent.service"
    
    cat > "$systemd_dir/monitoring-agent.service" << 'EOF'
[Unit]
Description=CheckCle Server Monitoring Agent
Documentation=https://github.com/operacle/checkcle-server-agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=monitoring-agent
Group=monitoring-agent
EOF

    # Add Docker group only if Docker is available
    if [[ "$docker_available" == "true" ]]; then
        echo "# Add docker group for Docker monitoring access" >> "$systemd_dir/monitoring-agent.service"
        echo "SupplementaryGroups=docker" >> "$systemd_dir/monitoring-agent.service"
    else
        echo "# Docker not available - skipping Docker group configuration" >> "$systemd_dir/monitoring-agent.service"
    fi

    # Continue with the rest of the service configuration
    cat >> "$systemd_dir/monitoring-agent.service" << 'EOF'
ExecStart=/usr/bin/monitoring-agent
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ProtectKernelTunables=yes
ProtectControlGroups=yes
RestrictSUIDSGID=yes
RemoveIPC=yes
RestrictRealtime=yes

# Allow access to system information and configuration
ReadWritePaths=/var/log/monitoring-agent
ReadOnlyPaths=/proc /sys /etc/monitoring-agent

# Network access
PrivateNetwork=no

# Environment
EnvironmentFile=/etc/monitoring-agent/monitoring-agent.env
WorkingDirectory=/var/lib/monitoring-agent

[Install]
WantedBy=multi-user.target
EOF
    
    log_success "Created systemd service file with Docker support: $docker_available"
}

# Install package based on detected package manager
install_package() {
    log_info "Installing monitoring agent package using $PACKAGE_MANAGER..."
    
    case $PACKAGE_MANAGER in
        dpkg)
            # Update package lists
            apt-get update -qq
            
            # Install the package
            if dpkg -i "$DOWNLOADED_PACKAGE" 2>/dev/null; then
                log_success "DEB package installed successfully"
            else
                log_warning "Package installation had dependency issues, fixing..."
                apt-get install -f -y
                log_success "Dependencies resolved and package installed"
            fi
            ;;
            
        rpm)
            # Install the package directly
            if rpm -ivh "$DOWNLOADED_PACKAGE" 2>/dev/null; then
                log_success "RPM package installed successfully"
            else
                log_error "RPM package installation failed"
                log_info "Try installing manually: sudo rpm -ivh $DOWNLOADED_PACKAGE"
                exit 1
            fi
            ;;
            
        yum)
            if yum localinstall -y "$DOWNLOADED_PACKAGE"; then
                log_success "Package installed successfully via YUM"
            else
                log_error "YUM package installation failed"
                exit 1
            fi
            ;;
            
        dnf)
            if dnf localinstall -y "$DOWNLOADED_PACKAGE"; then
                log_success "Package installed successfully via DNF"
            else
                log_error "DNF package installation failed"
                exit 1
            fi
            ;;
            
        tar)
            install_tar_package
            ;;
            
        *)
            log_error "Unsupported package manager: $PACKAGE_MANAGER"
            exit 1
            ;;
    esac
}

# Auto-detect system information
detect_system_info() {
    log_info "Auto-detecting system information..."
    
    # Detect hostname
    DETECTED_HOSTNAME=$(hostname)
    log_info "Detected hostname: $DETECTED_HOSTNAME"
    
    # Detect IP address
    DETECTED_IP=$(ip route get 8.8.8.8 2>/dev/null | head -1 | awk '{print $7}' | head -1)
    if [[ -z "$DETECTED_IP" ]]; then
        DETECTED_IP=$(hostname -I | awk '{print $1}')
    fi
    log_info "Detected IP address: $DETECTED_IP"
    
    # Detect OS
    if [[ -f /etc/os-release ]]; then
        DETECTED_OS=$(grep "^ID=" /etc/os-release | cut -d'=' -f2 | tr -d '"')
    else
        DETECTED_OS="linux"
    fi
    log_info "Detected OS: $DETECTED_OS"
}

# Configure agent using environment variables
configure_agent() {
    log_info "Configuring agent with provided settings..."
    
    # Use environment variables or auto-detected defaults
    SERVER_NAME="${SERVER_NAME:-$DETECTED_HOSTNAME}"
    POCKETBASE_URL="${POCKETBASE_URL:-http://localhost:8090}"
    IP_ADDRESS="${IP_ADDRESS:-$DETECTED_IP}"
    HOSTNAME="${HOSTNAME:-$DETECTED_HOSTNAME}"
    OS_TYPE="${OS_TYPE:-$DETECTED_OS}"
    AGENT_ID="${AGENT_ID:-monitoring-agent-$(hostname -s)}"
    HEALTH_CHECK_PORT="${HEALTH_CHECK_PORT:-8081}"
    
    log_info "Final configuration:"
    log_info "  Server Name: $SERVER_NAME"
    log_info "  Agent ID: $AGENT_ID"
    log_info "  PocketBase URL: $POCKETBASE_URL"
    log_info "  IP Address: $IP_ADDRESS"
    log_info "  OS Type: $OS_TYPE"
    log_info "  Health Check Port: $HEALTH_CHECK_PORT"
    
    write_config
}

# Write configuration to file
write_config() {
    log_info "Writing configuration to $CONFIG_FILE..."
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$CONFIG_FILE")"
    
    cat > "$CONFIG_FILE" << EOF
# CheckCle Server Monitoring Agent Configuration
# Generated on $(date)

# Basic Configuration
AGENT_ID=$AGENT_ID
CHECK_INTERVAL=30s
HEALTH_CHECK_PORT=$HEALTH_CHECK_PORT

# PocketBase Configuration
POCKETBASE_ENABLED=true
POCKETBASE_URL=$POCKETBASE_URL

# Server Configuration
SERVER_NAME=$SERVER_NAME
HOSTNAME=$HOSTNAME
IP_ADDRESS=$IP_ADDRESS
OS_TYPE=$OS_TYPE
SERVER_TOKEN=$SERVER_TOKEN

# Remote Control
REMOTE_CONTROL_ENABLED=true
COMMAND_CHECK_INTERVAL=10s

# Monitoring Settings
REPORT_INTERVAL=5m
MAX_RETRIES=3
REQUEST_TIMEOUT=10s
EOF
    
    # Set proper permissions
    chown root:monitoring-agent "$CONFIG_FILE" 2>/dev/null || chown root:root "$CONFIG_FILE"
    chmod 640 "$CONFIG_FILE"
    
    log_success "Configuration written successfully"
}

# Start and enable service
start_service() {
    log_info "Starting monitoring agent service..."
    
    # Reload systemd
    systemctl daemon-reload
    
    # Enable service
    systemctl enable monitoring-agent
    log_success "Service enabled for auto-start"
    
    # Start service
    if systemctl start monitoring-agent; then
        log_success "Service started successfully"
    else
        log_error "Failed to start service"
        log_info "Check logs with: journalctl -u monitoring-agent -f"
        return 1
    fi
    
    # Check service status
    sleep 2
    if systemctl is-active --quiet monitoring-agent; then
        log_success "Service is running"
    else
        log_warning "Service may have issues, checking status..."
        systemctl status monitoring-agent --no-pager
        return 1
    fi
}

# Test installation
test_installation() {
    log_info "Testing installation..."
    
    # Test health endpoint
    local health_port=${HEALTH_CHECK_PORT:-8081}
    log_info "Testing health endpoint at http://localhost:$health_port/health"
    
    # Wait a moment for service to fully start
    sleep 3
    
    if curl -s "http://localhost:$health_port/health" > /dev/null; then
        log_success "Health endpoint is responding"
    else
        log_warning "Health endpoint not responding yet (service may still be starting)"
    fi
    
    # Show recent logs
    log_info "Recent service logs:"
    journalctl -u monitoring-agent --no-pager -n 5
}

# Show post-installation information
show_post_install_info() {
    echo
    echo "============================================="
    echo "  Installation Complete!"
    echo "============================================="
    echo
    log_success "CheckCle Monitoring Agent installed and configured successfully"
    echo
    echo "System Information:"
    echo "  Architecture: $ARCH ($PACKAGE_ARCH)"
    echo "  Package: $PACKAGE_FILENAME"
    echo "  Package Manager: $PACKAGE_MANAGER"
    if [[ "$PACKAGE_FORMAT" == "tar.gz" ]]; then
        echo "  Installation Type: Manual (tar.gz)"
    else
        echo "  Installation Type: Native package ($PACKAGE_FORMAT)"
    fi
    echo
    echo "Configuration: $CONFIG_FILE"
    echo "Service status: systemctl status monitoring-agent"
    echo "Service logs: journalctl -u monitoring-agent -f"
    echo "Health check: curl http://localhost:${HEALTH_CHECK_PORT:-8081}/health"
    echo
    echo "The monitoring agent is now running and will appear in your dashboard."
    echo
}

# Clean up temporary files
cleanup() {
    if [[ -n "$DOWNLOADED_PACKAGE" && -f "$DOWNLOADED_PACKAGE" ]]; then
        rm -f "$DOWNLOADED_PACKAGE"
        rm -rf "$(dirname "$DOWNLOADED_PACKAGE")"
    fi
}

# Main installation function
main() {
    echo "============================================="
    echo "  CheckCle Server Monitoring Agent"
    echo "  One-Click Installation"
    echo "============================================="
    echo
    
    check_root
    validate_environment
    detect_system
    detect_system_info
    
    log_info "Starting automated installation..."
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Download and install package
    download_package
    install_package
    
    # Configure and start service
    configure_agent
    
    if start_service; then
        test_installation
        show_post_install_info
    else
        log_error "Service failed to start properly"
        log_info "Check configuration: sudo nano $CONFIG_FILE"
        log_info "Restart service: sudo systemctl restart monitoring-agent"
        exit 1
    fi
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "CheckCle Server Monitoring Agent - One-Click Installer"
        echo
        echo "Usage: SERVER_TOKEN=your-token [OPTIONS] sudo bash $0"
        echo "   or: sudo SERVER_TOKEN=your-token [OPTIONS] bash $0"
        echo "   or: curl -L script-url | SERVER_TOKEN=your-token [OPTIONS] sudo bash"
        echo
        echo "System Requirements:"
        echo "  - Linux with systemd"
        echo "  - Supported architectures: x86_64 (amd64), aarch64/arm64"
        echo "  - Supported distributions: Debian/Ubuntu (.deb), RHEL/CentOS/Fedora (.rpm), or any Linux (tar.gz fallback)"
        echo
        echo "Package Installation Logic:"
        echo "  1. Detects OS and tries native package format first (.deb for Debian-based, .rpm for RHEL-based)"
        echo "  2. If native package is not available, automatically falls back to tar.gz"
        echo "  3. tar.gz works on any Linux distribution with systemd"
        echo
        echo "Required Environment Variables:"
        echo "  SERVER_TOKEN          Server authentication token"
        echo
        echo "Optional Environment Variables:"
        echo "  POCKETBASE_URL        PocketBase URL (default: http://localhost:8090)"
        echo "  SERVER_NAME           Server name (default: hostname)"
        echo "  AGENT_ID              Agent identifier (default: monitoring-agent-hostname)"
        echo "  HEALTH_CHECK_PORT     Health check port (default: 8081)"
        echo
        echo "Examples:"
        echo "  SERVER_TOKEN=abc123 sudo -E bash $0"
        echo "  sudo SERVER_TOKEN=abc123 POCKETBASE_URL=https://pb.example.com bash $0"
        echo
        echo "Options:"
        echo "  --help, -h            Show this help message"
        echo "  --uninstall           Uninstall the monitoring agent"
        echo
        exit 0
        ;;
    --uninstall)
        check_root
        log_info "Uninstalling monitoring agent..."
        systemctl stop monitoring-agent 2>/dev/null || true
        systemctl disable monitoring-agent 2>/dev/null || true
        
        # Remove based on detected package manager
        if command -v dpkg >/dev/null 2>&1 && dpkg -l monitoring-agent >/dev/null 2>&1; then
            dpkg -r monitoring-agent 2>/dev/null || true
            log_success "DEB package removed"
        elif command -v rpm >/dev/null 2>&1 && rpm -q monitoring-agent >/dev/null 2>&1; then
            rpm -e monitoring-agent 2>/dev/null || true
            log_success "RPM package removed"
        else
            # Manual removal for tar.gz installations
            log_info "Performing manual cleanup for tar.gz installation..."
            rm -f /usr/bin/monitoring-agent 2>/dev/null || true
            rm -rf /etc/monitoring-agent 2>/dev/null || true
            rm -f /lib/systemd/system/monitoring-agent.service 2>/dev/null || true
            rm -f /usr/lib/systemd/system/monitoring-agent.service 2>/dev/null || true
            rm -f /etc/systemd/system/monitoring-agent.service 2>/dev/null || true
            userdel monitoring-agent 2>/dev/null || true
            groupdel monitoring-agent 2>/dev/null || true
            rm -rf /var/lib/monitoring-agent 2>/dev/null || true
            rm -rf /var/log/monitoring-agent 2>/dev/null || true
            log_success "Manual cleanup completed"
        fi
        
        systemctl daemon-reload 2>/dev/null || true
        log_success "Monitoring agent uninstalled"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac