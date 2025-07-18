
VERSION = 1.0.0
NAME = monitoring-agent

# Architecture variables
ARCH ?= amd64
GO_ARCH_MAP_amd64 = amd64
GO_ARCH_MAP_arm64 = arm64
GO_ARCH = $(GO_ARCH_MAP_$(ARCH))

# Build flags
CGO_ENABLED = 0
GOOS = linux
GO_FLAGS = -a -installsuffix cgo -ldflags '-w -s'

all: build

build:
	@echo "Building $(NAME) for $(GOOS)/$(GO_ARCH)..."
	@mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GO_ARCH) go build $(GO_FLAGS) -o bin/$(NAME)-$(ARCH) main.go

build-amd64:
	@$(MAKE) build ARCH=amd64

build-arm64:
	@$(MAKE) build ARCH=arm64

build-all: build-amd64 build-arm64

test:
	@echo "Testing..."
	go test ./...

deb: build
	@echo "Building .deb package for $(ARCH)..."
	@mkdir -p dist/$(NAME)-$(ARCH)/DEBIAN
	@mkdir -p dist/$(NAME)-$(ARCH)/usr/bin
	@mkdir -p dist/$(NAME)-$(ARCH)/etc/systemd/system
	@mkdir -p dist/$(NAME)-$(ARCH)/etc/$(NAME)
	
	# Copy binary
	cp bin/$(NAME)-$(ARCH) dist/$(NAME)-$(ARCH)/usr/bin/$(NAME)
	chmod +x dist/$(NAME)-$(ARCH)/usr/bin/$(NAME)
	
	# Copy systemd service file
	cp packaging/monitoring-agent.service dist/$(NAME)-$(ARCH)/etc/systemd/system/
	
	# Copy configuration files
	cp packaging/monitoring-agent.env dist/$(NAME)-$(ARCH)/etc/$(NAME)/
	cp .env.example dist/$(NAME)-$(ARCH)/etc/$(NAME)/monitoring-agent.env.example
	
	# Copy package control files with architecture substitution
	sed 's/Architecture: amd64/Architecture: $(ARCH)/' packaging/control > dist/$(NAME)-$(ARCH)/DEBIAN/control
	cp packaging/postinst dist/$(NAME)-$(ARCH)/DEBIAN/
	cp packaging/prerm dist/$(NAME)-$(ARCH)/DEBIAN/
	cp packaging/postrm dist/$(NAME)-$(ARCH)/DEBIAN/
	chmod 755 dist/$(NAME)-$(ARCH)/DEBIAN/postinst
	chmod 755 dist/$(NAME)-$(ARCH)/DEBIAN/prerm
	chmod 755 dist/$(NAME)-$(ARCH)/DEBIAN/postrm
	
	# Build .deb package
	dpkg-deb --build dist/$(NAME)-$(ARCH) dist/$(NAME)_$(VERSION)_$(ARCH).deb
	@echo "Package created: dist/$(NAME)_$(VERSION)_$(ARCH).deb"

deb-amd64:
	@$(MAKE) deb ARCH=amd64

deb-arm64:
	@$(MAKE) deb ARCH=arm64

deb-all: deb-amd64 deb-arm64

rpm: build
	@echo "Building .rpm package for $(ARCH)..."
	@mkdir -p rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	@mkdir -p rpmbuild/BUILD/$(NAME)-$(VERSION)-$(ARCH)
	
	# Copy files to build directory
	cp bin/$(NAME)-$(ARCH) rpmbuild/BUILD/$(NAME)-$(VERSION)-$(ARCH)/$(NAME)
	cp packaging/monitoring-agent.service rpmbuild/BUILD/$(NAME)-$(VERSION)-$(ARCH)/
	cp packaging/monitoring-agent.env rpmbuild/BUILD/$(NAME)-$(VERSION)-$(ARCH)/
	cp .env.example rpmbuild/BUILD/$(NAME)-$(VERSION)-$(ARCH)/monitoring-agent.env.example
	
	# Set RPM architecture
	$(eval RPM_ARCH := $(if $(filter arm64,$(ARCH)),aarch64,x86_64))
	
	# Copy and modify spec file for architecture
	sed -e 's/%{_version}/$(VERSION)/g' \
	    -e 's/%{_release}/1/g' \
	    -e 's/BuildArch:.*/BuildArch: $(RPM_ARCH)/' \
	    packaging/monitoring-agent.spec > rpmbuild/SPECS/monitoring-agent-$(ARCH).spec
	
	# Build RPM
	rpmbuild --define "_topdir $(PWD)/rpmbuild" \
	         --define "_arch $(RPM_ARCH)" \
	         --define "_build_arch $(RPM_ARCH)" \
	         -bb rpmbuild/SPECS/monitoring-agent-$(ARCH).spec
	
	# Move RPM to dist directory
	@mkdir -p dist
	cp rpmbuild/RPMS/$(RPM_ARCH)/$(NAME)-$(VERSION)-1.$(RPM_ARCH).rpm dist/$(NAME)_$(VERSION)_$(ARCH).rpm
	@echo "Package created: dist/$(NAME)_$(VERSION)_$(ARCH).rpm"

rpm-amd64:
	@$(MAKE) rpm ARCH=amd64

rpm-arm64:
	@$(MAKE) rpm ARCH=arm64

rpm-all: rpm-amd64 rpm-arm64

package-all: deb-all rpm-all

install: deb
	@echo "Installing $(ARCH) package..."
	sudo dpkg -i dist/$(NAME)_$(VERSION)_$(ARCH).deb

install-deb-amd64:
	@$(MAKE) install ARCH=amd64

install-deb-arm64:
	@$(MAKE) install ARCH=arm64

install-rpm-amd64:
	sudo rpm -ivh dist/$(NAME)_$(VERSION)_amd64.rpm

install-rpm-arm64:
	sudo rpm -ivh dist/$(NAME)_$(VERSION)_arm64.rpm

uninstall:
	@echo "Uninstalling monitoring agent..."
	sudo systemctl stop monitoring-agent || true
	sudo systemctl disable monitoring-agent || true
	sudo rm -f /usr/bin/monitoring-agent
	sudo rm -f /etc/systemd/system/monitoring-agent.service
	sudo rm -rf /etc/monitoring-agent
	sudo systemctl daemon-reload

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf dist/
	rm -rf rpmbuild/

help:
	@echo "Available targets:"
	@echo "  build           - Build binary for current architecture (default: amd64)"
	@echo "  build-amd64     - Build binary for AMD64"
	@echo "  build-arm64     - Build binary for ARM64"
	@echo "  build-all       - Build binaries for both architectures"
	@echo "  deb             - Create .deb package for current architecture"
	@echo "  deb-amd64       - Create .deb package for AMD64"
	@echo "  deb-arm64       - Create .deb package for ARM64"
	@echo "  deb-all         - Create .deb packages for both architectures"
	@echo "  rpm             - Create .rpm package for current architecture"
	@echo "  rpm-amd64       - Create .rpm package for AMD64"
	@echo "  rpm-arm64       - Create .rpm package for ARM64"
	@echo "  rpm-all         - Create .rpm packages for both architectures"
	@echo "  package-all     - Create all packages (DEB and RPM) for both architectures"
	@echo "  install         - Install the .deb package for current architecture"
	@echo "  uninstall       - Uninstall the monitoring agent"
	@echo "  clean           - Clean build artifacts"
	@echo "  test            - Run tests"
	@echo "  help            - Show this help"

.PHONY: all build build-amd64 build-arm64 build-all test deb deb-amd64 deb-arm64 deb-all rpm rpm-amd64 rpm-arm64 rpm-all package-all install install-deb-amd64 install-deb-arm64 install-rpm-amd64 install-rpm-arm64 uninstall clean help