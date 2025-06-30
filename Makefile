
# Server Monitoring Agent Makefile

# Build variables
BINARY_NAME=monitoring-agent
PACKAGE_NAME=monitoring-agent
VERSION=1.0.0
BUILD_DIR=build
DIST_DIR=dist

# Go build variables
GOOS=linux
GOARCH=amd64
CGO_ENABLED=0

.PHONY: all build clean test deps package install deb

all: clean deps build

# Install dependencies
deps:
	go mod tidy
	go mod download

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

# Package for distribution (alias for deb target)
package: deb

# Build .deb package
deb: build
	@echo "Creating .deb package..."
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/usr/bin
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/etc/monitoring-agent
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/etc/systemd/system
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN
	
	# Copy binary
	cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/$(PACKAGE_NAME)/usr/bin/
	
	# Copy configuration
	cp packaging/monitoring-agent.conf $(DIST_DIR)/$(PACKAGE_NAME)/etc/monitoring-agent/
	
	# Copy systemd service
	cp packaging/monitoring-agent.service $(DIST_DIR)/$(PACKAGE_NAME)/etc/systemd/system/
	
	# Copy packaging files
	cp packaging/control $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/
	cp packaging/postinst $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/
	cp packaging/prerm $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/
	cp packaging/postrm $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/
	
	# Set permissions
	chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/postinst
	chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/prerm
	chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/DEBIAN/postrm
	chmod +x $(DIST_DIR)/$(PACKAGE_NAME)/usr/bin/$(BINARY_NAME)
	
	# Build .deb package
	dpkg-deb --build $(DIST_DIR)/$(PACKAGE_NAME)
	
	@echo "Package created: $(DIST_DIR)/$(PACKAGE_NAME).deb"

# Install locally for development
install: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

# Run locally with .env
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development mode with auto-restart
dev:
	go run . 