
Name:           monitoring-agent
Version:        %{_version}
Release:        %{_release}%{?dist}
Summary:        Go-based monitoring agent with gRPC and PocketBase support
License:        MIT
URL:            https://github.com/operacle/server-monitoring-agent
Source0:        %{name}-%{version}.tar.gz
BuildArch:      %{_arch}

BuildRequires:  systemd-rpm-macros
Requires:       systemd
Requires(pre):  shadow-utils
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

%description
A comprehensive monitoring agent that collects system metrics (CPU, Memory, 
Disk, Network) and sends them to remote servers via multiple protocols including
gRPC, PocketBase, and HTTP REST API. Features remote control capabilities and
health check endpoints.

Key features:
- System metrics collection
- Multiple communication protocols (gRPC, PocketBase, HTTP)
- Remote control commands (start/stop monitoring)
- Health check endpoints
- Configurable via environment variables

%prep
%setup -q

%build
# Binary is already built and included in the source

%install
rm -rf $RPM_BUILD_ROOT

# Create directories
mkdir -p $RPM_BUILD_ROOT%{_bindir}
mkdir -p $RPM_BUILD_ROOT%{_sysconfdir}/monitoring-agent
mkdir -p $RPM_BUILD_ROOT%{_unitdir}
mkdir -p $RPM_BUILD_ROOT%{_sharedstatedir}/monitoring-agent
mkdir -p $RPM_BUILD_ROOT%{_localstatedir}/log/monitoring-agent

# Install binary
install -m 0755 monitoring-agent $RPM_BUILD_ROOT%{_bindir}/monitoring-agent

# Install configuration
install -m 0640 monitoring-agent.env $RPM_BUILD_ROOT%{_sysconfdir}/monitoring-agent/monitoring-agent.conf

# Install systemd service
install -m 0644 monitoring-agent.service $RPM_BUILD_ROOT%{_unitdir}/monitoring-agent.service

%pre
# Create monitoring-agent user and group
getent group monitoring-agent >/dev/null || groupadd -r monitoring-agent
getent passwd monitoring-agent >/dev/null || \
    useradd -r -g monitoring-agent -d %{_sharedstatedir}/monitoring-agent \
    -s /sbin/nologin -c "Monitoring Agent" monitoring-agent

%post
# Create environment file if it doesn't exist
if [ ! -f %{_sysconfdir}/monitoring-agent/monitoring-agent.env ]; then
    cp %{_sysconfdir}/monitoring-agent/monitoring-agent.conf \
       %{_sysconfdir}/monitoring-agent/monitoring-agent.env
fi

# Set ownership and permissions
chown monitoring-agent:monitoring-agent %{_sharedstatedir}/monitoring-agent
chown monitoring-agent:monitoring-agent %{_localstatedir}/log/monitoring-agent
chmod 755 %{_sharedstatedir}/monitoring-agent
chmod 755 %{_localstatedir}/log/monitoring-agent

# Set permissions for config files
chown root:monitoring-agent %{_sysconfdir}/monitoring-agent/monitoring-agent.env
chmod 640 %{_sysconfdir}/monitoring-agent/monitoring-agent.env

%systemd_post monitoring-agent.service

echo "Server Monitoring Agent installed successfully!"
echo ""
echo "⚠️  IMPORTANT: Server configuration required!"
echo ""
echo "Before starting the service, you must configure the server settings:"
echo "1. Edit %{_sysconfdir}/monitoring-agent/monitoring-agent.env"
echo "2. Set the following variables:"
echo "   - SERVER_NAME=your-server-name"
echo "   - HOSTNAME=your-hostname"  
echo "   - IP_ADDRESS=your-server-ip"
echo "   - OS_TYPE=your-os-type"
echo "   - SERVER_TOKEN=your-unique-token"
echo "   - POCKETBASE_URL=your-pocketbase-url"
echo ""
echo "To start the service after configuration:"
echo "   sudo systemctl enable monitoring-agent"
echo "   sudo systemctl start monitoring-agent"

%preun
%systemd_preun monitoring-agent.service

%postun
%systemd_postun_with_restart monitoring-agent.service

if [ $1 -eq 0 ]; then
    # Complete removal
    userdel monitoring-agent >/dev/null 2>&1 || :
    groupdel monitoring-agent >/dev/null 2>&1 || :
    rm -rf %{_sharedstatedir}/monitoring-agent
    rm -rf %{_localstatedir}/log/monitoring-agent
    rm -rf %{_sysconfdir}/monitoring-agent
fi

%files
%{_bindir}/monitoring-agent
%{_unitdir}/monitoring-agent.service
%config(noreplace) %{_sysconfdir}/monitoring-agent/monitoring-agent.conf
%attr(755,monitoring-agent,monitoring-agent) %dir %{_sharedstatedir}/monitoring-agent
%attr(755,monitoring-agent,monitoring-agent) %dir %{_localstatedir}/log/monitoring-agent

%changelog
* Thu Jul 17 2025 Tola Leng <hello@checkcle.com> - 1.0.0-1
- Initial RPM package with multi-architecture support
- System metrics collection support
- Multiple communication protocols (gRPC, PocketBase, HTTP)
- Remote control capabilities
- Health check endpoints
- Support for AMD64 and ARM64 architectures