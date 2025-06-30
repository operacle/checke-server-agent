
#!/bin/bash

# Build script for monitoring-agent .deb package

set -e

echo "Building Monitoring Agent .deb package..."

# Check if required tools are installed
command -v dpkg-deb >/dev/null 2>&1 || {
    echo "Error: dpkg-deb is required but not installed."
    echo "Install with: sudo apt-get install dpkg-dev"
    exit 1
}

command -v go >/dev/null 2>&1 || {
    echo "Error: Go is required but not installed."
    exit 1
}

# Clean previous builds
echo "Cleaning previous builds..."
make clean

# Generate protobuf files if protoc is available
if command -v protoc >/dev/null 2>&1; then
    echo "Generating protobuf files..."
    protoc --go_out=. --go-grpc_out=. proto/monitoring.proto || {
        echo "Warning: Failed to generate protobuf files. Using placeholder."
    }
else
    echo "Warning: protoc not found. Using placeholder protobuf files."
fi

# Build the .deb package
echo "Building .deb package..."
make deb

echo ""
echo "Build complete!"
echo ""
echo "To install the package:"
echo "  sudo dpkg -i build/monitoring-agent_1.0.0_amd64.deb"
echo ""
echo "To install dependencies if needed:"
echo "  sudo apt-get install -f"
echo ""
echo "To configure:"
echo "  sudo nano /etc/monitoring-agent/monitoring-agent.env"
echo ""
echo "To start:"
echo "  sudo systemctl enable monitoring-agent"
echo "  sudo systemctl start monitoring-agent"
