
#!/bin/bash

# Build script for monitoring-agent packages

set -e

echo "Building Monitoring Agent packages..."

# Check if required tools are installed
command -v dpkg-deb >/dev/null 2>&1 || {
    echo "Warning: dpkg-deb not found. DEB package creation will be skipped."
    echo "Install with: sudo apt-get install dpkg-dev"
    DEB_AVAILABLE=false
}
DEB_AVAILABLE=${DEB_AVAILABLE:-true}

command -v rpmbuild >/dev/null 2>&1 || {
    echo "Warning: rpmbuild not found. RPM package creation will be skipped."
    echo "Install with: sudo apt-get install rpm (Ubuntu/Debian) or sudo yum install rpm-build (CentOS/RHEL)"
    RPM_AVAILABLE=false
}
RPM_AVAILABLE=${RPM_AVAILABLE:-true}

command -v go >/dev/null 2>&1 || {
    echo "Error: Go is required but not installed."
    exit 1
}

# Parse command line arguments
PACKAGE_TYPE=${1:-"all"}
ARCH=${2:-"all"}

show_usage() {
    echo "Usage: $0 [PACKAGE_TYPE] [ARCHITECTURE]"
    echo ""
    echo "PACKAGE_TYPE:"
    echo "  deb  - Build only .deb packages"
    echo "  rpm  - Build only .rpm packages"
    echo "  all  - Build all available packages (default)"
    echo ""
    echo "ARCHITECTURE:"
    echo "  amd64 - Build only for AMD64"
    echo "  arm64 - Build only for ARM64"
    echo "  all   - Build for both architectures (default)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Build all packages for all architectures"
    echo "  $0 deb                # Build DEB packages for all architectures"
    echo "  $0 rpm arm64          # Build RPM package for ARM64 only"
    echo "  $0 all amd64          # Build all packages for AMD64 only"
}

build_packages() {
    local pkg_type=$1
    local arch=$2
    
    echo "Building $pkg_type packages for $arch architecture..."
    
    case $pkg_type in
        deb)
            if [ "$DEB_AVAILABLE" = true ]; then
                if [ "$arch" = "all" ]; then
                    make deb-all
                else
                    make deb-$arch
                fi
            else
                echo "Error: DEB packaging tools not available"
                return 1
            fi
            ;;
        rpm)
            if [ "$RPM_AVAILABLE" = true ]; then
                if [ "$arch" = "all" ]; then
                    make rpm-all
                else
                    make rpm-$arch
                fi
            else
                echo "Error: RPM packaging tools not available"
                return 1
            fi
            ;;
        all)
            if [ "$arch" = "all" ]; then
                [ "$DEB_AVAILABLE" = true ] && make deb-all
                [ "$RPM_AVAILABLE" = true ] && make rpm-all
            else
                [ "$DEB_AVAILABLE" = true ] && make deb-$arch
                [ "$RPM_AVAILABLE" = true ] && make rpm-$arch
            fi
            ;;
    esac
}

# Validate arguments
case $PACKAGE_TYPE in
    deb|rpm|all)
        ;;
    --help|-h)
        show_usage
        exit 0
        ;;
    *)
        echo "Error: Invalid package type '$PACKAGE_TYPE'"
        echo ""
        show_usage
        exit 1
        ;;
esac

case $ARCH in
    amd64|arm64|all)
        ;;
    *)
        echo "Error: Invalid architecture '$ARCH'"
        echo ""
        show_usage
        exit 1
        ;;
esac

# Clean previous builds
echo "Cleaning previous builds..."
make clean

# Generate protobuf files if protoc is available
if command -v protoc >/dev/null 2>&1; then
    echo "Generating protobuf files..."
    protoc --go_out=. --go-grpc_out=. proto/monitoring.proto 2>/dev/null || {
        echo "Warning: Failed to generate protobuf files. Using existing files."
    }
else
    echo "Warning: protoc not found. Using existing protobuf files."
fi

# Build packages
build_packages "$PACKAGE_TYPE" "$ARCH"

echo ""
echo "✅ Build complete!"
echo ""
echo "📦 Generated packages:"
echo ""

# Show DEB packages
if ls dist/*.deb >/dev/null 2>&1; then
    echo "📦 DEB Packages:"
    ls -la dist/*.deb | awk '{print "  " $9 " (" $5 " bytes)"}'
    echo ""
fi

# Show RPM packages
if ls dist/*.rpm >/dev/null 2>&1; then
    echo "📦 RPM Packages:"
    ls -la dist/*.rpm | awk '{print "  " $9 " (" $5 " bytes)"}'
    echo ""
fi

# Installation instructions
echo "🚀 Installation Instructions:"
echo ""

if ls dist/*.deb >/dev/null 2>&1; then
    echo "📋 DEB Package Installation:"
    for deb in dist/*.deb; do
        echo "  sudo dpkg -i $deb"
    done
    echo "  sudo apt-get install -f  # Install dependencies if needed"
    echo ""
fi

if ls dist/*.rpm >/dev/null 2>&1; then
    echo "📋 RPM Package Installation:"
    for rpm in dist/*.rpm; do
        echo "  sudo rpm -ivh $rpm"
        echo "  # or: sudo yum localinstall $rpm"
    done
    echo ""
fi

echo "⚙️  Configuration (REQUIRED before starting):"
echo "  sudo nano /etc/monitoring-agent/monitoring-agent.env"
echo ""
echo "🔧 Service Management:"
echo "  sudo systemctl enable monitoring-agent"
echo "  sudo systemctl start monitoring-agent"
echo "  sudo systemctl status monitoring-agent"
echo ""
echo "🩺 Health Check:"
echo "  curl http://localhost:8081/health"