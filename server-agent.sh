
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

# Detect system architecture and package format
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
    
    # Detect package format preference
    if command -v dpkg >/dev/null 2>&1; then
        PACKAGE_FORMAT="deb"
        PACKAGE_MANAGER="dpkg"
    elif command -v rpm >/dev/null 2>&1; then
        PACKAGE_FORMAT="rpm"
        if command -v yum >/dev/null 2>&1; then
            PACKAGE_MANAGER="yum"
        elif command -v dnf >/dev/null 2>&1; then
            PACKAGE_MANAGER="dnf"
        else
            PACKAGE_MANAGER="rpm"
        fi
    else
        log_error "No supported package manager found (dpkg or rpm required)"
        log_info "This script supports Debian/Ubuntu (.deb) and RHEL/CentOS/Fedora (.rpm) systems"
        exit 1
    fi
    
    # Construct package filename and URL
    PACKAGE_FILENAME="monitoring-agent_1.0.0_${PACKAGE_ARCH}.${PACKAGE_FORMAT}"
    PACKAGE_URL="${GITHUB_BASE_URL}/${PACKAGE_FILENAME}"
    
    log_success "System detection complete:"
    log_info "  Architecture: $ARCH -> $PACKAGE_ARCH"
    log_info "  Package format: $PACKAGE_FORMAT"
    log_info "  Package manager: $PACKAGE_MANAGER"
    log_info "  Package URL: $PACKAGE_URL"
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

# Download package based on detected system
download_package() {
    local temp_dir="/tmp/monitoring-agent-install"
    mkdir -p "$temp_dir"
    
    log_info "Downloading monitoring agent package..."
    log_info "URL: $PACKAGE_URL"
    
    if curl -L -f -o "$temp_dir/$PACKAGE_FILENAME" "$PACKAGE_URL"; then
        DOWNLOADED_PACKAGE="$temp_dir/$PACKAGE_FILENAME"
        log_success "Package downloaded successfully: $PACKAGE_FILENAME"
    else
        log_error "Failed to download package from: $PACKAGE_URL"
        log_info "Please check:"
        log_info "  1. Internet connectivity"
        log_info "  2. Package availability for your architecture ($PACKAGE_ARCH)"
        log_info "  3. GitHub repository access"
        exit 1
    fi
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
        echo "  - Supported distributions: Debian/Ubuntu (.deb), RHEL/CentOS/Fedora (.rpm)"
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
        if command -v dpkg >/dev/null 2>&1; then
            dpkg -r monitoring-agent 2>/dev/null || true
        elif command -v rpm >/dev/null 2>&1; then
            rpm -e monitoring-agent 2>/dev/null || true
        fi
        
        rm -rf /etc/monitoring-agent 2>/dev/null || true
        log_success "Monitoring agent uninstalled"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac